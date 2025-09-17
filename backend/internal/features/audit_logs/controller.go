package audit_logs

import (
	"net/http"

	user_models "logbull/internal/features/users/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type AuditLogController struct {
	auditLogService *AuditLogService
}

func (c *AuditLogController) RegisterRoutes(router *gin.RouterGroup) {
	// All audit log endpoints require authentication (handled in main.go)
	auditRoutes := router.Group("/audit-logs")

	auditRoutes.GET("/global", c.GetGlobalAuditLogs)
	auditRoutes.GET("/users/:userId", c.GetUserAuditLogs)
}

// GetGlobalAuditLogs
// @Summary Get global audit logs (ADMIN only)
// @Description Retrieve all audit logs across the system
// @Tags audit-logs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param limit query int false "Limit number of results" default(100)
// @Param offset query int false "Offset for pagination" default(0)
// @Param beforeDate query string false "Filter logs created before this date (RFC3339 format)" format(date-time)
// @Success 200 {object} GetAuditLogsResponse
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /audit-logs/global [get]
func (c *AuditLogController) GetGlobalAuditLogs(ctx *gin.Context) {
	user, isOk := ctx.MustGet("user").(*user_models.User)
	if !isOk {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user type in context"})
		return
	}

	request := &GetAuditLogsRequest{}
	if err := ctx.ShouldBindQuery(request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters"})
		return
	}

	response, err := c.auditLogService.GetGlobalAuditLogs(user, request)
	if err != nil {
		if err.Error() == "only administrators can view global audit logs" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve audit logs"})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// GetUserAuditLogs
// @Summary Get user audit logs
// @Description Retrieve audit logs for a specific user
// @Tags audit-logs
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param userId path string true "User ID"
// @Param limit query int false "Limit number of results" default(100)
// @Param offset query int false "Offset for pagination" default(0)
// @Param beforeDate query string false "Filter logs created before this date (RFC3339 format)" format(date-time)
// @Success 200 {object} GetAuditLogsResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /audit-logs/users/{userId} [get]
func (c *AuditLogController) GetUserAuditLogs(ctx *gin.Context) {
	user, isOk := ctx.MustGet("user").(*user_models.User)
	if !isOk {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user type in context"})
		return
	}

	userIDStr := ctx.Param("userId")
	targetUserID, err := uuid.Parse(userIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	request := &GetAuditLogsRequest{}
	if err := ctx.ShouldBindQuery(request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters"})
		return
	}

	response, err := c.auditLogService.GetUserAuditLogs(targetUserID, user, request)
	if err != nil {
		if err.Error() == "insufficient permissions to view user audit logs" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve audit logs"})
		return
	}

	ctx.JSON(http.StatusOK, response)
}
