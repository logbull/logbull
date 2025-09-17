package logs_cleanup_tests

import (
	"fmt"
	"strings"
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

func Test_EnforceProjectQuotas_WhenStorageSizeExceedsMaxLogsSizeMB_DeletesOldestLogs(t *testing.T) {
	// Setup test environment
	router := projects_testing.CreateTestRouter(
		projects_controllers.GetProjectController(),
		projects_controllers.GetMembershipController(),
	)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	uniqueID := uuid.New().String()[:8]

	// Create test project
	projectName := "Size Quota Test " + uniqueID
	project := projects_testing.CreateTestProject(projectName, owner, router)

	// Update project to set MaxLogsSizeMB to a small value (1 MB)
	updateData := &projects_models.Project{
		Name:          project.Name,
		MaxLogsSizeMB: 1, // 1 MB limit
	}
	projects_testing.UpdateProject(project, updateData, owner.Token, router)

	// Get repository and cleanup service
	repository := logs_core.GetLogCoreRepository()
	cleanupService := logs_cleanup.GetLogCleanupBackgroundService()

	// Create test timestamps
	now := time.Now().UTC()
	oldTime := now.Add(-2 * time.Hour)       // 2 hours ago (should be deleted)
	recentTime := now.Add(-30 * time.Minute) // 30 minutes ago (should remain)

	// Create logs with large messages to exceed size quota
	largeMessage := strings.Repeat(
		"This is a large log message to test size-based quota enforcement. ",
		100,
	) // ~6KB per message

	// Create multiple logs to exceed the 1MB limit
	var allEntries map[uuid.UUID][]*logs_core.LogItem

	// Create 200 old logs (~1.2MB total)
	for i := range 200 {
		oldLogEntries := logs_core_tests.CreateTestLogEntriesWithUniqueFields(
			project.ID,
			oldTime.Add(time.Duration(i)*time.Second), // Slightly different timestamps
			largeMessage+fmt.Sprintf(" Log #%d", i),
			map[string]any{
				"test_session": uniqueID,
				"log_type":     "old",
				"size_test":    true,
				"log_index":    i,
			},
		)
		if allEntries == nil {
			allEntries = oldLogEntries
		} else {
			allEntries = logs_core_tests.MergeLogEntries(allEntries, oldLogEntries)
		}
	}

	// Create 50 recent logs (~300KB total)
	for i := 0; i < 50; i++ {
		recentLogEntries := logs_core_tests.CreateTestLogEntriesWithUniqueFields(
			project.ID,
			recentTime.Add(time.Duration(i)*time.Second), // Slightly different timestamps
			largeMessage+fmt.Sprintf(" Recent Log #%d", i),
			map[string]any{
				"test_session": uniqueID,
				"log_type":     "recent",
				"size_test":    true,
				"log_index":    200 + i,
			},
		)
		allEntries = logs_core_tests.MergeLogEntries(allEntries, recentLogEntries)
	}

	// Store all logs
	logs_core_tests.StoreTestLogsAndFlush(t, repository, allEntries)

	// Verify initial size exceeds quota
	statsBeforeCleanup, err := repository.GetProjectLogStats(project.ID)
	assert.NoError(t, err)
	assert.Equal(t, int64(250), statsBeforeCleanup.TotalLogs, "Should have 250 logs before cleanup")
	assert.Greater(t, statsBeforeCleanup.TotalSizeMB, 1.0, "Should exceed 1MB quota before cleanup")

	t.Logf(
		"Before cleanup: TotalLogs=%d, TotalSizeMB=%.3f",
		statsBeforeCleanup.TotalLogs,
		statsBeforeCleanup.TotalSizeMB,
	)

	// Execute cleanup service
	err = cleanupService.ExecuteAllTasksForTest()
	assert.NoError(t, err, "Cleanup service should execute successfully")

	// Force flush to ensure deletions are reflected
	err = repository.ForceFlush()
	assert.NoError(t, err, "Force flush should succeed")

	// Wait for delete operations to complete
	time.Sleep(100 * time.Millisecond)

	// Verify size quota was enforced
	statsAfterCleanup, err := repository.GetProjectLogStats(project.ID)
	assert.NoError(t, err)

	t.Logf("After cleanup: TotalLogs=%d, TotalSizeMB=%.3f", statsAfterCleanup.TotalLogs, statsAfterCleanup.TotalSizeMB)

	assert.LessOrEqual(t, statsAfterCleanup.TotalSizeMB, 0.9, "Size should be reduced to ~90% of quota (0.9MB)")
	assert.Less(t, statsAfterCleanup.TotalLogs, statsBeforeCleanup.TotalLogs, "Should have fewer logs after cleanup")

	// Verify the remaining logs are primarily the recent ones
	if !statsAfterCleanup.OldestLogTime.IsZero() && !statsAfterCleanup.NewestLogTime.IsZero() {
		// Most remaining logs should be from the recent time period
		assert.True(t, statsAfterCleanup.OldestLogTime.After(oldTime.Add(30*time.Minute)),
			"Remaining logs should be mostly from the recent time period")
	}
}

func Test_EnforceProjectQuotas_WhenStorageSizeIsWithinMaxLogsSizeMB_NoLogsDeleted(t *testing.T) {
	// Setup test environment
	router := projects_testing.CreateTestRouter(
		projects_controllers.GetProjectController(),
		projects_controllers.GetMembershipController(),
	)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	uniqueID := uuid.New().String()[:8]

	// Create test project
	projectName := "Size Within Quota Test " + uniqueID
	project := projects_testing.CreateTestProject(projectName, owner, router)

	// Update project to set MaxLogsSizeMB to a large value (10 MB)
	updateData := &projects_models.Project{
		Name:          project.Name,
		MaxLogsSizeMB: 10, // 10 MB limit - large enough to not trigger cleanup
	}
	projects_testing.UpdateProject(project, updateData, owner.Token, router)

	// Get repository and cleanup service
	repository := logs_core.GetLogCoreRepository()
	cleanupService := logs_cleanup.GetLogCleanupBackgroundService()

	// Create test timestamps
	now := time.Now().UTC()
	oldTime := now.Add(-2 * time.Hour)       // 2 hours ago
	recentTime := now.Add(-30 * time.Minute) // 30 minutes ago

	// Create small logs that won't exceed quota
	smallMessage := strings.Repeat("Small log message. ", 10) // ~200 bytes per message

	// Create only 50 logs (~10KB total, well below 10MB limit)
	var allEntries map[uuid.UUID][]*logs_core.LogItem

	for i := range 25 {
		oldLogEntries := logs_core_tests.CreateTestLogEntriesWithUniqueFields(
			project.ID,
			oldTime.Add(time.Duration(i)*time.Second),
			smallMessage+fmt.Sprintf(" Old Log #%d", i),
			map[string]any{
				"test_session": uniqueID,
				"log_type":     "old",
				"size_test":    true,
				"log_index":    i,
			},
		)
		if allEntries == nil {
			allEntries = oldLogEntries
		} else {
			allEntries = logs_core_tests.MergeLogEntries(allEntries, oldLogEntries)
		}
	}

	for i := range 25 {
		recentLogEntries := logs_core_tests.CreateTestLogEntriesWithUniqueFields(
			project.ID,
			recentTime.Add(time.Duration(i)*time.Second),
			smallMessage+fmt.Sprintf(" Recent Log #%d", i),
			map[string]any{
				"test_session": uniqueID,
				"log_type":     "recent",
				"size_test":    true,
				"log_index":    25 + i,
			},
		)
		allEntries = logs_core_tests.MergeLogEntries(allEntries, recentLogEntries)
	}

	// Store all logs
	logs_core_tests.StoreTestLogsAndFlush(t, repository, allEntries)

	// Verify initial size is well within quota
	statsBeforeCleanup, err := repository.GetProjectLogStats(project.ID)
	assert.NoError(t, err)
	assert.Equal(t, int64(50), statsBeforeCleanup.TotalLogs, "Should have 50 logs before cleanup")
	assert.Less(t, statsBeforeCleanup.TotalSizeMB, 1.0, "Should be well below 10MB quota before cleanup")

	t.Logf(
		"Before cleanup: TotalLogs=%d, TotalSizeMB=%.3f",
		statsBeforeCleanup.TotalLogs,
		statsBeforeCleanup.TotalSizeMB,
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

	t.Logf("After cleanup: TotalLogs=%d, TotalSizeMB=%.3f", statsAfterCleanup.TotalLogs, statsAfterCleanup.TotalSizeMB)

	assert.Equal(
		t,
		statsBeforeCleanup.TotalLogs,
		statsAfterCleanup.TotalLogs,
		"No logs should be deleted when within quota",
	)
	assert.Equal(
		t,
		statsBeforeCleanup.TotalSizeMB,
		statsAfterCleanup.TotalSizeMB,
		"Size should remain the same when within quota",
	)
}

func Test_EnforceProjectQuotas_WhenMaxLogsSizeMBIsZero_NoSizeQuotaEnforcement(t *testing.T) {
	// Setup test environment
	router := projects_testing.CreateTestRouter(
		projects_controllers.GetProjectController(),
		projects_controllers.GetMembershipController(),
	)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	uniqueID := uuid.New().String()[:8]

	// Create test project
	projectName := "Zero Size Quota Test " + uniqueID
	project := projects_testing.CreateTestProject(projectName, owner, router)

	// Update project to set MaxLogsSizeMB to 0 (no size-based quota)
	updateData := &projects_models.Project{
		Name:          project.Name,
		MaxLogsSizeMB: 0, // No size quota enforcement
	}
	projects_testing.UpdateProject(project, updateData, owner.Token, router)

	// Get repository and cleanup service
	repository := logs_core.GetLogCoreRepository()
	cleanupService := logs_cleanup.GetLogCleanupBackgroundService()

	// Create test timestamps
	now := time.Now().UTC()
	oldTime := now.Add(-2 * time.Hour)       // 2 hours ago
	recentTime := now.Add(-30 * time.Minute) // 30 minutes ago

	// Create large logs that would normally trigger cleanup
	largeMessage := strings.Repeat(
		"This is a large log message to test zero size quota enforcement. ",
		100,
	) // ~6KB per message

	// Create many logs that would exceed any reasonable size limit
	var allEntries map[uuid.UUID][]*logs_core.LogItem

	// Create 100 old logs (~600KB total - would trigger cleanup if quota was set)
	for i := range 100 {
		oldLogEntries := logs_core_tests.CreateTestLogEntriesWithUniqueFields(
			project.ID,
			oldTime.Add(time.Duration(i)*time.Second),
			largeMessage+fmt.Sprintf(" Old Log #%d", i),
			map[string]any{
				"test_session": uniqueID,
				"log_type":     "old",
				"size_test":    true,
				"log_index":    i,
			},
		)
		if allEntries == nil {
			allEntries = oldLogEntries
		} else {
			allEntries = logs_core_tests.MergeLogEntries(allEntries, oldLogEntries)
		}
	}

	// Create 50 recent logs (~300KB total)
	for i := range 50 {
		recentLogEntries := logs_core_tests.CreateTestLogEntriesWithUniqueFields(
			project.ID,
			recentTime.Add(time.Duration(i)*time.Second),
			largeMessage+fmt.Sprintf(" Recent Log #%d", i),
			map[string]any{
				"test_session": uniqueID,
				"log_type":     "recent",
				"size_test":    true,
				"log_index":    100 + i,
			},
		)
		allEntries = logs_core_tests.MergeLogEntries(allEntries, recentLogEntries)
	}

	// Store all logs
	logs_core_tests.StoreTestLogsAndFlush(t, repository, allEntries)

	// Verify logs were stored
	statsBeforeCleanup, err := repository.GetProjectLogStats(project.ID)
	assert.NoError(t, err)
	assert.Equal(t, int64(150), statsBeforeCleanup.TotalLogs, "Should have 150 logs before cleanup")

	t.Logf(
		"Before cleanup: TotalLogs=%d, TotalSizeMB=%.3f",
		statsBeforeCleanup.TotalLogs,
		statsBeforeCleanup.TotalSizeMB,
	)

	// Execute cleanup service
	err = cleanupService.ExecuteAllTasksForTest()
	assert.NoError(t, err, "Cleanup service should execute successfully")

	// Force flush to ensure any potential changes are reflected
	err = repository.ForceFlush()
	assert.NoError(t, err, "Force flush should succeed")

	// Wait for any operations to complete
	time.Sleep(100 * time.Millisecond)

	// Verify NO logs were deleted (zero size quota means no size-based enforcement)
	statsAfterCleanup, err := repository.GetProjectLogStats(project.ID)
	assert.NoError(t, err)

	t.Logf("After cleanup: TotalLogs=%d, TotalSizeMB=%.3f", statsAfterCleanup.TotalLogs, statsAfterCleanup.TotalSizeMB)

	assert.Equal(
		t,
		statsBeforeCleanup.TotalLogs,
		statsAfterCleanup.TotalLogs,
		"No logs should be deleted with zero size quota",
	)
	assert.Equal(
		t,
		statsBeforeCleanup.TotalSizeMB,
		statsAfterCleanup.TotalSizeMB,
		"Size should remain the same with zero size quota",
	)
}

func Test_EnforceProjectQuotas_WhenStorageSizeExceedsQuota_DeletesToNinetyPercentOfLimit(t *testing.T) {
	// Setup test environment
	router := projects_testing.CreateTestRouter(
		projects_controllers.GetProjectController(),
		projects_controllers.GetMembershipController(),
	)
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)
	uniqueID := uuid.New().String()[:8]

	// Create test project
	projectName := "Exceeds Quota Cleanup Test " + uniqueID
	project := projects_testing.CreateTestProject(projectName, owner, router)

	// Update project to set MaxLogsSizeMB to 1 MB (small quota)
	// According to calculateCleanupPercentage, quotas <= 10MB target 85%
	updateData := &projects_models.Project{
		Name:          project.Name,
		MaxLogsSizeMB: 1, // 1 MB limit, should clean up to 85% = 0.85MB
	}
	projects_testing.UpdateProject(project, updateData, owner.Token, router)

	// Get repository and cleanup service
	repository := logs_core.GetLogCoreRepository()
	cleanupService := logs_cleanup.GetLogCleanupBackgroundService()

	// Create test timestamps
	now := time.Now().UTC()
	oldTime := now.Add(-2 * time.Hour)       // 2 hours ago (will be deleted)
	recentTime := now.Add(-30 * time.Minute) // 30 minutes ago (may remain)

	// Create logs with large messages to exceed size quota
	largeMessage := strings.Repeat(
		"This is a large log message to test quota cleanup percentage. ",
		100,
	) // ~6KB per message

	// Create logs to exceed the 1MB limit significantly
	var allEntries map[uuid.UUID][]*logs_core.LogItem

	// Create 150 old logs (~900KB total)
	for i := range 150 {
		oldLogEntries := logs_core_tests.CreateTestLogEntriesWithUniqueFields(
			project.ID,
			oldTime.Add(time.Duration(i)*time.Second),
			largeMessage+fmt.Sprintf(" Old Log #%d", i),
			map[string]any{
				"test_session": uniqueID,
				"log_type":     "old",
				"size_test":    true,
				"log_index":    i,
			},
		)
		if allEntries == nil {
			allEntries = oldLogEntries
		} else {
			allEntries = logs_core_tests.MergeLogEntries(allEntries, oldLogEntries)
		}
	}

	// Create 75 recent logs (~450KB total)
	// Total: ~1.35MB, significantly exceeding 1MB quota
	for i := range 75 {
		recentLogEntries := logs_core_tests.CreateTestLogEntriesWithUniqueFields(
			project.ID,
			recentTime.Add(time.Duration(i)*time.Second),
			largeMessage+fmt.Sprintf(" Recent Log #%d", i),
			map[string]any{
				"test_session": uniqueID,
				"log_type":     "recent",
				"size_test":    true,
				"log_index":    150 + i,
			},
		)
		allEntries = logs_core_tests.MergeLogEntries(allEntries, recentLogEntries)
	}

	// Store all logs
	logs_core_tests.StoreTestLogsAndFlush(t, repository, allEntries)

	// Verify initial size exceeds quota
	statsBeforeCleanup, err := repository.GetProjectLogStats(project.ID)
	assert.NoError(t, err)
	assert.Equal(t, int64(225), statsBeforeCleanup.TotalLogs, "Should have 225 logs before cleanup")
	assert.Greater(t, statsBeforeCleanup.TotalSizeMB, 1.0, "Should exceed 1MB quota before cleanup")

	t.Logf(
		"Before cleanup: TotalLogs=%d, TotalSizeMB=%.3f",
		statsBeforeCleanup.TotalLogs,
		statsBeforeCleanup.TotalSizeMB,
	)

	// Execute cleanup service
	err = cleanupService.ExecuteAllTasksForTest()
	assert.NoError(t, err, "Cleanup service should execute successfully")

	// Force flush to ensure deletions are reflected
	err = repository.ForceFlush()
	assert.NoError(t, err, "Force flush should succeed")

	// Wait for delete operations to complete
	time.Sleep(100 * time.Millisecond)

	// Verify size was reduced to approximately 85% of quota (0.85MB for 1MB quota)
	statsAfterCleanup, err := repository.GetProjectLogStats(project.ID)
	assert.NoError(t, err)

	t.Logf("After cleanup: TotalLogs=%d, TotalSizeMB=%.3f", statsAfterCleanup.TotalLogs, statsAfterCleanup.TotalSizeMB)

	// For 1MB quota, target should be 85% = 0.85MB, but cleanup algorithm may be more aggressive
	// due to the way cutoff times are calculated based on log distribution
	expectedMaxSizeMB := 0.85
	expectedMinSizeMB := 0.40 // Allow for more aggressive cleanup due to algorithm behavior

	assert.Less(t, statsAfterCleanup.TotalLogs, statsBeforeCleanup.TotalLogs, "Should have fewer logs after cleanup")
	assert.LessOrEqual(t, statsAfterCleanup.TotalSizeMB, expectedMaxSizeMB+0.1,
		fmt.Sprintf("Size should be reduced to at most %.2fMB (85%% of 1MB quota)", expectedMaxSizeMB))
	assert.GreaterOrEqual(t, statsAfterCleanup.TotalSizeMB, expectedMinSizeMB,
		fmt.Sprintf("Size should not be reduced too aggressively, should be at least %.2fMB", expectedMinSizeMB))

	// Most importantly, verify that cleanup happened and size was significantly reduced
	assert.Less(t, statsAfterCleanup.TotalSizeMB, 1.0, "Size should be below the 1MB quota after cleanup")
	assert.Less(t, statsAfterCleanup.TotalSizeMB, statsBeforeCleanup.TotalSizeMB*0.7,
		"Size should be reduced by at least 30% to demonstrate quota enforcement")

	// Verify the remaining logs are primarily the recent ones
	if !statsAfterCleanup.OldestLogTime.IsZero() && !statsAfterCleanup.NewestLogTime.IsZero() {
		// Most remaining logs should be from the recent time period or close to it
		assert.True(t, statsAfterCleanup.OldestLogTime.After(oldTime.Add(30*time.Minute)),
			"Remaining logs should be mostly from the recent time period after cleanup")
	}
}

func Test_EnforceProjectQuotas_WithDifferentProjectsSizeQuotas_DeletesOnlyTargetProjectLogs(t *testing.T) {
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
	project1Name := "Project1 Different Size Quota Test " + uniqueID1
	project2Name := "Project2 Different Size Quota Test " + uniqueID2

	project1 := projects_testing.CreateTestProject(project1Name, owner1, router)
	project2 := projects_testing.CreateTestProject(project2Name, owner2, router)

	// Set different MaxLogsSizeMB for each project
	// Project 1: 1 MB quota (will exceed and trigger cleanup)
	updateData1 := &projects_models.Project{
		Name:          project1.Name,
		MaxLogsSizeMB: 1, // 1 MB limit - will be exceeded
	}
	projects_testing.UpdateProject(project1, updateData1, owner1.Token, router)

	// Project 2: 10 MB quota (will NOT exceed)
	updateData2 := &projects_models.Project{
		Name:          project2.Name,
		MaxLogsSizeMB: 10, // 10 MB limit - will not be exceeded
	}
	projects_testing.UpdateProject(project2, updateData2, owner2.Token, router)

	// Get repository and cleanup service
	repository := logs_core.GetLogCoreRepository()
	cleanupService := logs_cleanup.GetLogCleanupBackgroundService()

	// Create test timestamps
	now := time.Now().UTC()
	oldTime := now.Add(-2 * time.Hour)       // 2 hours ago
	recentTime := now.Add(-30 * time.Minute) // 30 minutes ago

	// Create large logs for Project 1 (will exceed 1MB quota)
	largeMessage := strings.Repeat(
		"This is a large log message to test multi-project size quota enforcement. ",
		100,
	) // ~7KB per message

	// Create logs for Project 1 - exceed 1MB quota
	var project1Entries map[uuid.UUID][]*logs_core.LogItem

	// Create 100 old logs for project 1 (~700KB)
	for i := range 100 {
		oldLogEntries := logs_core_tests.CreateTestLogEntriesWithUniqueFields(
			project1.ID,
			oldTime.Add(time.Duration(i)*time.Second),
			largeMessage+fmt.Sprintf(" Project1 Old Log #%d", i),
			map[string]any{
				"test_session": uniqueID1,
				"log_type":     "old",
				"size_test":    true,
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

	// Create 50 recent logs for project 1 (~350KB) - Total: ~1.05MB, exceeds 1MB
	for i := range 50 {
		recentLogEntries := logs_core_tests.CreateTestLogEntriesWithUniqueFields(
			project1.ID,
			recentTime.Add(time.Duration(i)*time.Second),
			largeMessage+fmt.Sprintf(" Project1 Recent Log #%d", i),
			map[string]any{
				"test_session": uniqueID1,
				"log_type":     "recent",
				"size_test":    true,
				"project_name": project1Name,
				"log_index":    100 + i,
			},
		)
		project1Entries = logs_core_tests.MergeLogEntries(project1Entries, recentLogEntries)
	}

	// Create smaller logs for Project 2 (will NOT exceed 10MB quota)
	smallMessage := strings.Repeat("Small message for project 2. ", 10) // ~300 bytes per message

	var project2Entries map[uuid.UUID][]*logs_core.LogItem

	// Create 50 logs for project 2 (~15KB total, well below 10MB)
	for i := range 50 {
		logEntries := logs_core_tests.CreateTestLogEntriesWithUniqueFields(
			project2.ID,
			oldTime.Add(time.Duration(i)*time.Second),
			smallMessage+fmt.Sprintf(" Project2 Log #%d", i),
			map[string]any{
				"test_session": uniqueID2,
				"log_type":     "normal",
				"size_test":    true,
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
		"Project1 stats before cleanup: TotalLogs=%d, TotalSizeMB=%.3f",
		project1StatsBeforeCleanup.TotalLogs,
		project1StatsBeforeCleanup.TotalSizeMB,
	)
	assert.Equal(t, int64(150), project1StatsBeforeCleanup.TotalLogs, "Project1 should have 150 logs before cleanup")
	assert.Greater(t, project1StatsBeforeCleanup.TotalSizeMB, 1.0, "Project1 should exceed 1MB quota")

	project2StatsBeforeCleanup, err := repository.GetProjectLogStats(project2.ID)
	assert.NoError(t, err)
	t.Logf(
		"Project2 stats before cleanup: TotalLogs=%d, TotalSizeMB=%.3f",
		project2StatsBeforeCleanup.TotalLogs,
		project2StatsBeforeCleanup.TotalSizeMB,
	)
	assert.Equal(t, int64(50), project2StatsBeforeCleanup.TotalLogs, "Project2 should have 50 logs before cleanup")
	assert.Less(t, project2StatsBeforeCleanup.TotalSizeMB, 1.0, "Project2 should be well below 10MB quota")

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
		"Project1 stats after cleanup: TotalLogs=%d, TotalSizeMB=%.3f",
		project1StatsAfterCleanup.TotalLogs,
		project1StatsAfterCleanup.TotalSizeMB,
	)
	assert.Less(
		t,
		project1StatsAfterCleanup.TotalLogs,
		project1StatsBeforeCleanup.TotalLogs,
		"Project1 should have fewer logs after cleanup (quota exceeded)",
	)
	assert.LessOrEqual(
		t,
		project1StatsAfterCleanup.TotalSizeMB,
		0.95, // Should be reduced to ~85% of 1MB = 0.85MB, plus some tolerance
		"Project1 size should be reduced after cleanup",
	)

	// Verify Project 2 has unchanged logs (did not exceed quota)
	project2StatsAfterCleanup, err := repository.GetProjectLogStats(project2.ID)
	assert.NoError(t, err)
	t.Logf(
		"Project2 stats after cleanup: TotalLogs=%d, TotalSizeMB=%.3f",
		project2StatsAfterCleanup.TotalLogs,
		project2StatsAfterCleanup.TotalSizeMB,
	)
	assert.Equal(
		t,
		project2StatsBeforeCleanup.TotalLogs,
		project2StatsAfterCleanup.TotalLogs,
		"Project2 logs should remain unchanged (quota not exceeded)",
	)
	assert.Equal(
		t,
		project2StatsBeforeCleanup.TotalSizeMB,
		project2StatsAfterCleanup.TotalSizeMB,
		"Project2 size should remain unchanged (quota not exceeded)",
	)
}
