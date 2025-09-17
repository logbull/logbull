package api_keys

import (
	"github.com/google/uuid"
)

type CreateApiKeyRequestDTO struct {
	Name string `json:"name" binding:"required,min=1,max=100"`
}

type GetApiKeysResponseDTO struct {
	ApiKeys []*ApiKey `json:"apiKeys"`
}

type UpdateApiKeyRequestDTO struct {
	Name   *string       `json:"name,omitempty"   binding:"omitempty,min=1,max=100"`
	Status *ApiKeyStatus `json:"status,omitempty"`
}

type ValidateTokenRequest struct {
	Token     string    `json:"token"`
	ProjectID uuid.UUID `json:"projectId"`
}

type ValidateTokenResponse struct {
	IsValid   bool      `json:"isValid"`
	ApiKeyID  uuid.UUID `json:"apiKeyId,omitempty"`
	ProjectID uuid.UUID `json:"projectId,omitempty"`
}

type CachedApiKey struct {
	ID        uuid.UUID    `json:"id"`
	ProjectID uuid.UUID    `json:"projectId"`
	Status    ApiKeyStatus `json:"status"`
}
