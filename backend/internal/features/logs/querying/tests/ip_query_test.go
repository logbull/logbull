package logs_querying_tests

import (
	"fmt"
	"net/http"
	"strings"
	"testing"

	logs_core "logbull/internal/features/logs/core"
	logs_receiving "logbull/internal/features/logs/receiving"
	projects_testing "logbull/internal/features/projects/testing"
	users_enums "logbull/internal/features/users/enums"
	users_testing "logbull/internal/features/users/testing"
	test_utils "logbull/internal/util/testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_ExecuteQuery_FilterByClientIPEquals_ReturnsMatchingLogs(t *testing.T) {
	router := CreateLogQueryTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)

	uniqueID := uuid.New().String()
	projectName := fmt.Sprintf("IP Filter Test %s", uniqueID[:8])
	project, _ := projects_testing.CreateTestProjectWithToken(projectName, owner.Token, router)

	targetIP := "192.168.1.100"
	otherIPs := []string{"10.0.0.50", "203.0.113.45"}

	for i := range 2 {
		submitLogWithIP(t, router, project.ID, targetIP, uniqueID,
			fmt.Sprintf("Request from office IP %d", i+1), "target_ip")
	}

	for i, ip := range otherIPs {
		submitLogWithIP(t, router, project.ID, ip, uniqueID,
			fmt.Sprintf("Request from other IP %d", i+1), "other_ip")
	}

	workerService := logs_receiving.GetLogWorkerService()
	if err := workerService.ExecuteBackgroundTasksForTest(); err != nil {
		t.Fatalf("Failed to execute background tasks: %v", err)
	}

	WaitForLogsToBeIndexed(t, router, project.ID, 2, uniqueID, "Bearer "+owner.Token)

	query := BuildSimpleConditionQuery("client_ip", "equals", targetIP)
	queryResponse := ExecuteTestQuery(t, router, project.ID, query, owner.Token, http.StatusOK)

	AssertQueryResponseValid(t, queryResponse, 1)
	assertAllLogsHaveClientIP(t, queryResponse.Logs, targetIP)
	assert.Equal(t, 2, len(queryResponse.Logs), "Expected exactly 2 logs with client_ip %s", targetIP)
}

func Test_ExecuteQuery_FilterByClientIPContains_ReturnsMatchingLogs(t *testing.T) {
	router := CreateLogQueryTestRouter()
	owner := users_testing.CreateTestUser(users_enums.UserRoleMember)

	uniqueID := uuid.New().String()
	projectName := fmt.Sprintf("IP Contains Test %s", uniqueID[:8])
	project, _ := projects_testing.CreateTestProjectWithToken(projectName, owner.Token, router)

	testData := []struct {
		ip      string
		network string
		message string
	}{
		{"192.168.1.10", "internal", "Internal network request 1"},
		{"192.168.2.25", "internal", "Internal network request 2"},
		{"203.0.113.100", "external", "External request"},
	}

	for _, data := range testData {
		submitLogWithIPAndFields(t, router, project.ID, data.ip, uniqueID, data.message,
			map[string]any{"network": data.network})
	}

	workerService := logs_receiving.GetLogWorkerService()
	if err := workerService.ExecuteBackgroundTasksForTest(); err != nil {
		t.Fatalf("Failed to execute background tasks: %v", err)
	}

	WaitForLogsToBeIndexed(t, router, project.ID, 2, uniqueID, "Bearer "+owner.Token)

	ipPattern := "192.168"
	query := BuildSimpleConditionQuery("client_ip", "contains", ipPattern)
	queryResponse := ExecuteTestQuery(t, router, project.ID, query, owner.Token, http.StatusOK)

	AssertQueryResponseValid(t, queryResponse, 1)
	assertAllLogsContainIPPattern(t, queryResponse.Logs, ipPattern)
	assert.Equal(t, 2, len(queryResponse.Logs), "Expected exactly 2 logs with client_ip containing %s", ipPattern)
}

func submitLogWithIP(
	t *testing.T,
	router *gin.Engine,
	projectID uuid.UUID,
	clientIP, uniqueID, message, logType string,
) {
	logItems := []logs_receiving.LogItemRequestDTO{
		{
			Level:   logs_core.LogLevelInfo,
			Message: fmt.Sprintf("%s %s", message, uniqueID),
			Fields: map[string]any{
				"test_id":  uniqueID,
				"log_type": logType,
			},
		},
	}

	submitLogItemsWithIP(t, router, projectID, clientIP, logItems)
}

func submitLogWithIPAndFields(
	t *testing.T,
	router *gin.Engine,
	projectID uuid.UUID,
	clientIP, uniqueID, message string,
	additionalFields map[string]any,
) {
	fields := map[string]any{
		"test_id": uniqueID,
	}
	for k, v := range additionalFields {
		fields[k] = v
	}

	logItems := []logs_receiving.LogItemRequestDTO{
		{
			Level:   logs_core.LogLevelInfo,
			Message: fmt.Sprintf("%s %s", message, uniqueID),
			Fields:  fields,
		},
	}

	submitLogItemsWithIP(t, router, projectID, clientIP, logItems)
}

func submitLogItemsWithIP(
	t *testing.T,
	router *gin.Engine,
	projectID uuid.UUID,
	clientIP string,
	logItems []logs_receiving.LogItemRequestDTO,
) {
	request := &logs_receiving.SubmitLogsRequestDTO{
		Logs: logItems,
	}

	submitURL := fmt.Sprintf("/api/v1/logs/receiving/%s", projectID.String())
	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method: "POST",
		URL:    submitURL,
		Body:   request,
		Headers: map[string]string{
			"X-Real-IP": clientIP,
		},
		ExpectedStatus: 202,
	})

	if resp.StatusCode != 202 {
		t.Fatalf("Failed to submit log with IP %s: status %d, body: %s", clientIP, resp.StatusCode, string(resp.Body))
	}
}

func assertAllLogsHaveClientIP(t *testing.T, logs []logs_core.LogItemDTO, expectedIP string) {
	for _, log := range logs {
		if log.ClientIP != expectedIP {
			t.Errorf("Query returned log with unexpected ClientIP: %s (expected: %s)", log.ClientIP, expectedIP)
		}
	}
}

func assertAllLogsContainIPPattern(t *testing.T, logs []logs_core.LogItemDTO, pattern string) {
	for _, log := range logs {
		if !strings.Contains(log.ClientIP, pattern) {
			t.Errorf("Query returned log with ClientIP '%s' that doesn't contain pattern '%s'", log.ClientIP, pattern)
		}
	}
}
