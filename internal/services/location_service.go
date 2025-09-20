package services

import (
    "context"
    "fmt"

    "github.com/google/uuid"
    "github.com/redis/go-redis/v9"
    "gorm.io/gorm"

    "github.com/valpere/enterprise-weather-bot/internal/models"
)

type LocationService struct {
    db    *gorm.DB
    redis *redis.Client
}

func NewLocationService(db *gorm.DB, redis *redis.Client) *LocationService {
    return &LocationService{
        db:    db,
        redis: redis,
    }
}

func (s *LocationService) AddLocation(ctx context.Context, userID int64, name string, lat, lon float64) (*models.Location, error) {
    location := &models.Location{
        UserID:    userID,
        Name:      name,
        Latitude:  lat,
        Longitude: lon,
        IsActive:  true,
    }

    // Check if this is the user's first location - make it default
    var count int64
    s.db.WithContext(ctx).Model(&models.Location{}).Where("user_id = ? AND is_active = ?", userID, true).Count(&count)
    if count == 0 {
        location.IsDefault = true
    }

    if err := s.db.WithContext(ctx).Create(location).Error; err != nil {
        return nil, err
    }

    // Invalidate cache
    s.invalidateUserLocationsCache(ctx, userID)

    return location, nil
}

func (s *LocationService) GetUserLocations(ctx context.Context, userID int64) ([]models.Location, error) {
    var locations []models.Location
    err := s.db.WithContext(ctx).
        Where("user_id = ? AND is_active = ?", userID, true).
        Order("is_default DESC, created_at ASC").
        Find(&locations).Error

    return locations, err
}

func (s *LocationService) GetDefaultLocation(ctx context.Context, userID int64) (*models.Location, error) {
    var location models.Location
    err := s.db.WithContext(ctx).
        Where("user_id = ? AND is_default = ? AND is_active = ?", userID, true, true).
        First(&location).Error

    if err == gorm.ErrRecordNotFound {
        return nil, nil
    }

    return &location, err
}

func (s *LocationService) SetDefaultLocation(ctx context.Context, userID int64, locationID uuid.UUID) error {
    // Start transaction
    tx := s.db.WithContext(ctx).Begin()

    // Remove default from all user locations
    if err := tx.Model(&models.Location{}).
        Where("user_id = ?", userID).
        Update("is_default", false).Error; err != nil {
        tx.Rollback()
        return err
    }

    // Set new default
    if err := tx.Model(&models.Location{}).
        Where("id = ? AND user_id = ?", locationID, userID).
        Update("is_default", true).Error; err != nil {
        tx.Rollback()
        return err
    }

    tx.Commit()

    // Invalidate cache
    s.invalidateUserLocationsCache(ctx, userID)

    return nil
}

func (s *LocationService) DeleteLocation(ctx context.Context, userID int64, locationID uuid.UUID) error {
    err := s.db.WithContext(ctx).
        Model(&models.Location{}).
        Where("id = ? AND user_id = ?", locationID, userID).
        Update("is_active", false).Error

    if err != nil {
        return err
    }

    // Invalidate cache
    s.invalidateUserLocationsCache(ctx, userID)

    return nil
}

func (s *LocationService) invalidateUserLocationsCache(ctx context.Context, userID int64) {
    cacheKey := fmt.Sprintf("user:locations:%d", userID)
    s.redis.Del(ctx, cacheKey)
}
