package logs_querying

import (
	"context"
	"fmt"
	"log/slog"
	logs_core "logbull/internal/features/logs/core"
	"time"

	"github.com/google/uuid"
	"github.com/valkey-io/valkey-go"
)

type ConcurrentQueryLimiter struct {
	client valkey.Client
	logger *slog.Logger
}

const (
	maxConcurrentQueries = 3
	queryKeyPrefix       = "concurrent_queries:user:"
	queryTimeout         = 30 * time.Minute // Auto-cleanup stale queries
)

func (l *ConcurrentQueryLimiter) AcquireQuerySlot(userID uuid.UUID, queryID string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := queryKeyPrefix + userID.String()

	result := l.client.Do(ctx, l.client.B().Incr().Key(key).Build())
	if result.Error() != nil {
		return fmt.Errorf("failed to increment query counter: %w", result.Error())
	}

	currentCount, err := result.AsInt64()
	if err != nil {
		return fmt.Errorf("failed to get current count: %w", err)
	}

	if currentCount > int64(maxConcurrentQueries) {
		l.client.Do(ctx, l.client.B().Decr().Key(key).Build())
		return &ValidationError{
			Code:    logs_core.ErrorTooManyConcurrentQueries,
			Message: fmt.Sprintf("maximum concurrent queries exceeded (%d/%d)", currentCount-1, maxConcurrentQueries),
		}
	}

	// Set TTL for cleanup (only set if this is a new key or TTL expired)
	l.client.Do(ctx, l.client.B().Expire().Key(key).Seconds(int64(queryTimeout.Seconds())).Build())

	return nil
}

func (l *ConcurrentQueryLimiter) ReleaseQuerySlot(userID uuid.UUID, queryID string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := queryKeyPrefix + userID.String()

	result := l.client.Do(ctx, l.client.B().Decr().Key(key).Build())
	if result.Error() != nil {
		l.logger.Error("Failed to release query slot",
			slog.String("userId", userID.String()),
			slog.String("queryId", queryID),
			slog.String("error", result.Error().Error()))
	}
}

func (l *ConcurrentQueryLimiter) GetActiveQueryCount(userID uuid.UUID) (int, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := queryKeyPrefix + userID.String()
	result := l.client.Do(ctx, l.client.B().Get().Key(key).Build())

	if result.Error() != nil {
		// Key doesn't exist, return 0
		return 0, nil
	}

	count, err := result.AsInt64()
	if err != nil {
		// Key exists but not a number, return 0
		return 0, nil
	}

	return int(count), nil
}

// CleanupAllQuerySlots clears all concurrent query tracking on application startup
// This prevents stale query slots from previous application runs
func (l *ConcurrentQueryLimiter) CleanupAllQuerySlots() error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	pattern := queryKeyPrefix + "*"
	keysResult := l.client.Do(ctx, l.client.B().Keys().Pattern(pattern).Build())

	if keysResult.Error() != nil {
		l.logger.Error("Failed to find query tracking keys",
			slog.String("error", keysResult.Error().Error()))
		return fmt.Errorf("failed to find query tracking keys: %w", keysResult.Error())
	}

	keys, err := keysResult.AsStrSlice()
	if err != nil {
		return fmt.Errorf("failed to parse keys result: %w", err)
	}

	if len(keys) == 0 {
		return nil
	}

	delResult := l.client.Do(ctx, l.client.B().Del().Key(keys...).Build())
	if delResult.Error() != nil {
		l.logger.Error("Failed to delete stale query tracking keys",
			slog.String("error", delResult.Error().Error()))
		return fmt.Errorf("failed to delete stale keys: %w", delResult.Error())
	}

	deletedCount, _ := delResult.AsInt64()

	if deletedCount > 0 {
		l.logger.Info("Cleaned up stale query tracking keys",
			slog.Int("count", int(deletedCount)))
	}

	return nil
}
