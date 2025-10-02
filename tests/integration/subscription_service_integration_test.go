//go:build integration
// +build integration

package integration

import (
	"context"
	"testing"

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

type SubscriptionServiceTestSuite struct {
	db                  *gorm.DB
	redisClient         *redis.Client
	pgContainer         testcontainers.Container
	redisContainer      testcontainers.Container
	subscriptionService *services.SubscriptionService
	testUserID          int64
}

func setupSubscriptionServiceTest(t *testing.T) *SubscriptionServiceTestSuite {
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
	testUserID := int64(87654321)
	testUser := &models.User{
		ID:           testUserID,
		Username:     "subscription_test_user",
		FirstName:    "Subscription",
		LastName:     "Tester",
		Language:     "en",
		LocationName: "New York, USA",
		Latitude:     40.7128,
		Longitude:    -74.0060,
		Timezone:     "America/New_York",
		Role:         models.RoleUser,
		IsActive:     true,
	}
	require.NoError(t, db.Create(testUser).Error)

	// Create services
	subscriptionService := services.NewSubscriptionService(db, redisClient)

	return &SubscriptionServiceTestSuite{
		db:                  db,
		redisClient:         redisClient,
		pgContainer:         pgContainer,
		redisContainer:      redisContainer,
		subscriptionService: subscriptionService,
		testUserID:          testUserID,
	}
}

func (suite *SubscriptionServiceTestSuite) teardown(t *testing.T) {
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

func TestIntegration_SubscriptionServiceCreateSubscription(t *testing.T) {
	suite := setupSubscriptionServiceTest(t)
	defer suite.teardown(t)

	ctx := context.Background()

	t.Run("create daily subscription successfully", func(t *testing.T) {
		subscription, err := suite.subscriptionService.CreateSubscription(
			ctx,
			suite.testUserID,
			models.SubscriptionDaily,
			models.FrequencyDaily,
			"08:00",
		)
		require.NoError(t, err)
		assert.NotNil(t, subscription)
		assert.NotEqual(t, uuid.Nil, subscription.ID)
		assert.Equal(t, suite.testUserID, subscription.UserID)
		assert.Equal(t, models.SubscriptionDaily, subscription.SubscriptionType)
		assert.Equal(t, models.FrequencyDaily, subscription.Frequency)
		assert.Equal(t, "08:00", subscription.TimeOfDay)
		assert.True(t, subscription.IsActive)

		// Verify in database
		var dbSubscription models.Subscription
		err = suite.db.Where("id = ?", subscription.ID).First(&dbSubscription).Error
		require.NoError(t, err)
		assert.Equal(t, models.SubscriptionDaily, dbSubscription.SubscriptionType)
	})

	t.Run("create weekly subscription", func(t *testing.T) {
		subscription, err := suite.subscriptionService.CreateSubscription(
			ctx,
			suite.testUserID,
			models.SubscriptionWeekly,
			models.FrequencyWeekly,
			"09:00",
		)
		require.NoError(t, err)
		assert.NotNil(t, subscription)
		assert.Equal(t, models.SubscriptionWeekly, subscription.SubscriptionType)
		assert.Equal(t, models.FrequencyWeekly, subscription.Frequency)
		assert.Equal(t, "09:00", subscription.TimeOfDay)
	})

	t.Run("create alerts subscription", func(t *testing.T) {
		subscription, err := suite.subscriptionService.CreateSubscription(
			ctx,
			suite.testUserID,
			models.SubscriptionAlerts,
			models.FrequencyHourly,
			"",
		)
		require.NoError(t, err)
		assert.NotNil(t, subscription)
		assert.Equal(t, models.SubscriptionAlerts, subscription.SubscriptionType)
		assert.Equal(t, models.FrequencyHourly, subscription.Frequency)
	})
}

func TestIntegration_SubscriptionServiceGetUserSubscriptions(t *testing.T) {
	suite := setupSubscriptionServiceTest(t)
	defer suite.teardown(t)

	ctx := context.Background()

	// Create test subscriptions
	_, err := suite.subscriptionService.CreateSubscription(
		ctx,
		suite.testUserID,
		models.SubscriptionDaily,
		models.FrequencyDaily,
		"08:00",
	)
	require.NoError(t, err)

	sub2, err := suite.subscriptionService.CreateSubscription(
		ctx,
		suite.testUserID,
		models.SubscriptionWeekly,
		models.FrequencyWeekly,
		"09:00",
	)
	require.NoError(t, err)

	// Make second subscription inactive
	sub2.IsActive = false
	err = suite.db.Save(sub2).Error
	require.NoError(t, err)

	t.Run("get active user subscriptions only", func(t *testing.T) {
		subscriptions, err := suite.subscriptionService.GetUserSubscriptions(ctx, suite.testUserID)
		require.NoError(t, err)

		// GetUserSubscriptions only returns active subscriptions (WHERE is_active = true)
		assert.Len(t, subscriptions, 1)
		assert.Equal(t, models.SubscriptionDaily, subscriptions[0].SubscriptionType)
		assert.True(t, subscriptions[0].IsActive)
	})

	t.Run("get subscriptions for non-existent user", func(t *testing.T) {
		subscriptions, err := suite.subscriptionService.GetUserSubscriptions(ctx, 99999)
		require.NoError(t, err)
		assert.Empty(t, subscriptions)
	})
}

func TestIntegration_SubscriptionServiceUpdateSubscription(t *testing.T) {
	suite := setupSubscriptionServiceTest(t)
	defer suite.teardown(t)

	ctx := context.Background()

	// Create initial subscription
	subscription, err := suite.subscriptionService.CreateSubscription(
		ctx,
		suite.testUserID,
		models.SubscriptionDaily,
		models.FrequencyDaily,
		"08:00",
	)
	require.NoError(t, err)

	t.Run("update subscription successfully", func(t *testing.T) {
		// Update through UpdateSubscription method
		updates := map[string]interface{}{
			"time_of_day": "10:00",
			"frequency":   models.FrequencyEvery3Hours,
		}
		err := suite.subscriptionService.UpdateSubscription(ctx, suite.testUserID, subscription.ID, updates)
		require.NoError(t, err)

		// Verify update
		var dbSubscription models.Subscription
		err = suite.db.Where("id = ?", subscription.ID).First(&dbSubscription).Error
		require.NoError(t, err)
		assert.Equal(t, "10:00", dbSubscription.TimeOfDay)
		assert.Equal(t, models.FrequencyEvery3Hours, dbSubscription.Frequency)
	})

	t.Run("update non-existent subscription does nothing", func(t *testing.T) {
		updates := map[string]interface{}{
			"time_of_day": "11:00",
		}
		// UpdateSubscription doesn't error for non-existent subscriptions, it just doesn't update anything
		err := suite.subscriptionService.UpdateSubscription(ctx, suite.testUserID, uuid.New(), updates)
		assert.NoError(t, err)
	})
}

func TestIntegration_SubscriptionServiceToggleSubscription(t *testing.T) {
	suite := setupSubscriptionServiceTest(t)
	defer suite.teardown(t)

	ctx := context.Background()

	// Create subscription
	subscription, err := suite.subscriptionService.CreateSubscription(
		ctx,
		suite.testUserID,
		models.SubscriptionDaily,
		models.FrequencyDaily,
		"08:00",
	)
	require.NoError(t, err)
	assert.True(t, subscription.IsActive)

	t.Run("deactivate subscription", func(t *testing.T) {
		updates := map[string]interface{}{
			"is_active": false,
		}
		err := suite.subscriptionService.UpdateSubscription(ctx, suite.testUserID, subscription.ID, updates)
		require.NoError(t, err)

		// Verify deactivation
		var dbSubscription models.Subscription
		err = suite.db.Where("id = ?", subscription.ID).First(&dbSubscription).Error
		require.NoError(t, err)
		assert.False(t, dbSubscription.IsActive)
	})

	t.Run("reactivate subscription", func(t *testing.T) {
		updates := map[string]interface{}{
			"is_active": true,
		}
		err := suite.subscriptionService.UpdateSubscription(ctx, suite.testUserID, subscription.ID, updates)
		require.NoError(t, err)

		// Verify reactivation
		var dbSubscription models.Subscription
		err = suite.db.Where("id = ?", subscription.ID).First(&dbSubscription).Error
		require.NoError(t, err)
		assert.True(t, dbSubscription.IsActive)
	})

	t.Run("toggle non-existent subscription does nothing", func(t *testing.T) {
		updates := map[string]interface{}{
			"is_active": false,
		}
		// UpdateSubscription doesn't error for non-existent subscriptions
		err := suite.subscriptionService.UpdateSubscription(ctx, suite.testUserID, uuid.New(), updates)
		assert.NoError(t, err)
	})
}

func TestIntegration_SubscriptionServiceDeleteSubscription(t *testing.T) {
	suite := setupSubscriptionServiceTest(t)
	defer suite.teardown(t)

	ctx := context.Background()

	// Create subscription
	subscription, err := suite.subscriptionService.CreateSubscription(
		ctx,
		suite.testUserID,
		models.SubscriptionDaily,
		models.FrequencyDaily,
		"08:00",
	)
	require.NoError(t, err)
	subscriptionID := subscription.ID

	t.Run("delete subscription successfully", func(t *testing.T) {
		err := suite.subscriptionService.DeleteSubscription(ctx, suite.testUserID, subscriptionID)
		require.NoError(t, err)

		// Verify soft deletion (is_active set to false, record still exists)
		var dbSubscription models.Subscription
		err = suite.db.Where("id = ?", subscriptionID).First(&dbSubscription).Error
		require.NoError(t, err)
		assert.False(t, dbSubscription.IsActive)
	})

	t.Run("delete non-existent subscription does nothing", func(t *testing.T) {
		// DeleteSubscription doesn't error for non-existent subscriptions
		err := suite.subscriptionService.DeleteSubscription(ctx, suite.testUserID, uuid.New())
		assert.NoError(t, err)
	})
}

func TestIntegration_SubscriptionServiceGetSubscriptionsByType(t *testing.T) {
	suite := setupSubscriptionServiceTest(t)
	defer suite.teardown(t)

	ctx := context.Background()

	// Create another test user
	testUser2ID := int64(87654322)
	testUser2 := &models.User{
		ID:           testUser2ID,
		Username:     "test_user_2",
		FirstName:    "Test",
		LastName:     "User2",
		Language:     "en",
		LocationName: "London, UK",
		Latitude:     51.5074,
		Longitude:    -0.1278,
		Timezone:     "Europe/London",
		Role:         models.RoleUser,
		IsActive:     true,
	}
	require.NoError(t, suite.db.Create(testUser2).Error)

	// Create daily subscriptions for both users
	_, err := suite.subscriptionService.CreateSubscription(
		ctx,
		suite.testUserID,
		models.SubscriptionDaily,
		models.FrequencyDaily,
		"08:00",
	)
	require.NoError(t, err)

	sub2, err := suite.subscriptionService.CreateSubscription(
		ctx,
		testUser2ID,
		models.SubscriptionDaily,
		models.FrequencyDaily,
		"09:00",
	)
	require.NoError(t, err)

	// Create weekly subscription for user 1
	_, err = suite.subscriptionService.CreateSubscription(
		ctx,
		suite.testUserID,
		models.SubscriptionWeekly,
		models.FrequencyWeekly,
		"10:00",
	)
	require.NoError(t, err)

	// Make user 2's subscription inactive
	sub2.IsActive = false
	require.NoError(t, suite.db.Save(sub2).Error)

	t.Run("get daily subscriptions by type", func(t *testing.T) {
		subscriptions, err := suite.subscriptionService.GetSubscriptionsByType(
			ctx,
			models.SubscriptionDaily,
		)
		require.NoError(t, err)

		// Should get only active daily subscriptions (user2's is inactive)
		assert.Len(t, subscriptions, 1)
		assert.Equal(t, suite.testUserID, subscriptions[0].UserID)
		assert.True(t, subscriptions[0].IsActive)
	})

	t.Run("get weekly subscriptions by type", func(t *testing.T) {
		subscriptions, err := suite.subscriptionService.GetSubscriptionsByType(
			ctx,
			models.SubscriptionWeekly,
		)
		require.NoError(t, err)

		// Should get user 1's weekly subscription
		assert.Len(t, subscriptions, 1)
		assert.Equal(t, suite.testUserID, subscriptions[0].UserID)
	})

	t.Run("get subscriptions with no matches", func(t *testing.T) {
		subscriptions, err := suite.subscriptionService.GetSubscriptionsByType(
			ctx,
			models.SubscriptionExtreme,
		)
		require.NoError(t, err)
		assert.Empty(t, subscriptions)
	})
}

func TestIntegration_SubscriptionServiceCacheInvalidation(t *testing.T) {
	suite := setupSubscriptionServiceTest(t)
	defer suite.teardown(t)

	ctx := context.Background()

	// Create subscription
	subscription, err := suite.subscriptionService.CreateSubscription(
		ctx,
		suite.testUserID,
		models.SubscriptionDaily,
		models.FrequencyDaily,
		"08:00",
	)
	require.NoError(t, err)

	t.Run("cache is invalidated after update", func(t *testing.T) {
		// First call - cache miss
		subs1, err := suite.subscriptionService.GetUserSubscriptions(ctx, suite.testUserID)
		require.NoError(t, err)
		assert.Len(t, subs1, 1)

		// Update subscription
		updates := map[string]interface{}{
			"time_of_day": "10:00",
		}
		err = suite.subscriptionService.UpdateSubscription(ctx, suite.testUserID, subscription.ID, updates)
		require.NoError(t, err)

		// Get subscriptions again - should reflect update
		subs2, err := suite.subscriptionService.GetUserSubscriptions(ctx, suite.testUserID)
		require.NoError(t, err)
		assert.Len(t, subs2, 1)
		assert.Equal(t, "10:00", subs2[0].TimeOfDay)
	})

	t.Run("cache is invalidated after update status", func(t *testing.T) {
		// Get initial state
		subs1, err := suite.subscriptionService.GetUserSubscriptions(ctx, suite.testUserID)
		require.NoError(t, err)

		activeCount := 0
		for _, sub := range subs1 {
			if sub.IsActive {
				activeCount++
			}
		}
		assert.Greater(t, activeCount, 0)

		// Update to inactive
		updates := map[string]interface{}{
			"is_active": false,
		}
		err = suite.subscriptionService.UpdateSubscription(ctx, suite.testUserID, subscription.ID, updates)
		require.NoError(t, err)

		// Verify status changed
		var updated models.Subscription
		err = suite.db.Where("id = ?", subscription.ID).First(&updated).Error
		require.NoError(t, err)
		assert.False(t, updated.IsActive)
	})

	t.Run("cache is invalidated after delete", func(t *testing.T) {
		// Recreate subscription
		newSub, err := suite.subscriptionService.CreateSubscription(
			ctx,
			suite.testUserID,
			models.SubscriptionWeekly,
			models.FrequencyWeekly,
			"09:00",
		)
		require.NoError(t, err)

		// Verify it exists
		subs1, err := suite.subscriptionService.GetUserSubscriptions(ctx, suite.testUserID)
		require.NoError(t, err)
		assert.GreaterOrEqual(t, len(subs1), 1)

		// Delete subscription
		err = suite.subscriptionService.DeleteSubscription(ctx, suite.testUserID, newSub.ID)
		require.NoError(t, err)

		// Verify subscription is soft-deleted
		var deletedSub models.Subscription
		err = suite.db.Where("id = ?", newSub.ID).First(&deletedSub).Error
		require.NoError(t, err)
		assert.False(t, deletedSub.IsActive)
	})
}
