package helpers

import (
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

// MockRedis represents a mocked Redis connection for testing
type MockRedis struct {
	Client *redis.Client
	Mock   redismock.ClientMock
}

// NewMockRedis creates a new mock Redis client
func NewMockRedis() *MockRedis {
	client, mock := redismock.NewClientMock()

	return &MockRedis{
		Client: client,
		Mock:   mock,
	}
}

// Close closes the mock Redis connection
func (m *MockRedis) Close() error {
	return m.Client.Close()
}

// ExpectationsWereMet checks if all expected Redis interactions were met
func (m *MockRedis) ExpectationsWereMet(t *testing.T) {
	require.NoError(t, m.Mock.ExpectationsWereMet())
}

// ExpectCacheHit sets up expectation for a cache hit
func (m *MockRedis) ExpectCacheHit(key, value string) {
	m.Mock.ExpectGet(key).SetVal(value)
}

// ExpectCacheMiss sets up expectation for a cache miss
func (m *MockRedis) ExpectCacheMiss(key string) {
	m.Mock.ExpectGet(key).RedisNil()
}

// ExpectCacheSet sets up expectation for setting a cache value
func (m *MockRedis) ExpectCacheSet(key, value string) {
	m.Mock.ExpectSet(key, value, 0).SetVal("OK")
}

// ExpectCacheSetWithTTL sets up expectation for setting a cache value with TTL
func (m *MockRedis) ExpectCacheSetWithTTL(key, value string, ttlSeconds int) {
	m.Mock.ExpectSetEx(key, value, time.Duration(ttlSeconds)*time.Second).SetVal("OK")
}

// ExpectCacheDel sets up expectation for deleting a cache key
func (m *MockRedis) ExpectCacheDel(key string) {
	m.Mock.ExpectDel(key).SetVal(1)
}

// ExpectPing sets up expectation for ping command
func (m *MockRedis) ExpectPing() {
	m.Mock.ExpectPing().SetVal("PONG")
}

// ExpectFlushDB sets up expectation for flushing database
func (m *MockRedis) ExpectFlushDB() {
	m.Mock.ExpectFlushDB().SetVal("OK")
}

// ExpectExists sets up expectation for checking key existence
func (m *MockRedis) ExpectExists(key string, exists bool) {
	if exists {
		m.Mock.ExpectExists(key).SetVal(1)
	} else {
		m.Mock.ExpectExists(key).SetVal(0)
	}
}

// ExpectTTL sets up expectation for getting TTL
func (m *MockRedis) ExpectTTL(key string, ttlSeconds int) {
	m.Mock.ExpectTTL(key).SetVal(time.Duration(ttlSeconds) * time.Second)
}

// ExpectHSet sets up expectation for setting hash field
func (m *MockRedis) ExpectHSet(key, field, value string) {
	m.Mock.ExpectHSet(key, field, value).SetVal(1)
}

// ExpectHGet sets up expectation for getting hash field
func (m *MockRedis) ExpectHGet(key, field, value string) {
	m.Mock.ExpectHGet(key, field).SetVal(value)
}

// ExpectHGetAll sets up expectation for getting all hash fields
func (m *MockRedis) ExpectHGetAll(key string, values map[string]string) {
	m.Mock.ExpectHGetAll(key).SetVal(values)
}
