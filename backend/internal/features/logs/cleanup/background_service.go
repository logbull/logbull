package logs_cleanup

import (
	"context"
	"fmt"
	"log/slog"
	"sync"
	"time"

	"logbull/internal/config"
	logs_core "logbull/internal/features/logs/core"
	projects_models "logbull/internal/features/projects/models"
	projects_services "logbull/internal/features/projects/services"

	"github.com/google/uuid"
)

type LogCleanupBackgroundService struct {
	logCoreRepository *logs_core.LogCoreRepository
	projectService    *projects_services.ProjectService
	logger            *slog.Logger

	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
}

const (
	quotaEnforcementInterval = 1 * time.Minute
	retentionCleanupInterval = 1 * time.Minute
)

func (s *LogCleanupBackgroundService) StartWorkers() {
	s.ctx, s.cancel = context.WithCancel(context.Background())

	s.logger.Info("Starting log cleanup background workers",
		slog.Duration("quotaInterval", quotaEnforcementInterval),
		slog.Duration("retentionInterval", retentionCleanupInterval))

	s.wg.Add(2)
	go s.quotaEnforcerWorker()
	go s.retentionWorker()

	s.logger.Info("Log cleanup workers started successfully")
}

func (s *LogCleanupBackgroundService) ExecuteAllTasksForTest() error {
	if err := s.enforceAllProjectQuotas(); err != nil {
		s.logger.Error("Error during quota enforcement in test execution", slog.String("error", err.Error()))
		return err
	}

	if err := s.enforceAllProjectsRetention(); err != nil {
		s.logger.Error("Error during retention cleanup in test execution", slog.String("error", err.Error()))
		return err
	}

	return nil
}

func (s *LogCleanupBackgroundService) quotaEnforcerWorker() {
	defer s.wg.Done()

	ticker := time.NewTicker(quotaEnforcementInterval)
	defer ticker.Stop()

	s.logger.Info("Quota enforcer worker started",
		slog.Duration("interval", quotaEnforcementInterval))

	for {
		if config.IsShouldShutdown() {
			s.logger.Info("Quota enforcer worker shutting down due to shutdown signal")
			return
		}

		select {
		case <-s.ctx.Done():
			s.logger.Info("Quota enforcer worker shutting down")
			return

		case <-ticker.C:
			if err := s.enforceAllProjectQuotas(); err != nil {
				s.logger.Error("Error during quota enforcement", slog.String("error", err.Error()))
			}
		}
	}
}

func (s *LogCleanupBackgroundService) retentionWorker() {
	defer s.wg.Done()

	ticker := time.NewTicker(retentionCleanupInterval)
	defer ticker.Stop()

	s.logger.Info("Retention cleanup worker started",
		slog.Duration("interval", retentionCleanupInterval))

	for {
		if config.IsShouldShutdown() {
			s.logger.Info("Retention cleanup worker shutting down due to shutdown signal")
			return
		}

		select {
		case <-s.ctx.Done():
			s.logger.Info("Retention cleanup worker shutting down")
			return

		case <-ticker.C:
			if err := s.enforceAllProjectsRetention(); err != nil {
				s.logger.Error("Error during retention cleanup", slog.String("error", err.Error()))
			}
		}
	}
}

func (s *LogCleanupBackgroundService) enforceAllProjectQuotas() error {
	projects, err := s.projectService.GetAllProjects()
	if err != nil {
		return fmt.Errorf("failed to get all projects: %w", err)
	}

	s.logger.Info(fmt.Sprintf("Enforcing quota for %d projects", len(projects)))

	quotaViolations := 0
	processedProjects := 0

	for _, project := range projects {
		if err := s.enforceProjectQuotas(project.ID, project); err != nil {
			quotaViolations++
			s.logger.Error("Failed to enforce quotas for project",
				slog.String("projectId", project.ID.String()),
				slog.String("error", err.Error()))
		}
		processedProjects++
	}

	s.logger.Info("Quota enforcement completed",
		slog.Int("processedProjects", processedProjects),
		slog.Int("quotaViolations", quotaViolations))

	if quotaViolations > 0 {
		return fmt.Errorf("quota violations detected in %d projects", quotaViolations)
	}

	return nil
}

func (s *LogCleanupBackgroundService) enforceAllProjectsRetention() error {
	projects, err := s.projectService.GetAllProjects()
	if err != nil {
		return fmt.Errorf("failed to get all projects: %w", err)
	}

	cleanupFailures := 0
	processedProjects := 0
	totalCleaned := 0

	for _, project := range projects {
		if project.MaxLogsLifeDays > 0 {
			if err := s.enforceLogRetention(project.ID, project.MaxLogsLifeDays); err != nil {
				cleanupFailures++
				s.logger.Error("Failed to enforce retention for project",
					slog.String("projectId", project.ID.String()),
					slog.String("error", err.Error()))
			} else {
				totalCleaned++
			}
		}
		processedProjects++
	}

	s.logger.Info("Retention cleanup completed",
		slog.Int("processedProjects", processedProjects),
		slog.Int("cleanupFailures", cleanupFailures),
		slog.Int("projectsCleaned", totalCleaned))

	if cleanupFailures > 0 {
		return fmt.Errorf("retention cleanup failed for %d projects", cleanupFailures)
	}

	return nil
}

func (s *LogCleanupBackgroundService) enforceProjectQuotas(
	projectID uuid.UUID,
	project *projects_models.Project,
) error {
	cleanupPercentage := s.calculateCleanupPercentage(project.MaxLogsSizeMB)

	stats, err := s.logCoreRepository.GetProjectLogStats(projectID)
	if err != nil {
		return fmt.Errorf("failed to get project log stats: %w", err)
	}

	quotaViolated := false

	if project.MaxLogsAmount > 0 && stats.TotalLogs > project.MaxLogsAmount {
		s.logger.Info("Project exceeds log count quota, cleanup needed",
			slog.String("projectId", projectID.String()),
			slog.Int64("currentLogs", stats.TotalLogs),
			slog.Int64("maxLogs", project.MaxLogsAmount))

		targetLogs := int64(float64(project.MaxLogsAmount) * cleanupPercentage)
		logsToDelete := stats.TotalLogs - targetLogs

		if logsToDelete > 0 {
			cutoffTime := s.calculateCutoffTimeForLogCount(logsToDelete, stats)
			if err := s.logCoreRepository.DeleteOldLogs(projectID, cutoffTime); err != nil {
				s.logger.Error("Failed to delete old logs for count quota",
					slog.String("projectId", projectID.String()),
					slog.String("error", err.Error()))
				quotaViolated = true
			} else {
				s.logger.Info("Deleted logs to enforce count quota",
					slog.String("projectId", projectID.String()),
					slog.Int64("deletedLogs", logsToDelete))
			}
		}
	}

	if project.MaxLogsSizeMB > 0 && stats.TotalSizeMB > float64(project.MaxLogsSizeMB) {
		s.logger.Info("Project exceeds storage size quota, cleanup needed",
			slog.String("projectId", projectID.String()),
			slog.Float64("currentSizeMB", stats.TotalSizeMB),
			slog.Int("maxSizeMB", project.MaxLogsSizeMB))

		targetSizeMB := float64(project.MaxLogsSizeMB) * cleanupPercentage
		excessSizeMB := stats.TotalSizeMB - targetSizeMB

		if excessSizeMB > 0 {
			cutoffTime := s.calculateCutoffTimeForSize(excessSizeMB, stats)
			if err := s.logCoreRepository.DeleteOldLogs(projectID, cutoffTime); err != nil {
				s.logger.Error("Failed to delete old logs for size quota",
					slog.String("projectId", projectID.String()),
					slog.String("error", err.Error()))
				quotaViolated = true
			} else {
				s.logger.Info("Deleted logs to enforce size quota",
					slog.String("projectId", projectID.String()),
					slog.Float64("freedSizeMB", excessSizeMB))
			}
		}
	}

	if quotaViolated {
		return fmt.Errorf("quota enforcement failed for project %s", projectID.String())
	}

	return nil
}

func (s *LogCleanupBackgroundService) enforceLogRetention(projectID uuid.UUID, maxLifeDays int) error {
	if maxLifeDays <= 0 {
		return nil
	}

	cutoffTime := time.Now().UTC().AddDate(0, 0, -maxLifeDays)

	err := s.logCoreRepository.DeleteOldLogs(projectID, cutoffTime)
	if err != nil {
		return fmt.Errorf("failed to delete old logs: %w", err)
	}

	return nil
}

func (s *LogCleanupBackgroundService) calculateCutoffTimeForLogCount(
	logsToDelete int64,
	stats *logs_core.ProjectLogStats,
) time.Time {
	if stats.TotalLogs == 0 {
		return time.Now().UTC()
	}

	logLifespan := stats.NewestLogTime.Sub(stats.OldestLogTime)
	if logLifespan <= 0 {
		return time.Now().UTC().Add(-24 * time.Hour)
	}

	percentageToDelete := float64(logsToDelete) / float64(stats.TotalLogs)
	timeToDelete := time.Duration(float64(logLifespan) * percentageToDelete)

	return stats.OldestLogTime.Add(timeToDelete)
}

func (s *LogCleanupBackgroundService) calculateCutoffTimeForSize(
	sizeMBToDelete float64,
	stats *logs_core.ProjectLogStats,
) time.Time {
	if stats.TotalSizeMB == 0 {
		return time.Now().UTC()
	}

	logLifespan := stats.NewestLogTime.Sub(stats.OldestLogTime)
	if logLifespan <= 0 {
		return time.Now().UTC().Add(-24 * time.Hour)
	}

	percentageToDelete := sizeMBToDelete / stats.TotalSizeMB
	timeToDelete := time.Duration(float64(logLifespan) * percentageToDelete)

	return stats.OldestLogTime.Add(timeToDelete)
}

func (s *LogCleanupBackgroundService) calculateCleanupPercentage(quotaSizeMB int) float64 {
	switch {
	case quotaSizeMB <= 10:
		return 0.85 // Small quotas: 85% target (more aggressive cleanup)
	case quotaSizeMB <= 100:
		return 0.90 // Medium quotas: 90% target (current behavior)
	case quotaSizeMB <= 500:
		return 0.95 // Large quotas: 95% target (gentler cleanup)
	default:
		return 0.98 // Very large quotas: 98% target (minimal deletion)
	}
}
