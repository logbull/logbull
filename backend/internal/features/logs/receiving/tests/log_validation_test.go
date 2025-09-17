package logs_receiving_tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	logs_core "logbull/internal/features/logs/core"
	logs_receiving "logbull/internal/features/logs/receiving"
	projects_models "logbull/internal/features/projects/models"
	projects_testing "logbull/internal/features/projects/testing"
	users_dto "logbull/internal/features/users/dto"
	users_enums "logbull/internal/features/users/enums"
	users_testing "logbull/internal/features/users/testing"
	test_utils "logbull/internal/util/testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_SubmitLogs_WithValidLogLevels_LogsAccepted(t *testing.T) {
	testData := setupValidationTest("Valid Log Levels Test")

	validLevels := []logs_core.LogLevel{
		logs_core.LogLevelDebug,
		logs_core.LogLevelInfo,
		logs_core.LogLevelWarn,
		logs_core.LogLevelError,
		logs_core.LogLevelFatal,
	}

	for i, level := range validLevels {
		uniqueID := fmt.Sprintf("%s_%s_%d", testData.UniqueID, level, i)
		logItems := createLogItemsWithLevel(1, uniqueID, level)

		response := submitLogsForValidation(t, testData.Router, testData.Project.ID, logItems)

		assert.Equal(t, 1, response.Accepted, "Level %s should be accepted", level)
		assert.Equal(t, 0, response.Rejected, "Level %s should not be rejected", level)
		assert.Empty(t, response.Errors, "Level %s should not have errors", level)
	}
}

func Test_SubmitLogs_WithInvalidLogLevel_LogRejected(t *testing.T) {
	testData := setupValidationTest("Invalid Log Level Test")

	invalidLogItem := logs_receiving.LogItemRequestDTO{
		Level:   "INVALID_LEVEL",
		Message: fmt.Sprintf("Test invalid level log %s", testData.UniqueID),
		Fields: map[string]any{
			"test_id": testData.UniqueID,
		},
	}

	response := submitLogsForValidation(
		t,
		testData.Router,
		testData.Project.ID,
		[]logs_receiving.LogItemRequestDTO{invalidLogItem},
	)

	assert.Equal(t, 0, response.Accepted)
	assert.Equal(t, 1, response.Rejected)
	assert.Len(t, response.Errors, 1)
	assert.Contains(t, response.Errors[0].Message, "INVALID_LOG_LEVEL")
}

func Test_SubmitLogs_WithEmptyMessage_LogRejected(t *testing.T) {
	testData := setupValidationTest("Empty Message Test")

	emptyMessageLogItem := logs_receiving.LogItemRequestDTO{
		Level:   logs_core.LogLevelInfo,
		Message: "",
		Fields: map[string]any{
			"test_id": testData.UniqueID,
		},
	}

	response := submitLogsForValidation(
		t,
		testData.Router,
		testData.Project.ID,
		[]logs_receiving.LogItemRequestDTO{emptyMessageLogItem},
	)

	assert.Equal(t, 0, response.Accepted)
	assert.Equal(t, 1, response.Rejected)
	assert.Len(t, response.Errors, 1)
	assert.Contains(t, response.Errors[0].Message, "MESSAGE_EMPTY")
}

func Test_SubmitLogs_WithWhitespaceMessage_LogRejected(t *testing.T) {
	testData := setupValidationTest("Whitespace Message Test")

	whitespaceMessages := []string{
		"   ",
		"\t\t\t",
		"\n\n\n",
		"   \t  \n  ",
	}

	for i, message := range whitespaceMessages {
		uniqueID := fmt.Sprintf("%s_%d", testData.UniqueID, i)
		whitespaceLogItem := logs_receiving.LogItemRequestDTO{
			Level:   logs_core.LogLevelInfo,
			Message: message,
			Fields: map[string]any{
				"test_id": uniqueID,
			},
		}

		response := submitLogsForValidation(
			t,
			testData.Router,
			testData.Project.ID,
			[]logs_receiving.LogItemRequestDTO{whitespaceLogItem},
		)

		assert.Equal(t, 0, response.Accepted, "Whitespace message '%s' should be rejected", message)
		assert.Equal(t, 1, response.Rejected, "Whitespace message '%s' should be rejected", message)
		assert.Len(t, response.Errors, 1, "Whitespace message '%s' should have error", message)
		assert.Contains(t, response.Errors[0].Message, "MESSAGE_EMPTY", "Error should indicate empty message")
	}
}

func Test_SubmitLogs_WithLogExceedingMaxSize_LogRejected(t *testing.T) {
	testData := setupValidationTest("Log Exceeding Max Size Test")

	largeMessage := strings.Repeat("A", 65*1024)
	largeLogItem := logs_receiving.LogItemRequestDTO{
		Level:   logs_core.LogLevelInfo,
		Message: largeMessage,
		Fields: map[string]any{
			"test_id": testData.UniqueID,
		},
	}

	response := submitLogsForValidation(
		t,
		testData.Router,
		testData.Project.ID,
		[]logs_receiving.LogItemRequestDTO{largeLogItem},
	)

	assert.Equal(t, 0, response.Accepted)
	assert.Equal(t, 1, response.Rejected)
	assert.Len(t, response.Errors, 1)
	assert.Contains(t, response.Errors[0].Message, "LOG_TOO_LARGE")
}

func Test_SubmitLogs_WithCustomFields_LogsAccepted(t *testing.T) {
	testData := setupValidationTest("Custom Fields Test")

	customFieldsLogItem := logs_receiving.LogItemRequestDTO{
		Level:   logs_core.LogLevelInfo,
		Message: fmt.Sprintf("Test custom fields log %s", testData.UniqueID),
		Fields: map[string]any{
			"test_id":     testData.UniqueID,
			"user_id":     12345,
			"action":      "login",
			"success":     true,
			"ip_address":  "192.168.1.100",
			"user_agent":  "Mozilla/5.0 Test Browser",
			"duration_ms": 150.5,
			"metadata": map[string]any{
				"nested_field": "nested_value",
				"count":        42,
			},
			"tags": []string{"auth", "security", "important"},
		},
	}

	response := submitLogsForValidation(
		t,
		testData.Router,
		testData.Project.ID,
		[]logs_receiving.LogItemRequestDTO{customFieldsLogItem},
	)

	assert.Equal(t, 1, response.Accepted)
	assert.Equal(t, 0, response.Rejected)
	assert.Empty(t, response.Errors)
}

type ValidationTestData struct {
	Router   *gin.Engine
	User     *users_dto.SignInResponseDTO
	Project  *projects_models.Project
	UniqueID string
}

func setupValidationTest(testPrefix string) *ValidationTestData {
	router := CreateLogsTestRouter()
	user := users_testing.CreateTestUser(users_enums.UserRoleMember)
	uniqueID := uuid.New().String()
	projectName := fmt.Sprintf("%s %s", testPrefix, uniqueID[:8])

	project := projects_testing.CreateBasicTestProject(projectName, user, router)

	return &ValidationTestData{
		Router:   router,
		User:     user,
		Project:  project,
		UniqueID: uniqueID,
	}
}

func createLogItemsWithLevel(count int, uniqueID string, level logs_core.LogLevel) []logs_receiving.LogItemRequestDTO {
	logItems := make([]logs_receiving.LogItemRequestDTO, count)

	for i := 0; i < count; i++ {
		logItems[i] = logs_receiving.LogItemRequestDTO{
			Level:   level,
			Message: fmt.Sprintf("Test log message %s - %d", uniqueID, i+1),
			Fields: map[string]any{
				"test_id":    uniqueID,
				"log_index":  i + 1,
				"component":  "test_component",
				"request_id": fmt.Sprintf("req_%s_%d", uniqueID[:8], i+1),
			},
		}
	}

	return logItems
}

func submitLogsForValidation(
	t *testing.T,
	router *gin.Engine,
	projectID uuid.UUID,
	logItems []logs_receiving.LogItemRequestDTO,
) *logs_receiving.SubmitLogsResponseDTO {
	request := &logs_receiving.SubmitLogsRequestDTO{
		Logs: logItems,
	}

	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "POST",
		URL:            fmt.Sprintf("/api/v1/logs/receiving/%s", projectID.String()),
		Body:           request,
		ExpectedStatus: http.StatusAccepted,
	})

	var response logs_receiving.SubmitLogsResponseDTO
	if err := json.Unmarshal(resp.Body, &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	return &response
}
