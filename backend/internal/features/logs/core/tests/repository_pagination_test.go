package logs_core_tests

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	logs_core "logbull/internal/features/logs/core"
)

func Test_ExecuteQueryForProject_WithPaginationAndOffset_ReturnsSameTotalCount(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	projectID := uuid.New()
	currentTime := time.Now().UTC()
	uniqueTestSession := uuid.New().String()[:8]

	testLogEntries := CreateBatchLogEntries(projectID, 12, currentTime, uniqueTestSession)
	StoreTestLogsAndFlush(t, repository, testLogEntries)

	// Add additional wait to ensure indexing is complete
	time.Sleep(1 * time.Second)

	firstPageQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "test_session",
				Operator: logs_core.ConditionOperatorEquals,
				Value:    uniqueTestSession,
			},
		},
		Limit:     5,
		Offset:    0,
		SortOrder: "desc",
	}

	firstPageResult, firstPageErr := repository.ExecuteQueryForProject(projectID, firstPageQuery)
	assert.NoError(t, firstPageErr)
	assert.NotNil(t, firstPageResult)

	// Verify we have the expected test data
	assert.GreaterOrEqual(t, firstPageResult.Total, int64(12), "Should have at least 12 logs total")

	secondPageQuery := *firstPageQuery
	secondPageQuery.Offset = 5

	secondPageResult, secondPageErr := repository.ExecuteQueryForProject(projectID, &secondPageQuery)
	assert.NoError(t, secondPageErr)
	assert.NotNil(t, secondPageResult)

	// Both pages should report the same total count
	assert.Equal(t, firstPageResult.Total, secondPageResult.Total, "Total count should be consistent across pages")
	assert.GreaterOrEqual(t, firstPageResult.Total, int64(12), "Should have at least 12 logs total")
	assert.LessOrEqual(t, len(firstPageResult.Logs), 5, "First page should have at most 5 logs")
	assert.LessOrEqual(t, len(secondPageResult.Logs), 5, "Second page should have at most 5 logs")
}

func Test_ExecuteQueryForProject_WithMicrosecondPrecision_MaintainsProperDESCOrdering(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	projectID := uuid.New()
	baseTime := time.Now().UTC()
	uniqueTestSession := uuid.New().String()[:8]
	logCount := 20
	pageSize := 5

	testLogEntries := createMicrosecondLogEntries(projectID, logCount, baseTime, uniqueTestSession)
	StoreTestLogsAndFlush(t, repository, testLogEntries)

	time.Sleep(1 * time.Second)

	baseQuery := &logs_core.LogQueryRequestDTO{
		Query: &logs_core.QueryNode{
			Type: logs_core.QueryNodeTypeCondition,
			Condition: &logs_core.ConditionNode{
				Field:    "test_session",
				Operator: logs_core.ConditionOperatorEquals,
				Value:    uniqueTestSession,
			},
		},
		Limit:     pageSize,
		Offset:    0,
		SortOrder: "desc",
	}

	allLogsByPage := make([][]logs_core.LogItemDTO, 0, 4)
	expectedPageCount := logCount / pageSize

	for pageIndex := 0; pageIndex < expectedPageCount; pageIndex++ {
		pageQuery := *baseQuery
		pageQuery.Offset = pageIndex * pageSize

		pageResult, err := repository.ExecuteQueryForProject(projectID, &pageQuery)
		assert.NoError(t, err, "Page %d query should succeed", pageIndex)
		assert.NotNil(t, pageResult, "Page %d result should not be nil", pageIndex)
		assert.Len(t, pageResult.Logs, pageSize, "Page %d should have %d logs", pageIndex, pageSize)

		allLogsByPage = append(allLogsByPage, pageResult.Logs)
		verifyDescOrderingWithinPage(t, pageResult.Logs, pageIndex)
	}

	verifyDescOrderingAcrossPages(t, allLogsByPage, logCount, pageSize)
}

func verifyDescOrderingWithinPage(t *testing.T, logs []logs_core.LogItemDTO, pageIndex int) {
	for i := 1; i < len(logs); i++ {
		previousLog := logs[i-1]
		currentLog := logs[i]

		assert.True(
			t,
			previousLog.Timestamp.After(currentLog.Timestamp),
			"Page %d: Log at index %d (timestamp: %s) should be after log at index %d (timestamp: %s) in DESC order",
			pageIndex,
			i-1,
			previousLog.Timestamp.Format("2006-01-02T15:04:05.999999Z07:00"),
			i,
			currentLog.Timestamp.Format("2006-01-02T15:04:05.999999Z07:00"),
		)

		previousSeq := previousLog.Fields["sequence_num"]
		currentSeq := currentLog.Fields["sequence_num"]
		assert.Greater(
			t,
			previousSeq,
			currentSeq,
			"Page %d: sequence_num should be in DESC order. Previous: %v, Current: %v",
			pageIndex,
			previousSeq,
			currentSeq,
		)
	}
}

func verifyDescOrderingAcrossPages(t *testing.T, allLogsByPage [][]logs_core.LogItemDTO, logCount, pageSize int) {
	expectedSequencesDescOrder := [][]int{
		{20, 19, 18, 17, 16},
		{15, 14, 13, 12, 11},
		{10, 9, 8, 7, 6},
		{5, 4, 3, 2, 1},
	}

	for pageIndex, expectedSeqsForPage := range expectedSequencesDescOrder {
		if pageIndex >= len(allLogsByPage) {
			break
		}

		actualPage := allLogsByPage[pageIndex]
		for logIndex, expectedSeq := range expectedSeqsForPage {
			if logIndex >= len(actualPage) {
				break
			}

			actualSeq := extractSequenceNumber(actualPage[logIndex])
			assert.Equal(t, float64(expectedSeq), actualSeq,
				"Page %d, index %d: Expected sequence %d but got %v in DESC order",
				pageIndex, logIndex, expectedSeq, actualSeq)
		}
	}
}

func createMicrosecondLogEntries(
	projectID uuid.UUID,
	logCount int,
	baseTime time.Time,
	testSessionID string,
) map[uuid.UUID][]*logs_core.LogItem {
	allBatchEntries := make(map[uuid.UUID][]*logs_core.LogItem)

	for sequenceIndex := 1; sequenceIndex <= logCount; sequenceIndex++ {
		uniqueLogID := uuid.New().String()[:8]
		batchLogEntries := CreateTestLogEntriesWithMessageAndFields(projectID,
			baseTime.Add(time.Duration(sequenceIndex)*time.Microsecond),
			"Microsecond test log message",
			map[string]any{
				"unique_id":    uniqueLogID,
				"test_session": testSessionID,
				"sequence_num": sequenceIndex,
				"service":      "microsecond-test",
			})

		if len(allBatchEntries) == 0 {
			allBatchEntries = batchLogEntries
		} else {
			for projectKey, logItems := range batchLogEntries {
				allBatchEntries[projectKey] = append(allBatchEntries[projectKey], logItems...)
			}
		}
	}

	return allBatchEntries
}

func extractSequenceNumber(log logs_core.LogItemDTO) float64 {
	if actualSeq, ok := log.Fields["sequence_num"].(float64); ok {
		return actualSeq
	}
	if seqInt, ok := log.Fields["sequence_num"].(int); ok {
		return float64(seqInt)
	}
	return 0
}
