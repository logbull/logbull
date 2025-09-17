package logs_receiving_tests

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	api_keys "logbull/internal/features/api_keys"
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

func Test_SubmitLogs_WhenApiKeyRequired_WithValidKey_LogsAccepted(t *testing.T) {
	testData := setupApiKeyTest("API Key Required Test", true)
	apiKey := api_keys.CreateTestApiKey("Test API Key", testData.Project.ID, testData.User.Token, testData.Router)

	response := submitTestLogs(t, testData.Router, testData.Project.ID, apiKey.Token, testData.UniqueID)

	assert.Equal(t, 1, response.Accepted)
	assert.Equal(t, 0, response.Rejected)
	assert.Empty(t, response.Errors)
}

func Test_SubmitLogs_WhenApiKeyRequired_WithoutKey_ReturnsUnauthorized(t *testing.T) {
	testData := setupApiKeyTest("API Key Required No Key Test", true)

	resp := submitTestLogsExpectingError(
		t,
		testData.Router,
		testData.Project.ID,
		"",
		testData.UniqueID,
		http.StatusUnauthorized,
	)

	assert.Contains(t, string(resp.Body), "API key required")
}

func Test_SubmitLogs_WhenApiKeyRequired_WithInvalidKey_ReturnsUnauthorized(t *testing.T) {
	testData := setupApiKeyTest("API Key Required Invalid Key Test", true)
	invalidToken := generateInvalidApiKeyToken()

	resp := submitTestLogsExpectingError(
		t,
		testData.Router,
		testData.Project.ID,
		invalidToken,
		testData.UniqueID,
		http.StatusUnauthorized,
	)

	assert.Contains(t, string(resp.Body), "invalid API key")
}

func Test_SubmitLogs_WhenApiKeyRequired_WithDisabledKey_ReturnsUnauthorized(t *testing.T) {
	testData := setupApiKeyTest("API Key Required Disabled Key Test", true)
	apiKey := api_keys.CreateTestApiKey("Disabled API Key", testData.Project.ID, testData.User.Token, testData.Router)

	disableApiKey(t, testData.Router, testData.Project.ID, apiKey.ID, testData.User.Token)

	resp := submitTestLogsExpectingError(
		t,
		testData.Router,
		testData.Project.ID,
		apiKey.Token,
		testData.UniqueID,
		http.StatusUnauthorized,
	)

	assert.Contains(t, string(resp.Body), "invalid API key")
}

func Test_SubmitLogs_WhenApiKeyNotRequired_WithoutKey_LogsAccepted(t *testing.T) {
	testData := setupApiKeyTest("API Key Not Required Test", false)

	response := submitTestLogs(t, testData.Router, testData.Project.ID, "", testData.UniqueID)

	assert.Equal(t, 1, response.Accepted)
	assert.Equal(t, 0, response.Rejected)
	assert.Empty(t, response.Errors)
}

func Test_SubmitLogs_WhenApiKeyNotRequired_WithValidKey_LogsAccepted(t *testing.T) {
	testData := setupApiKeyTest("API Key Not Required With Key Test", false)
	apiKey := api_keys.CreateTestApiKey("Optional API Key", testData.Project.ID, testData.User.Token, testData.Router)

	response := submitTestLogs(t, testData.Router, testData.Project.ID, apiKey.Token, testData.UniqueID)

	assert.Equal(t, 1, response.Accepted)
	assert.Equal(t, 0, response.Rejected)
	assert.Empty(t, response.Errors)
}

func Test_SubmitLogs_WhenApiKeyNotRequired_WithInvalidKey_LogsAccepted(t *testing.T) {
	testData := setupApiKeyTest("API Key Not Required Invalid Key Test", false)
	invalidToken := generateInvalidApiKeyToken()

	response := submitTestLogs(t, testData.Router, testData.Project.ID, invalidToken, testData.UniqueID)

	assert.Equal(t, 1, response.Accepted)
	assert.Equal(t, 0, response.Rejected)
	assert.Empty(t, response.Errors)
}

func Test_SubmitLogs_WhenApiKeyFromDifferentProject_ReturnsUnauthorized(t *testing.T) {
	testData1 := setupApiKeyTest("API Key Project 1", true)
	testData2 := setupApiKeyTest("API Key Project 2", true)

	// Create API key for project 1, try to use it for project 2
	apiKey := api_keys.CreateTestApiKey(
		"Cross Project Key",
		testData1.Project.ID,
		testData1.User.Token,
		testData1.Router,
	)

	resp := submitTestLogsExpectingError(
		t,
		testData2.Router,
		testData2.Project.ID,
		apiKey.Token,
		testData2.UniqueID,
		http.StatusUnauthorized,
	)

	assert.Contains(t, string(resp.Body), "invalid API key")
}

type ApiKeyTestData struct {
	Router   *gin.Engine
	User     *users_dto.SignInResponseDTO
	Project  *projects_models.Project
	UniqueID string
}

func setupApiKeyTest(testPrefix string, isApiKeyRequired bool) *ApiKeyTestData {
	router := CreateLogsTestRouter()
	user := users_testing.CreateTestUser(users_enums.UserRoleMember)
	uniqueID := uuid.New().String()
	projectName := fmt.Sprintf("%s %s", testPrefix, uniqueID[:8])

	var project *projects_models.Project
	if isApiKeyRequired {
		config := &projects_testing.ProjectConfigurationDTO{
			IsApiKeyRequired:   true,
			IsFilterByDomain:   false,
			AllowedDomains:     nil,
			IsFilterByIP:       false,
			AllowedIPs:         nil,
			LogsPerSecondLimit: 1000,
			MaxLogSizeKB:       64,
		}
		project = projects_testing.CreateTestProjectWithConfiguration(projectName, user, router, config)
	} else {
		project = projects_testing.CreateBasicTestProject(projectName, user, router)
	}

	return &ApiKeyTestData{
		Router:   router,
		User:     user,
		Project:  project,
		UniqueID: uniqueID,
	}
}

func submitTestLogs(
	t *testing.T,
	router *gin.Engine,
	projectID uuid.UUID,
	apiKeyToken, uniqueID string,
) *logs_receiving.SubmitLogsResponseDTO {
	logItems := CreateValidLogItems(1, uniqueID)
	request := &logs_receiving.SubmitLogsRequestDTO{
		Logs: logItems,
	}

	headers := make(map[string]string)
	if apiKeyToken != "" {
		headers["X-API-Key"] = apiKeyToken
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

func submitTestLogsExpectingError(
	t *testing.T,
	router *gin.Engine,
	projectID uuid.UUID,
	apiKeyToken, uniqueID string,
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

	return test_utils.MakeRequest(t, router, test_utils.RequestOptions{
		Method:         "POST",
		URL:            fmt.Sprintf("/api/v1/logs/receiving/%s", projectID.String()),
		Body:           request,
		Headers:        headers,
		ExpectedStatus: expectedStatus,
	})
}

func disableApiKey(t *testing.T, router *gin.Engine, projectID, apiKeyID uuid.UUID, userToken string) {
	status := api_keys.ApiKeyStatusDisabled
	updateRequest := api_keys.UpdateApiKeyRequestDTO{
		Status: &status,
	}
	test_utils.MakePutRequest(
		t,
		router,
		fmt.Sprintf("/api/v1/projects/api-keys/%s/%s", projectID.String(), apiKeyID.String()),
		"Bearer "+userToken,
		updateRequest,
		http.StatusOK,
	)
}

func generateInvalidApiKeyToken() string {
	return fmt.Sprintf("lb_invalid_api_key_token_%s", uuid.New().String()[:20])
}
