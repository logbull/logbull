package downdetect

import (
	logs_core "logbull/internal/features/logs/core"
)

var downdetectService = &DowndetectService{
	logs_core.GetLogCoreRepository(),
}
var downdetectController = &DowndetectController{
	downdetectService,
}

func GetDowndetectController() *DowndetectController {
	return downdetectController
}
