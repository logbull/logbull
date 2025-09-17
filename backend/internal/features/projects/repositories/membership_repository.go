package projects_repositories

import (
	"errors"
	"time"

	projects_dto "logbull/internal/features/projects/dto"
	projects_models "logbull/internal/features/projects/models"
	users_enums "logbull/internal/features/users/enums"
	"logbull/internal/storage"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MembershipRepository struct{}

func (r *MembershipRepository) CreateMembership(membership *projects_models.ProjectMembership) error {
	if membership.ID == uuid.Nil {
		membership.ID = uuid.New()
	}

	if membership.CreatedAt.IsZero() {
		membership.CreatedAt = time.Now().UTC()
	}

	return storage.GetDb().Create(membership).Error
}

func (r *MembershipRepository) GetMembershipByUserAndProject(
	userID, projectID uuid.UUID,
) (*projects_models.ProjectMembership, error) {
	var membership projects_models.ProjectMembership

	if err := storage.GetDb().
		Where("user_id = ? AND project_id = ?", userID, projectID).
		First(&membership).Error; err != nil {
		return nil, err
	}

	return &membership, nil
}

func (r *MembershipRepository) GetProjectMembers(
	projectID uuid.UUID,
) ([]*projects_dto.ProjectMemberResponseDTO, error) {
	var members []*projects_dto.ProjectMemberResponseDTO

	err := storage.GetDb().
		Table("project_memberships pm").
		Select("pm.id, pm.user_id, u.email, pm.role, pm.created_at").
		Joins("JOIN users u ON pm.user_id = u.id").
		Where("pm.project_id = ?", projectID).
		Order("pm.created_at ASC").
		Scan(&members).Error

	return members, err
}

func (r *MembershipRepository) UpdateMemberRole(userID, projectID uuid.UUID, role users_enums.ProjectRole) error {
	return storage.GetDb().
		Model(&projects_models.ProjectMembership{}).
		Where("user_id = ? AND project_id = ?", userID, projectID).
		Update("role", role).Error
}

func (r *MembershipRepository) RemoveMember(userID, projectID uuid.UUID) error {
	return storage.GetDb().
		Where("user_id = ? AND project_id = ?", userID, projectID).
		Delete(&projects_models.ProjectMembership{}).Error
}

func (r *MembershipRepository) GetUserProjectRole(projectID, userID uuid.UUID) (*users_enums.ProjectRole, error) {
	var membership projects_models.ProjectMembership
	err := storage.GetDb().
		Where("project_id = ? AND user_id = ?", projectID, userID).
		First(&membership).Error

	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}

		return nil, err
	}

	return &membership.Role, nil
}

func (r *MembershipRepository) GetProjectOwner(projectID uuid.UUID) (*projects_models.ProjectMembership, error) {
	var membership projects_models.ProjectMembership

	err := storage.GetDb().
		Where("project_id = ? AND role = ?", projectID, users_enums.ProjectRoleOwner).
		First(&membership).Error

	if err != nil {
		return nil, err
	}

	return &membership, nil
}

func (r *MembershipRepository) GetProjectsWithRolesByUserID(
	userRole users_enums.UserRole,
	userID uuid.UUID,
) ([]projects_dto.ProjectResponseDTO, error) {
	results := make([]projects_dto.ProjectResponseDTO, 0)

	if userRole == users_enums.UserRoleAdmin {
		err := storage.GetDb().Table("projects").Order("name ASC").Scan(&results).Error
		return results, err
	}

	err := storage.GetDb().
		Table("projects p").
		Select("p.id, p.name, p.created_at, pm.role as user_role").
		Joins("JOIN project_memberships pm ON p.id = pm.project_id").
		Where("pm.user_id = ?", userID).
		Order("p.name ASC").
		Scan(&results).Error

	return results, err
}
