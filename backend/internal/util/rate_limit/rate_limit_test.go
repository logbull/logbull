package rate_limit

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
)

func Test_CheckRateLimit_WithinLimits_AllowsRequest(t *testing.T) {
	rateLimiter := NewRateLimiter()
	projectID := uuid.New()
	rpsLimit := 10
	burstLimit := 20

	// Clean up any existing data
	rateLimiter.ResetRateLimit(projectID)

	result, err := rateLimiter.CheckRateLimit(projectID, rpsLimit, burstLimit)

	assert.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, burstLimit-1, result.Remaining) // Should have burst - 1 tokens remaining
	assert.Equal(t, 0, result.RetryAfterSec)
	assert.True(t, result.ResetTime.After(time.Now().Add(-time.Second)))
}

func Test_CheckRateLimit_ExceedsBurstLimit_DeniesRequest(t *testing.T) {
	rateLimiter := NewRateLimiter()
	projectID := uuid.New()
	rpsLimit := 1   // Very low RPS to make it easy to exceed
	burstLimit := 2 // Small burst limit

	// Clean up any existing data
	rateLimiter.ResetRateLimit(projectID)

	// Consume the burst tokens
	for i := 0; i < burstLimit; i++ {
		result, err := rateLimiter.CheckRateLimit(projectID, rpsLimit, burstLimit)
		assert.NoError(t, err)
		assert.True(t, result.Allowed, "Request %d should be allowed", i+1)
	}

	// The next request should be denied
	result, err := rateLimiter.CheckRateLimit(projectID, rpsLimit, burstLimit)
	assert.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.Equal(t, 0, result.Remaining)
	assert.True(t, result.RetryAfterSec > 0)
	assert.True(t, result.ResetTime.After(time.Now()))
}

func Test_CheckRateLimit_TokensRefillOverTime_AllowsRequestsAfterWait(t *testing.T) {
	rateLimiter := NewRateLimiter()
	projectID := uuid.New()
	rpsLimit := 10  // 10 RPS means 1 token every 100ms
	burstLimit := 1 // Only 1 token in the bucket

	// Clean up any existing data
	rateLimiter.ResetRateLimit(projectID)

	// Use the only token
	result, err := rateLimiter.CheckRateLimit(projectID, rpsLimit, burstLimit)
	assert.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, 0, result.Remaining)

	// Immediately try again - should be denied
	result, err = rateLimiter.CheckRateLimit(projectID, rpsLimit, burstLimit)
	assert.NoError(t, err)
	assert.False(t, result.Allowed)

	// Wait for tokens to refill (100ms for 1 token at 10 RPS, plus some buffer)
	time.Sleep(150 * time.Millisecond)

	// Now it should be allowed again
	result, err = rateLimiter.CheckRateLimit(projectID, rpsLimit, burstLimit)
	assert.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, 0, result.Remaining)
}

func Test_CheckRateLimit_DifferentProjects_IsolatedLimits(t *testing.T) {
	rateLimiter := NewRateLimiter()
	projectID1 := uuid.New()
	projectID2 := uuid.New()
	rpsLimit := 1
	burstLimit := 1

	// Clean up any existing data
	rateLimiter.ResetRateLimit(projectID1)
	rateLimiter.ResetRateLimit(projectID2)

	// Use up project1's token
	result1, err := rateLimiter.CheckRateLimit(projectID1, rpsLimit, burstLimit)
	assert.NoError(t, err)
	assert.True(t, result1.Allowed)

	// Project1 should now be denied
	result1, err = rateLimiter.CheckRateLimit(projectID1, rpsLimit, burstLimit)
	assert.NoError(t, err)
	assert.False(t, result1.Allowed)

	// But project2 should still be allowed (isolated buckets)
	result2, err := rateLimiter.CheckRateLimit(projectID2, rpsLimit, burstLimit)
	assert.NoError(t, err)
	assert.True(t, result2.Allowed)
}

func Test_CheckRateLimit_WithDefaultValues_HandlesInvalidParameters(t *testing.T) {
	rateLimiter := NewRateLimiter()
	projectID := uuid.New()

	// Clean up any existing data
	rateLimiter.ResetRateLimit(projectID)

	// Test with invalid RPS limit (should use default)
	result, err := rateLimiter.CheckRateLimit(projectID, 0, 10)
	assert.NoError(t, err)
	assert.True(t, result.Allowed)

	// Test with invalid burst limit (should calculate default)
	rateLimiter.ResetRateLimit(projectID)
	result, err = rateLimiter.CheckRateLimit(projectID, 10, 0)
	assert.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.True(t, result.Remaining >= 49) // Should be at least 50 tokens (5x RPS) minus 1

	// Test with both invalid (should use defaults)
	rateLimiter.ResetRateLimit(projectID)
	result, err = rateLimiter.CheckRateLimit(projectID, -1, -1)
	assert.NoError(t, err)
	assert.True(t, result.Allowed)
}

func Test_ResetRateLimit_ClearsRateLimitData(t *testing.T) {
	rateLimiter := NewRateLimiter()
	projectID := uuid.New()
	rpsLimit := 1
	burstLimit := 1

	// Clean up any existing data
	rateLimiter.ResetRateLimit(projectID)

	// Use up the token
	result, err := rateLimiter.CheckRateLimit(projectID, rpsLimit, burstLimit)
	assert.NoError(t, err)
	assert.True(t, result.Allowed)

	// Should be denied now
	result, err = rateLimiter.CheckRateLimit(projectID, rpsLimit, burstLimit)
	assert.NoError(t, err)
	assert.False(t, result.Allowed)

	// Reset the rate limit
	err = rateLimiter.ResetRateLimit(projectID)
	assert.NoError(t, err)

	// Should be allowed again immediately (full bucket)
	result, err = rateLimiter.CheckRateLimit(projectID, rpsLimit, burstLimit)
	assert.NoError(t, err)
	assert.True(t, result.Allowed)
}

func Test_GetRateLimitInfo_ReturnsCurrentStatusWithoutConsumingToken(t *testing.T) {
	rateLimiter := NewRateLimiter()
	projectID := uuid.New()
	rpsLimit := 10
	burstLimit := 20

	// Clean up any existing data
	rateLimiter.ResetRateLimit(projectID)

	// Get info without consuming token
	info, err := rateLimiter.GetRateLimitInfo(projectID, rpsLimit, burstLimit)
	assert.NoError(t, err)
	assert.True(t, info.Allowed)
	assert.Equal(t, burstLimit-1, info.Remaining) // What would remain after taking a token

	// Consume a token
	result, err := rateLimiter.CheckRateLimit(projectID, rpsLimit, burstLimit)
	assert.NoError(t, err)
	assert.True(t, result.Allowed)
	assert.Equal(t, burstLimit-1, result.Remaining)

	// Check info again - should show one less token available
	info, err = rateLimiter.GetRateLimitInfo(projectID, rpsLimit, burstLimit)
	assert.NoError(t, err)
	assert.True(t, info.Allowed)
	assert.Equal(t, burstLimit-2, info.Remaining) // Should show what would remain after taking a token
}

func Test_GetRateLimitInfo_ForNonExistentProject_ReturnsFullBucket(t *testing.T) {
	rateLimiter := NewRateLimiter()
	projectID := uuid.New() // New project, no rate limit data
	rpsLimit := 10
	burstLimit := 20

	// Get info for non-existent project
	info, err := rateLimiter.GetRateLimitInfo(projectID, rpsLimit, burstLimit)
	assert.NoError(t, err)
	assert.True(t, info.Allowed)
	assert.Equal(t, burstLimit-1, info.Remaining) // Full bucket minus what would be consumed
}

func Test_CheckRateLimit_HighThroughput_MaintainsAccuracy(t *testing.T) {
	rateLimiter := NewRateLimiter()
	projectID := uuid.New()
	rpsLimit := 100
	burstLimit := 10 // Small burst to make testing easier

	// Clean up any existing data
	rateLimiter.ResetRateLimit(projectID)

	allowedCount := 0
	deniedCount := 0
	totalRequests := 50

	// Make many requests quickly
	for i := 0; i < totalRequests; i++ {
		result, err := rateLimiter.CheckRateLimit(projectID, rpsLimit, burstLimit)
		assert.NoError(t, err)

		if result.Allowed {
			allowedCount++
		} else {
			deniedCount++
		}
	}

	// Should allow exactly burstLimit requests initially
	assert.Equal(t, burstLimit, allowedCount)
	assert.Equal(t, totalRequests-burstLimit, deniedCount)
}

func Test_CheckRateLimit_RetryAfterSeconds_CalculatedCorrectly(t *testing.T) {
	rateLimiter := NewRateLimiter()
	projectID := uuid.New()
	rpsLimit := 2 // 2 RPS = 500ms per token
	burstLimit := 1

	// Clean up any existing data
	rateLimiter.ResetRateLimit(projectID)

	// Use the only token
	result, err := rateLimiter.CheckRateLimit(projectID, rpsLimit, burstLimit)
	assert.NoError(t, err)
	assert.True(t, result.Allowed)

	// Next request should be denied with appropriate retry-after
	result, err = rateLimiter.CheckRateLimit(projectID, rpsLimit, burstLimit)
	assert.NoError(t, err)
	assert.False(t, result.Allowed)
	assert.True(t, result.RetryAfterSec > 0)
	// For 2 RPS, retry after should be at most 1 second (500ms rounded up)
	assert.True(t, result.RetryAfterSec <= 1)
}

func Test_CheckRateLimit_ResetTimeCalculation_IsAccurate(t *testing.T) {
	rateLimiter := NewRateLimiter()
	projectID := uuid.New()
	rpsLimit := 10 // 10 RPS = 100ms per token
	burstLimit := 5

	// Clean up any existing data
	rateLimiter.ResetRateLimit(projectID)

	// Use all tokens
	for i := 0; i < burstLimit; i++ {
		result, err := rateLimiter.CheckRateLimit(projectID, rpsLimit, burstLimit)
		assert.NoError(t, err)
		assert.True(t, result.Allowed)
	}

	// Next request should be denied with reset time in the future
	result, err := rateLimiter.CheckRateLimit(projectID, rpsLimit, burstLimit)
	assert.NoError(t, err)
	assert.False(t, result.Allowed)

	now := time.Now()
	assert.True(t, result.ResetTime.After(now))
	// Reset time should be reasonable (within a few seconds for full refill)
	assert.True(t, result.ResetTime.Before(now.Add(10*time.Second)))
}
