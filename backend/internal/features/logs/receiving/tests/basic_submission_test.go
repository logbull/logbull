package logs_receiving_tests

import (
	"fmt"
	"net/http"
	"testing"

	logs_receiving "logbull/internal/features/logs/receiving"
	projects_testing "logbull/internal/features/projects/testing"
	users_enums "logbull/internal/features/users/enums"
	users_testing "logbull/internal/features/users/testing"
	test_utils "logbull/internal/util/testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_SubmitLogs_WithValidData_LogsAccepted(t *testing.T) {
	router := CreateLogsTestRouter()
	user := users_testing.CreateTestUser(users_enums.UserRoleMember)
	uniqueID := uuid.New().String()
	projectName := fmt.Sprintf("Valid Data Test %s", uniqueID[:8])
	project := projects_testing.CreateTestProject(projectName, user, router)

	logItems := CreateValidLogItems(1, uniqueID)
	request := &logs_receiving.SubmitLogsRequestDTO{
		Logs: logItems,
	}

	var response logs_receiving.SubmitLogsResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/logs/receiving/%s", project.ID.String()),
		"",
		request,
		http.StatusAccepted,
		&response,
	)

	assert.Equal(t, 1, response.Accepted)
	assert.Equal(t, 0, response.Rejected)
	assert.Empty(t, response.Errors)
}

func Test_SubmitLogs_WithMultipleLogs_AllLogsAccepted(t *testing.T) {
	router := CreateLogsTestRouter()
	user := users_testing.CreateTestUser(users_enums.UserRoleMember)
	uniqueID := uuid.New().String()
	projectName := fmt.Sprintf("Multiple Logs Test %s", uniqueID[:8])
	project := projects_testing.CreateTestProject(projectName, user, router)

	logCount := 5
	logItems := CreateValidLogItems(logCount, uniqueID)
	request := &logs_receiving.SubmitLogsRequestDTO{
		Logs: logItems,
	}

	var response logs_receiving.SubmitLogsResponseDTO
	test_utils.MakePostRequestAndUnmarshal(
		t,
		router,
		fmt.Sprintf("/api/v1/logs/receiving/%s", project.ID.String()),
		"",
		request,
		http.StatusAccepted,
		&response,
	)

	assert.Equal(t, logCount, response.Accepted)
	assert.Equal(t, 0, response.Rejected)
	assert.Empty(t, response.Errors)
}

func Test_SubmitLogs_WithEmptyBatch_ReturnsBadRequest(t *testing.T) {
	router := CreateLogsTestRouter()
	user := users_testing.CreateTestUser(users_enums.UserRoleMember)
	uniqueID := uuid.New().String()
	projectName := fmt.Sprintf("Empty Batch Test %s", uniqueID[:8])
	project := projects_testing.CreateTestProject(projectName, user, router)

	request := &logs_receiving.SubmitLogsRequestDTO{
		Logs: []logs_receiving.LogItemRequestDTO{},
	}

	resp := test_utils.MakePostRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/logs/receiving/%s", project.ID.String()),
		"",
		request,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(resp.Body), "Invalid request format")
}

func Test_SubmitLogs_WithInvalidProjectID_ReturnsBadRequest(t *testing.T) {
	router := CreateLogsTestRouter()
	uniqueID := uuid.New().String()

	logItems := CreateValidLogItems(1, uniqueID)
	request := &logs_receiving.SubmitLogsRequestDTO{
		Logs: logItems,
	}

	resp := test_utils.MakePostRequest(
		t,
		router,
		"/api/v1/logs/receiving/invalid-uuid",
		"",
		request,
		http.StatusBadRequest,
	)

	assert.Contains(t, string(resp.Body), "Invalid project ID")
}

func Test_SubmitLogs_WithNonExistentProject_ReturnsNotFound(t *testing.T) {
	router := CreateLogsTestRouter()
	uniqueID := uuid.New().String()
	nonExistentProjectID := uuid.New()

	logItems := CreateValidLogItems(1, uniqueID)
	request := &logs_receiving.SubmitLogsRequestDTO{
		Logs: logItems,
	}

	resp := test_utils.MakePostRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/logs/receiving/%s", nonExistentProjectID.String()),
		"",
		request,
		http.StatusNotFound,
	)

	assert.Contains(t, string(resp.Body), "project not found")
}
