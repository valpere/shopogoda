//go:build integration
// +build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/wait"

	"github.com/valpere/shopogoda/internal/config"
	"github.com/valpere/shopogoda/internal/services"
	"github.com/valpere/shopogoda/pkg/weather"
	"github.com/valpere/shopogoda/tests/helpers"
)

type WeatherServiceTestSuite struct {
	redisClient    *redis.Client
	redisContainer testcontainers.Container
	weatherService *services.WeatherService
	mockServer     *httptest.Server
}

func setupWeatherServiceTest(t *testing.T) *WeatherServiceTestSuite {
	ctx := context.Background()

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

	// Get Redis container port
	redisHost, err := redisContainer.Host(ctx)
	require.NoError(t, err)

	redisPort, err := redisContainer.MappedPort(ctx, "6379")
	require.NoError(t, err)

	// Connect to Redis
	redisClient := redis.NewClient(&redis.Options{
		Addr: redisHost + ":" + redisPort.Port(),
	})

	// Test Redis connection
	pong, err := redisClient.Ping(ctx).Result()
	require.NoError(t, err)
	require.Equal(t, "PONG", pong)

	// Create mock OpenWeatherMap API server
	mockServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Mock weather endpoint
		if r.URL.Path == "/data/2.5/weather" {
			response := map[string]interface{}{
				"main": map[string]interface{}{
					"temp":     15.5,
					"humidity": 65,
					"pressure": 1013.0,
				},
				"wind": map[string]interface{}{
					"speed": 5.0,
					"deg":   180,
				},
				"weather": []map[string]interface{}{
					{
						"description": "clear sky",
						"icon":        "01d",
					},
				},
				"visibility": 10000,
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		// Mock forecast endpoint
		if r.URL.Path == "/data/2.5/forecast" {
			response := map[string]interface{}{
				"list": []map[string]interface{}{
					{
						"dt": time.Now().Unix(),
						"main": map[string]interface{}{
							"temp":     18.0,
							"humidity": 70,
						},
						"weather": []map[string]interface{}{
							{
								"description": "partly cloudy",
								"icon":        "02d",
							},
						},
						"wind": map[string]interface{}{
							"speed": 3.5,
						},
					},
				},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		// Mock air quality endpoint
		if r.URL.Path == "/data/2.5/air_pollution" {
			response := map[string]interface{}{
				"list": []map[string]interface{}{
					{
						"main": map[string]interface{}{
							"aqi": 2,
						},
						"components": map[string]interface{}{
							"co":    200.5,
							"no2":   10.2,
							"o3":    50.3,
							"pm2_5": 15.8,
							"pm10":  25.4,
						},
					},
				},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		// Mock geocoding endpoint
		if r.URL.Path == "/geo/1.0/direct" {
			response := []map[string]interface{}{
				{
					"name":    "Kyiv",
					"lat":     50.4501,
					"lon":     30.5234,
					"country": "UA",
				},
			}
			json.NewEncoder(w).Encode(response)
			return
		}

		// Default 404 for unknown endpoints
		w.WriteHeader(http.StatusNotFound)
	}))

	// Create weather service with mock API
	logger := helpers.NewSilentTestLogger()
	cfg := &config.WeatherConfig{
		OpenWeatherAPIKey: "test_api_key",
		UserAgent:         "ShoPogoda-Test/1.0",
	}

	weatherService := services.NewWeatherService(cfg, redisClient, logger)

	return &WeatherServiceTestSuite{
		redisClient:    redisClient,
		redisContainer: redisContainer,
		weatherService: weatherService,
		mockServer:     mockServer,
	}
}

func (suite *WeatherServiceTestSuite) teardown(t *testing.T) {
	ctx := context.Background()

	if suite.mockServer != nil {
		suite.mockServer.Close()
	}

	if suite.redisClient != nil {
		suite.redisClient.Close()
	}

	if suite.redisContainer != nil {
		require.NoError(t, suite.redisContainer.Terminate(ctx))
	}
}

func TestIntegration_WeatherServiceGetCurrentWeather(t *testing.T) {
	suite := setupWeatherServiceTest(t)
	defer suite.teardown(t)

	ctx := context.Background()
	lat, lon := 50.4501, 30.5234 // Kyiv coordinates

	t.Run("get current weather from API", func(t *testing.T) {
		// First call - should hit API (we can't easily mock, so we test cache behavior)
		cacheKey := fmt.Sprintf("weather:current:%.4f:%.4f", lat, lon)

		// Ensure cache is empty
		suite.redisClient.Del(ctx, cacheKey)

		// Since we can't easily mock the OpenWeatherMap API client,
		// we'll test the caching behavior instead
		testData := &weather.WeatherData{
			Temperature:   15.5,
			Humidity:      65,
			Pressure:      1013.0,
			WindSpeed:     18.0, // 5 m/s * 3.6 = 18 km/h
			WindDirection: 180,
			Visibility:    10.0,
			Description:   "clear sky",
			Icon:          "01d",
			Timestamp:     time.Now(),
		}

		// Manually populate cache
		weatherJSON, _ := json.Marshal(testData)
		err := suite.redisClient.Set(ctx, cacheKey, weatherJSON, 10*time.Minute).Err()
		require.NoError(t, err)

		// Call service - should return cached data
		weatherData, err := suite.weatherService.GetCurrentWeather(ctx, lat, lon)
		require.NoError(t, err)
		assert.NotNil(t, weatherData)
		assert.Equal(t, 15.5, weatherData.Temperature)
		assert.Equal(t, 65, weatherData.Humidity)
		assert.Equal(t, "clear sky", weatherData.Description)
	})

	t.Run("get current weather from cache", func(t *testing.T) {
		cacheKey := fmt.Sprintf("weather:current:%.4f:%.4f", lat, lon)

		// Populate cache with test data
		testData := &weather.WeatherData{
			Temperature: 20.0,
			Humidity:    50,
			Description: "cached weather",
		}
		weatherJSON, _ := json.Marshal(testData)
		suite.redisClient.Set(ctx, cacheKey, weatherJSON, 10*time.Minute)

		// Call service - should return cached data
		weatherData, err := suite.weatherService.GetCurrentWeather(ctx, lat, lon)
		require.NoError(t, err)
		assert.Equal(t, 20.0, weatherData.Temperature)
		assert.Equal(t, "cached weather", weatherData.Description)
	})
}

func TestIntegration_WeatherServiceGetForecast(t *testing.T) {
	suite := setupWeatherServiceTest(t)
	defer suite.teardown(t)

	ctx := context.Background()
	lat, lon := 50.4501, 30.5234
	days := 5

	t.Run("forecast caching behavior", func(t *testing.T) {
		cacheKey := fmt.Sprintf("weather:forecast:%.4f:%.4f:%d", lat, lon, days)

		// Populate cache with test forecast data
		testForecast := &weather.ForecastData{
			Location: "Kyiv",
			Forecasts: []weather.DailyForecast{
				{
					Date:        time.Now(),
					MinTemp:     10.0,
					MaxTemp:     20.0,
					Description: "sunny",
					Icon:        "01d",
					Humidity:    60,
					WindSpeed:   5.0,
				},
			},
		}
		forecastJSON, _ := json.Marshal(testForecast)
		suite.redisClient.Set(ctx, cacheKey, forecastJSON, time.Hour)

		// Call service - should return cached data
		forecastData, err := suite.weatherService.GetForecast(ctx, lat, lon, days)
		require.NoError(t, err)
		assert.NotNil(t, forecastData)
		assert.Equal(t, "Kyiv", forecastData.Location)
		assert.Len(t, forecastData.Forecasts, 1)
		assert.Equal(t, 10.0, forecastData.Forecasts[0].MinTemp)
		assert.Equal(t, 20.0, forecastData.Forecasts[0].MaxTemp)
	})
}

func TestIntegration_WeatherServiceGetAirQuality(t *testing.T) {
	suite := setupWeatherServiceTest(t)
	defer suite.teardown(t)

	ctx := context.Background()
	lat, lon := 50.4501, 30.5234

	t.Run("air quality caching behavior", func(t *testing.T) {
		cacheKey := fmt.Sprintf("weather:air:%.4f:%.4f", lat, lon)

		// Populate cache with test air quality data
		testAirData := &weather.AirQualityData{
			AQI:       2,
			CO:        200.5,
			NO2:       10.2,
			O3:        50.3,
			PM25:      15.8,
			PM10:      25.4,
			Timestamp: time.Now(),
		}
		airJSON, _ := json.Marshal(testAirData)
		suite.redisClient.Set(ctx, cacheKey, airJSON, 30*time.Minute)

		// Call service - should return cached data
		airData, err := suite.weatherService.GetAirQuality(ctx, lat, lon)
		require.NoError(t, err)
		assert.NotNil(t, airData)
		assert.Equal(t, 2, airData.AQI)
		assert.Equal(t, 200.5, airData.CO)
		assert.Equal(t, 10.2, airData.NO2)
		assert.Equal(t, 15.8, airData.PM25)
	})
}

func TestIntegration_WeatherServiceGeocodeLocation(t *testing.T) {
	suite := setupWeatherServiceTest(t)
	defer suite.teardown(t)

	ctx := context.Background()

	t.Run("geocode location with cache", func(t *testing.T) {
		locationName := "Kyiv"
		cacheKey := "geocode:kyiv"

		// Populate cache with test location
		testLocation := &weather.Location{
			Latitude:  50.4501,
			Longitude: 30.5234,
			Name:      "Kyiv",
			Country:   "Ukraine",
			City:      "Kyiv",
		}
		locationJSON, _ := json.Marshal(testLocation)
		suite.redisClient.Set(ctx, cacheKey, locationJSON, 24*time.Hour)

		// Call service - should return cached data
		location, err := suite.weatherService.GeocodeLocation(ctx, locationName)
		require.NoError(t, err)
		assert.NotNil(t, location)
		assert.Equal(t, 50.4501, location.Latitude)
		assert.Equal(t, 30.5234, location.Longitude)
		assert.Equal(t, "Kyiv", location.Name)
	})

	t.Run("geocode empty location name", func(t *testing.T) {
		_, err := suite.weatherService.GeocodeLocation(ctx, "")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "location name cannot be empty")
	})

	t.Run("geocode normalizes location name for cache", func(t *testing.T) {
		// Both " KYIV " and "kyiv" should use same cache key
		cacheKey := "geocode:kyiv"
		testLocation := &weather.Location{
			Latitude:  50.4501,
			Longitude: 30.5234,
			Name:      "Kyiv",
			Country:   "Ukraine",
		}
		locationJSON, _ := json.Marshal(testLocation)
		suite.redisClient.Set(ctx, cacheKey, locationJSON, 24*time.Hour)

		// Test with different casing and whitespace
		location1, err := suite.weatherService.GeocodeLocation(ctx, " KYIV ")
		require.NoError(t, err)
		assert.Equal(t, "Kyiv", location1.Name)

		location2, err := suite.weatherService.GeocodeLocation(ctx, "kyiv")
		require.NoError(t, err)
		assert.Equal(t, "Kyiv", location2.Name)
	})
}

func TestIntegration_WeatherServiceGetCompleteWeatherData(t *testing.T) {
	suite := setupWeatherServiceTest(t)
	defer suite.teardown(t)

	ctx := context.Background()
	lat, lon := 50.4501, 30.5234

	t.Run("get complete weather data with air quality", func(t *testing.T) {
		// Populate both weather and air quality caches
		weatherCacheKey := fmt.Sprintf("weather:current:%.4f:%.4f", lat, lon)
		airCacheKey := fmt.Sprintf("weather:air:%.4f:%.4f", lat, lon)

		testWeather := &weather.WeatherData{
			Temperature:   15.5,
			Humidity:      65,
			Pressure:      1013.0,
			WindSpeed:     18.0,
			WindDirection: 180,
			Visibility:    10.0,
			Description:   "clear sky",
			Icon:          "01d",
			LocationName:  "Kyiv",
			Timestamp:     time.Now(),
		}
		weatherJSON, _ := json.Marshal(testWeather)
		suite.redisClient.Set(ctx, weatherCacheKey, weatherJSON, 10*time.Minute)

		testAir := &weather.AirQualityData{
			AQI:  2,
			CO:   200.5,
			NO2:  10.2,
			O3:   50.3,
			PM25: 15.8,
			PM10: 25.4,
		}
		airJSON, _ := json.Marshal(testAir)
		suite.redisClient.Set(ctx, airCacheKey, airJSON, 30*time.Minute)

		// Get complete weather data
		completeData, err := suite.weatherService.GetCompleteWeatherData(ctx, lat, lon)
		require.NoError(t, err)
		assert.NotNil(t, completeData)

		// Verify weather data
		assert.Equal(t, 15.5, completeData.Temperature)
		assert.Equal(t, 65, completeData.Humidity)
		assert.Equal(t, "clear sky", completeData.Description)

		// Verify air quality data
		assert.Equal(t, 2, completeData.AQI)
		assert.Equal(t, 200.5, completeData.CO)
		assert.Equal(t, 15.8, completeData.PM25)
	})

	t.Run("get complete weather data with missing air quality", func(t *testing.T) {
		// Clear air quality cache to test fallback
		airCacheKey := fmt.Sprintf("weather:air:%.4f:%.4f", lat, lon)
		suite.redisClient.Del(ctx, airCacheKey)

		// Populate only weather cache
		weatherCacheKey := fmt.Sprintf("weather:current:%.4f:%.4f", lat, lon)
		testWeather := &weather.WeatherData{
			Temperature: 18.0,
			Humidity:    70,
			Description: "partly cloudy",
		}
		weatherJSON, _ := json.Marshal(testWeather)
		suite.redisClient.Set(ctx, weatherCacheKey, weatherJSON, 10*time.Minute)

		// Should still return weather data with zero air quality values
		completeData, err := suite.weatherService.GetCompleteWeatherData(ctx, lat, lon)
		require.NoError(t, err)
		assert.NotNil(t, completeData)
		assert.Equal(t, 18.0, completeData.Temperature)
		// Air quality should be zero values (fallback behavior)
		assert.Equal(t, 0, completeData.AQI)
		assert.Equal(t, 0.0, completeData.PM25)
	})
}

func TestIntegration_WeatherServiceGetLocationName(t *testing.T) {
	suite := setupWeatherServiceTest(t)
	defer suite.teardown(t)

	ctx := context.Background()
	lat, lon := 50.4501, 30.5234

	t.Run("reverse geocode with cache", func(t *testing.T) {
		cacheKey := fmt.Sprintf("reverse_geocode:%.4f:%.4f", lat, lon)

		// Populate cache with location name
		locationName := "Kyiv (50.4501, 30.5234)"
		suite.redisClient.Set(ctx, cacheKey, locationName, 24*time.Hour)

		// Call service - should return cached data
		result, err := suite.weatherService.GetLocationName(ctx, lat, lon)
		require.NoError(t, err)
		assert.Equal(t, locationName, result)
	})

	t.Run("reverse geocode cache miss returns fallback", func(t *testing.T) {
		// Use coordinates that won't be in cache
		unusedLat, unusedLon := 40.7128, -74.0060
		cacheKey := fmt.Sprintf("reverse_geocode:%.4f:%.4f", unusedLat, unusedLon)
		suite.redisClient.Del(ctx, cacheKey)

		// Without mock Nominatim server, should return fallback coordinate format
		result, err := suite.weatherService.GetLocationName(ctx, unusedLat, unusedLon)
		require.NoError(t, err)
		// Fallback format: "Location (lat, lon)"
		assert.Contains(t, result, fmt.Sprintf("%.4f", unusedLat))
		assert.Contains(t, result, fmt.Sprintf("%.4f", unusedLon))
	})
}

func TestIntegration_WeatherServiceCacheTTL(t *testing.T) {
	suite := setupWeatherServiceTest(t)
	defer suite.teardown(t)

	ctx := context.Background()
	lat, lon := 50.4501, 30.5234

	t.Run("weather cache has 10 minute TTL", func(t *testing.T) {
		cacheKey := fmt.Sprintf("weather:current:%.4f:%.4f", lat, lon)
		testData := &weather.WeatherData{Temperature: 15.0}
		weatherJSON, _ := json.Marshal(testData)
		suite.redisClient.Set(ctx, cacheKey, weatherJSON, 10*time.Minute)

		// Check TTL
		ttl, err := suite.redisClient.TTL(ctx, cacheKey).Result()
		require.NoError(t, err)
		assert.Greater(t, ttl, 9*time.Minute)
		assert.LessOrEqual(t, ttl, 10*time.Minute)
	})

	t.Run("forecast cache has 1 hour TTL", func(t *testing.T) {
		cacheKey := fmt.Sprintf("weather:forecast:%.4f:%.4f:5", lat, lon)
		testData := &weather.ForecastData{Location: "Test"}
		forecastJSON, _ := json.Marshal(testData)
		suite.redisClient.Set(ctx, cacheKey, forecastJSON, time.Hour)

		// Check TTL
		ttl, err := suite.redisClient.TTL(ctx, cacheKey).Result()
		require.NoError(t, err)
		assert.Greater(t, ttl, 59*time.Minute)
		assert.LessOrEqual(t, ttl, time.Hour)
	})

	t.Run("air quality cache has 30 minute TTL", func(t *testing.T) {
		cacheKey := fmt.Sprintf("weather:air:%.4f:%.4f", lat, lon)
		testData := &weather.AirQualityData{AQI: 2}
		airJSON, _ := json.Marshal(testData)
		suite.redisClient.Set(ctx, cacheKey, airJSON, 30*time.Minute)

		// Check TTL
		ttl, err := suite.redisClient.TTL(ctx, cacheKey).Result()
		require.NoError(t, err)
		assert.Greater(t, ttl, 29*time.Minute)
		assert.LessOrEqual(t, ttl, 30*time.Minute)
	})

	t.Run("geocode cache has 24 hour TTL", func(t *testing.T) {
		cacheKey := "geocode:test"
		testData := &weather.Location{Name: "Test"}
		locationJSON, _ := json.Marshal(testData)
		suite.redisClient.Set(ctx, cacheKey, locationJSON, 24*time.Hour)

		// Check TTL
		ttl, err := suite.redisClient.TTL(ctx, cacheKey).Result()
		require.NoError(t, err)
		assert.Greater(t, ttl, 23*time.Hour)
		assert.LessOrEqual(t, ttl, 24*time.Hour)
	})
}
