package downdetect

import (
	"fmt"

	logs_core "logbull/internal/features/logs/core"
	"logbull/internal/storage"
	cache_utils "logbull/internal/util/cache"
)

type DowndetectService struct {
	logCoreRepository *logs_core.LogCoreRepository
}

func (s *DowndetectService) IsAvailable() error {
	// Check database connection
	if err := storage.GetDb().Exec("SELECT 1").Error; err != nil {
		return fmt.Errorf("database check failed: %w", err)
	}

	// Check Valkey cache connection
	if err := s.testCacheConnection(); err != nil {
		return fmt.Errorf("cache check failed: %w", err)
	}

	// Check OpenSearch connection
	if err := s.logCoreRepository.TestOpenSearchConnection(); err != nil {
		return fmt.Errorf("OpenSearch check failed: %w", err)
	}

	return nil
}

func (s *DowndetectService) testCacheConnection() (err error) {
	defer func() {
		if r := recover(); r != nil {
			err = fmt.Errorf("cache connection test panicked: %v", r)
		}
	}()

	cache_utils.TestCacheConnection()
	return nil
}
