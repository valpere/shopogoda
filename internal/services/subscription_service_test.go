package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"

	"github.com/valpere/shopogoda/internal/models"
	"github.com/valpere/shopogoda/tests/helpers"
)

func TestNewSubscriptionService(t *testing.T) {
	mockDB := helpers.NewMockDB(t)
	defer func() { _ = mockDB.Close() }()
	mockRedis := helpers.NewMockRedis()

	service := NewSubscriptionService(mockDB.DB, mockRedis.Client)

	assert.NotNil(t, service)
	assert.NotNil(t, service.db)
	assert.NotNil(t, service.redis)
}

func TestSubscriptionService_CreateSubscription(t *testing.T) {
	mockDB := helpers.NewMockDB(t)
	defer func() { _ = mockDB.Close() }()
	mockRedis := helpers.NewMockRedis()
	service := NewSubscriptionService(mockDB.DB, mockRedis.Client)

	t.Run("successful creation", func(t *testing.T) {
		userID := int64(123)
		subType := models.SubscriptionDaily
		frequency := models.FrequencyDaily
		timeOfDay := "08:00"

		mockDB.Mock.ExpectBegin()
		mockDB.Mock.ExpectQuery(`INSERT INTO "subscriptions"`).
			WithArgs(userID, subType, frequency, timeOfDay, true, helpers.AnyTime{}, helpers.AnyTime{}).
			WillReturnRows(mockDB.Mock.NewRows([]string{"id"}).AddRow(uuid.New()))
		mockDB.Mock.ExpectCommit()

		subscription, err := service.CreateSubscription(context.Background(), userID, subType, frequency, timeOfDay)

		assert.NoError(t, err)
		assert.NotNil(t, subscription)
		assert.Equal(t, userID, subscription.UserID)
		assert.Equal(t, subType, subscription.SubscriptionType)
		assert.Equal(t, frequency, subscription.Frequency)
		assert.Equal(t, timeOfDay, subscription.TimeOfDay)
		assert.True(t, subscription.IsActive)
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("database error", func(t *testing.T) {
		userID := int64(123)

		mockDB.Mock.ExpectBegin()
		mockDB.Mock.ExpectQuery(`INSERT INTO "subscriptions"`).
			WillReturnError(errors.New("database error"))
		mockDB.Mock.ExpectRollback()

		subscription, err := service.CreateSubscription(context.Background(), userID, models.SubscriptionDaily, models.FrequencyDaily, "08:00")

		assert.Error(t, err)
		assert.Nil(t, subscription)
		assert.Contains(t, err.Error(), "database error")
		mockDB.ExpectationsWereMet(t)
	})
}

func TestSubscriptionService_GetUserSubscriptions(t *testing.T) {
	mockDB := helpers.NewMockDB(t)
	defer func() { _ = mockDB.Close() }()
	mockRedis := helpers.NewMockRedis()
	service := NewSubscriptionService(mockDB.DB, mockRedis.Client)

	t.Run("successful retrieval", func(t *testing.T) {
		userID := int64(123)
		sub1 := helpers.MockSubscription(userID)
		sub2 := helpers.MockSubscription(userID)

		rows := mockDB.Mock.NewRows([]string{
			"id", "user_id", "subscription_type", "frequency", "time_of_day", "is_active", "created_at", "updated_at",
		})
		rows.AddRow(sub1.ID, sub1.UserID, sub1.SubscriptionType, sub1.Frequency, sub1.TimeOfDay, sub1.IsActive, sub1.CreatedAt, sub1.UpdatedAt)
		rows.AddRow(sub2.ID, sub2.UserID, sub2.SubscriptionType, sub2.Frequency, sub2.TimeOfDay, sub2.IsActive, sub2.CreatedAt, sub2.UpdatedAt)

		mockDB.Mock.ExpectQuery(`SELECT \* FROM "subscriptions" WHERE user_id = \$1 AND is_active = \$2`).
			WithArgs(userID, true).
			WillReturnRows(rows)

		subscriptions, err := service.GetUserSubscriptions(context.Background(), userID)

		assert.NoError(t, err)
		assert.Len(t, subscriptions, 2)
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("no subscriptions found", func(t *testing.T) {
		userID := int64(999)

		mockDB.Mock.ExpectQuery(`SELECT \* FROM "subscriptions" WHERE user_id = \$1 AND is_active = \$2`).
			WithArgs(userID, true).
			WillReturnRows(mockDB.Mock.NewRows([]string{"id", "user_id", "subscription_type"}))

		subscriptions, err := service.GetUserSubscriptions(context.Background(), userID)

		assert.NoError(t, err)
		assert.Len(t, subscriptions, 0)
		mockDB.ExpectationsWereMet(t)
	})
}

func TestSubscriptionService_UpdateSubscription(t *testing.T) {
	mockDB := helpers.NewMockDB(t)
	defer func() { _ = mockDB.Close() }()
	mockRedis := helpers.NewMockRedis()
	service := NewSubscriptionService(mockDB.DB, mockRedis.Client)

	t.Run("successful update", func(t *testing.T) {
		userID := int64(123)
		subscriptionID := uuid.New()
		updates := map[string]interface{}{
			"time_of_day": "10:00",
			"frequency":   models.FrequencyWeekly,
		}

		mockDB.Mock.ExpectBegin()
		mockDB.Mock.ExpectExec(`UPDATE "subscriptions" SET`).
			WithArgs(models.FrequencyWeekly, "10:00", helpers.AnyTime{}, subscriptionID, userID).
			WillReturnResult(helpers.NewResult(1, 1))
		mockDB.Mock.ExpectCommit()

		err := service.UpdateSubscription(context.Background(), userID, subscriptionID, updates)

		assert.NoError(t, err)
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("subscription not found", func(t *testing.T) {
		userID := int64(123)
		subscriptionID := uuid.New()
		updates := map[string]interface{}{"time_of_day": "10:00"}

		mockDB.Mock.ExpectBegin()
		mockDB.Mock.ExpectExec(`UPDATE "subscriptions" SET`).
			WillReturnResult(helpers.NewResult(0, 0))
		mockDB.Mock.ExpectCommit()

		err := service.UpdateSubscription(context.Background(), userID, subscriptionID, updates)

		assert.NoError(t, err) // GORM doesn't return error for zero rows affected
		mockDB.ExpectationsWereMet(t)
	})
}

func TestSubscriptionService_DeleteSubscription(t *testing.T) {
	mockDB := helpers.NewMockDB(t)
	defer func() { _ = mockDB.Close() }()
	mockRedis := helpers.NewMockRedis()
	service := NewSubscriptionService(mockDB.DB, mockRedis.Client)

	t.Run("successful deletion", func(t *testing.T) {
		userID := int64(123)
		subscriptionID := uuid.New()

		mockDB.Mock.ExpectBegin()
		mockDB.Mock.ExpectExec(`UPDATE "subscriptions" SET "is_active"=\$1,"updated_at"=\$2 WHERE id = \$3 AND user_id = \$4`).
			WithArgs(false, helpers.AnyTime{}, subscriptionID, userID).
			WillReturnResult(helpers.NewResult(1, 1))
		mockDB.Mock.ExpectCommit()

		err := service.DeleteSubscription(context.Background(), userID, subscriptionID)

		assert.NoError(t, err)
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("subscription not found", func(t *testing.T) {
		userID := int64(123)
		subscriptionID := uuid.New()

		mockDB.Mock.ExpectBegin()
		mockDB.Mock.ExpectExec(`UPDATE "subscriptions" SET "is_active"=\$1,"updated_at"=\$2 WHERE id = \$3 AND user_id = \$4`).
			WillReturnResult(helpers.NewResult(0, 0))
		mockDB.Mock.ExpectCommit()

		err := service.DeleteSubscription(context.Background(), userID, subscriptionID)

		assert.NoError(t, err) // GORM doesn't return error for zero rows affected
		mockDB.ExpectationsWereMet(t)
	})
}

func TestSubscriptionService_GetActiveSubscriptions(t *testing.T) {
	mockDB := helpers.NewMockDB(t)
	defer func() { _ = mockDB.Close() }()
	mockRedis := helpers.NewMockRedis()
	service := NewSubscriptionService(mockDB.DB, mockRedis.Client)

	t.Run("successful retrieval", func(t *testing.T) {
		sub1 := helpers.MockSubscription(int64(123))
		sub2 := helpers.MockSubscription(int64(456))

		// Mock the subscriptions query first (executed first by GORM)
		subRows := mockDB.Mock.NewRows([]string{
			"id", "user_id", "subscription_type", "frequency", "time_of_day", "is_active", "created_at", "updated_at",
		})
		subRows.AddRow(sub1.ID, sub1.UserID, sub1.SubscriptionType, sub1.Frequency, sub1.TimeOfDay, sub1.IsActive, sub1.CreatedAt, sub1.UpdatedAt)
		subRows.AddRow(sub2.ID, sub2.UserID, sub2.SubscriptionType, sub2.Frequency, sub2.TimeOfDay, sub2.IsActive, sub2.CreatedAt, sub2.UpdatedAt)

		mockDB.Mock.ExpectQuery(`SELECT \* FROM "subscriptions" WHERE is_active = \$1`).
			WithArgs(true).
			WillReturnRows(subRows)

		// Mock the User preload query (executed second)
		userRows := mockDB.Mock.NewRows([]string{"id", "first_name", "language"}).
			AddRow(int64(123), "User1", "en-US").
			AddRow(int64(456), "User2", "uk-UA")

		mockDB.Mock.ExpectQuery(`SELECT \* FROM "users" WHERE`).
			WillReturnRows(userRows)

		subscriptions, err := service.GetActiveSubscriptions(context.Background())

		assert.NoError(t, err)
		assert.Len(t, subscriptions, 2)
		mockDB.ExpectationsWereMet(t)
	})
}

func TestSubscriptionService_GetSubscriptionsByType(t *testing.T) {
	mockDB := helpers.NewMockDB(t)
	defer func() { _ = mockDB.Close() }()
	mockRedis := helpers.NewMockRedis()
	service := NewSubscriptionService(mockDB.DB, mockRedis.Client)

	t.Run("successful retrieval", func(t *testing.T) {
		subType := models.SubscriptionDaily
		sub1 := helpers.MockSubscription(int64(123))

		// Mock the subscriptions query first (executed first by GORM)
		subRows := mockDB.Mock.NewRows([]string{
			"id", "user_id", "subscription_type", "frequency", "time_of_day", "is_active", "created_at", "updated_at",
		})
		subRows.AddRow(sub1.ID, sub1.UserID, sub1.SubscriptionType, sub1.Frequency, sub1.TimeOfDay, sub1.IsActive, sub1.CreatedAt, sub1.UpdatedAt)

		mockDB.Mock.ExpectQuery(`SELECT \* FROM "subscriptions" WHERE subscription_type = \$1 AND is_active = \$2`).
			WithArgs(subType, true).
			WillReturnRows(subRows)

		// Mock the User preload query (executed second)
		userRows := mockDB.Mock.NewRows([]string{"id", "first_name", "language"}).
			AddRow(int64(123), "User1", "en-US")

		mockDB.Mock.ExpectQuery(`SELECT \* FROM "users" WHERE`).
			WillReturnRows(userRows)

		subscriptions, err := service.GetSubscriptionsByType(context.Background(), subType)

		assert.NoError(t, err)
		assert.Len(t, subscriptions, 1)
		mockDB.ExpectationsWereMet(t)
	})
}

func TestSubscriptionService_ShouldSendNotification(t *testing.T) {
	service := &SubscriptionService{}

	// Helper to create a subscription with a time near now
	createSubscription := func(timeOfDay string, frequency models.Frequency) *models.Subscription {
		return &models.Subscription{
			TimeOfDay: timeOfDay,
			Frequency: frequency,
		}
	}

	// Get current time
	now := time.Now().UTC()
	currentTimeStr := now.Format("15:04")

	t.Run("correct time and frequency - hourly", func(t *testing.T) {
		sub := createSubscription(currentTimeStr, models.FrequencyHourly)
		assert.True(t, service.ShouldSendNotification(sub))
	})

	t.Run("correct time and frequency - daily", func(t *testing.T) {
		sub := createSubscription(currentTimeStr, models.FrequencyDaily)
		assert.True(t, service.ShouldSendNotification(sub))
	})

	t.Run("correct time and frequency - every 3 hours", func(t *testing.T) {
		sub := createSubscription(currentTimeStr, models.FrequencyEvery3Hours)
		result := service.ShouldSendNotification(sub)
		// Should be true only if current hour is divisible by 3
		expected := now.Hour()%3 == 0
		assert.Equal(t, expected, result)
	})

	t.Run("correct time and frequency - every 6 hours", func(t *testing.T) {
		sub := createSubscription(currentTimeStr, models.FrequencyEvery6Hours)
		result := service.ShouldSendNotification(sub)
		// Should be true only if current hour is divisible by 6
		expected := now.Hour()%6 == 0
		assert.Equal(t, expected, result)
	})

	t.Run("correct time and frequency - weekly", func(t *testing.T) {
		sub := createSubscription(currentTimeStr, models.FrequencyWeekly)
		result := service.ShouldSendNotification(sub)
		// Should be true only on Monday
		expected := now.Weekday() == time.Monday
		assert.Equal(t, expected, result)
	})

	t.Run("wrong time - too early", func(t *testing.T) {
		// Set time 10 minutes in the future
		futureTime := now.Add(10 * time.Minute).Format("15:04")
		sub := createSubscription(futureTime, models.FrequencyDaily)
		assert.False(t, service.ShouldSendNotification(sub))
	})

	t.Run("wrong time - too late", func(t *testing.T) {
		// Set time 10 minutes in the past
		pastTime := now.Add(-10 * time.Minute).Format("15:04")
		sub := createSubscription(pastTime, models.FrequencyDaily)
		assert.False(t, service.ShouldSendNotification(sub))
	})

	t.Run("within 5 minute window", func(t *testing.T) {
		// 2 minutes in the past should be within window
		recentTime := now.Add(-2 * time.Minute).Format("15:04")
		sub := createSubscription(recentTime, models.FrequencyDaily)
		assert.True(t, service.ShouldSendNotification(sub))
	})

	t.Run("invalid time format", func(t *testing.T) {
		sub := createSubscription("invalid", models.FrequencyDaily)
		assert.False(t, service.ShouldSendNotification(sub))
	})

	t.Run("unknown frequency", func(t *testing.T) {
		sub := &models.Subscription{
			TimeOfDay: currentTimeStr,
			Frequency: models.Frequency(999), // Invalid frequency value
		}
		assert.False(t, service.ShouldSendNotification(sub))
	})
}
