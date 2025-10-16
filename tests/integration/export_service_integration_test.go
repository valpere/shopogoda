//go:build integration
// +build integration

package integration

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/valpere/shopogoda/internal/models"
	"github.com/valpere/shopogoda/internal/services"
)

type ExportServiceTestSuite struct {
	db            *gorm.DB
	pgContainer   testcontainers.Container
	exportService *services.ExportService
	testUserID    int64
	testUser      *models.User
}

func setupExportServiceTest(t *testing.T) *ExportServiceTestSuite {
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

	// Get container port
	pgHost, err := pgContainer.Host(ctx)
	require.NoError(t, err)

	pgPort, err := pgContainer.MappedPort(ctx, "5432")
	require.NoError(t, err)

	// Connect to PostgreSQL
	dsn := "host=" + pgHost + " user=testuser password=testpass dbname=testdb port=" + pgPort.Port() + " sslmode=disable"
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	require.NoError(t, err)

	// Test connection
	sqlDB, err := db.DB()
	require.NoError(t, err)
	require.NoError(t, sqlDB.Ping())

	// Run migrations
	require.NoError(t, models.Migrate(db))

	// Create test user
	testUserID := int64(98765432)
	testUser := &models.User{
		ID:           testUserID,
		Username:     "export_test_user",
		FirstName:    "Export",
		LastName:     "Tester",
		Language:     "en-US",
		LocationName: "Berlin, Germany",
		Latitude:     52.5200,
		Longitude:    13.4050,
		Country:      "Germany",
		City:         "Berlin",
		Timezone:     "Europe/Berlin",
		Units:        "metric",
		Role:         models.RoleUser,
		IsActive:     true,
	}
	require.NoError(t, db.Create(testUser).Error)

	// Create logger and localization service
	logger := zerolog.Nop()
	locService := services.NewLocalizationService(&logger)

	// Create export service
	exportService := services.NewExportService(db, &logger, locService)

	return &ExportServiceTestSuite{
		db:            db,
		pgContainer:   pgContainer,
		exportService: exportService,
		testUserID:    testUserID,
		testUser:      testUser,
	}
}

func (suite *ExportServiceTestSuite) teardown(t *testing.T) {
	ctx := context.Background()

	if suite.db != nil {
		sqlDB, _ := suite.db.DB()
		if sqlDB != nil {
			sqlDB.Close()
		}
	}

	if suite.pgContainer != nil {
		require.NoError(t, suite.pgContainer.Terminate(ctx))
	}
}

func TestIntegration_ExportServiceExportWeatherDataJSON(t *testing.T) {
	suite := setupExportServiceTest(t)
	defer suite.teardown(t)

	ctx := context.Background()

	// Create test weather data
	weatherData := &models.WeatherData{
		ID:          uuid.New(),
		UserID:      suite.testUserID,
		Temperature: 22.5,
		Humidity:    65,
		Pressure:    1013.25,
		WindSpeed:   12.5,
		WindDegree:  180,
		Visibility:  10.0,
		UVIndex:     5.0,
		Description: "Partly Cloudy",
		Icon:        "⛅",
		AQI:         45,
		PM25:        12.5,
		PM10:        20.0,
		Timestamp:   time.Now().UTC(),
	}
	require.NoError(t, suite.db.Create(weatherData).Error)

	t.Run("export weather data as JSON", func(t *testing.T) {
		buffer, filename, err := suite.exportService.ExportUserData(
			ctx,
			suite.testUserID,
			services.ExportTypeWeatherData,
			services.ExportFormatJSON,
			"en-US",
		)
		require.NoError(t, err)
		assert.NotNil(t, buffer)
		assert.Contains(t, filename, "shopogoda_weather_export_test_user")
		assert.Contains(t, filename, ".json")

		// Parse JSON
		var exportData services.ExportData
		err = json.Unmarshal(buffer.Bytes(), &exportData)
		require.NoError(t, err)

		// Verify structure
		assert.NotNil(t, exportData.User)
		assert.Equal(t, suite.testUserID, exportData.User.ID)
		assert.Len(t, exportData.WeatherData, 1)
		assert.Equal(t, 22.5, exportData.WeatherData[0].Temperature)
		assert.Equal(t, 65, exportData.WeatherData[0].Humidity)
		assert.Equal(t, services.ExportFormatJSON, exportData.Format)
		assert.Equal(t, services.ExportTypeWeatherData, exportData.Type)
	})
}

func TestIntegration_ExportServiceExportSubscriptionsCSV(t *testing.T) {
	suite := setupExportServiceTest(t)
	defer suite.teardown(t)

	ctx := context.Background()

	// Create test subscriptions
	subscription := &models.Subscription{
		ID:               uuid.New(),
		UserID:           suite.testUserID,
		SubscriptionType: models.SubscriptionDaily,
		Frequency:        models.FrequencyDaily,
		TimeOfDay:        "08:00",
		IsActive:         true,
	}
	require.NoError(t, suite.db.Create(subscription).Error)

	t.Run("export subscriptions as CSV", func(t *testing.T) {
		buffer, filename, err := suite.exportService.ExportUserData(
			ctx,
			suite.testUserID,
			services.ExportTypeSubscriptions,
			services.ExportFormatCSV,
			"en-US",
		)
		require.NoError(t, err)
		assert.NotNil(t, buffer)
		assert.Contains(t, filename, "shopogoda_subscriptions_export_test_user")
		assert.Contains(t, filename, ".csv")

		// Check CSV content (multi-section CSV with varying field counts)
		csvContent := buffer.String()
		assert.Contains(t, csvContent, "export_test_user")
		assert.Contains(t, csvContent, "Daily")
		assert.Contains(t, csvContent, "08:00")
		assert.Contains(t, csvContent, "true") // IsActive
	})
}

func TestIntegration_ExportServiceExportAlertsTXT(t *testing.T) {
	suite := setupExportServiceTest(t)
	defer suite.teardown(t)

	ctx := context.Background()

	// Create alert config
	alertConfig := &models.AlertConfig{
		ID:        uuid.New(),
		UserID:    suite.testUserID,
		AlertType: models.AlertTemperature,
		Condition: "greater_than",
		Threshold: 30.0,
		IsActive:  true,
	}
	require.NoError(t, suite.db.Create(alertConfig).Error)

	// Create triggered alert
	triggeredAlert := &models.EnvironmentalAlert{
		ID:          uuid.New(),
		UserID:      suite.testUserID,
		AlertType:   models.AlertTemperature,
		Severity:    models.SeverityHigh,
		Title:       "High Temperature Alert",
		Description: "Temperature exceeded threshold",
		Value:       32.5,
		Threshold:   30.0,
		IsResolved:  false,
		CreatedAt:   time.Now().UTC(),
	}
	require.NoError(t, suite.db.Create(triggeredAlert).Error)

	t.Run("export alerts as TXT", func(t *testing.T) {
		buffer, filename, err := suite.exportService.ExportUserData(
			ctx,
			suite.testUserID,
			services.ExportTypeAlerts,
			services.ExportFormatTXT,
			"en-US",
		)
		require.NoError(t, err)
		assert.NotNil(t, buffer)
		assert.Contains(t, filename, "shopogoda_alerts_export_test_user")
		assert.Contains(t, filename, ".txt")

		// Check TXT content
		txtContent := buffer.String()
		assert.Contains(t, txtContent, "Export Tester")    // FirstName from user
		assert.Contains(t, txtContent, "export_test_user") // Username
		assert.Contains(t, txtContent, "Temperature")
		assert.Contains(t, txtContent, "greater_than")
		assert.Contains(t, txtContent, "30.0")
		assert.Contains(t, txtContent, "High Temperature Alert")
		assert.Contains(t, txtContent, "Temperature exceeded threshold")
		assert.Contains(t, txtContent, "End of Export")
	})
}

func TestIntegration_ExportServiceExportAllData(t *testing.T) {
	suite := setupExportServiceTest(t)
	defer suite.teardown(t)

	ctx := context.Background()

	// Create weather data
	weatherData := &models.WeatherData{
		ID:          uuid.New(),
		UserID:      suite.testUserID,
		Temperature: 18.0,
		Humidity:    70,
		Pressure:    1015.0,
		WindSpeed:   8.0,
		Timestamp:   time.Now().UTC(),
	}
	require.NoError(t, suite.db.Create(weatherData).Error)

	// Create subscription
	subscription := &models.Subscription{
		ID:               uuid.New(),
		UserID:           suite.testUserID,
		SubscriptionType: models.SubscriptionWeekly,
		Frequency:        models.FrequencyWeekly,
		TimeOfDay:        "09:00",
		IsActive:         true,
	}
	require.NoError(t, suite.db.Create(subscription).Error)

	// Create alert config
	alertConfig := &models.AlertConfig{
		ID:        uuid.New(),
		UserID:    suite.testUserID,
		AlertType: models.AlertHumidity,
		Condition: "greater_than",
		Threshold: 80.0,
		IsActive:  true,
	}
	require.NoError(t, suite.db.Create(alertConfig).Error)

	t.Run("export all data as JSON", func(t *testing.T) {
		buffer, filename, err := suite.exportService.ExportUserData(
			ctx,
			suite.testUserID,
			services.ExportTypeAll,
			services.ExportFormatJSON,
			"en-US",
		)
		require.NoError(t, err)
		assert.NotNil(t, buffer)
		assert.Contains(t, filename, "shopogoda_all_export_test_user")
		assert.Contains(t, filename, ".json")

		// Parse JSON
		var exportData services.ExportData
		err = json.Unmarshal(buffer.Bytes(), &exportData)
		require.NoError(t, err)

		// Verify all data types are present
		assert.NotNil(t, exportData.User)
		assert.Len(t, exportData.WeatherData, 1)
		assert.Len(t, exportData.Subscriptions, 1)
		assert.Len(t, exportData.AlertConfigs, 1)
		assert.Equal(t, services.ExportTypeAll, exportData.Type)
	})
}

func TestIntegration_ExportServiceEmptyData(t *testing.T) {
	suite := setupExportServiceTest(t)
	defer suite.teardown(t)

	ctx := context.Background()

	t.Run("export weather data with no records", func(t *testing.T) {
		buffer, filename, err := suite.exportService.ExportUserData(
			ctx,
			suite.testUserID,
			services.ExportTypeWeatherData,
			services.ExportFormatJSON,
			"en-US",
		)
		require.NoError(t, err)
		assert.NotNil(t, buffer)
		assert.NotEmpty(t, filename)

		// Parse JSON
		var exportData services.ExportData
		err = json.Unmarshal(buffer.Bytes(), &exportData)
		require.NoError(t, err)

		// Weather data should be empty or nil
		assert.Empty(t, exportData.WeatherData)
	})
}

func TestIntegration_ExportServiceNonExistentUser(t *testing.T) {
	suite := setupExportServiceTest(t)
	defer suite.teardown(t)

	ctx := context.Background()

	t.Run("export for non-existent user returns error", func(t *testing.T) {
		buffer, filename, err := suite.exportService.ExportUserData(
			ctx,
			99999999,
			services.ExportTypeWeatherData,
			services.ExportFormatJSON,
			"en-US",
		)
		assert.Error(t, err)
		assert.Nil(t, buffer)
		assert.Empty(t, filename)
		assert.Contains(t, err.Error(), "failed to get user data")
	})
}

func TestIntegration_ExportServiceDateFiltering(t *testing.T) {
	suite := setupExportServiceTest(t)
	defer suite.teardown(t)

	ctx := context.Background()

	// Create weather data from different time periods
	now := time.Now().UTC()

	// Recent data (within 30 days)
	recentWeather := &models.WeatherData{
		ID:          uuid.New(),
		UserID:      suite.testUserID,
		Temperature: 20.0,
		Humidity:    60,
		Timestamp:   now.AddDate(0, 0, -10), // 10 days ago
	}
	require.NoError(t, suite.db.Create(recentWeather).Error)

	// Old data (beyond 30 days)
	oldWeather := &models.WeatherData{
		ID:          uuid.New(),
		UserID:      suite.testUserID,
		Temperature: 15.0,
		Humidity:    55,
		Timestamp:   now.AddDate(0, 0, -40), // 40 days ago
	}
	require.NoError(t, suite.db.Create(oldWeather).Error)

	// Create old triggered alert (beyond 90 days)
	oldAlert := &models.EnvironmentalAlert{
		ID:        uuid.New(),
		UserID:    suite.testUserID,
		AlertType: models.AlertTemperature,
		Severity:  models.SeverityMedium,
		Title:     "Old Alert",
		Value:     25.0,
		Threshold: 30.0,
		CreatedAt: now.AddDate(0, 0, -100), // 100 days ago
	}
	require.NoError(t, suite.db.Create(oldAlert).Error)

	// Recent triggered alert (within 90 days)
	recentAlert := &models.EnvironmentalAlert{
		ID:        uuid.New(),
		UserID:    suite.testUserID,
		AlertType: models.AlertHumidity,
		Severity:  models.SeverityLow,
		Title:     "Recent Alert",
		Value:     85.0,
		Threshold: 80.0,
		CreatedAt: now.AddDate(0, 0, -30), // 30 days ago
	}
	require.NoError(t, suite.db.Create(recentAlert).Error)

	t.Run("weather data filtered to last 30 days", func(t *testing.T) {
		buffer, _, err := suite.exportService.ExportUserData(
			ctx,
			suite.testUserID,
			services.ExportTypeWeatherData,
			services.ExportFormatJSON,
			"en-US",
		)
		require.NoError(t, err)

		var exportData services.ExportData
		err = json.Unmarshal(buffer.Bytes(), &exportData)
		require.NoError(t, err)

		// Should only have recent weather data
		assert.Len(t, exportData.WeatherData, 1)
		assert.Equal(t, 20.0, exportData.WeatherData[0].Temperature)
	})

	t.Run("triggered alerts filtered to last 90 days", func(t *testing.T) {
		buffer, _, err := suite.exportService.ExportUserData(
			ctx,
			suite.testUserID,
			services.ExportTypeAlerts,
			services.ExportFormatJSON,
			"en-US",
		)
		require.NoError(t, err)

		var exportData services.ExportData
		err = json.Unmarshal(buffer.Bytes(), &exportData)
		require.NoError(t, err)

		// Should only have recent triggered alert
		assert.Len(t, exportData.TriggeredAlerts, 1)
		assert.Equal(t, "Recent Alert", exportData.TriggeredAlerts[0].Title)
	})
}

func TestIntegration_ExportServiceAllFormats(t *testing.T) {
	suite := setupExportServiceTest(t)
	defer suite.teardown(t)

	ctx := context.Background()

	// Create minimal test data
	weatherData := &models.WeatherData{
		ID:          uuid.New(),
		UserID:      suite.testUserID,
		Temperature: 21.0,
		Humidity:    62,
		Timestamp:   time.Now().UTC(),
	}
	require.NoError(t, suite.db.Create(weatherData).Error)

	t.Run("export as JSON format", func(t *testing.T) {
		buffer, filename, err := suite.exportService.ExportUserData(
			ctx,
			suite.testUserID,
			services.ExportTypeWeatherData,
			services.ExportFormatJSON,
			"en-US",
		)
		require.NoError(t, err)
		assert.NotNil(t, buffer)
		assert.Contains(t, filename, ".json")

		// Should be valid JSON
		var exportData services.ExportData
		err = json.Unmarshal(buffer.Bytes(), &exportData)
		require.NoError(t, err)
	})

	t.Run("export as CSV format", func(t *testing.T) {
		buffer, filename, err := suite.exportService.ExportUserData(
			ctx,
			suite.testUserID,
			services.ExportTypeWeatherData,
			services.ExportFormatCSV,
			"en-US",
		)
		require.NoError(t, err)
		assert.NotNil(t, buffer)
		assert.Contains(t, filename, ".csv")

		// Check CSV content (multi-section CSV with varying field counts)
		csvContent := buffer.String()
		assert.Contains(t, csvContent, "export_test_user")
		assert.Contains(t, csvContent, "21.0") // Temperature
		assert.Greater(t, len(csvContent), 0)
	})

	t.Run("export as TXT format", func(t *testing.T) {
		buffer, filename, err := suite.exportService.ExportUserData(
			ctx,
			suite.testUserID,
			services.ExportTypeWeatherData,
			services.ExportFormatTXT,
			"en-US",
		)
		require.NoError(t, err)
		assert.NotNil(t, buffer)
		assert.Contains(t, filename, ".txt")

		// Should contain expected text
		txtContent := buffer.String()
		assert.Contains(t, txtContent, "Export Tester")    // FirstName
		assert.Contains(t, txtContent, "export_test_user") // Username
		assert.Contains(t, txtContent, "End of Export")
	})
}

func TestIntegration_ExportServiceFilenameFormat(t *testing.T) {
	suite := setupExportServiceTest(t)
	defer suite.teardown(t)

	ctx := context.Background()

	t.Run("filename contains correct format", func(t *testing.T) {
		_, filename, err := suite.exportService.ExportUserData(
			ctx,
			suite.testUserID,
			services.ExportTypeWeatherData,
			services.ExportFormatJSON,
			"en-US",
		)
		require.NoError(t, err)

		// Filename format: shopogoda_<type>_<username>_<date>.<format>
		assert.True(t, strings.HasPrefix(filename, "shopogoda_weather_export_test_user_"))
		assert.True(t, strings.HasSuffix(filename, ".json"))

		// Should contain today's date in YYYY-MM-DD format
		today := time.Now().Format("2006-01-02")
		assert.Contains(t, filename, today)
	})
}
