package projects_services

import (
	"errors"
	"fmt"

	audit_logs "logbull/internal/features/audit_logs"
	projects_dto "logbull/internal/features/projects/dto"
	projects_models "logbull/internal/features/projects/models"
	projects_repositories "logbull/internal/features/projects/repositories"
	users_dto "logbull/internal/features/users/dto"
	users_enums "logbull/internal/features/users/enums"
	users_models "logbull/internal/features/users/models"
	users_services "logbull/internal/features/users/services"

	"github.com/google/uuid"
)

type MembershipService struct {
	membershipRepository *projects_repositories.MembershipRepository
	projectRepository    *projects_repositories.ProjectRepository
	userService          *users_services.UserService
	auditLogService      *audit_logs.AuditLogService
	projectService       *ProjectService
	settingsService      *users_services.SettingsService
}

func (s *MembershipService) GetMembers(
	projectID uuid.UUID,
	user *users_models.User,
) (*projects_dto.GetMembersResponseDTO, error) {
	canAccess, _, err := s.projectService.CanUserAccessProject(projectID, user)

	if err != nil {
		return nil, err
	}
	if !canAccess {
		return nil, errors.New("insufficient permissions to view project members")
	}

	members, err := s.membershipRepository.GetProjectMembers(projectID)
	if err != nil {
		return nil, fmt.Errorf("failed to get project members: %w", err)
	}

	membersList := make([]projects_dto.ProjectMemberResponseDTO, len(members))
	for i, member := range members {
		membersList[i] = *member
	}

	return &projects_dto.GetMembersResponseDTO{
		Members: membersList,
	}, nil
}

func (s *MembershipService) AddMember(
	projectID uuid.UUID,
	request *projects_dto.AddMemberRequestDTO,
	addedBy *users_models.User,
) (*projects_dto.AddMemberResponseDTO, error) {
	if err := s.validateCanManageMembership(projectID, addedBy, request.Role); err != nil {
		return nil, err
	}

	targetUser, err := s.userService.GetUserByEmail(request.Email)
	if err != nil {
		return nil, err
	}

	if targetUser == nil {
		// User doesn't exist, invite them
		settings, err := s.settingsService.GetSettings()
		if err != nil {
			return nil, fmt.Errorf("failed to get settings: %w", err)
		}

		if !addedBy.CanInviteUsers(settings) {
			return nil, errors.New("insufficient permissions to invite users")
		}

		inviteRequest := &users_dto.InviteUserRequestDTO{
			Email:               request.Email,
			IntendedProjectID:   &projectID,
			IntendedProjectRole: &request.Role,
		}

		inviteResponse, err := s.userService.InviteUser(inviteRequest, addedBy)
		if err != nil {
			return nil, err
		}

		membership := &projects_models.ProjectMembership{
			UserID:    inviteResponse.ID,
			ProjectID: projectID,
			Role:      request.Role,
		}

		if err := s.membershipRepository.CreateMembership(membership); err != nil {
			return nil, fmt.Errorf("failed to add member: %w", err)
		}

		s.auditLogService.WriteAuditLog(
			fmt.Sprintf("User invited to project: %s and added as %s", request.Email, request.Role),
			&addedBy.ID,
			&projectID,
		)

		return &projects_dto.AddMemberResponseDTO{
			Status: projects_dto.AddStatusInvited,
		}, nil
	}

	existingMembership, _ := s.membershipRepository.GetMembershipByUserAndProject(targetUser.ID, projectID)
	if existingMembership != nil {
		return nil, errors.New("user is already a member of this project")
	}

	membership := &projects_models.ProjectMembership{
		UserID:    targetUser.ID,
		ProjectID: projectID,
		Role:      request.Role,
	}

	if err := s.membershipRepository.CreateMembership(membership); err != nil {
		return nil, fmt.Errorf("failed to add member: %w", err)
	}

	s.auditLogService.WriteAuditLog(
		fmt.Sprintf("User added to project: %s as %s", targetUser.Email, request.Role),
		&addedBy.ID,
		&projectID,
	)

	return &projects_dto.AddMemberResponseDTO{
		Status: projects_dto.AddStatusAdded,
	}, nil
}

func (s *MembershipService) ChangeMemberRole(
	projectID uuid.UUID,
	memberUserID uuid.UUID,
	request *projects_dto.ChangeMemberRoleRequestDTO,
	changedBy *users_models.User,
) error {
	if err := s.validateCanManageMembership(projectID, changedBy, request.Role); err != nil {
		return err
	}

	if memberUserID == changedBy.ID {
		return errors.New("cannot change your own role")
	}

	existingMembership, err := s.membershipRepository.GetMembershipByUserAndProject(memberUserID, projectID)
	if err != nil {
		return errors.New("user is not a member of this project")
	}

	if existingMembership.Role == users_enums.ProjectRoleOwner {
		return errors.New("cannot change owner role")
	}

	targetUser, err := s.userService.GetUserByID(memberUserID)
	if err != nil {
		return errors.New("user not found")
	}

	if err := s.membershipRepository.UpdateMemberRole(memberUserID, projectID, request.Role); err != nil {
		return fmt.Errorf("failed to update member role: %w", err)
	}

	s.auditLogService.WriteAuditLog(
		fmt.Sprintf(
			"Member role changed: %s from %s to %s",
			targetUser.Email,
			existingMembership.Role,
			request.Role,
		),
		&changedBy.ID,
		&projectID,
	)

	return nil
}

func (s *MembershipService) RemoveMember(
	projectID uuid.UUID,
	memberUserID uuid.UUID,
	removedBy *users_models.User,
) error {
	canManage, err := s.projectService.CanUserManageProject(projectID, removedBy)
	if err != nil {
		return err
	}

	if !canManage {
		return errors.New("insufficient permissions to remove members")
	}

	existingMembership, err := s.membershipRepository.GetMembershipByUserAndProject(memberUserID, projectID)
	if err != nil {
		return errors.New("user is not a member of this project")
	}

	if existingMembership.Role == users_enums.ProjectRoleOwner {
		return errors.New("cannot remove project owner, transfer ownership first")
	}

	targetUser, err := s.userService.GetUserByID(memberUserID)
	if err != nil {
		return errors.New("user not found")
	}

	if err := s.membershipRepository.RemoveMember(memberUserID, projectID); err != nil {
		return fmt.Errorf("failed to remove member: %w", err)
	}

	s.auditLogService.WriteAuditLog(
		fmt.Sprintf("Member removed from project: %s", targetUser.Email),
		&removedBy.ID,
		&projectID,
	)

	return nil
}

func (s *MembershipService) TransferOwnership(
	projectID uuid.UUID,
	request *projects_dto.TransferOwnershipRequestDTO,
	user *users_models.User,
) error {
	currentRole, err := s.membershipRepository.GetUserProjectRole(projectID, user.ID)
	if err != nil {
		return fmt.Errorf("failed to get current user role: %w", err)
	}

	if user.Role != users_enums.UserRoleAdmin &&
		(currentRole == nil || *currentRole != users_enums.ProjectRoleOwner) {
		return errors.New("only project owner or admin can transfer ownership")
	}

	newOwner, err := s.userService.GetUserByEmail(request.NewOwnerEmail)
	if err != nil {
		return errors.New("new owner not found")
	}

	if newOwner == nil {
		return errors.New("new owner not found")
	}

	_, err = s.membershipRepository.GetMembershipByUserAndProject(newOwner.ID, projectID)
	if err != nil {
		return errors.New("new owner must be a project member")
	}

	currentOwner, err := s.membershipRepository.GetProjectOwner(projectID)
	if err != nil {
		return fmt.Errorf("failed to find current project owner: %w", err)
	}

	if currentOwner == nil {
		return errors.New("no current project owner found")
	}

	if err := s.membershipRepository.UpdateMemberRole(newOwner.ID, projectID, users_enums.ProjectRoleOwner); err != nil {
		return fmt.Errorf("failed to update new owner role: %w", err)
	}

	if err := s.membershipRepository.UpdateMemberRole(currentOwner.UserID, projectID, users_enums.ProjectRoleAdmin); err != nil {
		return fmt.Errorf("failed to update previous owner role: %w", err)
	}

	s.auditLogService.WriteAuditLog(
		fmt.Sprintf("Project ownership transferred to: %s", newOwner.Email),
		&user.ID,
		&projectID,
	)

	return nil
}

func (s *MembershipService) validateCanManageMembership(
	projectID uuid.UUID,
	user *users_models.User,
	changesRoleTo users_enums.ProjectRole,
) error {
	canManageProject, err := s.projectService.CanUserManageProject(projectID, user)
	if err != nil {
		return err
	}

	if !canManageProject {
		return errors.New("insufficient permissions to manage members")
	}

	currentRole, err := s.membershipRepository.GetUserProjectRole(projectID, user.ID)
	if err != nil {
		return err
	}

	if changesRoleTo == users_enums.ProjectRoleAdmin || changesRoleTo == users_enums.ProjectRoleOwner {
		// Global admins can manage any role
		if user.Role == users_enums.UserRoleAdmin {
			return nil
		}

		if currentRole == nil || *currentRole != users_enums.ProjectRoleOwner {
			return errors.New("only project owner can add/manage admins")
		}
	}

	return nil
}
