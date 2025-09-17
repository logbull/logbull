package api_keys

import (
	"net/http"

	users_middleware "logbull/internal/features/users/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ApiKeyController struct {
	apiKeyService *ApiKeyService
}

func (c *ApiKeyController) RegisterRoutes(router *gin.RouterGroup) {
	apiKeyRoutes := router.Group("/projects/api-keys/:projectId")

	apiKeyRoutes.POST("", c.CreateApiKey)
	apiKeyRoutes.GET("", c.GetApiKeys)
	apiKeyRoutes.PUT("/:apiKeyId", c.UpdateApiKey)
	apiKeyRoutes.DELETE("/:apiKeyId", c.DeleteApiKey)
}

// CreateApiKey
// @Summary Create a new API key
// @Description Create a new API key for the project
// @Tags api-keys
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param projectId path string true "Project ID"
// @Param request body CreateApiKeyRequestDTO true "API key creation data"
// @Success 200 {object} ApiKey
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /projects/api-keys/{projectId} [post]
func (c *ApiKeyController) CreateApiKey(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	projectIDStr := ctx.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	var request CreateApiKeyRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	response, err := c.apiKeyService.CreateApiKey(projectID, &request, user)
	if err != nil {
		if err.Error() == "insufficient permissions to create API keys" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// GetApiKeys
// @Summary List project API keys
// @Description Get list of API keys for the project
// @Tags api-keys
// @Produce json
// @Security BearerAuth
// @Param projectId path string true "Project ID"
// @Success 200 {object} GetApiKeysResponseDTO
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /projects/api-keys/{projectId} [get]
func (c *ApiKeyController) GetApiKeys(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	projectIDStr := ctx.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	response, err := c.apiKeyService.GetProjectApiKeys(projectID, user)
	if err != nil {
		if err.Error() == "insufficient permissions to view API keys" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// UpdateApiKey
// @Summary Update API key
// @Description Update API key name or status
// @Tags api-keys
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param projectId path string true "Project ID"
// @Param apiKeyId path string true "API Key ID"
// @Param request body UpdateApiKeyRequestDTO true "API key update data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /projects/api-keys/{projectId}/{apiKeyId} [put]
func (c *ApiKeyController) UpdateApiKey(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	projectIDStr := ctx.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	apiKeyIDStr := ctx.Param("apiKeyId")
	apiKeyID, err := uuid.Parse(apiKeyIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid API key ID"})
		return
	}

	var request UpdateApiKeyRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	if err := c.apiKeyService.UpdateApiKey(projectID, apiKeyID, &request, user); err != nil {
		if err.Error() == "insufficient permissions to update API keys" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "API key updated successfully"})
}

// DeleteApiKey
// @Summary Delete API key
// @Description Delete an API key
// @Tags api-keys
// @Security BearerAuth
// @Param projectId path string true "Project ID"
// @Param apiKeyId path string true "API Key ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /projects/api-keys/{projectId}/{apiKeyId} [delete]
func (c *ApiKeyController) DeleteApiKey(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	projectIDStr := ctx.Param("projectId")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	apiKeyIDStr := ctx.Param("apiKeyId")
	apiKeyID, err := uuid.Parse(apiKeyIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid API key ID"})
		return
	}

	if err := c.apiKeyService.DeleteApiKey(projectID, apiKeyID, user); err != nil {
		if err.Error() == "insufficient permissions to delete API keys" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "API key deleted successfully"})
}
