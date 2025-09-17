package rate_limit

import (
	"context"
	"fmt"
	"logbull/internal/cache"
	"math"
	"time"

	"github.com/google/uuid"
	"github.com/valkey-io/valkey-go"
)

type RateLimiter struct {
	client valkey.Client
}

type RateLimitResult struct {
	Allowed       bool      `json:"allowed"`
	Remaining     int       `json:"remaining"`
	ResetTime     time.Time `json:"resetTime"`
	RetryAfterSec int       `json:"retryAfterSec,omitempty"`
}

const (
	defaultTimeout = 5 * time.Second
	keyPrefix      = "rate_limit:project:"
)

// Lua script for token bucket rate limiting
// This script atomically:
// 1. Gets current token count and last refill time
// 2. Calculates tokens to add based on time elapsed
// 3. Checks if request can be allowed
// 4. Updates token count and timestamp
const tokenBucketLuaScript = `
local key = KEYS[1]
local now = tonumber(ARGV[1])
local rps_limit = tonumber(ARGV[2])
local burst_limit = tonumber(ARGV[3])
local ttl = tonumber(ARGV[4])

-- Get current state
local current = redis.call('HMGET', key, 'tokens', 'last_refill')
local tokens = tonumber(current[1]) or burst_limit
local last_refill = tonumber(current[2]) or now

-- Calculate time elapsed and tokens to add
local elapsed = math.max(0, now - last_refill)
local tokens_to_add = math.floor(elapsed * rps_limit / 1000)
tokens = math.min(burst_limit, tokens + tokens_to_add)

-- Check if request can be allowed
local allowed = 0
local remaining = tokens
if tokens >= 1 then
    allowed = 1
    tokens = tokens - 1
    remaining = tokens
end

-- Update state
redis.call('HMSET', key, 'tokens', tokens, 'last_refill', now)
redis.call('EXPIRE', key, ttl)

-- Calculate reset time (when bucket will be full again)
local time_to_full = 0
if tokens < burst_limit then
    time_to_full = math.ceil((burst_limit - tokens) * 1000 / rps_limit)
end

return {allowed, remaining, time_to_full}
`

func NewRateLimiter() *RateLimiter {
	return &RateLimiter{
		client: cache.GetCache(),
	}
}

func (r *RateLimiter) CheckRateLimit(projectID uuid.UUID, rpsLimit, burstLimit int) (*RateLimitResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	// Validate parameters
	if rpsLimit <= 0 {
		rpsLimit = 100 // Default fallback
	}
	if burstLimit <= 0 {
		burstLimit = max(rpsLimit*5, 500) // Default burst is 5x RPS or 500, whichever is higher
	}

	key := keyPrefix + projectID.String()
	now := time.Now().UnixMilli()
	ttl := int64(300) // 5 minutes TTL for cleanup

	// Execute Lua script
	result := r.client.Do(ctx, r.client.B().Eval().
		Script(tokenBucketLuaScript).
		Numkeys(1).
		Key(key).
		Arg(fmt.Sprintf("%d", now)).
		Arg(fmt.Sprintf("%d", rpsLimit)).
		Arg(fmt.Sprintf("%d", burstLimit)).
		Arg(fmt.Sprintf("%d", ttl)).
		Build())

	if result.Error() != nil {
		return nil, fmt.Errorf("rate limit check failed: %w", result.Error())
	}

	values, err := result.AsIntSlice()
	if err != nil {
		return nil, fmt.Errorf("failed to parse rate limit result: %w", err)
	}

	if len(values) < 3 {
		return nil, fmt.Errorf("invalid rate limit result: expected 3 values, got %d", len(values))
	}

	allowed := values[0] == 1
	remaining := int(values[1])
	timeToFullMs := values[2]

	resetTime := time.Now().Add(time.Duration(timeToFullMs) * time.Millisecond)

	var retryAfterSec int
	if !allowed {
		// If not allowed, suggest retry after enough time for at least 1 token
		// Calculate milliseconds per token, then convert to seconds
		retryAfterMs := 1000.0 / float64(rpsLimit)
		retryAfterSec = int(math.Ceil(retryAfterMs / 1000.0))
		if retryAfterSec < 1 {
			retryAfterSec = 1
		}
	}

	return &RateLimitResult{
		Allowed:       allowed,
		Remaining:     remaining,
		ResetTime:     resetTime,
		RetryAfterSec: retryAfterSec,
	}, nil
}

func (r *RateLimiter) ResetRateLimit(projectID uuid.UUID) error {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	key := keyPrefix + projectID.String()

	result := r.client.Do(ctx, r.client.B().Del().Key(key).Build())
	return result.Error()
}

// GetRateLimitInfo returns current rate limit status without consuming a token
func (r *RateLimiter) GetRateLimitInfo(projectID uuid.UUID, rpsLimit, burstLimit int) (*RateLimitResult, error) {
	ctx, cancel := context.WithTimeout(context.Background(), defaultTimeout)
	defer cancel()

	key := keyPrefix + projectID.String()

	result := r.client.Do(ctx, r.client.B().Hmget().Key(key).Field("tokens", "last_refill").Build())
	if result.Error() != nil {
		// If key doesn't exist, assume full bucket
		return &RateLimitResult{
			Allowed:   true,
			Remaining: burstLimit,
			ResetTime: time.Now(),
		}, nil
	}

	values, err := result.AsStrSlice()
	if err != nil {
		return nil, fmt.Errorf("failed to get rate limit info: %w", err)
	}

	now := time.Now()
	var tokens = burstLimit
	var lastRefill = now.UnixMilli()

	if len(values) >= 2 && values[0] != "" && values[1] != "" {
		if t, err := fmt.Sscanf(values[0], "%d", &tokens); err != nil || t != 1 {
			tokens = burstLimit
		}
		if t, err := fmt.Sscanf(values[1], "%d", &lastRefill); err != nil || t != 1 {
			lastRefill = now.UnixMilli()
		}
	}

	// Calculate current tokens based on time elapsed
	elapsed := float64(now.UnixMilli() - lastRefill)
	tokensToAdd := int(math.Floor(elapsed * float64(rpsLimit) / 1000.0))
	currentTokens := min(burstLimit, tokens+tokensToAdd)

	// Calculate reset time
	var resetTime time.Time
	if currentTokens < burstLimit {
		timeToFullMs := int64(math.Ceil(float64(burstLimit-currentTokens) * 1000.0 / float64(rpsLimit)))
		resetTime = now.Add(time.Duration(timeToFullMs) * time.Millisecond)
	} else {
		resetTime = now
	}

	return &RateLimitResult{
		Allowed:   currentTokens >= 1,
		Remaining: max(0, currentTokens-1), // What would remain after taking a token
		ResetTime: resetTime,
	}, nil
}
