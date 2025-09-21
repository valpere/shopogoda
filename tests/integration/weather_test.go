//go:build integration

package integration

import (
    "context"
    "testing"
    "time"

    "github.com/stretchr/testify/assert"
    "github.com/stretchr/testify/require"
    "github.com/testcontainers/testcontainers-go"
    "github.com/testcontainers/testcontainers-go/wait"

    "github.com/valpere/shopogoda/internal/config"
    "github.com/valpere/shopogoda/internal/database"
    "github.com/valpere/shopogoda/internal/services"
)

func TestWeatherServiceIntegration(t *testing.T) {
    ctx := context.Background()

    // Start Redis container for testing
    redisContainer, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
        ContainerRequest: testcontainers.ContainerRequest{
            Image:        "redis:7-alpine",
            ExposedPorts: []string{"6379/tcp"},
            WaitingFor:   wait.ForLog("Ready to accept connections"),
        },
        Started: true,
    })
    require.NoError(t, err)
    defer redisContainer.Terminate(ctx)

    // Get Redis connection details
    redisHost, err := redisContainer.Host(ctx)
    require.NoError(t, err)
    redisPort, err := redisContainer.MappedPort(ctx, "6379")
    require.NoError(t, err)

    // Setup Redis client
    redisConfig := &config.RedisConfig{
        Host: redisHost,
        Port: redisPort.Int(),
        DB:   0,
    }

    rdb, err := database.ConnectRedis(redisConfig)
    require.NoError(t, err)

    // Setup weather service
    weatherConfig := &config.WeatherConfig{
        OpenWeatherAPIKey: "test_api_key", // Use a test API key
    }

    weatherService := services.NewWeatherService(weatherConfig, rdb)

    // Test getting coordinates (this will use real API)
    t.Run("GetCoordinates", func(t *testing.T) {
        if testing.Short() {
            t.Skip("Skipping API test in short mode")
        }

        coords, err := weatherService.GetCoordinates(ctx, "London")
        assert.NoError(t, err)
        assert.NotNil(t, coords)
        assert.InDelta(t, 51.5074, coords.Lat, 0.1)
        assert.InDelta(t, -0.1278, coords.Lon, 0.1)
    })

    t.Run("CacheCoordinates", func(t *testing.T) {
        // This test verifies caching works
        location := "TestCity"

        // First call should cache
        _, err := weatherService.GetCoordinates(ctx, location)
        if err != nil {
            t.Skip("Skipping cache test due to API error")
        }

        // Second call should be faster (cached)
        start := time.Now()
        _, err = weatherService.GetCoordinates(ctx, location)
        duration := time.Since(start)

        assert.NoError(t, err)
        assert.Less(t, duration, 100*time.Millisecond, "Cached call should be fast")
    })
}
