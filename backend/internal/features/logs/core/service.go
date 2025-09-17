package logs_core

import "github.com/google/uuid"

type LogCoreService struct {
	logCoreRepository *LogCoreRepository
}

func (s *LogCoreService) OnBeforeProjectDeletion(projectID uuid.UUID) error {
	return s.logCoreRepository.DeleteLogsByProject(projectID)
}
