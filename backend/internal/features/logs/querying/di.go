package logs_querying

import (
	"logbull/internal/cache"
	logs_core "logbull/internal/features/logs/core"
	projects_services "logbull/internal/features/projects/services"
	"logbull/internal/util/logger"
)

var concurrentQueryLimiter = &ConcurrentQueryLimiter{
	cache.GetCache(),
	logger.GetLogger(),
}

var queryValidator = &QueryValidator{
	logger.GetLogger(),
}

var logQueryService = &LogQueryService{
	logs_core.GetLogCoreRepository(),
	projects_services.GetProjectService(),
	concurrentQueryLimiter,
	queryValidator,
	logger.GetLogger(),
}

var logQueryController = &LogQueryController{
	logQueryService,
}

func GetLogQueryService() *LogQueryService {
	return logQueryService
}

func GetLogQueryController() *LogQueryController {
	return logQueryController
}
