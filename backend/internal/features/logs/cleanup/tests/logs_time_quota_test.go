package logs_cleanup_tests

import (
	"testing"
	"time"

	logs_cleanup "logbull/internal/features/logs/cleanup"
	logs_core "logbull/internal/features/logs/core"
	logs_core_tests "logbull/internal/features/logs/core/tests"
	projects_controllers "logbull/internal/features/projects/controllers"
	projects_models "logbull/internal/features/projects/models"
	projects_testing "logbull/internal/features/projects/testing"
	users_enums "logbull/internal/features/users/enums"
	users_testing "logbull/internal/features/users/testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_EnforceLogRetention_WhenMaxLogsLifeDaysIsSet_DeletesLogsOlderThanRetentionPeriod(t *testing.T) {
	// Setup test environment
	router := projects_testing.CreateTestRouter(
		projects_controllers.GetProjectController(),
		projects_controllers.GetMembershipController(),
	)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	uniqueID := uuid.New().String()[:8]

	// Create test project
	projectName := "Log Retention Test " + uniqueID
	project := projects_testing.CreateTestProject(projectName, owner, router)

	// Update project to set MaxLogsLifeDays to 7 days
	updateData := &projects_models.Project{
		Name:            project.Name,
		MaxLogsLifeDays: 7,
	}
	projects_testing.UpdateProject(project, updateData, owner.Token, router)

	// Get repository and cleanup service
	repository := logs_core.GetLogCoreRepository()
	cleanupService := logs_cleanup.GetLogCleanupBackgroundService()

	// Create test timestamps
	now := time.Now().UTC()
	oldTime := now.AddDate(0, 0, -10)   // 10 days ago (should be deleted)
	recentTime := now.AddDate(0, 0, -5) // 5 days ago (should remain)

	// Create old logs (should be deleted)
	oldLogEntries := logs_core_tests.CreateTestLogEntriesWithUniqueFields(
		project.ID,
		oldTime,
		"Old log message for retention test",
		map[string]any{
			"test_session": uniqueID,
			"log_type":     "old",
		},
	)

	// Create recent logs (should remain)
	recentLogEntries := logs_core_tests.CreateTestLogEntriesWithUniqueFields(
		project.ID,
		recentTime,
		"Recent log message for retention test",
		map[string]any{
			"test_session": uniqueID,
			"log_type":     "recent",
		},
	)

	// Merge and store all logs
	allEntries := logs_core_tests.MergeLogEntries(oldLogEntries, recentLogEntries)
	logs_core_tests.StoreTestLogsAndFlush(t, repository, allEntries)

	// Verify logs were stored (should have 2 total)
	statsBeforeCleanup, err := repository.GetProjectLogStats(project.ID)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), statsBeforeCleanup.TotalLogs, "Should have 2 logs before cleanup")

	// Execute cleanup service
	err = cleanupService.ExecuteAllTasksForTest()
	assert.NoError(t, err, "Cleanup service should execute successfully")

	// Force flush to ensure deletions are reflected
	err = repository.ForceFlush()
	assert.NoError(t, err, "Force flush should succeed")

	// Wait a moment for delete operations to complete
	time.Sleep(100 * time.Millisecond)

	// Verify old logs were deleted and recent logs remain
	statsAfterCleanup, err := repository.GetProjectLogStats(project.ID)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), statsAfterCleanup.TotalLogs, "Should have 1 log remaining after cleanup")

	// Verify the remaining log is the recent one by checking timestamp bounds
	if !statsAfterCleanup.OldestLogTime.IsZero() && !statsAfterCleanup.NewestLogTime.IsZero() {
		// The remaining log should be from around the recent time
		assert.True(t, statsAfterCleanup.OldestLogTime.After(recentTime.Add(-1*time.Minute)),
			"Remaining log should be newer than recent time boundary")
	}
}

func Test_EnforceLogRetention_WhenMaxLogsLifeDaysIsZero_NoRetentionEnforcement(t *testing.T) {
	// Setup test environment
	router := projects_testing.CreateTestRouter(
		projects_controllers.GetProjectController(),
		projects_controllers.GetMembershipController(),
	)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	uniqueID := uuid.New().String()[:8]

	// Create test project
	projectName := "Zero Retention Test " + uniqueID
	project := projects_testing.CreateTestProject(projectName, owner, router)

	// Update project to set MaxLogsLifeDays to 0 (no retention)
	updateData := &projects_models.Project{
		Name:            project.Name,
		MaxLogsLifeDays: 0,
	}
	projects_testing.UpdateProject(project, updateData, owner.Token, router)

	// Get repository and cleanup service
	repository := logs_core.GetLogCoreRepository()
	cleanupService := logs_cleanup.GetLogCleanupBackgroundService()

	// Create test timestamps
	now := time.Now().UTC()
	oldTime := now.AddDate(0, 0, -30)   // 30 days ago (would normally be deleted)
	recentTime := now.AddDate(0, 0, -1) // 1 day ago

	// Create old logs (should NOT be deleted when retention is 0)
	oldLogEntries := logs_core_tests.CreateTestLogEntriesWithUniqueFields(
		project.ID,
		oldTime,
		"Old log message for zero retention test",
		map[string]any{
			"test_session": uniqueID,
			"log_type":     "old",
		},
	)

	// Create recent logs (should remain)
	recentLogEntries := logs_core_tests.CreateTestLogEntriesWithUniqueFields(
		project.ID,
		recentTime,
		"Recent log message for zero retention test",
		map[string]any{
			"test_session": uniqueID,
			"log_type":     "recent",
		},
	)

	// Merge and store all logs
	allEntries := logs_core_tests.MergeLogEntries(oldLogEntries, recentLogEntries)
	logs_core_tests.StoreTestLogsAndFlush(t, repository, allEntries)

	// Verify logs were stored (should have 2 total)
	statsBeforeCleanup, err := repository.GetProjectLogStats(project.ID)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), statsBeforeCleanup.TotalLogs, "Should have 2 logs before cleanup")

	// Execute cleanup service
	err = cleanupService.ExecuteAllTasksForTest()
	assert.NoError(t, err, "Cleanup service should execute successfully")

	// Force flush to ensure any potential deletions are reflected
	err = repository.ForceFlush()
	assert.NoError(t, err, "Force flush should succeed")

	// Wait a moment for any operations to complete
	time.Sleep(100 * time.Millisecond)

	// Verify NO logs were deleted (both old and recent should remain)
	statsAfterCleanup, err := repository.GetProjectLogStats(project.ID)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), statsAfterCleanup.TotalLogs, "Should still have 2 logs after cleanup with zero retention")
}

func Test_EnforceLogRetention_WhenMaxLogsLifeDaysIsNegative_NoRetentionEnforcement(t *testing.T) {
	// Setup test environment
	router := projects_testing.CreateTestRouter(
		projects_controllers.GetProjectController(),
		projects_controllers.GetMembershipController(),
	)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	uniqueID := uuid.New().String()[:8]

	// Create test project
	projectName := "Negative Retention Test " + uniqueID
	project := projects_testing.CreateTestProject(projectName, owner, router)

	// Update project to set MaxLogsLifeDays to -1 (no retention)
	updateData := &projects_models.Project{
		Name:            project.Name,
		MaxLogsLifeDays: -1,
	}
	projects_testing.UpdateProject(project, updateData, owner.Token, router)

	// Get repository and cleanup service
	repository := logs_core.GetLogCoreRepository()
	cleanupService := logs_cleanup.GetLogCleanupBackgroundService()

	// Create test timestamps
	now := time.Now().UTC()
	oldTime := now.AddDate(0, 0, -30)   // 30 days ago (would normally be deleted)
	recentTime := now.AddDate(0, 0, -1) // 1 day ago

	// Create old logs (should NOT be deleted when retention is negative)
	oldLogEntries := logs_core_tests.CreateTestLogEntriesWithUniqueFields(
		project.ID,
		oldTime,
		"Old log message for negative retention test",
		map[string]any{
			"test_session": uniqueID,
			"log_type":     "old",
		},
	)

	// Create recent logs (should remain)
	recentLogEntries := logs_core_tests.CreateTestLogEntriesWithUniqueFields(
		project.ID,
		recentTime,
		"Recent log message for negative retention test",
		map[string]any{
			"test_session": uniqueID,
			"log_type":     "recent",
		},
	)

	// Merge and store all logs
	allEntries := logs_core_tests.MergeLogEntries(oldLogEntries, recentLogEntries)
	logs_core_tests.StoreTestLogsAndFlush(t, repository, allEntries)

	// Verify logs were stored (should have 2 total)
	statsBeforeCleanup, err := repository.GetProjectLogStats(project.ID)
	assert.NoError(t, err)
	assert.Equal(t, int64(2), statsBeforeCleanup.TotalLogs, "Should have 2 logs before cleanup")

	// Execute cleanup service
	err = cleanupService.ExecuteAllTasksForTest()
	assert.NoError(t, err, "Cleanup service should execute successfully")

	// Force flush to ensure any potential deletions are reflected
	err = repository.ForceFlush()
	assert.NoError(t, err, "Force flush should succeed")

	// Wait a moment for any operations to complete
	time.Sleep(100 * time.Millisecond)

	// Verify NO logs were deleted (both old and recent should remain)
	statsAfterCleanup, err := repository.GetProjectLogStats(project.ID)
	assert.NoError(t, err)
	assert.Equal(
		t,
		int64(2),
		statsAfterCleanup.TotalLogs,
		"Should still have 2 logs after cleanup with negative retention",
	)
}

func Test_EnforceProjectQuotas_WithDifferentProjectsTimeQuotas_DeletesOnlyTargetProjectLogs(t *testing.T) {
	// Setup test environment
	router := projects_testing.CreateTestRouter(
		projects_controllers.GetProjectController(),
		projects_controllers.GetMembershipController(),
	)

	// Create multiple users and projects
	owner1 := users_testing.CreateTestUser(users_enums.UserRoleMember)
	owner2 := users_testing.CreateTestUser(users_enums.UserRoleMember)
	uniqueID1 := uuid.New().String()[:8]
	uniqueID2 := uuid.New().String()[:8]

	// Create test projects
	project1Name := "Project1 Different Quota Test " + uniqueID1
	project2Name := "Project2 Different Quota Test " + uniqueID2

	project1 := projects_testing.CreateTestProject(project1Name, owner1, router)
	project2 := projects_testing.CreateTestProject(project2Name, owner2, router)

	// Set different MaxLogsLifeDays for each project
	// Project 1: 7 days retention (should delete old logs)
	updateData1 := &projects_models.Project{
		Name:            project1.Name,
		MaxLogsLifeDays: 7,
	}
	projects_testing.UpdateProject(project1, updateData1, owner1.Token, router)

	// Project 2: 30 days retention (should NOT delete old logs)
	updateData2 := &projects_models.Project{
		Name:            project2.Name,
		MaxLogsLifeDays: 30,
	}
	projects_testing.UpdateProject(project2, updateData2, owner2.Token, router)

	// Get repository and cleanup service
	repository := logs_core.GetLogCoreRepository()
	cleanupService := logs_cleanup.GetLogCleanupBackgroundService()

	// Create test timestamps
	now := time.Now().UTC()
	oldTime := now.AddDate(0, 0, -10)   // 10 days ago (should be deleted from project1 but not project2)
	recentTime := now.AddDate(0, 0, -5) // 5 days ago (should remain in both projects)

	// Create logs for Project 1
	project1OldLogs := logs_core_tests.CreateTestLogEntriesWithUniqueFields(
		project1.ID,
		oldTime,
		"Project1 old log for different quota test",
		map[string]any{
			"test_session": uniqueID1,
			"log_type":     "old",
			"project_name": project1Name,
		},
	)

	project1RecentLogs := logs_core_tests.CreateTestLogEntriesWithUniqueFields(
		project1.ID,
		recentTime,
		"Project1 recent log for different quota test",
		map[string]any{
			"test_session": uniqueID1,
			"log_type":     "recent",
			"project_name": project1Name,
		},
	)

	// Create logs for Project 2
	project2OldLogs := logs_core_tests.CreateTestLogEntriesWithUniqueFields(
		project2.ID,
		oldTime,
		"Project2 old log for different quota test",
		map[string]any{
			"test_session": uniqueID2,
			"log_type":     "old",
			"project_name": project2Name,
		},
	)

	project2RecentLogs := logs_core_tests.CreateTestLogEntriesWithUniqueFields(
		project2.ID,
		recentTime,
		"Project2 recent log for different quota test",
		map[string]any{
			"test_session": uniqueID2,
			"log_type":     "recent",
			"project_name": project2Name,
		},
	)

	// Store all logs
	project1Entries := logs_core_tests.MergeLogEntries(project1OldLogs, project1RecentLogs)
	project2Entries := logs_core_tests.MergeLogEntries(project2OldLogs, project2RecentLogs)

	logs_core_tests.StoreTestLogsAndFlush(t, repository, project1Entries)
	logs_core_tests.StoreTestLogsAndFlush(t, repository, project2Entries)

	// Additional force flush to ensure all logs are indexed
	err := repository.ForceFlush()
	assert.NoError(t, err, "Initial force flush should succeed")

	// Wait for logs to be fully indexed
	time.Sleep(500 * time.Millisecond)

	// Verify logs were stored for both projects
	project1StatsBeforeCleanup, err := repository.GetProjectLogStats(project1.ID)
	assert.NoError(t, err)
	t.Logf(
		"Project1 stats before cleanup: TotalLogs=%d, OldestTime=%v, NewestTime=%v",
		project1StatsBeforeCleanup.TotalLogs,
		project1StatsBeforeCleanup.OldestLogTime,
		project1StatsBeforeCleanup.NewestLogTime,
	)
	assert.Equal(t, int64(2), project1StatsBeforeCleanup.TotalLogs, "Project1 should have 2 logs before cleanup")

	project2StatsBeforeCleanup, err := repository.GetProjectLogStats(project2.ID)
	assert.NoError(t, err)
	t.Logf(
		"Project2 stats before cleanup: TotalLogs=%d, OldestTime=%v, NewestTime=%v",
		project2StatsBeforeCleanup.TotalLogs,
		project2StatsBeforeCleanup.OldestLogTime,
		project2StatsBeforeCleanup.NewestLogTime,
	)
	assert.Equal(t, int64(2), project2StatsBeforeCleanup.TotalLogs, "Project2 should have 2 logs before cleanup")

	// Execute cleanup service
	err = cleanupService.ExecuteAllTasksForTest()
	assert.NoError(t, err, "Cleanup service should execute successfully")

	// Force flush to ensure deletions are reflected
	err = repository.ForceFlush()
	assert.NoError(t, err, "Force flush should succeed")

	// Wait a moment for delete operations to complete
	time.Sleep(100 * time.Millisecond)

	// Verify Project 1 had old logs deleted (7 days retention)
	project1StatsAfterCleanup, err := repository.GetProjectLogStats(project1.ID)
	assert.NoError(t, err)
	t.Logf(
		"Project1 stats after cleanup: TotalLogs=%d, OldestTime=%v, NewestTime=%v",
		project1StatsAfterCleanup.TotalLogs,
		project1StatsAfterCleanup.OldestLogTime,
		project1StatsAfterCleanup.NewestLogTime,
	)
	assert.Equal(
		t,
		int64(1),
		project1StatsAfterCleanup.TotalLogs,
		"Project1 should have 1 log remaining after cleanup (old deleted)",
	)

	// Verify Project 2 still has all logs (30 days retention)
	project2StatsAfterCleanup, err := repository.GetProjectLogStats(project2.ID)
	assert.NoError(t, err)
	t.Logf(
		"Project2 stats after cleanup: TotalLogs=%d, OldestTime=%v, NewestTime=%v",
		project2StatsAfterCleanup.TotalLogs,
		project2StatsAfterCleanup.OldestLogTime,
		project2StatsAfterCleanup.NewestLogTime,
	)
	assert.Equal(
		t,
		int64(2),
		project2StatsAfterCleanup.TotalLogs,
		"Project2 should still have 2 logs after cleanup (none deleted)",
	)

	// Verify the remaining log in project1 is the recent one by checking timestamp bounds
	if !project1StatsAfterCleanup.OldestLogTime.IsZero() && !project1StatsAfterCleanup.NewestLogTime.IsZero() {
		// The remaining log should be from around the recent time
		assert.True(t, project1StatsAfterCleanup.OldestLogTime.After(recentTime.Add(-1*time.Minute)),
			"Remaining log in Project1 should be newer than recent time boundary")
	}
}
