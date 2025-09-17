package disk

import (
	"fmt"
	"runtime"

	"github.com/shirou/gopsutil/v4/disk"
)

type DiskService struct{}

func (s *DiskService) GetDiskUsage() (*DiskUsage, error) {
	platform := s.detectPlatform()

	// Set path based on platform
	path := "/"
	if platform == PlatformWindows {
		path = "C:\\"
	}

	diskUsage, err := disk.Usage(path)
	if err != nil {
		return nil, fmt.Errorf("failed to get disk usage for path %s: %w", path, err)
	}

	return &DiskUsage{
		Platform:        platform,
		TotalSpaceBytes: int64(diskUsage.Total),
		UsedSpaceBytes:  int64(diskUsage.Used),
		FreeSpaceBytes:  int64(diskUsage.Free),
	}, nil
}

func (s *DiskService) detectPlatform() Platform {
	switch runtime.GOOS {
	case "windows":
		return PlatformWindows
	default:
		return PlatformLinux
	}
}
