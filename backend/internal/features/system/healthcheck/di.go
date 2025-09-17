package system_healthcheck

import (
	"logbull/internal/features/disk"
)

var healthcheckService = &HealthcheckService{
	disk.GetDiskService(),
}
var healthcheckController = &HealthcheckController{
	healthcheckService,
}

func GetHealthcheckController() *HealthcheckController {
	return healthcheckController
}
