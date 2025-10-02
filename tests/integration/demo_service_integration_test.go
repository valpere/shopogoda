//go:build integration
// +build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/valpere/shopogoda/internal/models"
	"github.com/valpere/shopogoda/internal/services"
	"github.com/valpere/shopogoda/tests/helpers"
)

type DemoServiceTestSuite struct {
	db             *gorm.DB
	redisClient    *redis.Client
	pgContainer    testcontainers.Container
	redisContainer testcontainers.Container
	demoService    *services.DemoService
}

func setupDemoServiceTest(t *testing.T) *DemoServiceTestSuite {
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

	pong, err := redisClient.Ping(ctx).Result()
	require.NoError(t, err)
	require.Equal(t, "PONG", pong)

	// Run migrations
	require.NoError(t, models.Migrate(db))

	// Create demo service
	logger := helpers.NewSilentTestLogger()
	demoService := services.NewDemoService(db, logger)

	return &DemoServiceTestSuite{
		db:             db,
		redisClient:    redisClient,
		pgContainer:    pgContainer,
		redisContainer: redisContainer,
		demoService:    demoService,
	}
}

func (suite *DemoServiceTestSuite) teardown(t *testing.T) {
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

func TestIntegration_DemoServiceSeedData(t *testing.T) {
	suite := setupDemoServiceTest(t)
	defer suite.teardown(t)

	ctx := context.Background()

	t.Run("seed demo data creates user and related data", func(t *testing.T) {
		// Seed demo data
		err := suite.demoService.SeedDemoData(ctx)
		require.NoError(t, err)

		// Verify demo user was created
		var user models.User
		err = suite.db.Where("id = ?", services.DemoUserID).First(&user).Error
		require.NoError(t, err)
		assert.Equal(t, services.DemoUserID, user.ID)
		assert.Equal(t, "demo_user", user.Username)
		assert.Equal(t, "Demo", user.FirstName)
		assert.Equal(t, "User", user.LastName)
		assert.Equal(t, "Kyiv, Ukraine", user.LocationName)
		assert.InDelta(t, 50.4501, user.Latitude, 0.001)
		assert.InDelta(t, 30.5234, user.Longitude, 0.001)

		// Verify weather data was created
		var weatherCount int64
		err = suite.db.Model(&models.WeatherData{}).Where("user_id = ?", services.DemoUserID).Count(&weatherCount).Error
		require.NoError(t, err)
		assert.Equal(t, int64(24), weatherCount, "Should create 24 hours of weather data")

		// Verify alert configurations were created
		var alertCount int64
		err = suite.db.Model(&models.AlertConfig{}).Where("user_id = ?", services.DemoUserID).Count(&alertCount).Error
		require.NoError(t, err)
		assert.Equal(t, int64(3), alertCount, "Should create 3 alert configurations")

		// Verify subscriptions were created
		var subCount int64
		err = suite.db.Model(&models.Subscription{}).Where("user_id = ?", services.DemoUserID).Count(&subCount).Error
		require.NoError(t, err)
		assert.Equal(t, int64(3), subCount, "Should create 3 subscriptions")
	})

	t.Run("seed demo data is idempotent", func(t *testing.T) {
		// Clear first to ensure clean state
		err := suite.demoService.ClearDemoData(ctx)
		require.NoError(t, err)

		// Seed once
		err = suite.demoService.SeedDemoData(ctx)
		require.NoError(t, err)

		// Seed again - should not error
		err = suite.demoService.SeedDemoData(ctx)
		require.NoError(t, err)

		// Verify still only one user
		var userCount int64
		err = suite.db.Model(&models.User{}).Where("id = ?", services.DemoUserID).Count(&userCount).Error
		require.NoError(t, err)
		assert.Equal(t, int64(1), userCount, "Should have exactly one demo user")
	})
}

func TestIntegration_DemoServiceClearData(t *testing.T) {
	suite := setupDemoServiceTest(t)
	defer suite.teardown(t)

	ctx := context.Background()

	t.Run("clear demo data removes all demo user data", func(t *testing.T) {
		// First seed data
		err := suite.demoService.SeedDemoData(ctx)
		require.NoError(t, err)

		// Verify data exists
		var userCount int64
		err = suite.db.Model(&models.User{}).Where("id = ?", services.DemoUserID).Count(&userCount).Error
		require.NoError(t, err)
		assert.Equal(t, int64(1), userCount)

		// Clear demo data
		err = suite.demoService.ClearDemoData(ctx)
		require.NoError(t, err)

		// Verify user is deleted
		err = suite.db.Model(&models.User{}).Where("id = ?", services.DemoUserID).Count(&userCount).Error
		require.NoError(t, err)
		assert.Equal(t, int64(0), userCount, "Demo user should be deleted")

		// Verify weather data is deleted
		var weatherCount int64
		err = suite.db.Model(&models.WeatherData{}).Where("user_id = ?", services.DemoUserID).Count(&weatherCount).Error
		require.NoError(t, err)
		assert.Equal(t, int64(0), weatherCount, "Weather data should be deleted")

		// Verify alerts are deleted
		var alertCount int64
		err = suite.db.Model(&models.AlertConfig{}).Where("user_id = ?", services.DemoUserID).Count(&alertCount).Error
		require.NoError(t, err)
		assert.Equal(t, int64(0), alertCount, "Alerts should be deleted")

		// Verify subscriptions are deleted
		var subCount int64
		err = suite.db.Model(&models.Subscription{}).Where("user_id = ?", services.DemoUserID).Count(&subCount).Error
		require.NoError(t, err)
		assert.Equal(t, int64(0), subCount, "Subscriptions should be deleted")
	})

	t.Run("clear demo data is safe when no data exists", func(t *testing.T) {
		// Clear when already empty - should not error
		err := suite.demoService.ClearDemoData(ctx)
		require.NoError(t, err)
	})
}

func TestIntegration_DemoServiceResetData(t *testing.T) {
	suite := setupDemoServiceTest(t)
	defer suite.teardown(t)

	ctx := context.Background()

	t.Run("reset demo data clears and re-seeds", func(t *testing.T) {
		// First seed data
		err := suite.demoService.SeedDemoData(ctx)
		require.NoError(t, err)

		// Get initial timestamp
		var initialUser models.User
		err = suite.db.Where("id = ?", services.DemoUserID).First(&initialUser).Error
		require.NoError(t, err)
		initialCreatedAt := initialUser.CreatedAt

		// Wait a moment to ensure timestamp difference
		time.Sleep(100 * time.Millisecond)

		// Reset demo data
		err = suite.demoService.ResetDemoData(ctx)
		require.NoError(t, err)

		// Verify user exists with new timestamp
		var resetUser models.User
		err = suite.db.Where("id = ?", services.DemoUserID).First(&resetUser).Error
		require.NoError(t, err)
		assert.True(t, resetUser.CreatedAt.After(initialCreatedAt), "Reset should create new user with later timestamp")

		// Verify all data is recreated
		var weatherCount, alertCount, subCount int64

		err = suite.db.Model(&models.WeatherData{}).Where("user_id = ?", services.DemoUserID).Count(&weatherCount).Error
		require.NoError(t, err)
		assert.Equal(t, int64(24), weatherCount)

		err = suite.db.Model(&models.AlertConfig{}).Where("user_id = ?", services.DemoUserID).Count(&alertCount).Error
		require.NoError(t, err)
		assert.Equal(t, int64(3), alertCount)

		err = suite.db.Model(&models.Subscription{}).Where("user_id = ?", services.DemoUserID).Count(&subCount).Error
		require.NoError(t, err)
		assert.Equal(t, int64(3), subCount)
	})
}

func TestIntegration_DemoServiceWeatherData(t *testing.T) {
	suite := setupDemoServiceTest(t)
	defer suite.teardown(t)

	ctx := context.Background()

	t.Run("weather data has realistic values", func(t *testing.T) {
		// Seed demo data
		err := suite.demoService.SeedDemoData(ctx)
		require.NoError(t, err)

		// Get all weather data
		var weatherData []models.WeatherData
		err = suite.db.Where("user_id = ?", services.DemoUserID).
			Order("timestamp ASC").
			Find(&weatherData).Error
		require.NoError(t, err)
		require.Len(t, weatherData, 24)

		// Verify temperature variations (should vary throughout the day)
		temperatures := make([]float64, len(weatherData))
		for i, wd := range weatherData {
			temperatures[i] = wd.Temperature

			// All temperatures should be realistic
			assert.GreaterOrEqual(t, wd.Temperature, -20.0, "Temperature should be >= -20°C")
			assert.LessOrEqual(t, wd.Temperature, 40.0, "Temperature should be <= 40°C")

			// Humidity should be 0-100%
			assert.GreaterOrEqual(t, wd.Humidity, 0, "Humidity should be >= 0%")
			assert.LessOrEqual(t, wd.Humidity, 100, "Humidity should be <= 100%")

			// Pressure should be realistic (if set - may be 0)
			if wd.Pressure > 0 {
				assert.GreaterOrEqual(t, wd.Pressure, 950.0, "Pressure should be >= 950 hPa")
				assert.LessOrEqual(t, wd.Pressure, 1050.0, "Pressure should be <= 1050 hPa")
			}
		}

		// Temperature should vary (not all the same)
		allSame := true
		firstTemp := temperatures[0]
		for _, temp := range temperatures {
			if temp != firstTemp {
				allSame = false
				break
			}
		}
		assert.False(t, allSame, "Temperatures should vary throughout the day")
	})
}

func TestIntegration_DemoServiceAlertConfigs(t *testing.T) {
	suite := setupDemoServiceTest(t)
	defer suite.teardown(t)

	ctx := context.Background()

	t.Run("alert configs are properly configured", func(t *testing.T) {
		// Seed demo data
		err := suite.demoService.SeedDemoData(ctx)
		require.NoError(t, err)

		// Get all alert configs
		var alerts []models.AlertConfig
		err = suite.db.Where("user_id = ?", services.DemoUserID).Find(&alerts).Error
		require.NoError(t, err)
		require.Len(t, alerts, 3)

		// Verify alert types are different
		alertTypes := make(map[models.AlertType]bool)
		for _, alert := range alerts {
			alertTypes[alert.AlertType] = true
			assert.True(t, alert.IsActive, "Demo alerts should be active")
			assert.Greater(t, alert.Threshold, 0.0, "Alert threshold should be positive")
		}
		assert.Len(t, alertTypes, 3, "Should have 3 different alert types")
	})
}

func TestIntegration_DemoServiceSubscriptions(t *testing.T) {
	suite := setupDemoServiceTest(t)
	defer suite.teardown(t)

	ctx := context.Background()

	t.Run("subscriptions are properly configured", func(t *testing.T) {
		// Seed demo data
		err := suite.demoService.SeedDemoData(ctx)
		require.NoError(t, err)

		// Get all subscriptions
		var subscriptions []models.Subscription
		err = suite.db.Where("user_id = ?", services.DemoUserID).Find(&subscriptions).Error
		require.NoError(t, err)
		require.Len(t, subscriptions, 3)

		// Verify subscription types are different
		subTypes := make(map[models.SubscriptionType]bool)
		for _, sub := range subscriptions {
			subTypes[sub.SubscriptionType] = true
			assert.True(t, sub.IsActive, "Demo subscriptions should be active")
		}
		assert.Len(t, subTypes, 3, "Should have 3 different subscription types")
	})
}
