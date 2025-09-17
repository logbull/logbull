package system_healthcheck

import (
	"errors"
	"logbull/internal/features/disk"
	"logbull/internal/storage"
)

type HealthcheckService struct {
	diskService *disk.DiskService
}

func (s *HealthcheckService) IsHealthy() error {
	diskUsage, err := s.diskService.GetDiskUsage()
	if err != nil {
		return errors.New("cannot get disk usage")
	}

	if float64(diskUsage.UsedSpaceBytes) >= float64(diskUsage.TotalSpaceBytes)*0.95 {
		return errors.New("more than 95% of the disk is used")
	}

	db := storage.GetDb()
	err = db.Raw("SELECT 1").Error

	if err != nil {
		return errors.New("cannot connect to the database")
	}

	return nil
}
