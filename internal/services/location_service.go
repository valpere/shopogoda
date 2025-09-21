package services

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/google/uuid"
    "github.com/redis/go-redis/v9"
    "gorm.io/gorm"

    "github.com/valpere/shopogoda/internal/models"
    "github.com/valpere/shopogoda/pkg/weather"
)

type LocationService struct {
    db       *gorm.DB
    redis    *redis.Client
    geocoder *weather.GeocodingClient
}

func NewLocationService(db *gorm.DB, redis *redis.Client) *LocationService {
    return &LocationService{
        db:       db,
        redis:    redis,
        geocoder: nil, // Would be initialized with weather API key in production
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

// GetLocationByID gets a location by its ID
func (s *LocationService) GetLocationByID(ctx context.Context, locationID uuid.UUID) (*models.Location, error) {
    // Try cache first
    cacheKey := fmt.Sprintf("location:%s", locationID)
    cached, err := s.redis.Get(ctx, cacheKey).Result()
    if err == nil {
        var location models.Location
        if err := json.Unmarshal([]byte(cached), &location); err == nil {
            return &location, nil
        }
    }

    // Get from database
    var location models.Location
    err = s.db.WithContext(ctx).First(&location, "id = ?", locationID).Error
    if err != nil {
        return nil, err
    }

    // Cache for 1 hour
    s.cacheLocation(ctx, &location)

    return &location, nil
}

// SearchLocationByName searches for a location by name using geocoding
func (s *LocationService) SearchLocationByName(ctx context.Context, name string) (*weather.Location, error) {
    // This would use the geocoding service to find location by name
    // For now, return a placeholder implementation
    if s.geocoder != nil {
        return s.geocoder.GeocodeLocation(ctx, name)
    }

    // Fallback mock implementation for common Ukrainian cities
    switch name {
    case "Kyiv", "Kiev", "Київ":
        return &weather.Location{
            Name:      "Kyiv",
            Latitude:  50.4501,
            Longitude: 30.5234,
            Country:   "UA",
        }, nil
    case "Lviv", "Львів":
        return &weather.Location{
            Name:      "Lviv",
            Latitude:  49.8397,
            Longitude: 24.0297,
            Country:   "UA",
        }, nil
    case "Odesa", "Odessa", "Одеса":
        return &weather.Location{
            Name:      "Odesa",
            Latitude:  46.4825,
            Longitude: 30.7233,
            Country:   "UA",
        }, nil
    case "Kharkiv", "Харків":
        return &weather.Location{
            Name:      "Kharkiv",
            Latitude:  49.9935,
            Longitude: 36.2304,
            Country:   "UA",
        }, nil
    default:
        return &weather.Location{
            Name:      name,
            Latitude:  50.4501, // Default to Kyiv coordinates
            Longitude: 30.5234,
            Country:   "UA",
        }, nil
    }
}

// GetActiveLocations returns all active locations for monitoring
func (s *LocationService) GetActiveLocations(ctx context.Context) ([]models.Location, error) {
    var locations []models.Location
    err := s.db.WithContext(ctx).
        Where("is_active = ?", true).
        Find(&locations).Error

    return locations, err
}

// cacheLocation caches a location in Redis
func (s *LocationService) cacheLocation(ctx context.Context, location *models.Location) {
    cacheKey := fmt.Sprintf("location:%s", location.ID)
    locationJSON, _ := json.Marshal(location)
    s.redis.Set(ctx, cacheKey, locationJSON, time.Hour)
}
