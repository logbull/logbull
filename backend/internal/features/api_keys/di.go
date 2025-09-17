package api_keys

import (
	"logbull/internal/cache"
	audit_logs "logbull/internal/features/audit_logs"
	projects_services "logbull/internal/features/projects/services"
	cache_utils "logbull/internal/util/cache"

	"golang.org/x/sync/singleflight"
)

var apiKeyRepository = &ApiKeyRepository{}

var apiKeyService = &ApiKeyService{
	apiKeyRepository,
	projects_services.GetProjectService(),
	audit_logs.GetAuditLogService(),
	cache_utils.NewCacheUtil[CachedApiKey](cache.GetCache(), "lb_apikey:"),
	singleflight.Group{},
}

var apiKeyController = &ApiKeyController{
	apiKeyService,
}

func GetApiKeyService() *ApiKeyService {
	return apiKeyService
}

func GetApiKeyController() *ApiKeyController {
	return apiKeyController
}
