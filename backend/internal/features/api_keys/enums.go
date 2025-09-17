package api_keys

type ApiKeyStatus string

const (
	ApiKeyStatusActive   ApiKeyStatus = "ACTIVE"
	ApiKeyStatusDisabled ApiKeyStatus = "DISABLED"
	ApiKeyStatusNotFound ApiKeyStatus = "NOT_FOUND"
)
