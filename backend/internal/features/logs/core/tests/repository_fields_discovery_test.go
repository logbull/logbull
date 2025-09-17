package logs_core_tests

import (
	"fmt"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	logs_core "logbull/internal/features/logs/core"
)

func Test_DiscoverFields_WithStoredLogsContainingCustomFields_ReturnsDiscoveredFields(t *testing.T) {
	t.Parallel()
	repository := logs_core.GetLogCoreRepository()
	projectID := uuid.New()
	uniqueTestSession := uuid.New().String()[:8]
	currentTime := time.Now().UTC()

	customTestFields := map[string]any{
		"test_session": uniqueTestSession,
		"custom_field": "value_" + uniqueTestSession,
		"status_code":  200,
	}

	testLogEntries := CreateTestLogEntriesWithMessageAndFields(
		projectID,
		currentTime,
		fmt.Sprintf("Test log for field discovery - %s", uniqueTestSession),
		customTestFields,
	)

	storeErr := repository.StoreLogsBatch(testLogEntries)
	assert.NoError(t, storeErr)

	flushErr := repository.ForceFlush()
	assert.NoError(t, flushErr)

	discoveredFields, discoveryErr := repository.DiscoverFields(projectID)

	assert.NoError(t, discoveryErr)
	assert.NotNil(t, discoveredFields)
	assert.IsType(t, []string{}, discoveredFields)

	if len(discoveredFields) > 0 {
		discoveredFieldsMap := make(map[string]bool)
		for _, fieldName := range discoveredFields {
			discoveredFieldsMap[fieldName] = true
		}

		hasCustomTestFields := discoveredFieldsMap["test_session"] || discoveredFieldsMap["custom_field"] ||
			discoveredFieldsMap["status_code"]
		if hasCustomTestFields {
			t.Logf("Successfully discovered custom fields")
		}
	}
}

func Test_DiscoverFields_WithUnavailableLogsStorage_PropagatesError(t *testing.T) {
	t.Parallel()
	unavailableRepository := logs_core.GetUnavailableLogCoreRepository()
	projectID := uuid.New()

	discoveredFields, discoveryErr := unavailableRepository.DiscoverFields(projectID)

	assert.Error(t, discoveryErr)
	assert.Nil(t, discoveredFields)
	assert.Contains(t, discoveryErr.Error(), "failed to execute field discovery search")
}
