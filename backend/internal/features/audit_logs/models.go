package audit_logs

import (
	"time"

	"github.com/google/uuid"
)

type AuditLog struct {
	ID        uuid.UUID  `json:"id"        gorm:"column:id"`
	UserID    *uuid.UUID `json:"userId"    gorm:"column:user_id"`
	ProjectID *uuid.UUID `json:"projectId" gorm:"column:project_id"`
	Message   string     `json:"message"   gorm:"column:message"`
	CreatedAt time.Time  `json:"createdAt" gorm:"column:created_at"`
}

func (AuditLog) TableName() string {
	return "audit_logs"
}
