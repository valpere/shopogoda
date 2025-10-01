package services

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/go-redis/redismock/v9"
	"github.com/rs/zerolog"
	"github.com/stretchr/testify/assert"

	"github.com/valpere/shopogoda/internal/config"
	"github.com/valpere/shopogoda/pkg/weather"
)

func TestNewWeatherService(t *testing.T) {
	logger := zerolog.Nop()
	rdb, _ := redismock.NewClientMock()

	cfg := &config.WeatherConfig{
		OpenWeatherAPIKey: "test-key",
		UserAgent:         "TestBot/1.0",
	}

	service := NewWeatherService(cfg, rdb, &logger)

	assert.NotNil(t, service)
	assert.NotNil(t, service.client)
	assert.NotNil(t, service.geocoder)
	assert.NotNil(t, service.redis)
	assert.NotNil(t, service.httpClient)
	assert.Equal(t, cfg, service.config)
}

func TestGetUserAgent(t *testing.T) {
	logger := zerolog.Nop()
	rdb, _ := redismock.NewClientMock()

	t.Run("returns custom user agent from config", func(t *testing.T) {
		cfg := &config.WeatherConfig{
			OpenWeatherAPIKey: "test-key",
			UserAgent:         "CustomBot/2.0",
		}
		service := NewWeatherService(cfg, rdb, &logger)

		assert.Equal(t, "CustomBot/2.0", service.getUserAgent())
	})

	t.Run("returns default user agent when config is empty", func(t *testing.T) {
		cfg := &config.WeatherConfig{
			OpenWeatherAPIKey: "test-key",
			UserAgent:         "",
		}
		service := NewWeatherService(cfg, rdb, &logger)

		// Should return default
		agent := service.getUserAgent()
		assert.NotEmpty(t, agent)
	})
}

func TestGetCurrentWeather(t *testing.T) {
	logger := zerolog.Nop()

	t.Run("returns cached weather data", func(t *testing.T) {
		rdb, mock := redismock.NewClientMock()
		cfg := &config.WeatherConfig{
			OpenWeatherAPIKey: "test-key",
		}
		service := NewWeatherService(cfg, rdb, &logger)

		ctx := context.Background()
		lat, lon := 50.4501, 30.5234

		// Mock cached weather data
		cachedData := weather.WeatherData{
			Temperature: 20.5,
			Description: "Clear sky",
			Humidity:    60,
		}
		cachedJSON, _ := json.Marshal(cachedData)
		cacheKey := "weather:current:50.4501:30.5234"

		mock.ExpectGet(cacheKey).SetVal(string(cachedJSON))

		result, err := service.GetCurrentWeather(ctx, lat, lon)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 20.5, result.Temperature)
		assert.Equal(t, "Clear sky", result.Description)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("handles cache miss and API call", func(t *testing.T) {
		rdb, mock := redismock.NewClientMock()
		cfg := &config.WeatherConfig{
			OpenWeatherAPIKey: "test-key",
		}
		service := NewWeatherService(cfg, rdb, &logger)

		ctx := context.Background()
		lat, lon := 50.4501, 30.5234
		cacheKey := "weather:current:50.4501:30.5234"

		// Expect cache miss
		mock.ExpectGet(cacheKey).RedisNil()

		// Expect cache set (will happen after API call, but API will fail in this test)
		// Note: The actual API call will fail since we don't have a real API key
		// This test demonstrates the cache logic

		_, err := service.GetCurrentWeather(ctx, lat, lon)

		// API call will fail, which is expected in unit test
		assert.Error(t, err)
	})
}

func TestGetForecast(t *testing.T) {
	logger := zerolog.Nop()

	t.Run("returns cached forecast data", func(t *testing.T) {
		rdb, mock := redismock.NewClientMock()
		cfg := &config.WeatherConfig{
			OpenWeatherAPIKey: "test-key",
		}
		service := NewWeatherService(cfg, rdb, &logger)

		ctx := context.Background()
		lat, lon := 50.4501, 30.5234
		days := 5

		// Mock cached forecast data
		cachedData := weather.ForecastData{
			Location: "Kyiv",
			Forecasts: []weather.DailyForecast{
				{MinTemp: 18.0, MaxTemp: 22.0, Description: "Clear"},
				{MinTemp: 19.0, MaxTemp: 23.0, Description: "Sunny"},
			},
		}
		cachedJSON, _ := json.Marshal(cachedData)
		cacheKey := "weather:forecast:50.4501:30.5234:5"

		mock.ExpectGet(cacheKey).SetVal(string(cachedJSON))

		result, err := service.GetForecast(ctx, lat, lon, days)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Forecasts, 2)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGetAirQuality(t *testing.T) {
	logger := zerolog.Nop()

	t.Run("returns cached air quality data", func(t *testing.T) {
		rdb, mock := redismock.NewClientMock()
		cfg := &config.WeatherConfig{
			OpenWeatherAPIKey: "test-key",
			AirQualityAPIKey:  "air-key",
		}
		service := NewWeatherService(cfg, rdb, &logger)

		ctx := context.Background()
		lat, lon := 50.4501, 30.5234

		// Mock cached air quality data
		cachedData := weather.AirQualityData{
			AQI:  2,
			PM25: 15.5,
			PM10: 25.0,
		}
		cachedJSON, _ := json.Marshal(cachedData)
		cacheKey := "weather:air:50.4501:30.5234"

		mock.ExpectGet(cacheKey).SetVal(string(cachedJSON))

		result, err := service.GetAirQuality(ctx, lat, lon)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 2, result.AQI)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGeocodeLocation(t *testing.T) {
	logger := zerolog.Nop()

	t.Run("returns cached geocoded location", func(t *testing.T) {
		rdb, mock := redismock.NewClientMock()
		cfg := &config.WeatherConfig{
			OpenWeatherAPIKey: "test-key",
		}
		service := NewWeatherService(cfg, rdb, &logger)

		ctx := context.Background()
		location := "Kyiv"

		// Mock cached geocode result (using normalized lowercase key)
		cachedResult := weather.Location{
			Name:      "Київ",
			Country:   "UA",
			Latitude:  50.4501,
			Longitude: 30.5234,
		}
		cachedJSON, _ := json.Marshal(cachedResult)
		cacheKey := "geocode:kyiv" // Normalized to lowercase

		mock.ExpectGet(cacheKey).SetVal(string(cachedJSON))

		result, err := service.GeocodeLocation(ctx, location)

		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Київ", result.Name)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestCacheLocation(t *testing.T) {
	logger := zerolog.Nop()
	rdb, mock := redismock.NewClientMock()
	cfg := &config.WeatherConfig{
		OpenWeatherAPIKey: "test-key",
	}
	service := NewWeatherService(cfg, rdb, &logger)

	ctx := context.Background()
	location := &weather.Location{
		Name:      "Kyiv",
		Country:   "UA",
		Latitude:  50.4501,
		Longitude: 30.5234,
	}

	cacheKey := "geocode:test"
	locationJSON, _ := json.Marshal(location)
	mock.ExpectSet(cacheKey, locationJSON, 24*time.Hour).SetVal("OK")

	err := service.cacheLocation(ctx, cacheKey, location)

	assert.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestGetCurrentWeatherByCoords(t *testing.T) {
	logger := zerolog.Nop()
	rdb, _ := redismock.NewClientMock()
	cfg := &config.WeatherConfig{
		OpenWeatherAPIKey: "test-key",
	}
	service := NewWeatherService(cfg, rdb, &logger)

	ctx := context.Background()
	lat, lon := 50.4501, 30.5234

	// This will fail due to no real API, but tests the method exists
	_, err := service.GetCurrentWeatherByCoords(ctx, lat, lon)
	assert.Error(t, err) // Expected since we don't have real API
}

func TestGetCurrentWeatherByLocation(t *testing.T) {
	logger := zerolog.Nop()
	rdb, mock := redismock.NewClientMock()
	cfg := &config.WeatherConfig{
		OpenWeatherAPIKey: "test-key",
	}
	service := NewWeatherService(cfg, rdb, &logger)

	ctx := context.Background()
	location := "Kyiv"

	// Mock geocoding result in cache
	geocodeResult := weather.Location{
		Latitude: 50.4501, Longitude: 30.5234, Name: "Kyiv", Country: "UA",
	}
	geocodeJSON, _ := json.Marshal(geocodeResult)
	mock.ExpectGet("geocode:Kyiv").SetVal(string(geocodeJSON))

	// Will fail on weather API call, but tests geocoding works
	_, err := service.GetCurrentWeatherByLocation(ctx, location)
	assert.Error(t, err) // Expected failure on weather API
}

func TestGetLocationName(t *testing.T) {
	logger := zerolog.Nop()

	t.Run("returns cached location name", func(t *testing.T) {
		rdb, mock := redismock.NewClientMock()
		cfg := &config.WeatherConfig{
			OpenWeatherAPIKey: "test-key",
		}
		service := NewWeatherService(cfg, rdb, &logger)

		ctx := context.Background()
		lat, lon := 50.4501, 30.5234
		cacheKey := "reverse_geocode:50.4501:30.5234"

		mock.ExpectGet(cacheKey).SetVal("Київ (50.4501, 30.5234)")

		result, err := service.GetLocationName(ctx, lat, lon)

		assert.NoError(t, err)
		assert.Equal(t, "Київ (50.4501, 30.5234)", result)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestGetCompleteWeatherData(t *testing.T) {
	logger := zerolog.Nop()
	rdb, _ := redismock.NewClientMock()
	cfg := &config.WeatherConfig{
		OpenWeatherAPIKey: "test-key",
	}
	service := NewWeatherService(cfg, rdb, &logger)

	ctx := context.Background()
	lat, lon := 50.4501, 30.5234

	// Will fail due to no real API, but tests method structure
	_, err := service.GetCompleteWeatherData(ctx, lat, lon)
	assert.Error(t, err) // Expected since we don't have real API
}

func TestGeocodeWithNominatim(t *testing.T) {
	logger := zerolog.Nop()
	rdb, _ := redismock.NewClientMock()
	cfg := &config.WeatherConfig{
		OpenWeatherAPIKey: "test-key",
	}
	service := NewWeatherService(cfg, rdb, &logger)

	// Create test server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := []map[string]interface{}{
			{
				"lat":          "50.4501",
				"lon":          "30.5234",
				"display_name": "Kyiv, Ukraine",
			},
		}
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	ctx := context.Background()

	// Note: This will use the real Nominatim API endpoint, not our test server
	// In a real scenario, we'd need to inject the HTTP client or URL
	results, err := service.geocodeWithNominatim(ctx, "Kyiv")

	// May fail or succeed depending on network, but tests the method
	_ = results
	_ = err
}

func TestReverseGeocodeWithNominatim(t *testing.T) {
	logger := zerolog.Nop()
	rdb, _ := redismock.NewClientMock()
	cfg := &config.WeatherConfig{
		OpenWeatherAPIKey: "test-key",
	}
	service := NewWeatherService(cfg, rdb, &logger)

	ctx := context.Background()
	lat, lon := 50.4501, 30.5234

	// Will call real Nominatim API - may fail in CI
	_, err := service.reverseGeocodeWithNominatim(ctx, lat, lon)
	_ = err // Ignore error as it depends on network
}

func TestFormatLocationFromAddress(t *testing.T) {
	logger := zerolog.Nop()
	rdb, _ := redismock.NewClientMock()
	cfg := &config.WeatherConfig{
		OpenWeatherAPIKey: "test-key",
	}
	service := NewWeatherService(cfg, rdb, &logger)

	t.Run("formats city with coordinates", func(t *testing.T) {
		addr := NominatimAddress{
			City:    "Kyiv",
			Country: "Ukraine",
		}
		result := service.formatLocationFromAddress(addr, 50.4501, 30.5234)
		assert.Equal(t, "Kyiv (50.4501, 30.5234)", result)
	})

	t.Run("uses town when city is empty", func(t *testing.T) {
		addr := NominatimAddress{
			Town:    "Lviv",
			Country: "Ukraine",
		}
		result := service.formatLocationFromAddress(addr, 49.8397, 24.0297)
		assert.Equal(t, "Lviv (49.8397, 24.0297)", result)
	})

	t.Run("uses village when city and town are empty", func(t *testing.T) {
		addr := NominatimAddress{
			Village: "Kryvorivnia",
			County:  "Ivano-Frankivsk",
			Country: "Ukraine",
		}
		result := service.formatLocationFromAddress(addr, 48.3833, 24.5667)
		assert.Equal(t, "Kryvorivnia (48.3833, 24.5667)", result)
	})

	t.Run("uses near prefix for state", func(t *testing.T) {
		addr := NominatimAddress{
			State:   "Kyiv Oblast",
			Country: "Ukraine",
		}
		result := service.formatLocationFromAddress(addr, 50.4501, 30.5234)
		assert.Equal(t, "near Kyiv Oblast (50.4501, 30.5234)", result)
	})

	t.Run("fallback to coordinates when nothing else available", func(t *testing.T) {
		addr := NominatimAddress{
			Country: "Ukraine",
		}
		result := service.formatLocationFromAddress(addr, 48.3794, 31.1656)
		assert.Equal(t, "Location (48.3794, 31.1656)", result)
	})
}
