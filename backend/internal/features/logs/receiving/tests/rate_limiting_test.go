package logs_receiving_tests

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	logs_receiving "logbull/internal/features/logs/receiving"
	projects_models "logbull/internal/features/projects/models"
	projects_testing "logbull/internal/features/projects/testing"
	users_dto "logbull/internal/features/users/dto"
	users_enums "logbull/internal/features/users/enums"
	users_testing "logbull/internal/features/users/testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_SubmitLogs_WithinRateLimit_LogsAccepted(t *testing.T) {
	testData := setupRateLimitTest("Within Rate Limit Test", 10) // 10 logs per second

	// Submit 5 logs (well within limit)
	for i := 0; i < 5; i++ {
		response := submitTestLogsForRateLimit(
			t,
			testData.Router,
			testData.Project.ID,
			fmt.Sprintf("%s_%d", testData.UniqueID, i),
		)

		assert.Equal(t, 1, response.Accepted)
		assert.Equal(t, 0, response.Rejected)
		assert.Empty(t, response.Errors)

		// Small delay to avoid hitting any burst limits
		time.Sleep(50 * time.Millisecond)
	}
}

func Test_SubmitLogs_ExceedingRateLimit_ReturnsTooManyRequests(t *testing.T) {
	testData := setupRateLimitTest("Exceeding Rate Limit Test", 2) // Very low limit: 2 per second

	// Submit logs rapidly to exceed the rate limit
	// With 2 RPS limit and 5x burst multiplier, we start with 10 tokens
	// We need to submit more than 10 requests rapidly to hit the limit
	successCount := 0
	rateLimitHit := false

	// Try to submit 20 logs very quickly to exhaust token bucket
	for i := 0; i < 20; i++ {
		resp := submitTestLogsForRateLimitRaw(
			t,
			testData.Router,
			testData.Project.ID,
			fmt.Sprintf("%s_%d", testData.UniqueID, i),
		)

		if resp.StatusCode == http.StatusAccepted {
			successCount++
		} else if resp.StatusCode == http.StatusTooManyRequests {
			rateLimitHit = true
			assert.Contains(t, string(resp.Body), "logs per second limit exceeded")
			break
		}

		// No delay to simulate rapid burst requests
	}

	// Should have had some successful requests and hit rate limit
	assert.Greater(t, successCount, 0, "Should have at least some successful requests")
	assert.True(t, rateLimitHit, "Should have hit rate limit and received 429 status")
}

func Test_SubmitLogs_WithBurstCapacity_LogsAccepted(t *testing.T) {
	testData := setupRateLimitTest("Burst Capacity Test", 5) // 5 logs per second

	// Submit a small burst of logs quickly (within reasonable burst capacity)
	burstSize := 3
	successfulSubmissions := 0

	for i := 0; i < burstSize; i++ {
		response := submitTestLogsForRateLimit(
			t,
			testData.Router,
			testData.Project.ID,
			fmt.Sprintf("%s_burst_%d", testData.UniqueID, i),
		)

		// All burst requests should be accepted if within reasonable burst limit
		if response.Accepted > 0 {
			successfulSubmissions++
		}
	}

	// Should accept reasonable burst
	assert.Equal(t, burstSize, successfulSubmissions, "Should accept reasonable burst of requests")
}

func Test_SubmitLogs_AfterRateLimitReset_LogsAccepted(t *testing.T) {
	testData := setupRateLimitTest("Rate Limit Reset Test", 3) // 3 logs per second

	// First, exhaust the rate limit
	// With 3 RPS limit and 5x burst multiplier, we start with 15 tokens
	// We need to submit more than 15 requests rapidly to hit the limit
	rateLimitHit := false
	for i := 0; i < 25; i++ {
		resp := submitTestLogsForRateLimitRaw(
			t,
			testData.Router,
			testData.Project.ID,
			fmt.Sprintf("%s_exhaust_%d", testData.UniqueID, i),
		)

		if resp.StatusCode == http.StatusTooManyRequests {
			rateLimitHit = true
			break
		}

		// No delay to quickly exhaust the token bucket
	}

	assert.True(t, rateLimitHit, "Should have hit rate limit first")

	// Wait for rate limit window to reset (typically 1 second)
	time.Sleep(1200 * time.Millisecond)

	// Now submit should work again
	response := submitTestLogsForRateLimit(
		t,
		testData.Router,
		testData.Project.ID,
		fmt.Sprintf("%s_after_reset", testData.UniqueID),
	)

	assert.Equal(t, 1, response.Accepted)
	assert.Equal(t, 0, response.Rejected)
	assert.Empty(t, response.Errors)
}

func Test_SubmitLogs_WithHighRateLimit_AcceptsMultipleRequests(t *testing.T) {
	testData := setupRateLimitTest("High Rate Limit Test", 100) // High limit: 100 per second

	// Submit many logs quickly - should all be accepted with high limit
	requestCount := 20
	successCount := 0

	for i := 0; i < requestCount; i++ {
		response := submitTestLogsForRateLimit(
			t,
			testData.Router,
			testData.Project.ID,
			fmt.Sprintf("%s_high_rate_%d", testData.UniqueID, i),
		)

		if response.Accepted > 0 {
			successCount++
		}

		time.Sleep(5 * time.Millisecond) // Very short delay
	}

	// With high rate limit, most or all should succeed
	assert.GreaterOrEqual(t, successCount, requestCount-2, "Should accept most requests with high rate limit")
}

func Test_SubmitLogs_WithZeroRateLimit_UnlimitedAccess(t *testing.T) {
	testData := setupRateLimitTest("Zero Rate Limit Test", 0) // 0 = unlimited

	// Submit many logs rapidly - should all be accepted with unlimited rate
	requestCount := 15
	successCount := 0

	for i := 0; i < requestCount; i++ {
		response := submitTestLogsForRateLimit(
			t,
			testData.Router,
			testData.Project.ID,
			fmt.Sprintf("%s_unlimited_%d", testData.UniqueID, i),
		)

		if response.Accepted > 0 {
			successCount++
		}
	}

	// With unlimited rate (0), all should succeed
	assert.Equal(t, requestCount, successCount, "Should accept all requests with unlimited rate limit")
}

type RateLimitTestData struct {
	Router   *gin.Engine
	User     *users_dto.SignInResponseDTO
	Project  *projects_models.Project
	UniqueID string
}

func setupRateLimitTest(testPrefix string, logsPerSecondLimit int) *RateLimitTestData {
	router := CreateLogsTestRouter()
	user := users_testing.CreateTestUser(users_enums.UserRoleMember)
	uniqueID := uuid.New().String()
	projectName := fmt.Sprintf("%s %s", testPrefix, uniqueID[:8])

	config := &projects_testing.ProjectConfigurationDTO{
		IsApiKeyRequired:   false,
		IsFilterByDomain:   false,
		AllowedDomains:     nil,
		IsFilterByIP:       false,
		AllowedIPs:         nil,
		LogsPerSecondLimit: logsPerSecondLimit,
		MaxLogSizeKB:       64,
	}
	project := projects_testing.CreateTestProjectWithConfiguration(projectName, user, router, config)

	return &RateLimitTestData{
		Router:   router,
		User:     user,
		Project:  project,
		UniqueID: uniqueID,
	}
}

func submitTestLogsForRateLimit(
	t *testing.T,
	router *gin.Engine,
	projectID uuid.UUID,
	uniqueID string,
) *logs_receiving.SubmitLogsResponseDTO {
	resp := submitTestLogsForRateLimitRaw(t, router, projectID, uniqueID)

	if resp.StatusCode != http.StatusAccepted {
		t.Fatalf("Expected successful log submission, got status %d: %s", resp.StatusCode, string(resp.Body))
	}

	return resp.Response
}

type RateLimitTestResponse struct {
	StatusCode int
	Body       []byte
	Response   *logs_receiving.SubmitLogsResponseDTO
}

func submitTestLogsForRateLimitRaw(
	t *testing.T,
	router *gin.Engine,
	projectID uuid.UUID,
	uniqueID string,
) *RateLimitTestResponse {
	logItems := CreateValidLogItems(1, uniqueID)
	request := &logs_receiving.SubmitLogsRequestDTO{
		Logs: logItems,
	}

	resp := makeRateLimitTestRequest(t, router, projectID, request)

	result := &RateLimitTestResponse{
		StatusCode: resp.StatusCode,
		Body:       resp.Body,
	}

	// Only try to parse response if it was successful
	if resp.StatusCode == http.StatusAccepted {
		var response logs_receiving.SubmitLogsResponseDTO
		if err := resp.UnmarshalResponse(&response); err != nil {
			t.Fatalf("Failed to unmarshal successful response: %v", err)
		}
		result.Response = &response
	}

	return result
}

type TestResponse struct {
	StatusCode int
	Body       []byte
}

func (tr *TestResponse) UnmarshalResponse(target interface{}) error {
	return json.Unmarshal(tr.Body, target)
}

func makeRateLimitTestRequest(
	t *testing.T,
	router *gin.Engine,
	projectID uuid.UUID,
	request *logs_receiving.SubmitLogsRequestDTO,
) *TestResponse {
	jsonBody, err := json.Marshal(request)
	if err != nil {
		t.Fatalf("Failed to marshal request: %v", err)
	}

	req := httptest.NewRequest(
		"POST",
		fmt.Sprintf("/api/v1/logs/receiving/%s", projectID.String()),
		bytes.NewReader(jsonBody),
	)
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	return &TestResponse{
		StatusCode: w.Code,
		Body:       w.Body.Bytes(),
	}
}
