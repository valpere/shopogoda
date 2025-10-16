//go:build integration
// +build integration

package integration

import (
	"context"
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
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

type UserServiceTestSuite struct {
	db             *gorm.DB
	redisClient    *redis.Client
	pgContainer    testcontainers.Container
	redisContainer testcontainers.Container
	userService    *services.UserService
}

func setupUserServiceTest(t *testing.T) *UserServiceTestSuite {
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

	// Create user service
	userService := services.NewUserService(db, redisClient)

	return &UserServiceTestSuite{
		db:             db,
		redisClient:    redisClient,
		pgContainer:    pgContainer,
		redisContainer: redisContainer,
		userService:    userService,
	}
}

func (suite *UserServiceTestSuite) teardown(t *testing.T) {
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

func TestIntegration_UserServiceRegisterUser(t *testing.T) {
	suite := setupUserServiceTest(t)
	defer suite.teardown(t)

	ctx := context.Background()

	t.Run("register new user successfully", func(t *testing.T) {
		tgUser := &gotgbot.User{
			Id:           12345678,
			Username:     "testuser",
			FirstName:    "Test",
			LastName:     "User",
			LanguageCode: "en",
		}

		err := suite.userService.RegisterUser(ctx, tgUser)
		require.NoError(t, err)

		// Verify user was created in database
		var user models.User
		err = suite.db.Where("id = ?", tgUser.Id).First(&user).Error
		require.NoError(t, err)
		assert.Equal(t, tgUser.Id, user.ID)
		assert.Equal(t, tgUser.Username, user.Username)
		assert.Equal(t, tgUser.FirstName, user.FirstName)
		assert.Equal(t, tgUser.LastName, user.LastName)
		assert.Equal(t, tgUser.LanguageCode, user.Language)
		assert.True(t, user.IsActive)
	})

	t.Run("register user updates existing user", func(t *testing.T) {
		// Create initial user
		tgUser := &gotgbot.User{
			Id:           87654321,
			Username:     "oldusername",
			FirstName:    "Old",
			LastName:     "Name",
			LanguageCode: "uk",
		}

		err := suite.userService.RegisterUser(ctx, tgUser)
		require.NoError(t, err)

		// Update user with new data
		tgUser.Username = "newusername"
		tgUser.FirstName = "New"
		tgUser.LastName = "Name"

		err = suite.userService.RegisterUser(ctx, tgUser)
		require.NoError(t, err)

		// Verify user was updated
		var user models.User
		err = suite.db.Where("id = ?", tgUser.Id).First(&user).Error
		require.NoError(t, err)
		assert.Equal(t, "newusername", user.Username)
		assert.Equal(t, "New", user.FirstName)
		assert.Equal(t, "Name", user.LastName)

		// Verify only one user record exists
		var count int64
		err = suite.db.Model(&models.User{}).Where("id = ?", tgUser.Id).Count(&count).Error
		require.NoError(t, err)
		assert.Equal(t, int64(1), count)
	})
}

func TestIntegration_UserServiceGetUser(t *testing.T) {
	suite := setupUserServiceTest(t)
	defer suite.teardown(t)

	ctx := context.Background()

	t.Run("get user from database and cache", func(t *testing.T) {
		// Create test user
		userID := int64(11111111)
		user := &models.User{
			ID:        userID,
			Username:  "cachetest",
			FirstName: "Cache",
			LastName:  "Test",
			Language:  "en-US",
			IsActive:  true,
		}
		require.NoError(t, suite.db.Create(user).Error)

		// First call - should get from database and cache
		retrieved, err := suite.userService.GetUser(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, userID, retrieved.ID)
		assert.Equal(t, "cachetest", retrieved.Username)

		// Verify cached in Redis
		cacheKey := fmt.Sprintf("user:%d", userID)
		cached, err := suite.redisClient.Get(ctx, cacheKey).Result()
		require.NoError(t, err)
		assert.NotEmpty(t, cached)

		// Second call - should get from cache
		retrieved2, err := suite.userService.GetUser(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, userID, retrieved2.ID)
		assert.Equal(t, "cachetest", retrieved2.Username)
	})

	t.Run("get non-existent user returns error", func(t *testing.T) {
		_, err := suite.userService.GetUser(ctx, 99999999)
		assert.Error(t, err)
	})
}

func TestIntegration_UserServiceUpdateUserSettings(t *testing.T) {
	suite := setupUserServiceTest(t)
	defer suite.teardown(t)

	ctx := context.Background()

	t.Run("update user settings successfully", func(t *testing.T) {
		// Create test user
		userID := int64(22222222)
		user := &models.User{
			ID:        userID,
			Username:  "settingstest",
			FirstName: "Settings",
			Language:  "en-US",
			IsActive:  true,
		}
		require.NoError(t, suite.db.Create(user).Error)

		// Update settings
		settings := map[string]interface{}{
			"language": "uk",
			"timezone": "Europe/Kyiv",
			"units":    "metric",
		}

		err := suite.userService.UpdateUserSettings(ctx, userID, settings)
		require.NoError(t, err)

		// Verify updates
		var updated models.User
		err = suite.db.Where("id = ?", userID).First(&updated).Error
		require.NoError(t, err)
		assert.Equal(t, "uk", updated.Language)
		assert.Equal(t, "Europe/Kyiv", updated.Timezone)
		assert.Equal(t, "metric", updated.Units)
	})
}

func TestIntegration_UserServiceLocationManagement(t *testing.T) {
	suite := setupUserServiceTest(t)
	defer suite.teardown(t)

	ctx := context.Background()

	t.Run("set and get user location", func(t *testing.T) {
		// Create test user
		userID := int64(33333333)
		user := &models.User{
			ID:        userID,
			Username:  "locationtest",
			FirstName: "Location",
			IsActive:  true,
		}
		require.NoError(t, suite.db.Create(user).Error)

		// Set location
		err := suite.userService.SetUserLocation(ctx, userID, "Kyiv, Ukraine", "Ukraine", "Kyiv", 50.4501, 30.5234)
		require.NoError(t, err)

		// Get location
		locationName, lat, lon, err := suite.userService.GetUserLocation(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, "Kyiv, Ukraine", locationName)
		assert.InDelta(t, 50.4501, lat, 0.0001)
		assert.InDelta(t, 30.5234, lon, 0.0001)

		// Verify in database
		var updated models.User
		err = suite.db.Where("id = ?", userID).First(&updated).Error
		require.NoError(t, err)
		assert.Equal(t, "Kyiv, Ukraine", updated.LocationName)
		assert.Equal(t, "Ukraine", updated.Country)
		assert.Equal(t, "Kyiv", updated.City)
		assert.InDelta(t, 50.4501, updated.Latitude, 0.0001)
		assert.InDelta(t, 30.5234, updated.Longitude, 0.0001)
	})

	t.Run("clear user location", func(t *testing.T) {
		// Create test user with location
		userID := int64(44444444)
		user := &models.User{
			ID:           userID,
			Username:     "cleartest",
			FirstName:    "Clear",
			LocationName: "Test City",
			Country:      "Test Country",
			City:         "Test City",
			Latitude:     40.0,
			Longitude:    30.0,
			IsActive:     true,
		}
		require.NoError(t, suite.db.Create(user).Error)

		// Clear location
		err := suite.userService.ClearUserLocation(ctx, userID)
		require.NoError(t, err)

		// Verify location cleared
		var updated models.User
		err = suite.db.Where("id = ?", userID).First(&updated).Error
		require.NoError(t, err)
		assert.Empty(t, updated.LocationName)
		assert.Empty(t, updated.Country)
		assert.Empty(t, updated.City)
		assert.Equal(t, 0.0, updated.Latitude)
		assert.Equal(t, 0.0, updated.Longitude)
	})

	t.Run("get location for user without location", func(t *testing.T) {
		// Create test user without location
		userID := int64(55555555)
		user := &models.User{
			ID:        userID,
			Username:  "nolocation",
			FirstName: "NoLoc",
			IsActive:  true,
		}
		require.NoError(t, suite.db.Create(user).Error)

		// Get location should return error
		_, _, _, err := suite.userService.GetUserLocation(ctx, userID)
		assert.Error(t, err)
	})
}

func TestIntegration_UserServiceTimezoneManagement(t *testing.T) {
	suite := setupUserServiceTest(t)
	defer suite.teardown(t)

	ctx := context.Background()

	t.Run("get user timezone defaults to UTC", func(t *testing.T) {
		// Create test user without timezone
		userID := int64(66666666)
		user := &models.User{
			ID:        userID,
			Username:  "tztest",
			FirstName: "TZ",
			IsActive:  true,
		}
		require.NoError(t, suite.db.Create(user).Error)

		// Get timezone should return UTC
		tz := suite.userService.GetUserTimezone(ctx, userID)
		assert.Equal(t, "UTC", tz)
	})

	t.Run("get user timezone with set timezone", func(t *testing.T) {
		// Create test user with timezone
		userID := int64(77777777)
		user := &models.User{
			ID:        userID,
			Username:  "tzset",
			FirstName: "TZSet",
			Timezone:  "Europe/Kyiv",
			IsActive:  true,
		}
		require.NoError(t, suite.db.Create(user).Error)

		// Get timezone
		tz := suite.userService.GetUserTimezone(ctx, userID)
		assert.Equal(t, "Europe/Kyiv", tz)
	})

	t.Run("convert time to user timezone", func(t *testing.T) {
		// Create test user with timezone
		userID := int64(88888888)
		user := &models.User{
			ID:        userID,
			Username:  "tzconvert",
			FirstName: "TZConvert",
			Timezone:  "America/New_York",
			IsActive:  true,
		}
		require.NoError(t, suite.db.Create(user).Error)

		// Convert UTC time to user timezone
		utcTime := time.Date(2025, 1, 1, 12, 0, 0, 0, time.UTC)
		userTime := suite.userService.ConvertToUserTime(ctx, userID, utcTime)

		// Verify timezone conversion (EST/EDT is UTC-5 or UTC-4)
		assert.NotEqual(t, utcTime, userTime)
		assert.Equal(t, "America/New_York", userTime.Location().String())
	})

	t.Run("convert user time to UTC", func(t *testing.T) {
		// Create test user with timezone
		userID := int64(99999999)
		user := &models.User{
			ID:        userID,
			Username:  "tzconvertutc",
			FirstName: "TZConvertUTC",
			Timezone:  "Europe/London",
			IsActive:  true,
		}
		require.NoError(t, suite.db.Create(user).Error)

		// Create time in user timezone
		loc, err := time.LoadLocation("Europe/London")
		require.NoError(t, err)
		localTime := time.Date(2025, 1, 1, 12, 0, 0, 0, loc)

		// Convert to UTC
		utcTime := suite.userService.ConvertToUTC(ctx, userID, localTime)

		// Verify UTC timezone
		assert.Equal(t, "UTC", utcTime.Location().String())
	})
}

func TestIntegration_UserServiceGetActiveUsers(t *testing.T) {
	suite := setupUserServiceTest(t)
	defer suite.teardown(t)

	ctx := context.Background()

	t.Run("get only active users", func(t *testing.T) {
		// Create active user
		activeUser := &models.User{
			ID:        111111,
			Username:  "active1",
			FirstName: "Active",
			IsActive:  true,
		}
		require.NoError(t, suite.db.Create(activeUser).Error)

		// Create inactive user
		inactiveUser := &models.User{
			ID:        222222,
			Username:  "inactive1",
			FirstName: "Inactive",
			IsActive:  false,
		}
		require.NoError(t, suite.db.Create(inactiveUser).Error)

		// Get active users
		activeUsers, err := suite.userService.GetActiveUsers(ctx)
		require.NoError(t, err)

		// Verify only active user returned
		assert.GreaterOrEqual(t, len(activeUsers), 1)

		// Verify all returned users are active
		for _, user := range activeUsers {
			assert.True(t, user.IsActive)
		}
	})
}

func TestIntegration_UserServiceCacheInvalidation(t *testing.T) {
	suite := setupUserServiceTest(t)
	defer suite.teardown(t)

	ctx := context.Background()

	t.Run("cache invalidated after settings update", func(t *testing.T) {
		// Create test user
		userID := int64(13579)
		user := &models.User{
			ID:        userID,
			Username:  "cacheinvalidate",
			FirstName: "CacheTest",
			Language:  "en-US",
			IsActive:  true,
		}
		require.NoError(t, suite.db.Create(user).Error)

		// Get user to populate cache
		retrieved, err := suite.userService.GetUser(ctx, userID)
		require.NoError(t, err)
		assert.Equal(t, "en", retrieved.Language)

		// Update settings
		settings := map[string]interface{}{
			"language": "uk",
		}
		err = suite.userService.UpdateUserSettings(ctx, userID, settings)
		require.NoError(t, err)

		// Verify cache was invalidated by checking Redis directly
		cacheKey := fmt.Sprintf("user:%d", userID)
		cached, err := suite.redisClient.Get(ctx, cacheKey).Result()
		if err == nil {
			// If cached, verify it has updated data
			var cachedUser models.User
			err = json.Unmarshal([]byte(cached), &cachedUser)
			require.NoError(t, err)
			assert.Equal(t, "uk", cachedUser.Language)
		}
	})
}
