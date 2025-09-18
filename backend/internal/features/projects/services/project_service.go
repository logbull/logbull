package projects_services

import (
	"errors"
	"fmt"
	"time"

	audit_logs "logbull/internal/features/audit_logs"
	projects_dto "logbull/internal/features/projects/dto"
	projects_interfaces "logbull/internal/features/projects/interfaces"
	projects_models "logbull/internal/features/projects/models"
	projects_repositories "logbull/internal/features/projects/repositories"
	users_enums "logbull/internal/features/users/enums"
	users_models "logbull/internal/features/users/models"
	users_services "logbull/internal/features/users/services"
	cache_utils "logbull/internal/util/cache"

	"github.com/google/uuid"
	"golang.org/x/sync/singleflight"
)

type ProjectService struct {
	projectRepository        *projects_repositories.ProjectRepository
	membershipRepository     *projects_repositories.MembershipRepository
	userService              *users_services.UserService
	auditLogService          *audit_logs.AuditLogService
	settingsService          *users_services.SettingsService
	projectDeletionListeners []projects_interfaces.ProjectDeletionListener

	projectCacheUtil *cache_utils.CacheUtil[projects_models.Project]
	singleflight     singleflight.Group // Prevents thundering herd on DB calls
}

func (s *ProjectService) AddProjectDeletionListener(listener projects_interfaces.ProjectDeletionListener) {
	s.projectDeletionListeners = append(s.projectDeletionListeners, listener)
}

func (s *ProjectService) CreateProject(
	request *projects_dto.CreateProjectRequestDTO,
	creator *users_models.User,
) (*projects_dto.ProjectResponseDTO, error) {
	settings, err := s.settingsService.GetSettings()

	if err != nil {
		return nil, fmt.Errorf("failed to get settings: %w", err)
	}

	if !creator.CanCreateProjects(settings) {
		return nil, errors.New("insufficient permissions to create projects")
	}

	project := &projects_models.Project{
		ID:                 uuid.New(),
		Name:               request.Name,
		IsApiKeyRequired:   false,
		IsFilterByDomain:   false,
		IsFilterByIP:       false,
		AllowedDomainsRaw:  "",
		AllowedIPsRaw:      "",
		LogsPerSecondLimit: 1000,
		MaxLogsAmount:      100_000_000,
		MaxLogsSizeMB:      100_000, // 100 GB
		MaxLogsLifeDays:    180,
		MaxLogSizeKB:       64,
		CreatedAt:          time.Now().UTC(),
	}

	if err := s.projectRepository.CreateProject(project); err != nil {
		return nil, fmt.Errorf("failed to create project: %w", err)
	}

	// Pre-warm cache with new project for immediate availability
	s.projectCacheUtil.Set(project.ID.String(), project)

	membership := &projects_models.ProjectMembership{
		UserID:    creator.ID,
		ProjectID: project.ID,
		Role:      users_enums.ProjectRoleOwner,
		CreatedAt: time.Now().UTC(),
	}

	if err := s.membershipRepository.CreateMembership(membership); err != nil {
		return nil, fmt.Errorf("failed to create project membership: %w", err)
	}

	s.auditLogService.WriteAuditLog(
		fmt.Sprintf("Project created: %s", project.Name),
		&creator.ID,
		&project.ID,
	)

	ownerRole := users_enums.ProjectRoleOwner
	return &projects_dto.ProjectResponseDTO{
		ID:        project.ID,
		Name:      project.Name,
		CreatedAt: project.CreatedAt,
		UserRole:  &ownerRole,
	}, nil
}

func (s *ProjectService) GetProject(projectID uuid.UUID, user *users_models.User) (*projects_models.Project, error) {
	isCanAccess, _, err := s.CanUserAccessProject(projectID, user)

	if err != nil {
		return nil, err
	}
	if !isCanAccess {
		return nil, errors.New("insufficient permissions to view project")
	}

	return s.projectRepository.GetProjectByID(projectID)
}

func (s *ProjectService) GetUserProjects(user *users_models.User) (*projects_dto.ListProjectsResponseDTO, error) {
	projects, err := s.membershipRepository.GetProjectsWithRolesByUserID(user.Role, user.ID)
	if err != nil {
		return nil, fmt.Errorf("failed to get user projects: %w", err)
	}

	return &projects_dto.ListProjectsResponseDTO{
		Projects: projects,
	}, nil
}

func (s *ProjectService) UpdateProject(
	projectID uuid.UUID,
	project *projects_models.Project,
	user *users_models.User,
) (*projects_models.Project, error) {
	canManage, err := s.CanUserManageProject(projectID, user)

	if err != nil {
		return nil, err
	}
	if !canManage {
		return nil, errors.New("insufficient permissions to update project")
	}

	existingProject, err := s.projectRepository.GetProjectByID(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project: %w", err)
	}

	project.ID = projectID
	project.CreatedAt = existingProject.CreatedAt

	if err := s.projectRepository.UpdateProject(project); err != nil {
		return nil, fmt.Errorf("failed to update project: %w", err)
	}

	s.projectCacheUtil.Invalidate(projectID.String())

	s.auditLogService.WriteAuditLog(
		fmt.Sprintf("Project updated: %s", project.Name),
		&user.ID,
		&projectID,
	)

	return project, nil
}

func (s *ProjectService) DeleteProject(projectID uuid.UUID, user *users_models.User) error {
	if user.Role != users_enums.UserRoleAdmin {
		userProjectRole, err := s.GetUserProjectRole(projectID, user.ID)
		if err != nil {
			return fmt.Errorf("failed to get user role: %w", err)
		}

		if userProjectRole == nil || *userProjectRole != users_enums.ProjectRoleOwner {
			return errors.New("only project owner or admin can delete project")
		}
	}

	project, err := s.projectRepository.GetProjectByID(projectID)
	if err != nil {
		return fmt.Errorf("failed to get project: %w", err)
	}

	for _, listener := range s.projectDeletionListeners {
		if err := listener.OnBeforeProjectDeletion(projectID); err != nil {
			return fmt.Errorf("failed to delete project: %w", err)
		}
	}

	if err := s.projectRepository.DeleteProject(projectID); err != nil {
		return fmt.Errorf("failed to delete project: %w", err)
	}

	s.projectCacheUtil.Invalidate(projectID.String())

	s.auditLogService.WriteAuditLog(
		fmt.Sprintf("Project deleted: %s", project.Name),
		&user.ID,
		&projectID,
	)

	return nil
}

func (s *ProjectService) GetUserProjectRole(projectID uuid.UUID, userID uuid.UUID) (*users_enums.ProjectRole, error) {
	return s.membershipRepository.GetUserProjectRole(projectID, userID)
}

func (s *ProjectService) CanUserAccessProject(
	projectID uuid.UUID,
	user *users_models.User,
) (bool, *users_enums.ProjectRole, error) {
	if user.Role == users_enums.UserRoleAdmin {
		adminRole := users_enums.ProjectRoleOwner
		return true, &adminRole, nil
	}

	role, err := s.membershipRepository.GetUserProjectRole(projectID, user.ID)
	if err != nil {
		return false, nil, nil
	}

	return role != nil, role, nil
}

func (s *ProjectService) CanUserManageProject(projectID uuid.UUID, user *users_models.User) (bool, error) {
	if user.Role == users_enums.UserRoleAdmin {
		return true, nil
	}

	role, err := s.membershipRepository.GetUserProjectRole(projectID, user.ID)
	if err != nil {
		return false, err
	}

	if role == nil {
		return false, nil
	}

	return *role == users_enums.ProjectRoleOwner || *role == users_enums.ProjectRoleAdmin, nil
}

func (s *ProjectService) GetProjectAuditLogs(
	projectID uuid.UUID,
	user *users_models.User,
	request *audit_logs.GetAuditLogsRequest,
) (*audit_logs.GetAuditLogsResponse, error) {
	isCanAccess, _, err := s.CanUserAccessProject(projectID, user)
	if err != nil {
		return nil, err
	}
	if !isCanAccess {
		return nil, errors.New("insufficient permissions to view project audit logs")
	}

	return s.auditLogService.GetProjectAuditLogs(projectID, request)
}

func (s *ProjectService) GetProjectWithCache(projectID uuid.UUID) (*projects_models.Project, error) {
	projectIDStr := projectID.String()

	// Tier 1: Check  cache
	if cachedProject := s.projectCacheUtil.Get(projectIDStr); cachedProject != nil {
		if cachedProject.IsNotExists {
			return nil, errors.New("project not found")
		}

		return cachedProject, nil
	}

	// Tier 2: Database lookup with singleflight protection (prevents thundering herd)
	result, err, _ := s.singleflight.Do(projectIDStr, func() (any, error) {
		return s.projectRepository.GetProjectByID(projectID)
	})

	if err != nil {
		// Cache the invalid project to prevent future DB hits
		invalidCachedProject := &projects_models.Project{
			ID:          projectID,
			IsNotExists: true,
		}
		s.projectCacheUtil.Set(projectIDStr, invalidCachedProject)
		return nil, errors.New("project not found")
	}

	project, ok := result.(*projects_models.Project)
	if !ok {
		return nil, fmt.Errorf("failed to cast result to Project")
	}

	// Cache the valid project
	s.projectCacheUtil.Set(projectIDStr, project)

	return project, nil
}

func (s *ProjectService) GetAllProjects() ([]*projects_models.Project, error) {
	return s.projectRepository.GetAllProjects()
}
