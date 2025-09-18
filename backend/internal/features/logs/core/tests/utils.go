package logs_core_tests

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	logs_core "logbull/internal/features/logs/core"
)

func CreateTestLogEntries() map[uuid.UUID][]*logs_core.LogItem {
	projectID := uuid.New()
	logID := uuid.New()
	currentTime := time.Now().UTC()

	testLogItem := &logs_core.LogItem{
		ID:        logID,
		ProjectID: projectID,
		Timestamp: currentTime,
		Level:     logs_core.LogLevelInfo,
		Message:   "Test log message",
		Fields: map[string]any{
			"component": "test",
			"action":    "test_action",
		},
		ClientIP: "127.0.0.1",
	}

	return map[uuid.UUID][]*logs_core.LogItem{
		projectID: {testLogItem},
	}
}

func CreateTestLogEntriesWithTimestamp(
	projectID uuid.UUID,
	timestamp time.Time,
	message string,
) map[uuid.UUID][]*logs_core.LogItem {
	logEntryID := uuid.New()

	testLogEntry := &logs_core.LogItem{
		ID:        logEntryID,
		ProjectID: projectID,
		Timestamp: timestamp,
		Level:     logs_core.LogLevelInfo,
		Message:   message,
		Fields: map[string]any{
			"component": "test",
			"action":    "test_action",
		},
		ClientIP: "127.0.0.1",
	}

	return map[uuid.UUID][]*logs_core.LogItem{
		projectID: {testLogEntry},
	}
}

func CreateTestLogEntriesWithMessageAndFields(
	projectID uuid.UUID,
	timestamp time.Time,
	message string,
	fields map[string]any,
) map[uuid.UUID][]*logs_core.LogItem {
	logEntryID := uuid.New()

	testLogEntryWithFields := &logs_core.LogItem{
		ID:        logEntryID,
		ProjectID: projectID,
		Timestamp: timestamp,
		Level:     logs_core.LogLevelInfo,
		Message:   message,
		Fields:    fields,
		ClientIP:  "127.0.0.1",
	}

	return map[uuid.UUID][]*logs_core.LogItem{
		projectID: {testLogEntryWithFields},
	}
}

func GetFirstProjectID(testLogEntries map[uuid.UUID][]*logs_core.LogItem) uuid.UUID {
	for projectID := range testLogEntries {
		return projectID
	}
	return uuid.UUID{}
}

func StoreTestLogsAndFlush(
	t *testing.T,
	repository *logs_core.LogCoreRepository,
	testLogEntries map[uuid.UUID][]*logs_core.LogItem,
) {
	storeErr := repository.StoreLogsBatch(testLogEntries)
	assert.NoError(t, storeErr, "Failed to store test data")

	flushErr := repository.ForceFlush()
	assert.NoError(t, flushErr, "Failed to refresh index")
}

func CreateBatchLogEntries(
	projectID uuid.UUID,
	logCount int,
	baseTime time.Time,
	testSessionID string,
) map[uuid.UUID][]*logs_core.LogItem {
	var allBatchEntries map[uuid.UUID][]*logs_core.LogItem

	for sequenceIndex := 1; sequenceIndex <= logCount; sequenceIndex++ {
		uniqueLogID := uuid.New().String()[:8]
		batchLogEntries := CreateTestLogEntriesWithMessageAndFields(projectID,
			baseTime.Add(time.Duration(sequenceIndex)*time.Second),
			"Test batch log message",
			map[string]any{
				"unique_id":    uniqueLogID,
				"test_session": testSessionID,
				"sequence_num": sequenceIndex,
				"service":      "api",
			})

		if allBatchEntries == nil {
			allBatchEntries = batchLogEntries
		} else {
			for projectKey, logItems := range batchLogEntries {
				allBatchEntries[projectKey] = append(allBatchEntries[projectKey], logItems...)
			}
		}
	}

	return allBatchEntries
}

// Private helper functions

func CreateTestLogEntriesWithUniqueFields(
	projectID uuid.UUID,
	timestamp time.Time,
	message string,
	customFields map[string]any,
) map[uuid.UUID][]*logs_core.LogItem {
	logEntryID := uuid.New()

	uniqueLogEntry := &logs_core.LogItem{
		ID:        logEntryID,
		ProjectID: projectID,
		Timestamp: timestamp,
		Level:     logs_core.LogLevelInfo,
		Message:   message,
		Fields:    customFields,
		ClientIP:  "127.0.0.1",
	}

	return map[uuid.UUID][]*logs_core.LogItem{
		projectID: {uniqueLogEntry},
	}
}

func MergeLogEntries(
	firstLogEntries, secondLogEntries map[uuid.UUID][]*logs_core.LogItem,
) map[uuid.UUID][]*logs_core.LogItem {
	mergedLogEntries := make(map[uuid.UUID][]*logs_core.LogItem)

	for projectID, logItems := range firstLogEntries {
		mergedLogEntries[projectID] = append(mergedLogEntries[projectID], logItems...)
	}

	for projectID, logItems := range secondLogEntries {
		mergedLogEntries[projectID] = append(mergedLogEntries[projectID], logItems...)
	}

	return mergedLogEntries
}

func WaitForLogsDeletion(
	t *testing.T,
	repository *logs_core.LogCoreRepository,
	projectID uuid.UUID,
	query *logs_core.LogQueryRequestDTO,
	timeout time.Duration,
) {
	checkInterval := 50 * time.Millisecond
	startTime := time.Now()

	for {
		result, err := repository.ExecuteQueryForProject(projectID, query)
		assert.NoError(t, err, "Repository query should work during deletion wait")

		// If no logs found, deletion was successful
		if len(result.Logs) == 0 && result.Total == 0 {
			return
		}

		// Check if timeout exceeded
		if time.Since(startTime) > timeout {
			t.Fatalf("Timeout: logs still exist after %v. Found %d logs, total: %d",
				timeout, len(result.Logs), result.Total)
		}

		// Wait before next check
		time.Sleep(checkInterval)
	}
}

func WaitForLogsPartialDeletion(
	t *testing.T,
	repository *logs_core.LogCoreRepository,
	projectID uuid.UUID,
	query *logs_core.LogQueryRequestDTO,
	expectedCount int64,
	timeout time.Duration,
) {
	checkInterval := 50 * time.Millisecond
	startTime := time.Now()

	for {
		result, err := repository.ExecuteQueryForProject(projectID, query)
		assert.NoError(t, err, "Repository query should work during deletion wait")

		// If expected count is reached, deletion was successful
		if result.Total == expectedCount {
			return
		}

		// Check if timeout exceeded
		if time.Since(startTime) > timeout {
			t.Fatalf("Timeout: expected %d logs but found %d after %v",
				expectedCount, result.Total, timeout)
		}

		// Wait before next check
		time.Sleep(checkInterval)
	}
}
