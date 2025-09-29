//go:build integration
// +build integration

package integration

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/valpere/shopogoda/internal/config"
	"github.com/valpere/shopogoda/internal/models"
	"github.com/valpere/shopogoda/internal/services"
	"github.com/valpere/shopogoda/tests/helpers"
)

type IntegrationTestSuite struct {
	db          *gorm.DB
	redisClient *redis.Client
	pgContainer testcontainers.Container
	redisContainer testcontainers.Container
	services    *services.Services
}

func setupIntegrationTest(t *testing.T) *IntegrationTestSuite {
	ctx := context.Background()

	// Start PostgreSQL container
	pgReq := testcontainers.ContainerRequest{
		Image:        "postgres:15-alpine",
		ExposedPorts: []string{"5432/tcp"},
		Env: map[string]string{
			"POSTGRES_DB":       "testdb",
			"POSTGRES_USER":     "testuser",
			"POSTGRES_PASSWORD": "testpass",
		},
		WaitingFor: wait.ForListeningPort("5432/tcp"),
	}

	pgContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: pgReq,
		Started:          true,
	})
	require.NoError(t, err)

	// Start Redis container
	redisReq := testcontainers.ContainerRequest{
		Image:        "redis:7-alpine",
		ExposedPorts: []string{"6379/tcp"},
		WaitingFor:   wait.ForListeningPort("6379/tcp"),
	}

	redisContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: redisReq,
		Started:          true,
	})
	require.NoError(t, err)

	// Get container ports
	pgHost, err := pgContainer.Host(ctx)
	require.NoError(t, err)

	pgPort, err := pgContainer.MappedPort(ctx, "5432")
	require.NoError(t, err)

	redisHost, err := redisContainer.Host(ctx)
	require.NoError(t, err)

	redisPort, err := redisContainer.MappedPort(ctx, "6379")
	require.NoError(t, err)

	// Connect to PostgreSQL
	dsn := "host=" + pgHost + " user=testuser password=testpass dbname=testdb port=" + pgPort.Port() + " sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	// Connect to Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr: redisHost + ":" + redisPort.Port(),
	})

	// Test connections
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Ping())

	_, err = redisClient.Ping(ctx).Result()
	require.NoError(t, err)

	// Run migrations
	require.NoError(t, models.Migrate(db))

	// Create services
	logger := helpers.NewSilentTestLogger()
	testConfig := helpers.GetTestConfig()
	testConfig.Database.Host = pgHost
	testConfig.Database.Port = int(pgPort.Int())
	testConfig.Redis.Addr = redisHost + ":" + redisPort.Port()

	services := services.New(db, redisClient, testConfig, logger)

	return &IntegrationTestSuite{
		db:             db,
		redisClient:    redisClient,
		pgContainer:    pgContainer,
		redisContainer: redisContainer,
		services:       services,
	}
}

func (suite *IntegrationTestSuite) teardown(t *testing.T) {
	ctx := context.Background()

	if suite.redisClient != nil {
		suite.redisClient.Close()
	}

	if suite.db != nil {
		sqlDB, _ := suite.db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}

	if suite.pgContainer != nil {
		require.NoError(t, suite.pgContainer.Terminate(ctx))
	}

	if suite.redisContainer != nil {
		require.NoError(t, suite.redisContainer.Terminate(ctx))
	}
}

func TestIntegration_UserWorkflow(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.teardown(t)

	ctx := context.Background()

	t.Run("complete user lifecycle", func(t *testing.T) {
		userID := int64(123456)

		// Create user
		user := &models.User{
			ID:           userID,
			FirstName:    "Integration",
			LastName:     "Test",
			Username:     "integrationtest",
			LanguageCode: "en",
			Role:         models.UserRole,
		}

		err := suite.services.User.CreateUser(ctx, user)
		require.NoError(t, err)

		// Retrieve user
		retrievedUser, err := suite.services.User.GetUser(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, user.ID, retrievedUser.ID)
		assert.Equal(t, user.FirstName, retrievedUser.FirstName)

		// Set user location
		err = suite.services.User.SetUserLocation(ctx, userID, "London", "UK", "London", 51.5074, -0.1278)
		require.NoError(t, err)

		// Get user location
		locationName, lat, lon, err := suite.services.User.GetUserLocation(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, "London", locationName)
		assert.Equal(t, 51.5074, lat)
		assert.Equal(t, -0.1278, lon)

		// Set user language
		err = suite.services.User.SetUserLanguage(ctx, userID, "de")
		require.NoError(t, err)

		// Get user language
		language := suite.services.User.GetUserLanguage(ctx, userID)
		assert.Equal(t, "de", language)
	})
}

func TestIntegration_WeatherDataWorkflow(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.teardown(t)

	ctx := context.Background()

	t.Run("weather data storage and retrieval", func(t *testing.T) {
		userID := int64(123456)

		// Create user first
		user := helpers.MockUser(userID)
		err := suite.services.User.CreateUser(ctx, user)
		require.NoError(t, err)

		// Create weather data
		weatherData := &models.WeatherData{
			UserID:        userID,
			LocationName:  "London",
			Temperature:   20.5,
			FeelsLike:     22.0,
			Humidity:      65,
			Pressure:      1013,
			Visibility:    10,
			UVIndex:       3,
			WindSpeed:     5.2,
			WindDirection: 180,
			Description:   "Clear sky",
			AQI:           2,
			CO:            0.3,
			NO:            0.1,
			NO2:           15.2,
			O3:            45.3,
			SO2:           2.1,
			PM25:          8.5,
			PM10:          12.3,
			NH3:           1.2,
			Timestamp:     time.Now().UTC(),
		}

		// Save weather data
		err = suite.services.Weather.SaveWeatherData(ctx, weatherData)
		require.NoError(t, err)

		// Verify weather data was saved
		var savedWeatherData models.WeatherData
		err = suite.db.Where("user_id = ?", userID).First(&savedWeatherData).Error
		require.NoError(t, err)
		assert.Equal(t, weatherData.UserID, savedWeatherData.UserID)
		assert.Equal(t, weatherData.Temperature, savedWeatherData.Temperature)
		assert.Equal(t, weatherData.LocationName, savedWeatherData.LocationName)
	})
}

func TestIntegration_AlertWorkflow(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.teardown(t)

	ctx := context.Background()

	t.Run("complete alert lifecycle", func(t *testing.T) {
		userID := int64(123456)

		// Create user first
		user := helpers.MockUser(userID)
		err := suite.services.User.CreateUser(ctx, user)
		require.NoError(t, err)

		// Create alert
		alert := &models.EnvironmentalAlert{
			UserID:      userID,
			Type:        models.AlertTemperature,
			Threshold:   25.0,
			Condition:   "greater_than",
			Severity:    models.SeverityMedium,
			Title:       "High Temperature Alert",
			Description: "Temperature exceeded threshold",
			IsActive:    true,
		}

		err = suite.services.Alert.CreateAlert(ctx, alert)
		require.NoError(t, err)

		// Retrieve alerts for user
		userAlerts, err := suite.services.Alert.GetUserAlerts(ctx, userID)
		require.NoError(t, err)
		assert.Len(t, userAlerts, 1)
		assert.Equal(t, alert.UserID, userAlerts[0].UserID)
		assert.Equal(t, alert.Type, userAlerts[0].Type)

		// Create weather data that should trigger the alert
		weatherData := &models.WeatherData{
			UserID:      userID,
			Temperature: 26.5, // Above threshold of 25.0
			Timestamp:   time.Now().UTC(),
		}

		// Check if alerts are triggered
		triggeredAlerts, err := suite.services.Alert.CheckAlertsForUser(ctx, userID, weatherData)
		require.NoError(t, err)
		assert.Len(t, triggeredAlerts, 1)
		assert.Equal(t, models.AlertTemperature, triggeredAlerts[0].Type)
	})
}

func TestIntegration_CacheOperations(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.teardown(t)

	ctx := context.Background()

	t.Run("redis cache operations", func(t *testing.T) {
		// Test basic cache operations
		key := "test:key"
		value := "test_value"

		// Set cache
		err := suite.redisClient.Set(ctx, key, value, time.Minute).Err()
		require.NoError(t, err)

		// Get cache
		cachedValue, err := suite.redisClient.Get(ctx, key).Result()
		require.NoError(t, err)
		assert.Equal(t, value, cachedValue)

		// Test expiration
		err = suite.redisClient.Set(ctx, "exp:key", "exp_value", time.Millisecond*100).Err()
		require.NoError(t, err)

		// Wait for expiration
		time.Sleep(time.Millisecond * 150)

		// Should be expired
		_, err = suite.redisClient.Get(ctx, "exp:key").Result()
		assert.Error(t, err)
		assert.Equal(t, redis.Nil, err)
	})
}

func TestIntegration_DatabaseTransactions(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.teardown(t)

	ctx := context.Background()

	t.Run("transaction rollback on error", func(t *testing.T) {
		userID := int64(999999)

		// Start transaction
		tx := suite.db.Begin()
		defer tx.Rollback()

		// Create user in transaction
		user := helpers.MockUser(userID)
		err := tx.Create(user).Error
		require.NoError(t, err)

		// Verify user exists in transaction
		var count int64
		tx.Model(&models.User{}).Where("id = ?", userID).Count(&count)
		assert.Equal(t, int64(1), count)

		// Rollback transaction
		tx.Rollback()

		// Verify user doesn't exist after rollback
		suite.db.Model(&models.User{}).Where("id = ?", userID).Count(&count)
		assert.Equal(t, int64(0), count)
	})

	t.Run("transaction commit", func(t *testing.T) {
		userID := int64(888888)

		// Start transaction
		tx := suite.db.Begin()

		// Create user in transaction
		user := helpers.MockUser(userID)
		err := tx.Create(user).Error
		require.NoError(t, err)

		// Commit transaction
		err = tx.Commit().Error
		require.NoError(t, err)

		// Verify user exists after commit
		var count int64
		suite.db.Model(&models.User{}).Where("id = ?", userID).Count(&count)
		assert.Equal(t, int64(1), count)
	})
}

func TestIntegration_ConcurrentOperations(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.teardown(t)

	ctx := context.Background()

	t.Run("concurrent user operations", func(t *testing.T) {
		const numGoroutines = 10
		const numOperationsPerGoroutine = 5

		errChan := make(chan error, numGoroutines*numOperationsPerGoroutine)
		done := make(chan bool, numGoroutines)

		// Launch concurrent goroutines
		for i := 0; i < numGoroutines; i++ {
			go func(routineID int) {
				defer func() { done <- true }()

				for j := 0; j < numOperationsPerGoroutine; j++ {
					userID := int64(routineID*1000 + j + 1)
					user := helpers.MockUser(userID)

					// Create user
					if err := suite.services.User.CreateUser(ctx, user); err != nil {
						errChan <- err
						return
					}

					// Retrieve user
					if _, err := suite.services.User.GetUser(ctx, userID); err != nil {
						errChan <- err
						return
					}
				}
			}(i)
		}

		// Wait for all goroutines to complete
		for i := 0; i < numGoroutines; i++ {
			<-done
		}

		close(errChan)

		// Check for errors
		for err := range errChan {
			require.NoError(t, err)
		}

		// Verify all users were created
		var totalUsers int64
		suite.db.Model(&models.User{}).Count(&totalUsers)
		assert.Equal(t, int64(numGoroutines*numOperationsPerGoroutine), totalUsers)
	})
}

func TestIntegration_DatabaseConnection(t *testing.T) {
	suite := setupIntegrationTest(t)
	defer suite.teardown(t)

	t.Run("database connection health", func(t *testing.T) {
		sqlDB, err := suite.db.DB()
		require.NoError(t, err)

		// Test basic connectivity
		err = sqlDB.Ping()
		require.NoError(t, err)

		// Test connection stats
		stats := sqlDB.Stats()
		assert.GreaterOrEqual(t, stats.OpenConnections, 0)
		assert.GreaterOrEqual(t, stats.MaxOpenConnections, 0)

		// Test a simple query
		var result int
		err = sqlDB.QueryRow("SELECT 1").Scan(&result)
		require.NoError(t, err)
		assert.Equal(t, 1, result)
	})
}