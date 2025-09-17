package projects_services

import (
	"logbull/internal/cache"
	"logbull/internal/features/audit_logs"
	projects_interfaces "logbull/internal/features/projects/interfaces"
	projects_models "logbull/internal/features/projects/models"
	projects_repositories "logbull/internal/features/projects/repositories"
	users_services "logbull/internal/features/users/services"
	cache_utils "logbull/internal/util/cache"

	"golang.org/x/sync/singleflight"
)

var projectRepository = &projects_repositories.ProjectRepository{}
var membershipRepository = &projects_repositories.MembershipRepository{}

var projectService = &ProjectService{
	projectRepository,
	membershipRepository,
	users_services.GetUserService(),
	audit_logs.GetAuditLogService(),
	users_services.GetSettingsService(),
	[]projects_interfaces.ProjectDeletionListener{},
	cache_utils.NewCacheUtil[projects_models.Project](cache.GetCache(), "lb_project:"),
	singleflight.Group{},
}

var membershipService = &MembershipService{
	membershipRepository,
	projectRepository,
	users_services.GetUserService(),
	audit_logs.GetAuditLogService(),
	projectService,
	users_services.GetSettingsService(),
}

func GetProjectService() *ProjectService {
	return projectService
}

func GetMembershipService() *MembershipService {
	return membershipService
}
