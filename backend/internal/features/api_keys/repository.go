package api_keys

import (
	"time"

	"logbull/internal/storage"

	"github.com/google/uuid"
)

type ApiKeyRepository struct{}

func (r *ApiKeyRepository) CreateApiKey(apiKey *ApiKey) error {
	if apiKey.ID == uuid.Nil {
		apiKey.ID = uuid.New()
	}

	if apiKey.CreatedAt.IsZero() {
		apiKey.CreatedAt = time.Now().UTC()
	}

	return storage.GetDb().Create(apiKey).Error
}

func (r *ApiKeyRepository) GetApiKeysByProjectID(projectID uuid.UUID) ([]*ApiKey, error) {
	var apiKeys []*ApiKey

	err := storage.GetDb().
		Where("project_id = ?", projectID).
		Order("created_at DESC").
		Find(&apiKeys).Error

	return apiKeys, err
}

func (r *ApiKeyRepository) GetApiKeyByID(apiKeyID uuid.UUID) (*ApiKey, error) {
	var apiKey ApiKey

	err := storage.GetDb().
		Where("id = ?", apiKeyID).
		First(&apiKey).Error

	if err != nil {
		return nil, err
	}

	return &apiKey, nil
}

func (r *ApiKeyRepository) GetApiKeyByTokenHash(tokenHash string) (*ApiKey, error) {
	var apiKey ApiKey

	err := storage.GetDb().
		Where("token_hash = ? AND status = ?", tokenHash, ApiKeyStatusActive).
		First(&apiKey).Error

	if err != nil {
		return nil, err
	}

	return &apiKey, nil
}

func (r *ApiKeyRepository) UpdateApiKey(apiKey *ApiKey) error {
	return storage.GetDb().Save(apiKey).Error
}

func (r *ApiKeyRepository) DeleteApiKey(apiKeyID uuid.UUID) error {
	return storage.GetDb().Delete(&ApiKey{}, apiKeyID).Error
}
