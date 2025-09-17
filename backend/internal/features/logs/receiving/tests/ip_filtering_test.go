package logs_receiving_tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

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

func Test_SubmitLogs_WhenIPFilterEnabled_WithAllowedIP_LogsAccepted(t *testing.T) {
	testData := setupIPTest("Allowed IP Test", []string{"192.168.1.100", "10.0.0.5"})

	response := submitTestLogsWithIP(
		t,
		testData.Router,
		testData.Project.ID,
		"",
		testData.UniqueID,
		"192.168.1.100",
	)

	assert.Equal(t, 1, response.Accepted)
	assert.Equal(t, 0, response.Rejected)
	assert.Empty(t, response.Errors)
}

func Test_SubmitLogs_WhenIPFilterEnabled_WithDisallowedIP_ReturnsForbidden(t *testing.T) {
	testData := setupIPTest("Disallowed IP Test", []string{"192.168.1.100"})

	resp := submitTestLogsWithIPExpectingError(
		t,
		testData.Router,
		testData.Project.ID,
		"",
		testData.UniqueID,
		"10.0.0.5", // Not in allowed list
		http.StatusForbidden,
	)

	assert.Contains(t, string(resp.Body), "IP address not allowed")
}

func Test_SubmitLogs_WhenIPFilterEnabled_WithCIDRRange_LogsAccepted(t *testing.T) {
	testData := setupIPTest("CIDR Range Test", []string{"192.168.1.0/24", "10.0.0.0/8"})

	// Test IP within first CIDR range
	response1 := submitTestLogsWithIP(
		t,
		testData.Router,
		testData.Project.ID,
		"",
		testData.UniqueID+"_1",
		"192.168.1.50",
	)
	assert.Equal(t, 1, response1.Accepted)

	// Test IP within second CIDR range
	response2 := submitTestLogsWithIP(
		t,
		testData.Router,
		testData.Project.ID,
		"",
		testData.UniqueID+"_2",
		"10.5.10.20",
	)
	assert.Equal(t, 1, response2.Accepted)

	// Test IP outside both CIDR ranges should fail
	resp := submitTestLogsWithIPExpectingError(
		t,
		testData.Router,
		testData.Project.ID,
		"",
		testData.UniqueID+"_3",
		"172.16.0.10", // Not in allowed CIDR ranges
		http.StatusForbidden,
	)
	assert.Contains(t, string(resp.Body), "IP address not allowed")
}

func Test_SubmitLogs_WhenIPFilterEnabled_WithXForwardedFor_LogsAccepted(t *testing.T) {
	testData := setupIPTest("X-Forwarded-For Test", []string{"203.0.113.45"})

	response := submitTestLogsWithXForwardedFor(
		t,
		testData.Router,
		testData.Project.ID,
		"",
		testData.UniqueID,
		"203.0.113.45, 192.168.1.1", // Real IP is first in the list
	)

	assert.Equal(t, 1, response.Accepted)
	assert.Equal(t, 0, response.Rejected)
	assert.Empty(t, response.Errors)
}

func Test_SubmitLogs_WhenIPFilterEnabled_WithXRealIP_LogsAccepted(t *testing.T) {
	testData := setupIPTest("X-Real-IP Test", []string{"198.51.100.23"})

	response := submitTestLogsWithXRealIP(
		t,
		testData.Router,
		testData.Project.ID,
		"",
		testData.UniqueID,
		"198.51.100.23",
	)

	assert.Equal(t, 1, response.Accepted)
	assert.Equal(t, 0, response.Rejected)
	assert.Empty(t, response.Errors)
}

func Test_SubmitLogs_WhenIPFilterEnabled_WithXForwardedForDisallowed_ReturnsForbidden(t *testing.T) {
	testData := setupIPTest("X-Forwarded-For Disallowed Test", []string{"203.0.113.45"})

	resp := submitTestLogsWithXForwardedForExpectingError(
		t,
		testData.Router,
		testData.Project.ID,
		"",
		testData.UniqueID,
		"192.168.1.100, 10.0.0.1", // Neither IP is in allowed list
		http.StatusForbidden,
	)

	assert.Contains(t, string(resp.Body), "IP address not allowed")
}

func Test_SubmitLogs_WhenIPFilterEnabled_WithXRealIPDisallowed_ReturnsForbidden(t *testing.T) {
	testData := setupIPTest("X-Real-IP Disallowed Test", []string{"198.51.100.23"})

	resp := submitTestLogsWithXRealIPExpectingError(
		t,
		testData.Router,
		testData.Project.ID,
		"",
		testData.UniqueID,
		"10.0.0.100", // Not in allowed list
		http.StatusForbidden,
	)

	assert.Contains(t, string(resp.Body), "IP address not allowed")
}

func Test_SubmitLogs_WhenIPFilterDisabled_WithAnyIP_LogsAccepted(t *testing.T) {
	testData := setupIPTest("IP Filter Disabled Test", nil)

	// Test with random IP when filtering is disabled
	response := submitTestLogsWithIP(
		t,
		testData.Router,
		testData.Project.ID,
		"",
		testData.UniqueID,
		"1.2.3.4", // Any IP should work
	)

	assert.Equal(t, 1, response.Accepted)
	assert.Equal(t, 0, response.Rejected)
	assert.Empty(t, response.Errors)
}

func Test_SubmitLogs_WhenIPFilterEnabled_WithMultipleAllowedIPs_LogsAccepted(t *testing.T) {
	testData := setupIPTest("Multiple IPs Test", []string{"192.168.1.100", "10.0.0.5", "203.0.113.45"})

	// Test first allowed IP
	response1 := submitTestLogsWithIP(
		t,
		testData.Router,
		testData.Project.ID,
		"",
		testData.UniqueID+"_1",
		"192.168.1.100",
	)
	assert.Equal(t, 1, response1.Accepted)

	// Test second allowed IP
	response2 := submitTestLogsWithIP(
		t,
		testData.Router,
		testData.Project.ID,
		"",
		testData.UniqueID+"_2",
		"10.0.0.5",
	)
	assert.Equal(t, 1, response2.Accepted)

	// Test third allowed IP
	response3 := submitTestLogsWithIP(
		t,
		testData.Router,
		testData.Project.ID,
		"",
		testData.UniqueID+"_3",
		"203.0.113.45",
	)
	assert.Equal(t, 1, response3.Accepted)
}

func Test_SubmitLogs_WhenIPFilterEnabled_WithMixedIPsAndCIDR_LogsAccepted(t *testing.T) {
	testData := setupIPTest("Mixed IPs and CIDR Test", []string{"192.168.1.100", "10.0.0.0/24"})

	// Test specific IP
	response1 := submitTestLogsWithIP(
		t,
		testData.Router,
		testData.Project.ID,
		"",
		testData.UniqueID+"_1",
		"192.168.1.100",
	)
	assert.Equal(t, 1, response1.Accepted)

	// Test IP within CIDR range
	response2 := submitTestLogsWithIP(
		t,
		testData.Router,
		testData.Project.ID,
		"",
		testData.UniqueID+"_2",
		"10.0.0.50",
	)
	assert.Equal(t, 1, response2.Accepted)

	// Test IP outside both specific IP and CIDR range
	resp := submitTestLogsWithIPExpectingError(
		t,
		testData.Router,
		testData.Project.ID,
		"",
		testData.UniqueID+"_3",
		"172.16.0.10",
		http.StatusForbidden,
	)
	assert.Contains(t, string(resp.Body), "IP address not allowed")
}

type IPTestData struct {
	Router   *gin.Engine
	User     *users_dto.SignInResponseDTO
	Project  *projects_models.Project
	UniqueID string
}

func setupIPTest(testPrefix string, allowedIPs []string) *IPTestData {
	router := CreateLogsTestRouter()
	user := users_testing.CreateTestUser(users_enums.UserRoleMember)
	uniqueID := uuid.New().String()
	projectName := fmt.Sprintf("%s %s", testPrefix, uniqueID[:8])

	var project *projects_models.Project
	if allowedIPs != nil {
		config := &projects_testing.ProjectConfigurationDTO{
			IsApiKeyRequired:   false,
			IsFilterByDomain:   false,
			AllowedDomains:     nil,
			IsFilterByIP:       true,
			AllowedIPs:         allowedIPs,
			LogsPerSecondLimit: 1000,
			MaxLogSizeKB:       64,
		}
		project = projects_testing.CreateTestProjectWithConfiguration(projectName, user, router, config)
	} else {
		project = projects_testing.CreateBasicTestProject(projectName, user, router)
	}

	return &IPTestData{
		Router:   router,
		User:     user,
		Project:  project,
		UniqueID: uniqueID,
	}
}

func submitTestLogsWithIP(
	t *testing.T,
	router *gin.Engine,
	projectID uuid.UUID,
	apiKeyToken, uniqueID, clientIP string,
) *logs_receiving.SubmitLogsResponseDTO {
	logItems := CreateValidLogItems(1, uniqueID)
	request := &logs_receiving.SubmitLogsRequestDTO{
		Logs: logItems,
	}

	headers := make(map[string]string)
	if apiKeyToken != "" {
		headers["X-API-Key"] = apiKeyToken
	}

	// Use X-Real-IP to simulate client IP
	headers["X-Real-IP"] = clientIP

	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "POST",
		URL:            fmt.Sprintf("/api/v1/logs/receiving/%s", projectID.String()),
		Body:           request,
		Headers:        headers,
		ExpectedStatus: http.StatusAccepted,
	})

	var response logs_receiving.SubmitLogsResponseDTO
	if err := json.Unmarshal(resp.Body, &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	return &response
}

func submitTestLogsWithIPExpectingError(
	t *testing.T,
	router *gin.Engine,
	projectID uuid.UUID,
	apiKeyToken, uniqueID, clientIP string,
	expectedStatus int,
) *test_utils.TestResponse {
	logItems := CreateValidLogItems(1, uniqueID)
	request := &logs_receiving.SubmitLogsRequestDTO{
		Logs: logItems,
	}

	headers := make(map[string]string)
	if apiKeyToken != "" {
		headers["X-API-Key"] = apiKeyToken
	}
	// Use X-Real-IP to simulate client IP
	headers["X-Real-IP"] = clientIP

	return test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "POST",
		URL:            fmt.Sprintf("/api/v1/logs/receiving/%s", projectID.String()),
		Body:           request,
		Headers:        headers,
		ExpectedStatus: expectedStatus,
	})
}

func submitTestLogsWithXForwardedFor(
	t *testing.T,
	router *gin.Engine,
	projectID uuid.UUID,
	apiKeyToken, uniqueID, xForwardedFor string,
) *logs_receiving.SubmitLogsResponseDTO {
	logItems := CreateValidLogItems(1, uniqueID)
	request := &logs_receiving.SubmitLogsRequestDTO{
		Logs: logItems,
	}

	headers := make(map[string]string)
	if apiKeyToken != "" {
		headers["X-API-Key"] = apiKeyToken
	}
	headers["X-Forwarded-For"] = xForwardedFor

	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "POST",
		URL:            fmt.Sprintf("/api/v1/logs/receiving/%s", projectID.String()),
		Body:           request,
		Headers:        headers,
		ExpectedStatus: http.StatusAccepted,
	})

	var response logs_receiving.SubmitLogsResponseDTO
	if err := json.Unmarshal(resp.Body, &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	return &response
}

func submitTestLogsWithXForwardedForExpectingError(
	t *testing.T,
	router *gin.Engine,
	projectID uuid.UUID,
	apiKeyToken, uniqueID, xForwardedFor string,
	expectedStatus int,
) *test_utils.TestResponse {
	logItems := CreateValidLogItems(1, uniqueID)
	request := &logs_receiving.SubmitLogsRequestDTO{
		Logs: logItems,
	}

	headers := make(map[string]string)
	if apiKeyToken != "" {
		headers["X-API-Key"] = apiKeyToken
	}
	headers["X-Forwarded-For"] = xForwardedFor

	return test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "POST",
		URL:            fmt.Sprintf("/api/v1/logs/receiving/%s", projectID.String()),
		Body:           request,
		Headers:        headers,
		ExpectedStatus: expectedStatus,
	})
}

func submitTestLogsWithXRealIP(
	t *testing.T,
	router *gin.Engine,
	projectID uuid.UUID,
	apiKeyToken, uniqueID, xRealIP string,
) *logs_receiving.SubmitLogsResponseDTO {
	logItems := CreateValidLogItems(1, uniqueID)
	request := &logs_receiving.SubmitLogsRequestDTO{
		Logs: logItems,
	}

	headers := make(map[string]string)
	if apiKeyToken != "" {
		headers["X-API-Key"] = apiKeyToken
	}
	headers["X-Real-IP"] = xRealIP

	resp := test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "POST",
		URL:            fmt.Sprintf("/api/v1/logs/receiving/%s", projectID.String()),
		Body:           request,
		Headers:        headers,
		ExpectedStatus: http.StatusAccepted,
	})

	var response logs_receiving.SubmitLogsResponseDTO
	if err := json.Unmarshal(resp.Body, &response); err != nil {
		t.Fatalf("Failed to unmarshal response: %v", err)
	}

	return &response
}

func submitTestLogsWithXRealIPExpectingError(
	t *testing.T,
	router *gin.Engine,
	projectID uuid.UUID,
	apiKeyToken, uniqueID, xRealIP string,
	expectedStatus int,
) *test_utils.TestResponse {
	logItems := CreateValidLogItems(1, uniqueID)
	request := &logs_receiving.SubmitLogsRequestDTO{
		Logs: logItems,
	}

	headers := make(map[string]string)
	if apiKeyToken != "" {
		headers["X-API-Key"] = apiKeyToken
	}
	headers["X-Real-IP"] = xRealIP

	return test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "POST",
		URL:            fmt.Sprintf("/api/v1/logs/receiving/%s", projectID.String()),
		Body:           request,
		Headers:        headers,
		ExpectedStatus: expectedStatus,
	})
}
