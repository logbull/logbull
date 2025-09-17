package projects_controllers

import (
	"net/http"

	projects_dto "logbull/internal/features/projects/dto"
	projects_services "logbull/internal/features/projects/services"
	users_middleware "logbull/internal/features/users/middleware"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MembershipController struct {
	membershipService *projects_services.MembershipService
}

func (c *MembershipController) RegisterRoutes(router *gin.RouterGroup) {
	projectRoutes := router.Group("/projects/memberships/:id")

	projectRoutes.GET("/members", c.ListMembers)
	projectRoutes.POST("/members", c.AddMember)
	projectRoutes.PUT("/members/:userId/role", c.ChangeMemberRole)
	projectRoutes.DELETE("/members/:userId", c.RemoveMember)
	projectRoutes.POST("/transfer-ownership", c.TransferOwnership)
}

// ListMembers
// @Summary List project members
// @Description Get list of all project members
// @Tags project-membership
// @Produce json
// @Security BearerAuth
// @Param id path string true "Project ID"
// @Success 200 {object} projects_dto.GetMembersResponseDTO
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /projects/memberships/{id}/members [get]
func (c *MembershipController) ListMembers(ctx *gin.Context) {
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

	response, err := c.membershipService.GetMembers(projectID, user)
	if err != nil {
		if err.Error() == "insufficient permissions to view project members" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// AddMember
// @Summary Add member to project (supports both existing and new users)
// @Description Add an existing user to the project or invite a new user if they don't exist
// @Tags project-membership
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Project ID"
// @Param request body projects_dto.AddMemberRequestDTO true "Member addition data"
// @Success 200 {object} projects_dto.AddMemberResponseDTO
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /projects/memberships/{id}/members [post]
func (c *MembershipController) AddMember(ctx *gin.Context) {
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

	var request projects_dto.AddMemberRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	if !request.Role.IsValid() {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid role"})
		return
	}

	response, err := c.membershipService.AddMember(projectID, &request, user)
	if err != nil {
		if err.Error() == "insufficient permissions to manage members" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, response)
}

// ChangeMemberRole
// @Summary Change member role
// @Description Change the role of an existing project member
// @Tags project-membership
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Project ID"
// @Param userId path string true "User ID"
// @Param request body projects_dto.ChangeMemberRoleRequestDTO true "Role change data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /projects/memberships/{id}/members/{userId}/role [put]
func (c *MembershipController) ChangeMemberRole(ctx *gin.Context) {
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

	userIDStr := ctx.Param("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var request projects_dto.ChangeMemberRoleRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	if err := c.membershipService.ChangeMemberRole(projectID, userID, &request, user); err != nil {
		if err.Error() == "insufficient permissions to manage members" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Member role changed successfully"})
}

// RemoveMember
// @Summary Remove member from project
// @Description Remove a member from the project
// @Tags project-membership
// @Security BearerAuth
// @Param id path string true "Project ID"
// @Param userId path string true "User ID"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /projects/memberships/{id}/members/{userId} [delete]
func (c *MembershipController) RemoveMember(ctx *gin.Context) {
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

	userIDStr := ctx.Param("userId")
	userID, err := uuid.Parse(userIDStr)
	if err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	if err := c.membershipService.RemoveMember(projectID, userID, user); err != nil {
		if err.Error() == "insufficient permissions to remove members" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Member removed successfully"})
}

// TransferOwnership
// @Summary Transfer project ownership
// @Description Transfer project ownership to another project admin
// @Tags project-membership
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path string true "Project ID"
// @Param request body projects_dto.TransferOwnershipRequestDTO true "Ownership transfer data"
// @Success 200 {object} map[string]string
// @Failure 400 {object} map[string]string
// @Failure 401 {object} map[string]string
// @Failure 403 {object} map[string]string
// @Router /projects/memberships/{id}/transfer-ownership [post]
func (c *MembershipController) TransferOwnership(ctx *gin.Context) {
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

	var request projects_dto.TransferOwnershipRequestDTO
	if err := ctx.ShouldBindJSON(&request); err != nil {
		ctx.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request format"})
		return
	}

	if err := c.membershipService.TransferOwnership(projectID, &request, user); err != nil {
		if err.Error() == "only project owner or admin can transfer ownership" {
			ctx.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
			return
		}
		ctx.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	ctx.JSON(http.StatusOK, gin.H{"message": "Ownership transferred successfully"})
}
