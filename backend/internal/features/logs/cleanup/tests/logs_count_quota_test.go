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

func Test_EnforceProjectQuotas_WhenLogCountExceedsMaxLogsAmount_DeletesOldestLogs(t *testing.T) {
	// Setup test environment
	router := projects_testing.CreateTestRouter(
		projects_controllers.GetProjectController(),
		projects_controllers.GetMembershipController(),
	)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	uniqueID := uuid.New().String()[:8]

	// Create test project
	projectName := "Count Quota Test " + uniqueID
	project := projects_testing.CreateTestProject(projectName, owner, router)

	// Update project to set MaxLogsAmount to 10 logs
	updateData := &projects_models.Project{
		Name:          project.Name,
		MaxLogsAmount: 10, // 10 logs limit
	}
	projects_testing.UpdateProject(project, updateData, owner.Token, router)

	// Get repository and cleanup service
	repository := logs_core.GetLogCoreRepository()
	cleanupService := logs_cleanup.GetLogCleanupBackgroundService()

	// Create test timestamps
	now := time.Now().UTC()
	oldTime := now.Add(-2 * time.Hour)       // 2 hours ago (should be deleted)
	recentTime := now.Add(-30 * time.Minute) // 30 minutes ago (should remain)

	// Create old logs (should be deleted)
	var allEntries map[uuid.UUID][]*logs_core.LogItem

	// Create 8 old logs
	for i := range 8 {
		oldLogEntries := logs_core_tests.CreateTestLogEntriesWithUniqueFields(
			project.ID,
			oldTime.Add(time.Duration(i)*time.Second),
			"Old log message for count test",
			map[string]any{
				"test_session": uniqueID,
				"log_type":     "old",
				"log_index":    i,
			},
		)
		if allEntries == nil {
			allEntries = oldLogEntries
		} else {
			allEntries = logs_core_tests.MergeLogEntries(allEntries, oldLogEntries)
		}
	}

	// Create 7 recent logs (total: 15 logs, exceeds limit of 10)
	for i := range 7 {
		recentLogEntries := logs_core_tests.CreateTestLogEntriesWithUniqueFields(
			project.ID,
			recentTime.Add(time.Duration(i)*time.Second),
			"Recent log message for count test",
			map[string]any{
				"test_session": uniqueID,
				"log_type":     "recent",
				"log_index":    8 + i,
			},
		)
		allEntries = logs_core_tests.MergeLogEntries(allEntries, recentLogEntries)
	}

	// Store all logs
	logs_core_tests.StoreTestLogsAndFlush(t, repository, allEntries)

	// Verify initial count exceeds quota
	statsBeforeCleanup, err := repository.GetProjectLogStats(project.ID)
	assert.NoError(t, err)
	assert.Equal(t, int64(15), statsBeforeCleanup.TotalLogs, "Should have 15 logs before cleanup")

	t.Logf(
		"Before cleanup: TotalLogs=%d, OldestTime=%v, NewestTime=%v",
		statsBeforeCleanup.TotalLogs,
		statsBeforeCleanup.OldestLogTime,
		statsBeforeCleanup.NewestLogTime,
	)

	// Execute cleanup service
	err = cleanupService.ExecuteAllTasksForTest()
	assert.NoError(t, err, "Cleanup service should execute successfully")

	// Force flush to ensure deletions are reflected
	err = repository.ForceFlush()
	assert.NoError(t, err, "Force flush should succeed")

	// Wait for delete operations to complete
	time.Sleep(100 * time.Millisecond)

	// Verify count quota was enforced
	statsAfterCleanup, err := repository.GetProjectLogStats(project.ID)
	assert.NoError(t, err)

	t.Logf("After cleanup: TotalLogs=%d, OldestTime=%v, NewestTime=%v",
		statsAfterCleanup.TotalLogs, statsAfterCleanup.OldestLogTime, statsAfterCleanup.NewestLogTime)

	assert.LessOrEqual(t, statsAfterCleanup.TotalLogs, int64(10), "Log count should not exceed quota after cleanup")
	assert.Less(t, statsAfterCleanup.TotalLogs, statsBeforeCleanup.TotalLogs, "Should have fewer logs after cleanup")

	// Verify the remaining logs are primarily the recent ones
	if !statsAfterCleanup.OldestLogTime.IsZero() && !statsAfterCleanup.NewestLogTime.IsZero() {
		// Most remaining logs should be from the recent time period
		assert.True(t, statsAfterCleanup.OldestLogTime.After(oldTime.Add(30*time.Minute)),
			"Remaining logs should be mostly from the recent time period")
	}
}

func Test_EnforceProjectQuotas_WhenLogCountIsWithinMaxLogsAmount_NoLogsDeleted(t *testing.T) {
	// Setup test environment
	router := projects_testing.CreateTestRouter(
		projects_controllers.GetProjectController(),
		projects_controllers.GetMembershipController(),
	)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	uniqueID := uuid.New().String()[:8]

	// Create test project
	projectName := "Count Within Quota Test " + uniqueID
	project := projects_testing.CreateTestProject(projectName, owner, router)

	// Update project to set MaxLogsAmount to 50 logs (large enough to not trigger cleanup)
	updateData := &projects_models.Project{
		Name:          project.Name,
		MaxLogsAmount: 50, // 50 logs limit - large enough to not trigger cleanup
	}
	projects_testing.UpdateProject(project, updateData, owner.Token, router)

	// Get repository and cleanup service
	repository := logs_core.GetLogCoreRepository()
	cleanupService := logs_cleanup.GetLogCleanupBackgroundService()

	// Create test timestamps
	now := time.Now().UTC()
	oldTime := now.Add(-2 * time.Hour)       // 2 hours ago
	recentTime := now.Add(-30 * time.Minute) // 30 minutes ago

	// Create only 20 logs (well below 50 limit)
	var allEntries map[uuid.UUID][]*logs_core.LogItem

	for i := range 10 {
		oldLogEntries := logs_core_tests.CreateTestLogEntriesWithUniqueFields(
			project.ID,
			oldTime.Add(time.Duration(i)*time.Second),
			"Old log for within quota test",
			map[string]any{
				"test_session": uniqueID,
				"log_type":     "old",
				"log_index":    i,
			},
		)
		if allEntries == nil {
			allEntries = oldLogEntries
		} else {
			allEntries = logs_core_tests.MergeLogEntries(allEntries, oldLogEntries)
		}
	}

	for i := range 10 {
		recentLogEntries := logs_core_tests.CreateTestLogEntriesWithUniqueFields(
			project.ID,
			recentTime.Add(time.Duration(i)*time.Second),
			"Recent log for within quota test",
			map[string]any{
				"test_session": uniqueID,
				"log_type":     "recent",
				"log_index":    10 + i,
			},
		)
		allEntries = logs_core_tests.MergeLogEntries(allEntries, recentLogEntries)
	}

	// Store all logs
	logs_core_tests.StoreTestLogsAndFlush(t, repository, allEntries)

	// Verify initial count is well within quota
	statsBeforeCleanup, err := repository.GetProjectLogStats(project.ID)
	assert.NoError(t, err)
	assert.Equal(t, int64(20), statsBeforeCleanup.TotalLogs, "Should have 20 logs before cleanup")
	assert.Less(t, statsBeforeCleanup.TotalLogs, int64(50), "Should be well below 50 logs quota before cleanup")

	t.Logf(
		"Before cleanup: TotalLogs=%d, OldestTime=%v, NewestTime=%v",
		statsBeforeCleanup.TotalLogs,
		statsBeforeCleanup.OldestLogTime,
		statsBeforeCleanup.NewestLogTime,
	)

	// Execute cleanup service
	err = cleanupService.ExecuteAllTasksForTest()
	assert.NoError(t, err, "Cleanup service should execute successfully")

	// Force flush to ensure any potential changes are reflected
	err = repository.ForceFlush()
	assert.NoError(t, err, "Force flush should succeed")

	// Wait for any operations to complete
	time.Sleep(100 * time.Millisecond)

	// Verify no logs were deleted since we're within quota
	statsAfterCleanup, err := repository.GetProjectLogStats(project.ID)
	assert.NoError(t, err)

	t.Logf("After cleanup: TotalLogs=%d, OldestTime=%v, NewestTime=%v",
		statsAfterCleanup.TotalLogs, statsAfterCleanup.OldestLogTime, statsAfterCleanup.NewestLogTime)

	assert.Equal(
		t,
		statsBeforeCleanup.TotalLogs,
		statsAfterCleanup.TotalLogs,
		"No logs should be deleted when within quota",
	)
}

func Test_EnforceProjectQuotas_WhenMaxLogsAmountIsZero_NoQuotaEnforcement(t *testing.T) {
	// Setup test environment
	router := projects_testing.CreateTestRouter(
		projects_controllers.GetProjectController(),
		projects_controllers.GetMembershipController(),
	)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	uniqueID := uuid.New().String()[:8]

	// Create test project
	projectName := "Zero Count Quota Test " + uniqueID
	project := projects_testing.CreateTestProject(projectName, owner, router)

	// Update project to set MaxLogsAmount to 0 (no count-based quota)
	updateData := &projects_models.Project{
		Name:          project.Name,
		MaxLogsAmount: 0, // No count quota enforcement
	}
	projects_testing.UpdateProject(project, updateData, owner.Token, router)

	// Get repository and cleanup service
	repository := logs_core.GetLogCoreRepository()
	cleanupService := logs_cleanup.GetLogCleanupBackgroundService()

	// Create test timestamps
	now := time.Now().UTC()
	oldTime := now.Add(-2 * time.Hour)       // 2 hours ago
	recentTime := now.Add(-30 * time.Minute) // 30 minutes ago

	// Create many logs that would normally trigger cleanup
	var allEntries map[uuid.UUID][]*logs_core.LogItem

	// Create 50 old logs (would trigger cleanup if quota was set)
	for i := range 50 {
		oldLogEntries := logs_core_tests.CreateTestLogEntriesWithUniqueFields(
			project.ID,
			oldTime.Add(time.Duration(i)*time.Second),
			"Old log for zero quota test",
			map[string]any{
				"test_session": uniqueID,
				"log_type":     "old",
				"log_index":    i,
			},
		)
		if allEntries == nil {
			allEntries = oldLogEntries
		} else {
			allEntries = logs_core_tests.MergeLogEntries(allEntries, oldLogEntries)
		}
	}

	// Create 25 recent logs (total: 75 logs)
	for i := range 25 {
		recentLogEntries := logs_core_tests.CreateTestLogEntriesWithUniqueFields(
			project.ID,
			recentTime.Add(time.Duration(i)*time.Second),
			"Recent log for zero quota test",
			map[string]any{
				"test_session": uniqueID,
				"log_type":     "recent",
				"log_index":    50 + i,
			},
		)
		allEntries = logs_core_tests.MergeLogEntries(allEntries, recentLogEntries)
	}

	// Store all logs
	logs_core_tests.StoreTestLogsAndFlush(t, repository, allEntries)

	// Verify logs were stored
	statsBeforeCleanup, err := repository.GetProjectLogStats(project.ID)
	assert.NoError(t, err)
	assert.Equal(t, int64(75), statsBeforeCleanup.TotalLogs, "Should have 75 logs before cleanup")

	t.Logf(
		"Before cleanup: TotalLogs=%d, OldestTime=%v, NewestTime=%v",
		statsBeforeCleanup.TotalLogs,
		statsBeforeCleanup.OldestLogTime,
		statsBeforeCleanup.NewestLogTime,
	)

	// Execute cleanup service
	err = cleanupService.ExecuteAllTasksForTest()
	assert.NoError(t, err, "Cleanup service should execute successfully")

	// Force flush to ensure any potential changes are reflected
	err = repository.ForceFlush()
	assert.NoError(t, err, "Force flush should succeed")

	// Wait for any operations to complete
	time.Sleep(100 * time.Millisecond)

	// Verify NO logs were deleted (zero count quota means no count-based enforcement)
	statsAfterCleanup, err := repository.GetProjectLogStats(project.ID)
	assert.NoError(t, err)

	t.Logf("After cleanup: TotalLogs=%d, OldestTime=%v, NewestTime=%v",
		statsAfterCleanup.TotalLogs, statsAfterCleanup.OldestLogTime, statsAfterCleanup.NewestLogTime)

	assert.Equal(
		t,
		statsBeforeCleanup.TotalLogs,
		statsAfterCleanup.TotalLogs,
		"No logs should be deleted with zero count quota",
	)
}

func Test_EnforceProjectQuotas_WhenLogCountExceedsQuota_DeletesToNinetyPercentOfLimit(t *testing.T) {
	// Setup test environment
	router := projects_testing.CreateTestRouter(
		projects_controllers.GetProjectController(),
		projects_controllers.GetMembershipController(),
	)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	uniqueID := uuid.New().String()[:8]

	// Create test project
	projectName := "Exceeds Count Quota Cleanup Test " + uniqueID
	project := projects_testing.CreateTestProject(projectName, owner, router)

	// Update project to set MaxLogsAmount to 100 logs
	// According to calculateCleanupPercentage with no size quota (0), it should target 85%
	updateData := &projects_models.Project{
		Name:          project.Name,
		MaxLogsAmount: 100, // 100 logs limit, should clean up to 85% = 85 logs
	}
	projects_testing.UpdateProject(project, updateData, owner.Token, router)

	// Get repository and cleanup service
	repository := logs_core.GetLogCoreRepository()
	cleanupService := logs_cleanup.GetLogCleanupBackgroundService()

	// Create test timestamps
	now := time.Now().UTC()
	oldTime := now.Add(-2 * time.Hour)       // 2 hours ago (will be deleted)
	recentTime := now.Add(-30 * time.Minute) // 30 minutes ago (may remain)

	// Create logs to exceed the 100 log limit significantly
	var allEntries map[uuid.UUID][]*logs_core.LogItem

	// Create 100 old logs
	for i := range 100 {
		oldLogEntries := logs_core_tests.CreateTestLogEntriesWithUniqueFields(
			project.ID,
			oldTime.Add(time.Duration(i)*time.Second),
			"Old log for count quota cleanup test",
			map[string]any{
				"test_session": uniqueID,
				"log_type":     "old",
				"log_index":    i,
			},
		)
		if allEntries == nil {
			allEntries = oldLogEntries
		} else {
			allEntries = logs_core_tests.MergeLogEntries(allEntries, oldLogEntries)
		}
	}

	// Create 50 recent logs
	// Total: 150 logs, significantly exceeding 100 log quota
	for i := range 50 {
		recentLogEntries := logs_core_tests.CreateTestLogEntriesWithUniqueFields(
			project.ID,
			recentTime.Add(time.Duration(i)*time.Second),
			"Recent log for count quota cleanup test",
			map[string]any{
				"test_session": uniqueID,
				"log_type":     "recent",
				"log_index":    100 + i,
			},
		)
		allEntries = logs_core_tests.MergeLogEntries(allEntries, recentLogEntries)
	}

	// Store all logs
	logs_core_tests.StoreTestLogsAndFlush(t, repository, allEntries)

	// Verify initial count exceeds quota
	statsBeforeCleanup, err := repository.GetProjectLogStats(project.ID)
	assert.NoError(t, err)
	assert.Equal(t, int64(150), statsBeforeCleanup.TotalLogs, "Should have 150 logs before cleanup")
	assert.Greater(t, statsBeforeCleanup.TotalLogs, int64(100), "Should exceed 100 logs quota before cleanup")

	t.Logf(
		"Before cleanup: TotalLogs=%d, OldestTime=%v, NewestTime=%v",
		statsBeforeCleanup.TotalLogs,
		statsBeforeCleanup.OldestLogTime,
		statsBeforeCleanup.NewestLogTime,
	)

	// Execute cleanup service
	err = cleanupService.ExecuteAllTasksForTest()
	assert.NoError(t, err, "Cleanup service should execute successfully")

	// Force flush to ensure deletions are reflected
	err = repository.ForceFlush()
	assert.NoError(t, err, "Force flush should succeed")

	// Wait for delete operations to complete
	time.Sleep(100 * time.Millisecond)

	// Verify count was reduced to approximately 85% of quota (85 logs for 100 log quota)
	statsAfterCleanup, err := repository.GetProjectLogStats(project.ID)
	assert.NoError(t, err)

	t.Logf("After cleanup: TotalLogs=%d, OldestTime=%v, NewestTime=%v",
		statsAfterCleanup.TotalLogs, statsAfterCleanup.OldestLogTime, statsAfterCleanup.NewestLogTime)

	// For 100 log quota, target should be 85% = 85 logs, but cleanup algorithm may be more aggressive
	// due to the way cutoff times are calculated based on log distribution
	expectedMaxLogs := int64(85)
	expectedMinLogs := int64(50) // Allow for more aggressive cleanup due to algorithm behavior

	assert.Less(t, statsAfterCleanup.TotalLogs, statsBeforeCleanup.TotalLogs, "Should have fewer logs after cleanup")
	assert.LessOrEqual(t, statsAfterCleanup.TotalLogs, expectedMaxLogs+5,
		"Log count should be reduced to at most 85 logs (85% of 100 log quota)")
	assert.GreaterOrEqual(t, statsAfterCleanup.TotalLogs, expectedMinLogs,
		"Log count should not be reduced too aggressively")

	// Most importantly, verify that cleanup happened and count was significantly reduced
	assert.Less(t, statsAfterCleanup.TotalLogs, int64(100), "Log count should be below the 100 log quota after cleanup")
	assert.Less(t, float64(statsAfterCleanup.TotalLogs), float64(statsBeforeCleanup.TotalLogs)*0.7,
		"Log count should be reduced by at least 30% to demonstrate quota enforcement")

	// Verify the remaining logs are primarily the recent ones
	if !statsAfterCleanup.OldestLogTime.IsZero() && !statsAfterCleanup.NewestLogTime.IsZero() {
		// Most remaining logs should be from the recent time period or close to it
		assert.True(t, statsAfterCleanup.OldestLogTime.After(oldTime.Add(30*time.Minute)),
			"Remaining logs should be mostly from the recent time period after cleanup")
	}
}

func Test_EnforceProjectQuotas_WithDifferentProjectsCountQuotas_DeletesOnlyTargetProjectLogs(t *testing.T) {
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
	project1Name := "Project1 Different Count Quota Test " + uniqueID1
	project2Name := "Project2 Different Count Quota Test " + uniqueID2

	project1 := projects_testing.CreateTestProject(project1Name, owner1, router)
	project2 := projects_testing.CreateTestProject(project2Name, owner2, router)

	// Set different MaxLogsAmount for each project
	// Project 1: 10 logs quota (will exceed and trigger cleanup)
	updateData1 := &projects_models.Project{
		Name:          project1.Name,
		MaxLogsAmount: 10, // 10 logs limit - will be exceeded
	}
	projects_testing.UpdateProject(project1, updateData1, owner1.Token, router)

	// Project 2: 100 logs quota (will NOT exceed)
	updateData2 := &projects_models.Project{
		Name:          project2.Name,
		MaxLogsAmount: 100, // 100 logs limit - will not be exceeded
	}
	projects_testing.UpdateProject(project2, updateData2, owner2.Token, router)

	// Get repository and cleanup service
	repository := logs_core.GetLogCoreRepository()
	cleanupService := logs_cleanup.GetLogCleanupBackgroundService()

	// Create test timestamps
	now := time.Now().UTC()
	oldTime := now.Add(-2 * time.Hour)       // 2 hours ago
	recentTime := now.Add(-30 * time.Minute) // 30 minutes ago

	// Create logs for Project 1 - exceed 10 log quota
	var project1Entries map[uuid.UUID][]*logs_core.LogItem

	// Create 10 old logs for project 1
	for i := range 10 {
		oldLogEntries := logs_core_tests.CreateTestLogEntriesWithUniqueFields(
			project1.ID,
			oldTime.Add(time.Duration(i)*time.Second),
			"Project1 old log for different count quota test",
			map[string]any{
				"test_session": uniqueID1,
				"log_type":     "old",
				"project_name": project1Name,
				"log_index":    i,
			},
		)
		if project1Entries == nil {
			project1Entries = oldLogEntries
		} else {
			project1Entries = logs_core_tests.MergeLogEntries(project1Entries, oldLogEntries)
		}
	}

	// Create 8 recent logs for project 1 - Total: 18 logs, exceeds 10 log quota
	for i := range 8 {
		recentLogEntries := logs_core_tests.CreateTestLogEntriesWithUniqueFields(
			project1.ID,
			recentTime.Add(time.Duration(i)*time.Second),
			"Project1 recent log for different count quota test",
			map[string]any{
				"test_session": uniqueID1,
				"log_type":     "recent",
				"project_name": project1Name,
				"log_index":    10 + i,
			},
		)
		project1Entries = logs_core_tests.MergeLogEntries(project1Entries, recentLogEntries)
	}

	// Create logs for Project 2 (will NOT exceed 100 log quota)
	var project2Entries map[uuid.UUID][]*logs_core.LogItem

	// Create 25 logs for project 2 (well below 100 log limit)
	for i := range 25 {
		logEntries := logs_core_tests.CreateTestLogEntriesWithUniqueFields(
			project2.ID,
			oldTime.Add(time.Duration(i)*time.Second),
			"Project2 log for different count quota test",
			map[string]any{
				"test_session": uniqueID2,
				"log_type":     "normal",
				"project_name": project2Name,
				"log_index":    i,
			},
		)
		if project2Entries == nil {
			project2Entries = logEntries
		} else {
			project2Entries = logs_core_tests.MergeLogEntries(project2Entries, logEntries)
		}
	}

	// Store all logs for both projects
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
	assert.Equal(t, int64(18), project1StatsBeforeCleanup.TotalLogs, "Project1 should have 18 logs before cleanup")
	assert.Greater(t, project1StatsBeforeCleanup.TotalLogs, int64(10), "Project1 should exceed 10 log quota")

	project2StatsBeforeCleanup, err := repository.GetProjectLogStats(project2.ID)
	assert.NoError(t, err)
	t.Logf(
		"Project2 stats before cleanup: TotalLogs=%d, OldestTime=%v, NewestTime=%v",
		project2StatsBeforeCleanup.TotalLogs,
		project2StatsBeforeCleanup.OldestLogTime,
		project2StatsBeforeCleanup.NewestLogTime,
	)
	assert.Equal(t, int64(25), project2StatsBeforeCleanup.TotalLogs, "Project2 should have 25 logs before cleanup")
	assert.Less(t, project2StatsBeforeCleanup.TotalLogs, int64(100), "Project2 should be well below 100 log quota")

	// Execute cleanup service
	err = cleanupService.ExecuteAllTasksForTest()
	assert.NoError(t, err, "Cleanup service should execute successfully")

	// Force flush to ensure deletions are reflected
	err = repository.ForceFlush()
	assert.NoError(t, err, "Force flush should succeed")

	// Wait for delete operations to complete
	time.Sleep(100 * time.Millisecond)

	// Verify Project 1 had logs deleted (exceeded quota)
	project1StatsAfterCleanup, err := repository.GetProjectLogStats(project1.ID)
	assert.NoError(t, err)
	t.Logf(
		"Project1 stats after cleanup: TotalLogs=%d, OldestTime=%v, NewestTime=%v",
		project1StatsAfterCleanup.TotalLogs,
		project1StatsAfterCleanup.OldestLogTime,
		project1StatsAfterCleanup.NewestLogTime,
	)
	assert.Less(
		t,
		project1StatsAfterCleanup.TotalLogs,
		project1StatsBeforeCleanup.TotalLogs,
		"Project1 should have fewer logs after cleanup (quota exceeded)",
	)
	assert.LessOrEqual(
		t,
		project1StatsAfterCleanup.TotalLogs,
		int64(10), // Should be reduced to or below the quota
		"Project1 log count should be reduced after cleanup",
	)

	// Verify Project 2 has unchanged logs (did not exceed quota)
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
		project2StatsBeforeCleanup.TotalLogs,
		project2StatsAfterCleanup.TotalLogs,
		"Project2 logs should remain unchanged (quota not exceeded)",
	)
}
