package api_keys

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"

	audit_logs "logbull/internal/features/audit_logs"
	projects_services "logbull/internal/features/projects/services"
	users_models "logbull/internal/features/users/models"
	cache_utils "logbull/internal/util/cache"

	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"
)

type ApiKeyService struct {
	apiKeyRepository *ApiKeyRepository
	projectService   *projects_services.ProjectService
	auditLogService  *audit_logs.AuditLogService

	apiKeyCacheUtil *cache_utils.CacheUtil[CachedApiKey]
	singleflight    singleflight.Group // Prevents thundering herd on DB calls
}

const (
	TokenPrefix = "lb_"
	TokenLength = 32
)

func (s *ApiKeyService) CreateApiKey(
	projectID uuid.UUID,
	request *CreateApiKeyRequestDTO,
	creator *users_models.User,
) (*ApiKey, error) {
	canManage, err := s.projectService.CanUserManageProject(projectID, creator)
	if err != nil {
		return nil, err
	}
	if !canManage {
		return nil, errors.New("insufficient permissions to create API keys")
	}

	fullToken, tokenPrefix, tokenHash, err := s.generateSecureToken()
	if err != nil {
		return nil, fmt.Errorf("failed to generate token: %w", err)
	}

	apiKey := &ApiKey{
		ID:          uuid.New(),
		Name:        request.Name,
		ProjectID:   projectID,
		TokenPrefix: tokenPrefix,
		TokenHash:   tokenHash,
		Status:      ApiKeyStatusActive,
	}

	if err := s.apiKeyRepository.CreateApiKey(apiKey); err != nil {
		return nil, fmt.Errorf("failed to create API key: %w", err)
	}

	// Pre-warm cache with new API key for immediate availability
	cachedKey := &CachedApiKey{
		ID:        apiKey.ID,
		ProjectID: apiKey.ProjectID,
		Status:    apiKey.Status,
	}
	s.apiKeyCacheUtil.Set(tokenHash, cachedKey)

	s.auditLogService.WriteAuditLog(
		fmt.Sprintf("API key created: %s (%s)", request.Name, tokenPrefix),
		&creator.ID,
		&projectID,
	)

	// Set the full token in the response (only returned once)
	apiKey.Token = fullToken

	return apiKey, nil
}

func (s *ApiKeyService) GetProjectApiKeys(
	projectID uuid.UUID,
	user *users_models.User,
) (*GetApiKeysResponseDTO, error) {
	canAccess, _, err := s.projectService.CanUserAccessProject(projectID, user)
	if err != nil {
		return nil, err
	}
	if !canAccess {
		return nil, errors.New("insufficient permissions to view API keys")
	}

	apiKeys, err := s.apiKeyRepository.GetApiKeysByProjectID(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get API keys: %w", err)
	}

	return &GetApiKeysResponseDTO{
		ApiKeys: apiKeys,
	}, nil
}

func (s *ApiKeyService) UpdateApiKey(
	projectID uuid.UUID,
	apiKeyID uuid.UUID,
	request *UpdateApiKeyRequestDTO,
	updater *users_models.User,
) error {
	canManage, err := s.projectService.CanUserManageProject(projectID, updater)
	if err != nil {
		return err
	}
	if !canManage {
		return errors.New("insufficient permissions to update API keys")
	}

	apiKey, err := s.apiKeyRepository.GetApiKeyByID(apiKeyID)
	if err != nil {
		return errors.New("API key not found")
	}

	if apiKey.ProjectID != projectID {
		return errors.New("API key does not belong to this project")
	}

	if request.Name != nil {
		apiKey.Name = *request.Name
	}

	if request.Status != nil {
		apiKey.Status = *request.Status
	}

	if err := s.apiKeyRepository.UpdateApiKey(apiKey); err != nil {
		return fmt.Errorf("failed to update API key: %w", err)
	}

	s.apiKeyCacheUtil.Invalidate(apiKey.TokenHash)

	s.auditLogService.WriteAuditLog(
		fmt.Sprintf("API key updated: %s (%s)", apiKey.Name, apiKey.TokenPrefix),
		&updater.ID,
		&projectID,
	)

	return nil
}

func (s *ApiKeyService) DeleteApiKey(
	projectID uuid.UUID,
	apiKeyID uuid.UUID,
	deleter *users_models.User,
) error {
	canManage, err := s.projectService.CanUserManageProject(projectID, deleter)
	if err != nil {
		return err
	}
	if !canManage {
		return errors.New("insufficient permissions to delete API keys")
	}

	apiKey, err := s.apiKeyRepository.GetApiKeyByID(apiKeyID)
	if err != nil {
		return errors.New("API key not found")
	}

	if apiKey.ProjectID != projectID {
		return errors.New("API key does not belong to this project")
	}

	if err := s.apiKeyRepository.DeleteApiKey(apiKeyID); err != nil {
		return fmt.Errorf("failed to delete API key: %w", err)
	}

	s.apiKeyCacheUtil.Invalidate(apiKey.TokenHash)

	s.auditLogService.WriteAuditLog(
		fmt.Sprintf("API key deleted: %s (%s)", apiKey.Name, apiKey.TokenPrefix),
		&deleter.ID,
		&projectID,
	)

	return nil
}

func (s *ApiKeyService) ValidateApiKey(token string, projectID uuid.UUID) (*ValidateTokenResponse, error) {
	if !strings.HasPrefix(token, TokenPrefix) {
		return &ValidateTokenResponse{IsValid: false}, nil
	}

	tokenHash := s.hashToken(token)

	// Tier 1: Check cache
	if cachedKey := s.apiKeyCacheUtil.Get(tokenHash); cachedKey != nil {
		if cachedKey.ProjectID != projectID || cachedKey.Status != ApiKeyStatusActive {
			return &ValidateTokenResponse{IsValid: false}, nil
		}

		return &ValidateTokenResponse{
			IsValid:   true,
			ApiKeyID:  cachedKey.ID,
			ProjectID: cachedKey.ProjectID,
		}, nil
	}

	// Tier 2: Database lookup with singleflight protection (prevents thundering herd)
	result, err, _ := s.singleflight.Do(tokenHash, func() (any, error) {
		return s.apiKeyRepository.GetApiKeyByTokenHash(tokenHash)
	})

	if err != nil {
		// Cache the invalid key to prevent future DB hits
		invalidCachedKey := &CachedApiKey{
			ID:        uuid.Nil,
			ProjectID: uuid.Nil,
			Status:    ApiKeyStatusNotFound,
		}

		s.apiKeyCacheUtil.Set(tokenHash, invalidCachedKey)
		return &ValidateTokenResponse{IsValid: false}, nil
	}

	apiKey, ok := result.(*ApiKey)
	if !ok {
		return &ValidateTokenResponse{IsValid: false}, fmt.Errorf("failed to cast result to ApiKey")
	}

	// Verify project matches and is active
	if apiKey.ProjectID != projectID || apiKey.Status != ApiKeyStatusActive {
		return &ValidateTokenResponse{IsValid: false}, nil
	}

	cachedKey := &CachedApiKey{
		ID:        apiKey.ID,
		ProjectID: apiKey.ProjectID,
		Status:    apiKey.Status,
	}
	s.apiKeyCacheUtil.Set(tokenHash, cachedKey)

	return &ValidateTokenResponse{
		IsValid:   true,
		ApiKeyID:  apiKey.ID,
		ProjectID: apiKey.ProjectID,
	}, nil
}

func (s *ApiKeyService) generateSecureToken() (fullToken, prefix, hash string, err error) {
	// Generate random bytes
	tokenBytes := make([]byte, TokenLength/2) // hex encoding doubles the length
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", "", "", err
	}

	// Create token parts
	tokenSuffix := hex.EncodeToString(tokenBytes)
	fullToken = TokenPrefix + tokenSuffix
	prefix = TokenPrefix + tokenSuffix[:6] + "..."
	hash = s.hashToken(fullToken)

	return fullToken, prefix, hash, nil
}

func (s *ApiKeyService) hashToken(token string) string {
	hasher := sha256.New()
	hasher.Write([]byte(token))
	return hex.EncodeToString(hasher.Sum(nil))
}
