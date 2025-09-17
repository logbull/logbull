package logs_receiving

import (
	logs_core "logbull/internal/features/logs/core"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ReceivingController struct {
	logReceivingService *LogReceivingService
}

func (c *ReceivingController) RegisterRoutes(router *gin.RouterGroup) {
	// Log ingestion endpoints - no authentication middleware required
	// Authentication is handled via API keys at the service level
	logRoutes := router.Group("/logs/receiving")

	logRoutes.POST("/:projectId", c.SubmitLogs)
}

// SubmitLogs
// @Summary Submit logs to project
// @Description Submit one or more log items to the specified project. Validates project access, API keys (if required), domain/IP filtering (if enabled), rate limits, and individual log requirements.
// @Description
// @Description **Validation Requirements:**
// @Description - Project must exist and be accessible
// @Description - API Key: Required if project has `isApiKeyRequired=true`
// @Description - Domain Filtering: Origin header required if project has `isFilterByDomain=true` with matching allowed domain
// @Description - IP Filtering: Client IP must match allowed IPs/CIDRs if project has `isFilterByIP=true`
// @Description - Rate Limiting: Requests limited by project's `logsPerSecondLimit` with burst capability (5x multiplier)
// @Description - Batch Limits: Maximum 1000 logs per batch, maximum 10MB total batch size
// @Description - Log Requirements: Valid log level (DEBUG/INFO/WARN/ERROR), non-empty message, log size within project's `maxLogSizeKB` limit (timestamp automatically set by server)
// @Tags logs
// @Accept json
// @Produce json
// @Param projectId path string true "Project ID (UUID format)"
// @Param X-API-Key header string false "API Key (required if project has isApiKeyRequired=true)"
// @Param Origin header string false "Origin header (required if project has domain filtering enabled)"
// @Param X-Forwarded-For header string false "Client IP for IP filtering (auto-detected from various headers)"
// @Param request body SubmitLogsRequestDTO true "Log items to submit (1-1000 logs, max 10MB total, timestamp automatically set by server)"
// @Success 202 {object} SubmitLogsResponseDTO "Logs accepted (may include partial rejection for invalid logs)"
// @Failure 400 {object} map[string]string "Invalid request format, project ID, or batch limits exceeded"
// @Failure 401 {object} map[string]string "API key required or invalid"
// @Failure 403 {object} map[string]string "Domain not allowed or IP not allowed"
// @Failure 404 {object} map[string]string "Project not found"
// @Failure 413 {object} map[string]string "Project quota exceeded"
// @Failure 429 {object} map[string]string "Rate limit exceeded"
// @Router /logs/receiving/{projectId} [post]
func (c *ReceivingController) SubmitLogs(ctx *gin.Context) {
	projectIDStr := ctx.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	var request SubmitLogsRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	// Extract headers and client information
	apiKey := ctx.GetHeader("X-API-Key")
	origin := c.extractOrigin(ctx)
	clientIP := c.extractClientIP(ctx)

	response, err := c.logReceivingService.SubmitLogs(projectID, &request, clientIP, apiKey, origin)
	if err != nil {
		c.handleError(ctx, err)
		return
	}

	// Return 202 Accepted for successful log submission
	ctx.JSON(http.StatusAccepted, response)
}

func (c *ReceivingController) extractOrigin(ctx *gin.Context) string {
	// Try Origin header first (CORS requests)
	origin := ctx.GetHeader("Origin")
	if origin != "" {
		// Extract domain from origin URL (remove scheme)
		if after, ok := strings.CutPrefix(origin, "http://"); ok {
			origin = after
		} else if after, ok := strings.CutPrefix(origin, "https://"); ok {
			origin = after
		}

		// Remove port if present
		if idx := strings.Index(origin, ":"); idx != -1 {
			origin = origin[:idx]
		}

		return origin
	}

	// Try Referer header as fallback
	referer := ctx.GetHeader("Referer")
	if referer != "" {
		// Extract domain from referer URL
		if after, ok := strings.CutPrefix(referer, "http://"); ok {
			referer = after
		} else if after, ok := strings.CutPrefix(referer, "https://"); ok {
			referer = after
		}

		// Remove path and query parameters
		if idx := strings.Index(referer, "/"); idx != -1 {
			referer = referer[:idx]
		}
		if idx := strings.Index(referer, "?"); idx != -1 {
			referer = referer[:idx]
		}

		// Remove port if present
		if idx := strings.Index(referer, ":"); idx != -1 {
			referer = referer[:idx]
		}

		return referer
	}

	return ""
}

func (c *ReceivingController) extractClientIP(ctx *gin.Context) string {
	// Check X-Forwarded-For header first (for proxied requests)
	forwarded := ctx.GetHeader("X-Forwarded-For")
	if forwarded != "" {
		// Take the first IP if multiple are present
		if idx := strings.Index(forwarded, ","); idx != -1 {
			return strings.TrimSpace(forwarded[:idx])
		}
		return strings.TrimSpace(forwarded)
	}

	// Check X-Real-IP header
	realIP := ctx.GetHeader("X-Real-IP")
	if realIP != "" {
		return strings.TrimSpace(realIP)
	}

	// Fall back to RemoteAddr
	return ctx.ClientIP()
}

func (c *ReceivingController) handleError(ctx *gin.Context, err error) {
	// Check if it's a validation error
	if validationErr, ok := err.(*logs_core.ValidationError); ok {
		statusCode := c.getStatusCodeForValidationError(validationErr.Code)

		// Set Retry-After header for rate limit errors
		if validationErr.Code == logs_core.ErrorRateLimitExceeded {
			ctx.Header("Retry-After", "60") // Default retry after 60 seconds
		}

		ctx.JSON(statusCode, gin.H{
			"error": validationErr.Message,
			"code":  validationErr.Code,
		})
		return
	}

	// Default to internal server error
	ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to process logs"})
}

func (c *ReceivingController) getStatusCodeForValidationError(errorCode string) int {
	switch errorCode {
	case logs_core.ErrorProjectNotFound:
		return http.StatusNotFound
	case logs_core.ErrorAPIKeyRequired, logs_core.ErrorAPIKeyInvalid:
		return http.StatusUnauthorized
	case logs_core.ErrorDomainNotAllowed, logs_core.ErrorIPNotAllowed:
		return http.StatusForbidden
	case logs_core.ErrorRateLimitExceeded:
		return http.StatusTooManyRequests
	case logs_core.ErrorLogTooLarge, logs_core.ErrorInvalidLogLevel,
		logs_core.ErrorBatchTooLarge, logs_core.ErrorMessageEmpty:
		return http.StatusBadRequest
	case logs_core.ErrorProjectQuotaExceeded:
		return http.StatusRequestEntityTooLarge
	default:
		return http.StatusBadRequest
	}
}
