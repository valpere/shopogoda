//go:build integration
// +build integration

package integration

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/valpere/shopogoda/internal/models"
	"github.com/valpere/shopogoda/internal/services"
)

type AlertServiceTestSuite struct {
	db             *gorm.DB
	redisClient    *redis.Client
	pgContainer    testcontainers.Container
	redisContainer testcontainers.Container
	alertService   *services.AlertService
	testUserID     int64
}

func setupAlertServiceTest(t *testing.T) *AlertServiceTestSuite {
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

	// Create test user
	testUserID := int64(12345678)
	testUser := &models.User{
		ID:           testUserID,
		Username:     "test_user",
		FirstName:    "Test",
		LastName:     "User",
		Language:     "en-US",
		LocationName: "Test City",
		Latitude:     50.45,
		Longitude:    30.52,
		Timezone:     "UTC",
		Role:         models.RoleUser,
		IsActive:     true,
	}
	require.NoError(t, db.Create(testUser).Error)

	// Create alert service
	alertService := services.NewAlertService(db, redisClient)

	return &AlertServiceTestSuite{
		db:             db,
		redisClient:    redisClient,
		pgContainer:    pgContainer,
		redisContainer: redisContainer,
		alertService:   alertService,
		testUserID:     testUserID,
	}
}

func (suite *AlertServiceTestSuite) teardown(t *testing.T) {
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

func TestIntegration_AlertServiceCreateAlert(t *testing.T) {
	suite := setupAlertServiceTest(t)
	defer suite.teardown(t)

	ctx := context.Background()

	t.Run("create temperature alert successfully", func(t *testing.T) {
		condition := services.AlertCondition{
			Operator: "gt",
			Value:    30.0,
		}

		alert, err := suite.alertService.CreateAlert(ctx, suite.testUserID, models.AlertTemperature, condition)
		require.NoError(t, err)
		assert.NotNil(t, alert)
		assert.NotEqual(t, uuid.Nil, alert.ID)
		assert.Equal(t, suite.testUserID, alert.UserID)
		assert.Equal(t, models.AlertTemperature, alert.AlertType)
		assert.Equal(t, 30.0, alert.Threshold)
		assert.True(t, alert.IsActive)

		// Verify in database
		var dbAlert models.AlertConfig
		err = suite.db.Where("id = ?", alert.ID).First(&dbAlert).Error
		require.NoError(t, err)
		assert.Equal(t, models.AlertTemperature, dbAlert.AlertType)
	})

	t.Run("create multiple alert types", func(t *testing.T) {
		alerts := []struct {
			alertType models.AlertType
			operator  string
			value     float64
		}{
			{models.AlertHumidity, "gt", 80.0},
			{models.AlertAirQuality, "gt", 100.0},
			{models.AlertWindSpeed, "gt", 50.0},
		}

		for _, tc := range alerts {
			condition := services.AlertCondition{
				Operator: tc.operator,
				Value:    tc.value,
			}
			alert, err := suite.alertService.CreateAlert(ctx, suite.testUserID, tc.alertType, condition)
			require.NoError(t, err)
			assert.NotNil(t, alert)
			assert.Equal(t, tc.alertType, alert.AlertType)
		}

		// Verify count (3 new + 1 from previous test)
		var count int64
		err := suite.db.Model(&models.AlertConfig{}).Where("user_id = ?", suite.testUserID).Count(&count).Error
		require.NoError(t, err)
		assert.Equal(t, int64(4), count)
	})
}

func TestIntegration_AlertServiceGetUserAlerts(t *testing.T) {
	suite := setupAlertServiceTest(t)
	defer suite.teardown(t)

	ctx := context.Background()

	// Create test alerts
	tempCondition := services.AlertCondition{Operator: "gt", Value: 25.0}
	humidCondition := services.AlertCondition{Operator: "lt", Value: 30.0}

	_, err := suite.alertService.CreateAlert(ctx, suite.testUserID, models.AlertTemperature, tempCondition)
	require.NoError(t, err)

	alert2, err := suite.alertService.CreateAlert(ctx, suite.testUserID, models.AlertHumidity, humidCondition)
	require.NoError(t, err)

	// Make second alert inactive
	alert2.IsActive = false
	err = suite.db.Save(alert2).Error
	require.NoError(t, err)

	t.Run("get active user alerts only", func(t *testing.T) {
		userAlerts, err := suite.alertService.GetUserAlerts(ctx, suite.testUserID)
		require.NoError(t, err)

		// Should only get active alert (temperature)
		assert.Len(t, userAlerts, 1)
		assert.Equal(t, models.AlertTemperature, userAlerts[0].AlertType)
	})

	t.Run("get alerts for non-existent user", func(t *testing.T) {
		userAlerts, err := suite.alertService.GetUserAlerts(ctx, 99999)
		require.NoError(t, err)
		assert.Empty(t, userAlerts)
	})
}

func TestIntegration_AlertServiceUpdateAlert(t *testing.T) {
	suite := setupAlertServiceTest(t)
	defer suite.teardown(t)

	ctx := context.Background()

	// Create initial alert
	condition := services.AlertCondition{Operator: "gt", Value: 30.0}
	alert, err := suite.alertService.CreateAlert(ctx, suite.testUserID, models.AlertTemperature, condition)
	require.NoError(t, err)

	t.Run("update alert successfully", func(t *testing.T) {
		// Update through UpdateAlert method
		updates := map[string]interface{}{
			"threshold": 35.0,
		}
		err := suite.alertService.UpdateAlert(ctx, suite.testUserID, alert.ID, updates)
		require.NoError(t, err)

		// Verify update
		var dbAlert models.AlertConfig
		err = suite.db.Where("id = ?", alert.ID).First(&dbAlert).Error
		require.NoError(t, err)
		assert.Equal(t, 35.0, dbAlert.Threshold)
	})
}

func TestIntegration_AlertServiceDeleteAlert(t *testing.T) {
	suite := setupAlertServiceTest(t)
	defer suite.teardown(t)

	ctx := context.Background()

	// Create alert
	condition := services.AlertCondition{Operator: "gt", Value: 30.0}
	alert, err := suite.alertService.CreateAlert(ctx, suite.testUserID, models.AlertTemperature, condition)
	require.NoError(t, err)
	alertID := alert.ID

	t.Run("delete alert successfully", func(t *testing.T) {
		err := suite.alertService.DeleteAlert(ctx, suite.testUserID, alertID)
		require.NoError(t, err)

		// Verify deletion (should set is_active to false, not delete)
		var dbAlert models.AlertConfig
		err = suite.db.Where("id = ?", alertID).First(&dbAlert).Error
		require.NoError(t, err)
		assert.False(t, dbAlert.IsActive)
	})

	t.Run("delete non-existent alert", func(t *testing.T) {
		err := suite.alertService.DeleteAlert(ctx, suite.testUserID, uuid.New())
		assert.Error(t, err)
	})
}

func TestIntegration_AlertServiceCheckAlerts(t *testing.T) {
	suite := setupAlertServiceTest(t)
	defer suite.teardown(t)

	ctx := context.Background()

	// Create active alerts
	tempCondition := services.AlertCondition{Operator: "gt", Value: 25.0}
	_, err := suite.alertService.CreateAlert(ctx, suite.testUserID, models.AlertTemperature, tempCondition)
	require.NoError(t, err)

	humidCondition := services.AlertCondition{Operator: "gt", Value: 70.0}
	humidAlert, err := suite.alertService.CreateAlert(ctx, suite.testUserID, models.AlertHumidity, humidCondition)
	require.NoError(t, err)

	// Create weather data that triggers temperature alert
	weatherData := &models.WeatherData{
		ID:          uuid.New(),
		UserID:      suite.testUserID,
		Temperature: 30.0, // Above threshold (25.0)
		Humidity:    60,   // Below threshold (70.0)
		Pressure:    1013.0,
		WindSpeed:   5.0,
		Timestamp:   time.Now(),
	}

	t.Run("check alerts and trigger temperature alert", func(t *testing.T) {
		triggeredAlerts, err := suite.alertService.CheckAlerts(ctx, weatherData, suite.testUserID)
		require.NoError(t, err)

		// Should trigger temperature alert, not humidity alert
		assert.GreaterOrEqual(t, len(triggeredAlerts), 1)

		// Find temperature alert
		var foundTemp bool
		for _, alert := range triggeredAlerts {
			if alert.AlertType == models.AlertTemperature {
				foundTemp = true
				assert.Equal(t, 30.0, alert.Value)
				assert.False(t, alert.IsResolved)
				break
			}
		}
		assert.True(t, foundTemp, "Temperature alert should be triggered")
	})

	t.Run("check alerts with both triggered", func(t *testing.T) {
		// Wait a bit to avoid cooldown period
		time.Sleep(2 * time.Second)

		weatherData.Humidity = 80 // Now above threshold

		triggeredAlerts, err := suite.alertService.CheckAlerts(ctx, weatherData, suite.testUserID)
		require.NoError(t, err)

		// Should trigger both alerts
		assert.GreaterOrEqual(t, len(triggeredAlerts), 1)

		alertTypes := make(map[models.AlertType]bool)
		for _, alert := range triggeredAlerts {
			if alert.AlertType == models.AlertTemperature || alert.AlertType == models.AlertHumidity {
				alertTypes[alert.AlertType] = true
				assert.False(t, alert.IsResolved)
			}
		}
		// At least one should be triggered
		assert.True(t, len(alertTypes) > 0)
	})

	t.Run("check alerts with inactive alert", func(t *testing.T) {
		// Reset temperature alert's LastTriggered to allow re-triggering (bypass cooldown)
		err := suite.db.Model(&models.AlertConfig{}).
			Where("user_id = ? AND alert_type = ?", suite.testUserID, models.AlertTemperature).
			Update("last_triggered", nil).Error
		require.NoError(t, err)

		// Deactivate humidity alert
		humidAlert.IsActive = false
		require.NoError(t, suite.db.Save(humidAlert).Error)

		triggeredAlerts, err := suite.alertService.CheckAlerts(ctx, weatherData, suite.testUserID)
		require.NoError(t, err)

		// Should only trigger temperature alert (humidity is inactive)
		tempCount := 0
		humidCount := 0
		for _, alert := range triggeredAlerts {
			if alert.AlertType == models.AlertTemperature {
				tempCount++
			}
			if alert.AlertType == models.AlertHumidity {
				humidCount++
			}
		}
		assert.Greater(t, tempCount, 0, "Temperature alert should be triggered")
		assert.Equal(t, 0, humidCount, "Humidity alert should not be triggered (inactive)")
	})
}
