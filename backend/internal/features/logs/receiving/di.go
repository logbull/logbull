package logs_receiving

import (
	api_keys "logbull/internal/features/api_keys"
	logs_core "logbull/internal/features/logs/core"
	projects_services "logbull/internal/features/projects/services"
	"logbull/internal/util/logger"
	rate_limit "logbull/internal/util/rate_limit"
)

var rateLimiter = rate_limit.NewRateLimiter()

var logWorkerService = NewLogWorkerService(
	logs_core.GetLogCoreRepository(),
	logger.GetLogger(),
)

var logReceivingService = &LogReceivingService{
	logs_core.GetLogCoreRepository(),
	rateLimiter,
	projects_services.GetProjectService(),
	api_keys.GetApiKeyService(),
	logWorkerService,
	logger.GetLogger(),
}

var receivingController = &ReceivingController{
	logReceivingService,
}

func GetLogReceivingService() *LogReceivingService {
	return logReceivingService
}

func GetLogWorkerService() *LogWorkerService {
	return logWorkerService
}

func GetReceivingController() *ReceivingController {
	return receivingController
}
