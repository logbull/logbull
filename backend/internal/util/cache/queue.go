package cache_utils

import (
	"context"
	"logbull/internal/cache"
	"time"

	"github.com/valkey-io/valkey-go"
)

type ValkeyQueueService struct {
	client  valkey.Client
	timeout time.Duration
}

func NewValkeyQueueService() *ValkeyQueueService {
	return &ValkeyQueueService{
		client:  cache.GetCache(),
		timeout: DefaultQueueTimeout,
	}
}

func (q *ValkeyQueueService) EnqueueBatch(queueKey string, items [][]byte) error {
	if len(items) == 0 {
		return nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), q.timeout)
	defer cancel()

	// Use pipeline for batch operations to handle high throughput
	cmds := make([]valkey.Completed, 0, len(items))

	for _, item := range items {
		cmd := q.client.B().Lpush().Key(queueKey).Element(string(item)).Build()
		cmds = append(cmds, cmd)
	}

	// Execute all commands in a pipeline for maximum performance
	results := q.client.DoMulti(ctx, cmds...)

	// Check for errors in any of the operations
	for _, result := range results {
		if result.Error() != nil {
			return result.Error()
		}
	}

	return nil
}

func (q *ValkeyQueueService) DequeueBatch(queueKey string, maxCount int, timeout time.Duration) ([][]byte, error) {
	if maxCount <= 0 {
		return nil, nil
	}

	ctx, cancel := context.WithTimeout(context.Background(), q.timeout)
	defer cancel()

	var results [][]byte

	// For batch dequeue, we use multiple RPOP operations in a pipeline
	// since BRPOP only returns one item at a time
	cmds := make([]valkey.Completed, 0, maxCount)

	for range maxCount {
		cmd := q.client.B().Rpop().Key(queueKey).Build()
		cmds = append(cmds, cmd)
	}

	// Execute pipeline
	responses := q.client.DoMulti(ctx, cmds...)

	for _, response := range responses {
		if response.Error() != nil {
			// If error is "nil reply", it means queue is empty - this is expected
			if response.Error() == valkey.Nil {
				break
			}
			return results, response.Error()
		}

		data, err := response.AsBytes()
		if err != nil {
			return results, err
		}

		results = append(results, data)
	}

	return results, nil
}

func (q *ValkeyQueueService) DequeueBlocking(queueKey string, timeout time.Duration) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), q.timeout)
	defer cancel()

	cmd := q.client.B().Brpop().Key(queueKey).Timeout(timeout.Seconds()).Build()
	result := q.client.Do(ctx, cmd)

	if result.Error() != nil {
		return nil, result.Error()
	}

	// BRPOP returns [key, value], we want the value
	arr, err := result.AsStrSlice()
	if err != nil {
		return nil, err
	}

	if len(arr) < 2 {
		return nil, valkey.Nil
	}

	return []byte(arr[1]), nil
}

func (q *ValkeyQueueService) QueueLength(queueKey string) (int64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), q.timeout)
	defer cancel()

	cmd := q.client.B().Llen().Key(queueKey).Build()
	result := q.client.Do(ctx, cmd)

	if result.Error() != nil {
		return 0, result.Error()
	}

	return result.AsInt64()
}

func (q *ValkeyQueueService) ClearQueue(queueKey string) error {
	ctx, cancel := context.WithTimeout(context.Background(), q.timeout)
	defer cancel()

	cmd := q.client.B().Del().Key(queueKey).Build()
	result := q.client.Do(ctx, cmd)

	return result.Error()
}
