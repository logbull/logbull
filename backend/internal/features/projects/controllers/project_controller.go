package projects_controllers

import (
	"net/http"

	audit_logs "logbull/internal/features/audit_logs"
	projects_dto "logbull/internal/features/projects/dto"
	projects_models "logbull/internal/features/projects/models"
	projects_services "logbull/internal/features/projects/services"
	users_middleware "logbull/internal/features/users/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ProjectController struct {
	projectService *projects_services.ProjectService
}

func (c *ProjectController) RegisterRoutes(router *gin.RouterGroup) {
	projectRoutes := router.Group("/projects")

	projectRoutes.POST("", c.CreateProject)
	projectRoutes.GET("", c.GetProjects)
	projectRoutes.GET("/:id", c.GetProject)
	projectRoutes.PUT("/:id", c.UpdateProject)
	projectRoutes.DELETE("/:id", c.DeleteProject)
	projectRoutes.GET("/:id/audit-logs", c.GetProjectAuditLogs)
}

// CreateProject
// @Summary Create a new project
// @Description Create a new project with default settings
// @Tags projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param request body projects_dto.CreateProjectRequestDTO true "Project creation data"
// @Success 200 {object} projects_dto.ProjectResponseDTO
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /projects [post]
func (c *ProjectController) CreateProject(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	var request projects_dto.CreateProjectRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	response, err := c.projectService.CreateProject(&request, user)
	if err != nil {
		if err.Error() == "insufficient permissions to create projects" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})

		return
	}

	ctx.JSON(http.StatusOK, response)
}

// GetProjects
// @Summary List user's projects
// @Description Get list of projects the user is a member of
// @Tags projects
// @Produce json
// @Security BearerAuth
// @Success 200 {object} projects_dto.ListProjectsResponseDTO
// @Failure 401 {object} map[string]string
// @Router /projects [get]
func (c *ProjectController) GetProjects(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	response, err := c.projectService.GetUserProjects(user)
	if err != nil {
		ctx.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve projects"})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// GetProject
// @Summary Get project details
// @Description Get detailed information about a specific project
// @Tags projects
// @Produce json
// @Security BearerAuth
// @Param id path string true "Project ID"
// @Success 200 {object} projects_models.Project
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /projects/{id} [get]
func (c *ProjectController) GetProject(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	projectIDStr := ctx.Param("id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	project, err := c.projectService.GetProject(projectID, user)
	if err != nil {
		if err.Error() == "insufficient permissions to view project" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, project)
}

// UpdateProject
// @Summary Update project settings
// @Description Update project configuration and settings
// @Tags projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Project ID"
// @Param request body projects_models.Project true "Project update data"
// @Success 200 {object} projects_models.Project
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /projects/{id} [put]
func (c *ProjectController) UpdateProject(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	projectIDStr := ctx.Param("id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	var project projects_models.Project
	if err := ctx.ShouldBindJSON(&project); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	updatedProject, err := c.projectService.UpdateProject(projectID, &project, user)
	if err != nil {
		if err.Error() == "insufficient permissions to update project" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, updatedProject)
}

// DeleteProject
// @Summary Delete project
// @Description Delete a project (owner only)
// @Tags projects
// @Security BearerAuth
// @Param id path string true "Project ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /projects/{id} [delete]
func (c *ProjectController) DeleteProject(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	projectIDStr := ctx.Param("id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	if err := c.projectService.DeleteProject(projectID, user); err != nil {
		if err.Error() == "only project owner or admin can delete project" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Project deleted successfully"})
}

// GetProjectAuditLogs
// @Summary Get project audit logs
// @Description Retrieve audit logs for a specific project (member access required)
// @Tags projects
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Project ID"
// @Param limit query int false "Limit number of results" default(100)
// @Param offset query int false "Offset for pagination" default(0)
// @Param beforeDate query string false "Filter logs created before this date (RFC3339 format)" format(date-time)
// @Success 200 {object} audit_logs.GetAuditLogsResponse
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /projects/{id}/audit-logs [get]
func (c *ProjectController) GetProjectAuditLogs(ctx *gin.Context) {
	user, ok := users_middleware.GetUserFromContext(ctx)
	if !ok {
		ctx.JSON(http.StatusUnauthorized, gin.H{"error": "User not authenticated"})
		return
	}

	projectIDStr := ctx.Param("id")
	projectID, err := uuid.Parse(projectIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid project ID"})
		return
	}

	request := &audit_logs.GetAuditLogsRequest{}
	if err := ctx.ShouldBindQuery(request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid query parameters"})
		return
	}

	response, err := c.projectService.GetProjectAuditLogs(projectID, user, request)
	if err != nil {
		if err.Error() == "insufficient permissions to view project audit logs" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}
