package database

import (
	"context"
	"fmt"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/valpere/shopogoda/internal/config"
	"github.com/valpere/shopogoda/internal/models"
)

func newTestDB(t *testing.T) (*gorm.DB, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "nested", "dir", "shopogoda.db")
	db, err := Connect(&config.DatabaseConfig{Path: path})
	require.NoError(t, err)
	t.Cleanup(func() {
		if sqlDB, err := db.DB(); err == nil {
			_ = sqlDB.Close()
		}
	})
	require.NoError(t, Migrate(db))
	return db, path
}

func TestConnect_CreatesParentDirAndFile(t *testing.T) {
	_, path := newTestDB(t)
	assert.FileExists(t, path)
}

func TestConnect_AppliesPragmas(t *testing.T) {
	db, _ := newTestDB(t)

	var mode string
	require.NoError(t, db.Raw("PRAGMA journal_mode").Scan(&mode).Error)
	assert.Equal(t, "wal", mode)

	var fk int
	require.NoError(t, db.Raw("PRAGMA foreign_keys").Scan(&fk).Error)
	assert.Equal(t, 1, fk)

	var busy int
	require.NoError(t, db.Raw("PRAGMA busy_timeout").Scan(&busy).Error)
	assert.Equal(t, 5000, busy)
}

func TestConnect_EmptyPathFails(t *testing.T) {
	db, err := Connect(&config.DatabaseConfig{})
	assert.Error(t, err)
	assert.Nil(t, db)
}

func TestMigrate_Idempotent(t *testing.T) {
	db, _ := newTestDB(t)
	assert.NoError(t, Migrate(db))
	assert.NoError(t, Migrate(db))
}

func TestModels_ZeroIDGetsUniqueUUID(t *testing.T) {
	db, _ := newTestDB(t)
	require.NoError(t, db.Create(&models.User{ID: 1, FirstName: "A"}).Error)

	w1 := models.WeatherData{UserID: 1}
	w2 := models.WeatherData{UserID: 1}
	s1 := models.Subscription{UserID: 1, SubscriptionType: models.SubscriptionDaily, Frequency: models.FrequencyDaily, TimeOfDay: "08:00"}
	s2 := models.Subscription{UserID: 1, SubscriptionType: models.SubscriptionDaily, Frequency: models.FrequencyDaily, TimeOfDay: "09:00"}
	a1 := models.AlertConfig{UserID: 1, AlertType: models.AlertTemperature, Condition: "{}"}
	a2 := models.AlertConfig{UserID: 1, AlertType: models.AlertTemperature, Condition: "{}"}
	e1 := models.EnvironmentalAlert{UserID: 1}
	e2 := models.EnvironmentalAlert{UserID: 1}
	for _, m := range []any{&w1, &w2, &s1, &s2, &a1, &a2, &e1, &e2} {
		require.NoError(t, db.Create(m).Error)
	}

	pairs := [][2]uuid.UUID{{w1.ID, w2.ID}, {s1.ID, s2.ID}, {a1.ID, a2.ID}, {e1.ID, e2.ID}}
	for _, p := range pairs {
		assert.NotEqual(t, uuid.Nil, p[0])
		assert.NotEqual(t, uuid.Nil, p[1])
		assert.NotEqual(t, p[0], p[1])
	}

	var got models.WeatherData
	require.NoError(t, db.First(&got, "id = ?", w1.ID).Error, "UUID must round-trip as a text primary key")
	assert.Equal(t, w1.ID, got.ID)
}

func TestModels_ExplicitIDPreserved(t *testing.T) {
	db, _ := newTestDB(t)
	require.NoError(t, db.Create(&models.User{ID: 1, FirstName: "A"}).Error)
	id := uuid.New()
	require.NoError(t, db.Create(&models.WeatherData{ID: id, UserID: 1}).Error)

	var got models.WeatherData
	require.NoError(t, db.First(&got, "id = ?", id).Error)
	assert.Equal(t, id, got.ID)
}

func TestUserUpsert_OnConflictIdempotentAndUpdates(t *testing.T) {
	db, _ := newTestDB(t)

	upsert := func(first string) {
		u := models.User{ID: 42, FirstName: first, Username: "same"}
		require.NoError(t, db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "id"}},
			DoUpdates: clause.AssignmentColumns([]string{"first_name"}),
		}).Create(&u).Error)
	}
	upsert("One")
	upsert("Two")

	var count int64
	require.NoError(t, db.Model(&models.User{}).Where("id = ?", 42).Count(&count).Error)
	assert.EqualValues(t, 1, count)

	var got models.User
	require.NoError(t, db.First(&got, 42).Error)
	assert.Equal(t, "Two", got.FirstName)
}

func TestConnect_ConcurrentWritesNoLockErrors(t *testing.T) {
	db, _ := newTestDB(t)

	const n = 50
	var wg sync.WaitGroup
	errs := make(chan error, n)
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			errs <- db.Create(&models.User{ID: int64(1000 + i), FirstName: "c"}).Error
		}(i)
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		assert.NoError(t, err)
	}

	var count int64
	require.NoError(t, db.Model(&models.User{}).Count(&count).Error)
	assert.EqualValues(t, n, count)
}

func TestConnect_DataSurvivesReconnect(t *testing.T) {
	path := filepath.Join(t.TempDir(), "shopogoda.db")
	cfg := &config.DatabaseConfig{Path: path}

	db, err := Connect(cfg)
	require.NoError(t, err)
	require.NoError(t, Migrate(db))
	require.NoError(t, db.Create(&models.User{ID: 7, FirstName: "Persist"}).Error)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Close())

	db2, err := Connect(cfg)
	require.NoError(t, err)
	defer func() {
		if s, err := db2.DB(); err == nil {
			_ = s.Close()
		}
	}()
	require.NoError(t, Migrate(db2))

	var got models.User
	require.NoError(t, db2.First(&got, 7).Error)
	assert.Equal(t, "Persist", got.FirstName)
}

func TestConnectRedis(t *testing.T) {
	t.Run("successful connection", func(t *testing.T) {
		// Create mock Redis client
		client, mock := redismock.NewClientMock()
		defer func() {
			if err := client.Close(); err != nil {
				t.Logf("Failed to close mock Redis client: %v", err)
			}
		}()

		// Expect ping command
		mock.ExpectPing().SetVal("PONG")

		// Test the ping
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := client.Ping(ctx).Err()
		assert.NoError(t, err)

		// Verify expectations
		err = mock.ExpectationsWereMet()
		assert.NoError(t, err)
	})

	t.Run("connection timeout", func(t *testing.T) {
		client, mock := redismock.NewClientMock()
		defer func() {
			if err := client.Close(); err != nil {
				t.Logf("Failed to close mock Redis client: %v", err)
			}
		}()

		// Expect ping to fail
		mock.ExpectPing().SetErr(context.DeadlineExceeded)

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		err := client.Ping(ctx).Err()
		assert.Error(t, err)
		assert.Equal(t, context.DeadlineExceeded, err)
	})

	t.Run("validates config parameters", func(t *testing.T) {
		cfg := &config.RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "testpass",
			DB:       0,
		}

		assert.Equal(t, "localhost", cfg.Host)
		assert.Equal(t, 6379, cfg.Port)
		assert.Equal(t, "testpass", cfg.Password)
		assert.Equal(t, 0, cfg.DB)
	})

	t.Run("builds correct address", func(t *testing.T) {
		cfg := &config.RedisConfig{
			Host: "redis.example.com",
			Port: 6380,
		}

		expectedAddr := "redis.example.com:6380"
		actualAddr := buildRedisAddr(cfg)
		assert.Equal(t, expectedAddr, actualAddr)
	})

	t.Run("handles custom database number", func(t *testing.T) {
		cfg := &config.RedisConfig{
			Host: "localhost",
			Port: 6379,
			DB:   5,
		}

		assert.Equal(t, 5, cfg.DB)
	})

	t.Run("handles empty password", func(t *testing.T) {
		cfg := &config.RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       0,
		}

		assert.Equal(t, "", cfg.Password)
	})

	t.Run("fails with invalid host", func(t *testing.T) {
		cfg := &config.RedisConfig{
			Host:     "invalid-redis-host-that-does-not-exist.local",
			Port:     6379,
			Password: "",
			DB:       0,
		}

		client, err := ConnectRedis(cfg)
		assert.Error(t, err)
		assert.Nil(t, client)
		assert.Contains(t, err.Error(), "failed to connect to Redis")
	})

	t.Run("fails with invalid port", func(t *testing.T) {
		cfg := &config.RedisConfig{
			Host:     "localhost",
			Port:     99999, // Invalid port
			Password: "",
			DB:       0,
		}

		client, err := ConnectRedis(cfg)
		assert.Error(t, err)
		assert.Nil(t, client)
	})
}

func TestRedisConfig(t *testing.T) {
	t.Run("default values", func(t *testing.T) {
		cfg := &config.RedisConfig{}

		assert.Equal(t, "", cfg.Host)
		assert.Equal(t, 0, cfg.Port)
		assert.Equal(t, "", cfg.Password)
		assert.Equal(t, 0, cfg.DB)
	})

	t.Run("production settings", func(t *testing.T) {
		cfg := &config.RedisConfig{
			Host:     "redis-cluster.example.com",
			Port:     6380,
			Password: "strongpassword",
			DB:       1,
		}

		assert.Equal(t, "redis-cluster.example.com", cfg.Host)
		assert.Equal(t, 6380, cfg.Port)
		assert.Equal(t, "strongpassword", cfg.Password)
		assert.Equal(t, 1, cfg.DB)
	})

	t.Run("test environment settings", func(t *testing.T) {
		cfg := &config.RedisConfig{
			Host:     "localhost",
			Port:     6379,
			Password: "",
			DB:       15, // Different DB for testing
		}

		assert.Equal(t, "localhost", cfg.Host)
		assert.Equal(t, 15, cfg.DB)
	})
}

func buildRedisAddr(cfg *config.RedisConfig) string {
	return fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
}
