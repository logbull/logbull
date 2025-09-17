package projects_models

import (
	"time"

	users_enums "logbull/internal/features/users/enums"

	"github.com/google/uuid"
)

type ProjectMembership struct {
	ID        uuid.UUID               `json:"id"        gorm:"column:id"`
	UserID    uuid.UUID               `json:"userId"    gorm:"column:user_id"`
	ProjectID uuid.UUID               `json:"projectId" gorm:"column:project_id"`
	Role      users_enums.ProjectRole `json:"role"      gorm:"column:role"`
	CreatedAt time.Time               `json:"createdAt" gorm:"column:created_at"`
}

func (ProjectMembership) TableName() string {
	return "project_memberships"
}
