package logs_receiving

import (
	"encoding/json"
	"fmt"
	"log/slog"
	"net"
	"strings"
	"time"

	api_keys "logbull/internal/features/api_keys"
	logs_core "logbull/internal/features/logs/core"
	projects_models "logbull/internal/features/projects/models"
	projects_services "logbull/internal/features/projects/services"
	rate_limit "logbull/internal/util/rate_limit"

	"github.com/google/uuid"
)

const (
	// Rate limiting
	LogsBurstMultiplier = 5 // 5x base limit for burst handling

	// Batch limits
	MaxBatchSize      = 1000             // Maximum number of logs per batch
	MaxBatchSizeBytes = 10 * 1024 * 1024 // 10MB maximum batch size

	// Individual log limits
	MaxLogSizeFactor = 1024 // Convert KB to bytes
)

type LogReceivingService struct {
	logRepository    *logs_core.LogCoreRepository
	rateLimiter      *rate_limit.RateLimiter
	projectService   *projects_services.ProjectService
	apiKeyService    *api_keys.ApiKeyService
	logWorkerService *LogWorkerService
	logger           *slog.Logger
}

func (s *LogReceivingService) SubmitLogs(
	projectID uuid.UUID,
	request *SubmitLogsRequestDTO,
	clientIP, apiKey, origin string,
) (*SubmitLogsResponseDTO, error) {
	if err := s.validateBasicBatchLimits(request); err != nil {
		return nil, err
	}

	project, err := s.validateBasicProjectConstraints(projectID, origin, clientIP)
	if err != nil {
		return nil, err
	}

	if err := s.validateApiKey(project, apiKey); err != nil {
		return nil, err
	}

	_, err = s.validateRateLimit(project)
	if err != nil {
		return nil, err
	}

	validLogs, errors, totalBatchSize := s.processLogItems(request.Logs, project, projectID, clientIP)

	if err := s.validateTotalBatchSize(totalBatchSize); err != nil {
		return nil, err
	}

	s.queueValidLogs(validLogs, projectID)

	return &SubmitLogsResponseDTO{
		Accepted: len(validLogs),
		Rejected: len(errors),
		Errors:   errors,
	}, nil
}

func (s *LogReceivingService) processLogItems(
	logRequests []LogItemRequestDTO,
	project *projects_models.Project,
	projectID uuid.UUID,
	clientIP string,
) ([]*logs_core.LogItem, []LogSubmissionError, int) {
	var validLogs []*logs_core.LogItem
	var errors []LogSubmissionError
	var totalBatchSize int

	for i, logRequest := range logRequests {
		logSize, err := s.calculateLogSize(&logRequest)

		if err != nil {
			message := fmt.Sprintf("failed to calculate log size: %v", err)
			if validationErr, ok := err.(*logs_core.ValidationError); ok {
				message = validationErr.Code
			}

			errors = append(errors, LogSubmissionError{
				Index:   i,
				Message: message,
			})

			continue
		}

		totalBatchSize += logSize

		if err := s.validateLogItemWithSize(&logRequest, project, logSize); err != nil {
			message := err.Error()
			if validationErr, ok := err.(*logs_core.ValidationError); ok {
				message = validationErr.Code
			}

			errors = append(errors, LogSubmissionError{
				Index:   i,
				Message: message,
			})

			continue
		}

		formattedMessage := s.prettyFormatIfMessageJSON(logRequest.Message)

		logItem := &logs_core.LogItem{
			ID:        uuid.New(),
			ProjectID: projectID,
			Timestamp: time.Now().UTC(),
			Level:     logRequest.Level,
			Message:   formattedMessage,
			Fields:    logRequest.Fields,
			ClientIP:  clientIP,
		}

		validLogs = append(validLogs, logItem)
	}

	return validLogs, errors, totalBatchSize
}

func (s *LogReceivingService) queueValidLogs(validLogs []*logs_core.LogItem, projectID uuid.UUID) {
	if len(validLogs) == 0 {
		return
	}

	// Queue each log individually - they will be accumulated internally and flushed every second
	successCount := 0
	for _, log := range validLogs {
		if err := s.logWorkerService.QueueLog(log); err != nil {
			s.logger.Error("Failed to queue log",
				slog.String("projectId", projectID.String()),
				slog.String("logId", log.ID.String()),
				slog.String("error", err.Error()))
		} else {
			successCount++
		}
	}
}

func (s *LogReceivingService) validateBasicBatchLimits(request *SubmitLogsRequestDTO) error {
	if len(request.Logs) == 0 {
		return &logs_core.ValidationError{
			Code:    logs_core.ErrorBatchTooLarge,
			Message: "batch cannot be empty",
		}
	}

	if len(request.Logs) > MaxBatchSize {
		return &logs_core.ValidationError{
			Code:    logs_core.ErrorBatchTooLarge,
			Message: fmt.Sprintf("batch size cannot exceed %d logs", MaxBatchSize),
		}
	}

	return nil
}

func (s *LogReceivingService) validateBasicProjectConstraints(
	projectID uuid.UUID,
	origin, clientIP string,
) (*projects_models.Project, error) {
	project, err := s.projectService.GetProjectWithCache(projectID)
	if err != nil {
		return nil, &logs_core.ValidationError{
			Code:    logs_core.ErrorProjectNotFound,
			Message: "project not found",
		}
	}

	if err := s.validateDomainFilter(project, origin); err != nil {
		return nil, err
	}

	if err := s.validateIPFilter(project, clientIP); err != nil {
		return nil, err
	}

	return project, nil
}

func (s *LogReceivingService) validateApiKey(project *projects_models.Project, apiKey string) error {
	if !project.IsApiKeyRequired {
		return nil
	}

	if apiKey == "" {
		return &logs_core.ValidationError{
			Code:    logs_core.ErrorAPIKeyRequired,
			Message: "API key required for this project",
		}
	}

	result, err := s.apiKeyService.ValidateApiKey(apiKey, project.ID)
	if err != nil {
		return fmt.Errorf("failed to validate API key: %w", err)
	}

	if !result.IsValid {
		return &logs_core.ValidationError{
			Code:    logs_core.ErrorAPIKeyInvalid,
			Message: "invalid API key",
		}
	}

	return nil
}

func (s *LogReceivingService) validateDomainFilter(project *projects_models.Project, origin string) error {
	if !project.IsFilterByDomain {
		return nil
	}

	if origin == "" {
		return &logs_core.ValidationError{
			Code:    logs_core.ErrorDomainNotAllowed,
			Message: "origin header required for domain filtering",
		}
	}

	for _, allowedDomain := range project.AllowedDomains {
		if s.matchesDomain(origin, allowedDomain) {
			return nil
		}
	}

	return &logs_core.ValidationError{
		Code:    logs_core.ErrorDomainNotAllowed,
		Message: "domain not allowed",
	}
}

func (s *LogReceivingService) validateIPFilter(project *projects_models.Project, clientIP string) error {
	if !project.IsFilterByIP {
		return nil
	}

	if clientIP == "" {
		return &logs_core.ValidationError{
			Code:    logs_core.ErrorIPNotAllowed,
			Message: "client IP required for IP filtering",
		}
	}

	ip := net.ParseIP(clientIP)
	if ip == nil {
		return &logs_core.ValidationError{
			Code:    logs_core.ErrorIPNotAllowed,
			Message: "invalid client IP format",
		}
	}

	for _, allowedIP := range project.AllowedIPs {
		if s.matchesIPOrCIDR(ip, allowedIP) {
			return nil
		}
	}

	return &logs_core.ValidationError{
		Code:    logs_core.ErrorIPNotAllowed,
		Message: "IP address not allowed",
	}
}

func (s *LogReceivingService) validateRateLimit(project *projects_models.Project) (*rate_limit.RateLimitResult, error) {
	// If LogsPerSecondLimit is 0, it means unlimited - skip rate limiting
	if project.LogsPerSecondLimit == 0 {
		return &rate_limit.RateLimitResult{
			Allowed:   true,
			Remaining: 1000, // Arbitrary high number for unlimited
		}, nil
	}

	burstLimit := project.LogsPerSecondLimit * LogsBurstMultiplier

	result, err := s.rateLimiter.CheckRateLimit(project.ID, project.LogsPerSecondLimit, burstLimit)
	if err != nil {
		return nil, fmt.Errorf("rate limit check failed: %w", err)
	}

	if !result.Allowed {
		return nil, &logs_core.ValidationError{
			Code:    logs_core.ErrorRateLimitExceeded,
			Message: fmt.Sprintf("logs per second limit exceeded, retry after %d seconds", result.RetryAfterSec),
		}
	}

	return result, nil
}

func (s *LogReceivingService) validateLogItemWithSize(
	entry *LogItemRequestDTO,
	project *projects_models.Project,
	logSize int,
) error {
	if !entry.Level.IsValid() {
		return &logs_core.ValidationError{
			Code:    logs_core.ErrorInvalidLogLevel,
			Message: "invalid log level",
			Field:   "level",
		}
	}

	if strings.TrimSpace(entry.Message) == "" {
		return &logs_core.ValidationError{
			Code:    logs_core.ErrorMessageEmpty,
			Message: "message cannot be empty",
			Field:   "message",
		}
	}

	maxSizeBytes := project.MaxLogSizeKB * MaxLogSizeFactor
	if logSize > maxSizeBytes {
		return &logs_core.ValidationError{
			Code:    logs_core.ErrorLogTooLarge,
			Message: fmt.Sprintf("log size %d bytes exceeds maximum %d bytes", logSize, maxSizeBytes),
			Field:   "size",
		}
	}

	return nil
}

func (s *LogReceivingService) validateTotalBatchSize(totalBatchSize int) error {
	if totalBatchSize > MaxBatchSizeBytes {
		return &logs_core.ValidationError{
			Code:    logs_core.ErrorBatchTooLarge,
			Message: fmt.Sprintf("batch size %d bytes exceeds maximum %d bytes", totalBatchSize, MaxBatchSizeBytes),
		}
	}

	return nil
}

func (s *LogReceivingService) calculateLogSize(entry *LogItemRequestDTO) (int, error) {
	jsonData, err := json.Marshal(entry)
	if err != nil {
		return 0, err
	}

	return len(jsonData), nil
}

func (s *LogReceivingService) matchesDomain(origin, allowedDomain string) bool {
	origin = strings.ToLower(origin)
	allowedDomain = strings.ToLower(allowedDomain)

	if strings.HasPrefix(allowedDomain, "*.") {
		domain := allowedDomain[2:]
		return strings.HasSuffix(origin, "."+domain) || origin == domain
	}

	return origin == allowedDomain
}

func (s *LogReceivingService) matchesIPOrCIDR(ip net.IP, allowedIP string) bool {
	_, cidr, err := net.ParseCIDR(allowedIP)
	if err == nil {
		return cidr.Contains(ip)
	}

	allowed := net.ParseIP(allowedIP)
	if allowed != nil {
		return ip.Equal(allowed)
	}

	return false
}

func (s *LogReceivingService) prettyFormatIfMessageJSON(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		return message
	}

	var jsonData any
	if err := json.Unmarshal([]byte(message), &jsonData); err != nil {
		return message
	}

	prettyJSON, err := json.MarshalIndent(jsonData, "", "  ")
	if err != nil {
		return message
	}

	return string(prettyJSON)
}
