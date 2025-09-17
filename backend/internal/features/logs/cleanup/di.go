package logs_cleanup

import (
	logs_core "logbull/internal/features/logs/core"
	projects_services "logbull/internal/features/projects/services"
	"logbull/internal/util/logger"
	"sync"
)

var logCleanupBackgroundService = &LogCleanupBackgroundService{
	logs_core.GetLogCoreRepository(),
	projects_services.GetProjectService(),
	logger.GetLogger(),
	nil,
	nil,
	sync.WaitGroup{},
}

func GetLogCleanupBackgroundService() *LogCleanupBackgroundService {
	return logCleanupBackgroundService
}
