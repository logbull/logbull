package logs_core_tests

import (
	"math"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	logs_core "logbull/internal/features/logs/core"
)

func Test_GetProjectLogStats_WithMultipleLogs_ReturnsCorrectStats(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	projectID := uuid.New()
	uniqueTestSession := uuid.New().String()[:8]
	baseTime := time.Now().UTC()

	// Create logs at different timestamps
	oldTime := baseTime.Add(-2 * time.Hour)
	recentTime := baseTime.Add(-1 * time.Hour)
	newestTime := baseTime.Add(-30 * time.Minute)

	oldLogEntries := CreateTestLogEntriesWithUniqueFields(projectID, oldTime,
		"Old log for stats test", map[string]any{
			"test_session": uniqueTestSession,
			"log_order":    1,
		})

	recentLogEntries := CreateTestLogEntriesWithUniqueFields(projectID, recentTime,
		"Recent log for stats test", map[string]any{
			"test_session": uniqueTestSession,
			"log_order":    2,
		})

	newestLogEntries := CreateTestLogEntriesWithUniqueFields(projectID, newestTime,
		"Newest log for stats test", map[string]any{
			"test_session": uniqueTestSession,
			"log_order":    3,
		})

	allEntries := MergeLogEntries(oldLogEntries, recentLogEntries)
	allEntries = MergeLogEntries(allEntries, newestLogEntries)
	StoreTestLogsAndFlush(t, repository, allEntries)

	stats, err := repository.GetProjectLogStats(projectID)
	assert.NoError(t, err)
	assert.NotNil(t, stats)

	assert.Equal(t, int64(3), stats.TotalLogs, "Should have 3 total logs")
	assert.Equal(t, float64(0), math.Round(stats.TotalSizeMB*100)/100, "TotalSizeMB should be 0")

	// Verify oldest and newest timestamps (allow some tolerance for timestamp precision)
	timeTolerance := 10 * time.Second
	assert.WithinDuration(t, oldTime, stats.OldestLogTime, timeTolerance,
		"Oldest log time should match the earliest log timestamp")
	assert.WithinDuration(t, newestTime, stats.NewestLogTime, timeTolerance,
		"Newest log time should match the latest log timestamp")
}

func Test_GetProjectLogStats_WithNoLogs_ReturnsZeroStats(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	projectID := uuid.New() // Empty project with no logs

	stats, err := repository.GetProjectLogStats(projectID)
	assert.NoError(t, err)
	assert.NotNil(t, stats)

	assert.Equal(t, int64(0), stats.TotalLogs, "Should have 0 total logs for empty project")
	assert.Equal(t, float64(0), stats.TotalSizeMB, "TotalSizeMB should be 0")
	assert.True(t, stats.OldestLogTime.IsZero(), "OldestLogTime should be zero time for empty project")
	assert.True(t, stats.NewestLogTime.IsZero(), "NewestLogTime should be zero time for empty project")
}

func Test_GetProjectLogStats_WithSingleLog_ReturnsCorrectStats(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	projectID := uuid.New()
	uniqueTestSession := uuid.New().String()[:8]
	logTime := time.Now().UTC()

	singleLogEntries := CreateTestLogEntriesWithUniqueFields(projectID, logTime,
		"Single log for stats test", map[string]any{
			"test_session": uniqueTestSession,
			"single_log":   true,
		})

	StoreTestLogsAndFlush(t, repository, singleLogEntries)

	stats, err := repository.GetProjectLogStats(projectID)
	assert.NoError(t, err)
	assert.NotNil(t, stats)

	assert.Equal(t, int64(1), stats.TotalLogs, "Should have 1 total log")
	assert.Equal(t, float64(0), math.Round(stats.TotalSizeMB*100)/100, "TotalSizeMB should be 0")

	// For single log, oldest and newest should be the same
	timeTolerance := 10 * time.Second
	assert.WithinDuration(t, logTime, stats.OldestLogTime, timeTolerance,
		"Oldest log time should match the single log timestamp")
	assert.WithinDuration(t, logTime, stats.NewestLogTime, timeTolerance,
		"Newest log time should match the single log timestamp")
}
