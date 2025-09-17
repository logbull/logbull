package cache_utils

import (
	"context"
	"encoding/json"
	"time"

	"logbull/internal/cache"

	"github.com/valkey-io/valkey-go"
)

const (
	DefaultCacheTimeout = 10 * time.Second
	DefaultCacheExpiry  = 10 * time.Minute
	DefaultQueueTimeout = 30 * time.Second
)

type CacheUtil[T any] struct {
	client  valkey.Client
	prefix  string
	timeout time.Duration
	expiry  time.Duration
}

func NewCacheUtil[T any](client valkey.Client, prefix string) *CacheUtil[T] {
	return &CacheUtil[T]{
		client:  client,
		prefix:  prefix,
		timeout: DefaultCacheTimeout,
		expiry:  DefaultCacheExpiry,
	}
}

func TestCacheConnection() {
	// Get Valkey client from cache package
	client := cache.GetCache()

	// Create a simple test cache util for strings
	cacheUtil := NewCacheUtil[string](client, "test:")

	// Test data
	testKey := "connection_test"
	testValue := "valkey_is_working"

	// Test Set operation
	cacheUtil.Set(testKey, &testValue)

	// Test Get operation
	retrievedValue := cacheUtil.Get(testKey)

	// Verify the value was retrieved correctly
	if retrievedValue == nil {
		panic("Cache test failed: could not retrieve cached value")
	}

	if *retrievedValue != testValue {
		panic("Cache test failed: retrieved value does not match expected")
	}

	// Clean up - remove test key
	cacheUtil.Invalidate(testKey)

	// Verify cleanup worked
	cleanupCheck := cacheUtil.Get(testKey)
	if cleanupCheck != nil {
		panic("Cache test failed: test key was not properly invalidated")
	}
}

func (c *CacheUtil[T]) Get(key string) *T {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	fullKey := c.prefix + key
	result := c.client.Do(ctx, c.client.B().Get().Key(fullKey).Build())

	if result.Error() != nil {
		return nil
	}

	data, err := result.AsBytes()
	if err != nil {
		return nil
	}

	var item T
	if err := json.Unmarshal(data, &item); err != nil {
		return nil
	}

	return &item
}

func (c *CacheUtil[T]) Set(key string, item *T) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	data, err := json.Marshal(item)
	if err != nil {
		return
	}

	fullKey := c.prefix + key
	c.client.Do(ctx, c.client.B().Set().Key(fullKey).Value(string(data)).Ex(c.expiry).Build())
}

func (c *CacheUtil[T]) Invalidate(key string) {
	ctx, cancel := context.WithTimeout(context.Background(), c.timeout)
	defer cancel()

	fullKey := c.prefix + key
	c.client.Do(ctx, c.client.B().Del().Key(fullKey).Build())
}
