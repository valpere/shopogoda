package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jarcoal/httpmock"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/valpere/shopogoda/internal/config"
	"github.com/valpere/shopogoda/internal/models"
	"github.com/valpere/shopogoda/tests/fixtures"
	"github.com/valpere/shopogoda/tests/helpers"
)

func TestWeatherService_GeocodeLocation(t *testing.T) {
	// Setup
	mockRedis := helpers.NewMockRedis(t)
	defer mockRedis.Close()

	logger := helpers.NewSilentTestLogger()
	config := &config.WeatherConfig{
		OpenWeatherAPIKey: "test_api_key",
		UserAgent:        "Test-Bot/1.0",
	}

	service := NewWeatherService(config, mockRedis.Client, logger)

	// Setup HTTP mocks
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	t.Run("successful geocoding", func(t *testing.T) {
		// Mock cache miss
		mockRedis.ExpectCacheMiss("geocode:london")

		// Mock successful API response
		httpmock.RegisterResponder("GET", "http://api.openweathermap.org/geo/1.0/direct",
			httpmock.NewStringResponder(200, fixtures.GetMockGeocodeResponse()))

		// Mock cache set
		mockRedis.ExpectCacheSetWithTTL("geocode:london", "", 86400) // 24 hours

		result, err := service.GeocodeLocation(context.Background(), "london")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 51.5074, result.Latitude)
		assert.Equal(t, -0.1278, result.Longitude)

		mockRedis.ExpectationsWereMet(t)
	})

	t.Run("cache hit", func(t *testing.T) {
		// Mock cache hit
		cacheData := `[{"lat": 51.5074, "lon": -0.1278, "name": "London", "country": "GB"}]`
		mockRedis.ExpectCacheHit("geocode:paris", cacheData)

		result, err := service.GeocodeLocation(context.Background(), "paris")

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 51.5074, result.Latitude)
		assert.Equal(t, -0.1278, result.Longitude)

		mockRedis.ExpectationsWereMet(t)
	})

	t.Run("location not found", func(t *testing.T) {
		// Mock cache miss
		mockRedis.ExpectCacheMiss("geocode:nonexistent")

		// Mock API response with empty array
		httpmock.RegisterResponder("GET", "http://api.openweathermap.org/geo/1.0/direct",
			httpmock.NewStringResponder(200, "[]"))

		result, err := service.GeocodeLocation(context.Background(), "nonexistent")

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "location not found")

		mockRedis.ExpectationsWereMet(t)
	})

	t.Run("API error", func(t *testing.T) {
		// Mock cache miss
		mockRedis.ExpectCacheMiss("geocode:error")

		// Mock API error response
		httpmock.RegisterResponder("GET", "http://api.openweathermap.org/geo/1.0/direct",
			httpmock.NewStringResponder(500, "Internal Server Error"))

		result, err := service.GeocodeLocation(context.Background(), "error")

		assert.Error(t, err)
		assert.Nil(t, result)

		mockRedis.ExpectationsWereMet(t)
	})

	t.Run("invalid JSON response", func(t *testing.T) {
		// Mock cache miss
		mockRedis.ExpectCacheMiss("geocode:invalid")

		// Mock invalid JSON response
		httpmock.RegisterResponder("GET", "http://api.openweathermap.org/geo/1.0/direct",
			httpmock.NewStringResponder(200, fixtures.GetInvalidJSONResponse()))

		result, err := service.GeocodeLocation(context.Background(), "invalid")

		assert.Error(t, err)
		assert.Nil(t, result)

		mockRedis.ExpectationsWereMet(t)
	})
}

func TestWeatherService_GetCurrentWeather(t *testing.T) {
	// Setup
	mockDB := helpers.NewMockDB(t)
	defer mockDB.Close()

	mockRedis := helpers.NewMockRedis(t)
	defer mockRedis.Close()

	logger := helpers.NewSilentTestLogger()
	config := &config.WeatherConfig{
		OpenWeatherAPIKey: "test_api_key",
		UserAgent:        "Test-Bot/1.0",
	}

	service := NewWeatherService(config, mockRedis.Client, logger)

	// Setup HTTP mocks
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	lat, lon := 51.5074, -0.1278

	t.Run("successful weather fetch", func(t *testing.T) {
		// Mock cache miss
		mockRedis.ExpectCacheMiss("weather:51.507400:-0.127800")

		// Mock successful API responses
		httpmock.RegisterResponder("GET", "https://api.openweathermap.org/data/2.5/weather",
			httpmock.NewStringResponder(200, fixtures.GetMockWeatherResponse()))

		httpmock.RegisterResponder("GET", "http://api.openweathermap.org/data/2.5/air_pollution",
			httpmock.NewStringResponder(200, fixtures.GetMockAirQualityResponse()))

		// Mock cache set
		mockRedis.ExpectCacheSetWithTTL("weather:51.507400:-0.127800", "", 600) // 10 minutes

		result, err := service.GetCurrentWeather(context.Background(), lat, lon)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 20.5, result.Temperature)
		assert.Equal(t, "London", result.LocationName)
		assert.Equal(t, 2, result.AQI)

		mockRedis.ExpectationsWereMet(t)
	})

	t.Run("cache hit", func(t *testing.T) {
		// Create cached weather data
		cachedData := &models.WeatherData{
			LocationName:  "London",
			Temperature:   18.5,
			Humidity:      70,
			Pressure:      1015,
			AQI:          1,
			Timestamp:    time.Now(),
		}

		// Mock cache hit
		cacheJSON := `{"location_name":"London","temperature":18.5,"humidity":70,"pressure":1015,"aqi":1}`
		mockRedis.ExpectCacheHit("weather:51.507400:-0.127800", cacheJSON)

		result, err := service.GetCurrentWeather(context.Background(), lat, lon)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "London", result.LocationName)
		assert.Equal(t, 18.5, result.Temperature)

		mockRedis.ExpectationsWereMet(t)
	})

	t.Run("API error", func(t *testing.T) {
		// Mock cache miss
		mockRedis.ExpectCacheMiss("weather:51.507400:-0.127800")

		// Mock API error
		httpmock.RegisterResponder("GET", "https://api.openweathermap.org/data/2.5/weather",
			httpmock.NewStringResponder(401, `{"cod": 401, "message": "Invalid API key"}`))

		result, err := service.GetCurrentWeather(context.Background(), lat, lon)

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "weather API request failed")

		mockRedis.ExpectationsWereMet(t)
	})
}

func TestWeatherService_GetForecast(t *testing.T) {
	// Setup
	mockRedis := helpers.NewMockRedis(t)
	defer mockRedis.Close()

	logger := helpers.NewSilentTestLogger()
	config := &config.WeatherConfig{
		OpenWeatherAPIKey: "test_api_key",
		UserAgent:        "Test-Bot/1.0",
	}

	service := NewWeatherService(config, mockRedis.Client, logger)

	// Setup HTTP mocks
	httpmock.Activate()
	defer httpmock.DeactivateAndReset()

	lat, lon := 51.5074, -0.1278

	t.Run("successful forecast fetch", func(t *testing.T) {
		// Mock cache miss
		mockRedis.ExpectCacheMiss("forecast:51.507400:-0.127800")

		// Mock successful API response
		httpmock.RegisterResponder("GET", "https://api.openweathermap.org/data/2.5/forecast",
			httpmock.NewStringResponder(200, fixtures.GetMockForecastResponse()))

		// Mock cache set
		mockRedis.ExpectCacheSetWithTTL("forecast:51.507400:-0.127800", "", 3600) // 1 hour

		result, err := service.GetForecast(context.Background(), lat, lon)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Greater(t, len(result), 0)
		assert.Equal(t, "London", result[0].LocationName)
		assert.Equal(t, 18.5, result[0].Temperature)

		mockRedis.ExpectationsWereMet(t)
	})

	t.Run("empty forecast", func(t *testing.T) {
		// Mock cache miss
		mockRedis.ExpectCacheMiss("forecast:51.507400:-0.127800")

		// Mock empty forecast response
		httpmock.RegisterResponder("GET", "https://api.openweathermap.org/data/2.5/forecast",
			httpmock.NewStringResponder(200, `{"list": [], "city": {"name": "London", "country": "GB"}}`))

		result, err := service.GetForecast(context.Background(), lat, lon)

		assert.NoError(t, err)
		assert.Equal(t, 0, len(result))

		mockRedis.ExpectationsWereMet(t)
	})
}

func TestWeatherService_SaveWeatherData(t *testing.T) {
	// Setup
	mockDB := helpers.NewMockDB(t)
	defer mockDB.Close()

	logger := helpers.NewSilentTestLogger()
	config := &config.WeatherConfig{
		OpenWeatherAPIKey: "test_api_key",
		UserAgent:        "Test-Bot/1.0",
	}

	service := NewWeatherService(config, nil, logger)
	service.db = mockDB.DB // Inject mock database

	t.Run("successful save", func(t *testing.T) {
		weatherData := helpers.MockWeatherData(123)

		// Mock database expectations
		mockDB.Mock.ExpectBegin()
		mockDB.Mock.ExpectExec(`INSERT INTO "weather_data"`).
			WillReturnResult(helpers.NewResult(1, 1))
		mockDB.Mock.ExpectCommit()

		err := service.SaveWeatherData(context.Background(), weatherData)

		assert.NoError(t, err)
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("database error", func(t *testing.T) {
		weatherData := helpers.MockWeatherData(123)

		// Mock database error
		mockDB.Mock.ExpectBegin()
		mockDB.Mock.ExpectExec(`INSERT INTO "weather_data"`).
			WillReturnError(errors.New("database error"))
		mockDB.Mock.ExpectRollback()

		err := service.SaveWeatherData(context.Background(), weatherData)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		mockDB.ExpectationsWereMet(t)
	})
}

func TestWeatherService_Integration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping integration test in short mode")
	}

	// This test requires real Redis and database connections
	// It should be moved to integration tests directory
	t.Skip("Move to integration tests")
}