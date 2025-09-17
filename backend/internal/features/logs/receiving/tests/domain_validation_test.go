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

func Test_SubmitLogs_WhenDomainFilterEnabled_WithAllowedDomain_LogsAccepted(t *testing.T) {
	testData := setupDomainTest("Allowed Domain Test", []string{"example.com", "test.org"})

	response := submitTestLogsWithOrigin(
		t,
		testData.Router,
		testData.Project.ID,
		"",
		testData.UniqueID,
		"https://example.com",
	)

	assert.Equal(t, 1, response.Accepted)
	assert.Equal(t, 0, response.Rejected)
	assert.Empty(t, response.Errors)
}

func Test_SubmitLogs_WhenDomainFilterEnabled_WithDisallowedDomain_ReturnsForbidden(t *testing.T) {
	testData := setupDomainTest("Disallowed Domain Test", []string{"example.com"})

	resp := submitTestLogsWithOriginExpectingError(
		t,
		testData.Router,
		testData.Project.ID,
		"",
		testData.UniqueID,
		"https://malicious.com",
		http.StatusForbidden,
	)

	assert.Contains(t, string(resp.Body), "domain not allowed")
}

func Test_SubmitLogs_WhenDomainFilterEnabled_WithoutOriginHeader_ReturnsForbidden(t *testing.T) {
	testData := setupDomainTest("No Origin Header Test", []string{"example.com"})

	resp := submitTestLogsExpectingError(
		t,
		testData.Router,
		testData.Project.ID,
		"",
		testData.UniqueID,
		http.StatusForbidden,
	)

	assert.Contains(t, string(resp.Body), "origin header required for domain filtering")
}

func Test_SubmitLogs_WhenDomainFilterEnabled_WithRefererHeader_LogsAccepted(t *testing.T) {
	testData := setupDomainTest("Referer Header Test", []string{"example.com"})

	response := submitTestLogsWithReferer(
		t,
		testData.Router,
		testData.Project.ID,
		"",
		testData.UniqueID,
		"https://example.com/page",
	)

	assert.Equal(t, 1, response.Accepted)
	assert.Equal(t, 0, response.Rejected)
	assert.Empty(t, response.Errors)
}

func Test_SubmitLogs_WhenDomainFilterDisabled_WithoutOriginHeader_LogsAccepted(t *testing.T) {
	testData := setupDomainTest("Domain Filter Disabled Test", nil)

	response := submitTestLogs(t, testData.Router, testData.Project.ID, "", testData.UniqueID)

	assert.Equal(t, 1, response.Accepted)
	assert.Equal(t, 0, response.Rejected)
	assert.Empty(t, response.Errors)
}

func Test_SubmitLogs_WhenDomainFilterEnabled_WithExactSubdomain_LogsAccepted(t *testing.T) {
	testData := setupDomainTest("Exact Subdomain Test", []string{"api.example.com"})

	response := submitTestLogsWithOrigin(
		t,
		testData.Router,
		testData.Project.ID,
		"",
		testData.UniqueID,
		"https://api.example.com",
	)

	assert.Equal(t, 1, response.Accepted)
	assert.Equal(t, 0, response.Rejected)
	assert.Empty(t, response.Errors)
}

func Test_SubmitLogs_WhenDomainFilterEnabled_WithMultipleDomains_LogsAccepted(t *testing.T) {
	testData := setupDomainTest("Multiple Domains Test", []string{"example.com", "test.org", "domain.net"})

	// Test first allowed domain
	response1 := submitTestLogsWithOrigin(
		t,
		testData.Router,
		testData.Project.ID,
		"",
		testData.UniqueID+"_1",
		"https://example.com",
	)
	assert.Equal(t, 1, response1.Accepted)

	// Test second allowed domain
	response2 := submitTestLogsWithOrigin(
		t,
		testData.Router,
		testData.Project.ID,
		"",
		testData.UniqueID+"_2",
		"https://test.org",
	)
	assert.Equal(t, 1, response2.Accepted)

	// Test third allowed domain
	response3 := submitTestLogsWithOrigin(
		t,
		testData.Router,
		testData.Project.ID,
		"",
		testData.UniqueID+"_3",
		"https://domain.net",
	)
	assert.Equal(t, 1, response3.Accepted)
}

func Test_SubmitLogs_WhenDomainFilterEnabled_WithParentDomainOnlySubdomainAllowed_ReturnsForbidden(t *testing.T) {
	testData := setupDomainTest("Parent Domain With Subdomain Test", []string{"api.example.com"})

	resp := submitTestLogsWithOriginExpectingError(
		t,
		testData.Router,
		testData.Project.ID,
		"",
		testData.UniqueID,
		"https://example.com",
		http.StatusForbidden,
	)

	assert.Contains(t, string(resp.Body), "domain not allowed")
}

func Test_SubmitLogs_WhenDomainFilterEnabled_WithSubdomainOnlyParentAllowed_ReturnsForbidden(t *testing.T) {
	testData := setupDomainTest("Subdomain With Parent Test", []string{"example.com"})

	resp := submitTestLogsWithOriginExpectingError(
		t,
		testData.Router,
		testData.Project.ID,
		"",
		testData.UniqueID,
		"https://api.example.com",
		http.StatusForbidden,
	)

	assert.Contains(t, string(resp.Body), "domain not allowed")
}

func Test_SubmitLogs_WhenDomainFilterEnabled_WithPortInOrigin_LogsAccepted(t *testing.T) {
	testData := setupDomainTest("Port In Origin Test", []string{"localhost"})

	response := submitTestLogsWithOrigin(
		t,
		testData.Router,
		testData.Project.ID,
		"",
		testData.UniqueID,
		"http://localhost:3000",
	)

	assert.Equal(t, 1, response.Accepted)
	assert.Equal(t, 0, response.Rejected)
	assert.Empty(t, response.Errors)
}

func Test_SubmitLogs_WhenDomainFilterEnabled_WithInvalidOriginFormat_ReturnsForbidden(t *testing.T) {
	testData := setupDomainTest("Invalid Origin Format Test", []string{"example.com"})

	resp := submitTestLogsWithOriginExpectingError(
		t,
		testData.Router,
		testData.Project.ID,
		"",
		testData.UniqueID,
		"not-a-valid-url",
		http.StatusForbidden,
	)

	assert.Contains(t, string(resp.Body), "domain not allowed")
}

type DomainTestData struct {
	Router   *gin.Engine
	User     *users_dto.SignInResponseDTO
	Project  *projects_models.Project
	UniqueID string
}

func setupDomainTest(testPrefix string, allowedDomains []string) *DomainTestData {
	router := CreateLogsTestRouter()
	user := users_testing.CreateTestUser(users_enums.UserRoleMember)
	uniqueID := uuid.New().String()
	projectName := fmt.Sprintf("%s %s", testPrefix, uniqueID[:8])

	var project *projects_models.Project
	if allowedDomains != nil {
		config := &projects_testing.ProjectConfigurationDTO{
			IsApiKeyRequired:   false,
			IsFilterByDomain:   true,
			AllowedDomains:     allowedDomains,
			IsFilterByIP:       false,
			AllowedIPs:         nil,
			LogsPerSecondLimit: 1000,
			MaxLogSizeKB:       64,
		}
		project = projects_testing.CreateTestProjectWithConfiguration(projectName, user, router, config)
	} else {
		project = projects_testing.CreateBasicTestProject(projectName, user, router)
	}

	return &DomainTestData{
		Router:   router,
		User:     user,
		Project:  project,
		UniqueID: uniqueID,
	}
}

func submitTestLogsWithOrigin(
	t *testing.T,
	router *gin.Engine,
	projectID uuid.UUID,
	apiKeyToken, uniqueID, origin string,
) *logs_receiving.SubmitLogsResponseDTO {
	logItems := CreateValidLogItems(1, uniqueID)
	request := &logs_receiving.SubmitLogsRequestDTO{
		Logs: logItems,
	}

	headers := make(map[string]string)
	if apiKeyToken != "" {
		headers["X-API-Key"] = apiKeyToken
	}
	if origin != "" {
		headers["Origin"] = origin
	}

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

func submitTestLogsWithReferer(
	t *testing.T,
	router *gin.Engine,
	projectID uuid.UUID,
	apiKeyToken, uniqueID, referer string,
) *logs_receiving.SubmitLogsResponseDTO {
	logItems := CreateValidLogItems(1, uniqueID)
	request := &logs_receiving.SubmitLogsRequestDTO{
		Logs: logItems,
	}

	headers := make(map[string]string)
	if apiKeyToken != "" {
		headers["X-API-Key"] = apiKeyToken
	}
	if referer != "" {
		headers["Referer"] = referer
	}

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

func submitTestLogsWithOriginExpectingError(
	t *testing.T,
	router *gin.Engine,
	projectID uuid.UUID,
	apiKeyToken, uniqueID, origin string,
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

	if origin != "" {
		headers["Origin"] = origin
	}

	return test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "POST",
		URL:            fmt.Sprintf("/api/v1/logs/receiving/%s", projectID.String()),
		Body:           request,
		Headers:        headers,
		ExpectedStatus: expectedStatus,
	})
}
