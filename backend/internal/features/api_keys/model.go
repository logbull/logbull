package api_keys

import (
	"time"

	"github.com/google/uuid"
)

type ApiKey struct {
	ID          uuid.UUID    `json:"id"          gorm:"column:id"`
	Name        string       `json:"name"        gorm:"column:name"`
	ProjectID   uuid.UUID    `json:"projectId"   gorm:"column:project_id"`
	TokenPrefix string       `json:"tokenPrefix" gorm:"column:token_prefix"`
	TokenHash   string       `json:"-"           gorm:"column:token_hash"` // Never expose in JSON
	Status      ApiKeyStatus `json:"status"      gorm:"column:status"`
	CreatedAt   time.Time    `json:"createdAt"   gorm:"column:created_at"`

	Token string `json:"token,omitempty" gorm:"-"` //  Temporary field only populated during creation
}

func (ApiKey) TableName() string {
	return "api_keys"
}
