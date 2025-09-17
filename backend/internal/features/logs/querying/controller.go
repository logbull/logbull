package logs_querying

import (
	"net/http"
	"strings"

	logs_core "logbull/internal/features/logs/core"
	users_models "logbull/internal/features/users/models"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type LogQueryController struct {
	logQueryService *LogQueryService
}

func (c *LogQueryController) RegisterRoutes(router *gin.RouterGroup) {
	queryRoutes := router.Group("/logs/query")

	queryRoutes.POST("/execute/:projectId", c.ExecuteQuery)
	queryRoutes.GET("/fields/:projectId", c.GetQueryableFields)
}

// ExecuteQuery
// @Summary Execute log query
// @Description Execute a structured query against project logs. timeRange.to is required for pagination consistency.
// @Tags logs-query
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param projectId path string true "Project ID (UUID format)"
// @Param request body logs_core.LogQueryRequestDTO true "Query request"
// @Success 200 {object} logs_core.LogQueryResponseDTO
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Failure 408 {object} map[string]string
// @Failure 429 {object} map[string]string
// @Router /logs/query/execute/{projectId} [post]
func (c *LogQueryController) ExecuteQuery(ctx *gin.Context) {
	user, isOk := ctx.MustGet("user").(*users_models.User)
	if !isOk {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user type in context"})
		return
	}

	projectIDStr := ctx.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID format"})
		return
	}

	var request logs_core.LogQueryRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	response, err := c.logQueryService.ExecuteQuery(projectID, &request, user)
	if err != nil {
		c.handleError(ctx, err)
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// GetQueryableFields
// @Summary Get available queryable fields
// @Description Get list of fields that can be queried for a project, with optional search query
// @Tags logs-query
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param projectId path string true "Project ID (UUID format)"
// @Param query query string false "Search query to filter field names"
// @Success 200 {object} logs_core.GetQueryableFieldsResponseDTO
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /logs/query/fields/{projectId} [get]
func (c *LogQueryController) GetQueryableFields(ctx *gin.Context) {
	user, isOk := ctx.MustGet("user").(*users_models.User)
	if !isOk {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Invalid user type in context"})
		return
	}

	projectIDStr := ctx.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID format"})
		return
	}

	var request logs_core.GetQueryableFieldsRequestDTO
	if err := ctx.ShouldBindQuery(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters"})
		return
	}

	response, err := c.logQueryService.GetQueryableFields(projectID, &request, user)
	if err != nil {
		if strings.Contains(err.Error(), "insufficient permissions") {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		} else {
			ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get queryable fields"})
		}
		return
	}

	ctx.JSON(http.StatusOK, response)
}

func (c *LogQueryController) handleError(ctx *gin.Context, err error) {
	if validationErr, ok := err.(*ValidationError); ok {
		statusCode := c.getStatusCodeForQueryValidationError(validationErr.Code)
		ctx.JSON(statusCode, gin.H{
			"error": validationErr.Message,
			"code":  validationErr.Code,
		})
		return
	}

	if strings.Contains(err.Error(), "insufficient permissions") {
		ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
		return
	}

	if strings.Contains(err.Error(), "invalid query") {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if strings.Contains(err.Error(), "timeout") || strings.Contains(err.Error(), "context deadline") {
		ctx.JSON(http.StatusRequestTimeout, gin.H{"error": "Query execution timed out"})
		return
	}

	ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to execute query"})
}

func (c *LogQueryController) getStatusCodeForQueryValidationError(errorCode string) int {
	switch errorCode {
	case logs_core.ErrorTooManyConcurrentQueries:
		return http.StatusTooManyRequests
	case logs_core.ErrorInvalidQueryStructure, logs_core.ErrorQueryTooComplex, logs_core.ErrorMissingTimeRangeTo:
		return http.StatusBadRequest
	case logs_core.ErrorQueryTimeout:
		return http.StatusRequestTimeout
	default:
		return http.StatusBadRequest
	}
}
