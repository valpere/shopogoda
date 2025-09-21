package services

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"

	"github.com/valpere/shopogoda/internal/models"
)

type SubscriptionService struct {
	db    *gorm.DB
	redis *redis.Client
}

func NewSubscriptionService(db *gorm.DB, redis *redis.Client) *SubscriptionService {
	return &SubscriptionService{
		db:    db,
		redis: redis,
	}
}

func (s *SubscriptionService) CreateSubscription(ctx context.Context, userID int64, locationID uuid.UUID, subType models.SubscriptionType, frequency models.Frequency, timeOfDay string) (*models.Subscription, error) {
	subscription := &models.Subscription{
		UserID:           userID,
		LocationID:       locationID,
		SubscriptionType: subType,
		Frequency:        frequency,
		TimeOfDay:        timeOfDay,
		IsActive:         true,
	}

	if err := s.db.WithContext(ctx).Create(subscription).Error; err != nil {
		return nil, err
	}

	return subscription, nil
}

func (s *SubscriptionService) GetUserSubscriptions(ctx context.Context, userID int64) ([]models.Subscription, error) {
	var subscriptions []models.Subscription
	err := s.db.WithContext(ctx).
		Preload("Location").
		Where("user_id = ? AND is_active = ?", userID, true).
		Find(&subscriptions).Error

	return subscriptions, err
}

func (s *SubscriptionService) UpdateSubscription(ctx context.Context, userID int64, subscriptionID uuid.UUID, updates map[string]interface{}) error {
	updates["updated_at"] = time.Now()

	return s.db.WithContext(ctx).
		Model(&models.Subscription{}).
		Where("id = ? AND user_id = ?", subscriptionID, userID).
		Updates(updates).Error
}

func (s *SubscriptionService) DeleteSubscription(ctx context.Context, userID int64, subscriptionID uuid.UUID) error {
	return s.db.WithContext(ctx).
		Model(&models.Subscription{}).
		Where("id = ? AND user_id = ?", subscriptionID, userID).
		Update("is_active", false).Error
}

func (s *SubscriptionService) GetActiveSubscriptions(ctx context.Context) ([]models.Subscription, error) {
	var subscriptions []models.Subscription
	err := s.db.WithContext(ctx).
		Preload("User").
		Preload("Location").
		Where("is_active = ?", true).
		Find(&subscriptions).Error

	return subscriptions, err
}

func (s *SubscriptionService) GetSubscriptionsByType(ctx context.Context, subType models.SubscriptionType) ([]models.Subscription, error) {
	var subscriptions []models.Subscription
	err := s.db.WithContext(ctx).
		Preload("User").
		Preload("Location").
		Where("subscription_type = ? AND is_active = ?", subType, true).
		Find(&subscriptions).Error

	return subscriptions, err
}

func (s *SubscriptionService) ShouldSendNotification(subscription *models.Subscription) bool {
	now := time.Now()

	// Parse time of day
	timeOfDay, err := time.Parse("15:04", subscription.TimeOfDay)
	if err != nil {
		return false
	}

	// Check if it's the right time of day (within 5 minute window)
	currentTime := time.Date(1, 1, 1, now.Hour(), now.Minute(), 0, 0, time.UTC)
	targetTime := time.Date(1, 1, 1, timeOfDay.Hour(), timeOfDay.Minute(), 0, 0, time.UTC)
	timeDiff := currentTime.Sub(targetTime).Minutes()

	if timeDiff < 0 || timeDiff > 5 {
		return false
	}

	// Check frequency
	switch subscription.Frequency {
	case models.FrequencyHourly:
		return true
	case models.FrequencyEvery3Hours:
		return now.Hour()%3 == 0
	case models.FrequencyEvery6Hours:
		return now.Hour()%6 == 0
	case models.FrequencyDaily:
		return true
	case models.FrequencyWeekly:
		return now.Weekday() == time.Monday
	default:
		return false
	}
}