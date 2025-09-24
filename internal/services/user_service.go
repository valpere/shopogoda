package services

import (
	"context"
	"encoding/json"
	"fmt"
	"strconv"
	"time"

	"github.com/PaulSonOfLars/gotgbot/v2"
	"github.com/redis/go-redis/v9"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/valpere/shopogoda/internal/models"
)

// User table column names for upsert operations
var userUpsertColumns = []string{
	"username",
	"first_name",
	"last_name",
	"language",
	"updated_at",
}

type UserService struct {
	db    *gorm.DB
	redis *redis.Client
}

type SystemStats struct {
	TotalUsers          int64   `json:"total_users"`
	ActiveUsers         int64   `json:"active_users"`
	NewUsers24h         int64   `json:"new_users_24h"`
	UsersWithLocation   int64   `json:"users_with_location"`
	ActiveSubscriptions int64   `json:"active_subscriptions"`
	AlertsConfigured    int64   `json:"alerts_configured"`
	MessagesSent24h     int64   `json:"messages_sent_24h"`
	WeatherRequests24h  int64   `json:"weather_requests_24h"`
	CacheHitRate        float64 `json:"cache_hit_rate"`
	AvgResponseTime     int     `json:"avg_response_time"`
	Uptime              float64 `json:"uptime"`
}

func NewUserService(db *gorm.DB, redis *redis.Client) *UserService {
	return &UserService{
		db:    db,
		redis: redis,
	}
}

func (s *UserService) RegisterUser(ctx context.Context, tgUser *gotgbot.User) error {
	user := &models.User{
		ID:        tgUser.Id,
		Username:  tgUser.Username,
		FirstName: tgUser.FirstName,
		LastName:  tgUser.LastName,
		Language:  tgUser.LanguageCode,
		IsActive:  true,
	}

	// Use GORM's upsert functionality with proper conflict resolution
	result := s.db.WithContext(ctx).Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "id"}},
		DoUpdates: clause.AssignmentColumns(userUpsertColumns),
	}).Create(user)

	return result.Error
}

func (s *UserService) GetUser(ctx context.Context, userID int64) (*models.User, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("user:%d", userID)
	cached, err := s.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var user models.User
		if err := json.Unmarshal([]byte(cached), &user); err == nil {
			return &user, nil
		}
	}

	// Get from database
	var user models.User
	if err := s.db.WithContext(ctx).First(&user, userID).Error; err != nil {
		return nil, err
	}

	// Cache for 1 hour
	userJSON, _ := json.Marshal(user)
	s.redis.Set(ctx, cacheKey, userJSON, time.Hour)

	return &user, nil
}

func (s *UserService) UpdateUserSettings(ctx context.Context, userID int64, settings map[string]interface{}) error {
	err := s.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).Updates(settings).Error
	if err != nil {
		return err
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("user:%d", userID)
	s.redis.Del(ctx, cacheKey)

	return nil
}

func (s *UserService) GetSystemStats(ctx context.Context) (*SystemStats, error) {
	stats := &SystemStats{}

	// Get user statistics
	s.db.WithContext(ctx).Model(&models.User{}).Count(&stats.TotalUsers)
	s.db.WithContext(ctx).Model(&models.User{}).Where("is_active = ?", true).Count(&stats.ActiveUsers)

	yesterday := time.Now().UTC().AddDate(0, 0, -1)
	s.db.WithContext(ctx).Model(&models.User{}).Where("created_at > ?", yesterday).Count(&stats.NewUsers24h)

	// Get users with location configured
	s.db.WithContext(ctx).Model(&models.User{}).Where("location_name != '' AND location_name IS NOT NULL").Count(&stats.UsersWithLocation)

	// Get subscription statistics
	s.db.WithContext(ctx).Model(&models.Subscription{}).Where("is_active = ?", true).Count(&stats.ActiveSubscriptions)
	s.db.WithContext(ctx).Model(&models.AlertConfig{}).Where("is_active = ?", true).Count(&stats.AlertsConfigured)

	// Cache hit rate and performance metrics would come from monitoring systems
	stats.CacheHitRate = 85.5   // Placeholder
	stats.AvgResponseTime = 150 // Placeholder
	stats.Uptime = 99.9         // Placeholder

	return stats, nil
}

// GetActiveUsers returns all active users
func (s *UserService) GetActiveUsers(ctx context.Context) ([]models.User, error) {
	var users []models.User
	err := s.db.WithContext(ctx).
		Where("is_active = ?", true).
		Find(&users).Error
	return users, err
}

type UserStatistics struct {
	TotalUsers         int64 `json:"total_users"`
	ActiveUsers        int64 `json:"active_users"`
	NewUsers24h        int64 `json:"new_users_24h"`
	AdminCount         int64 `json:"admin_count"`
	ModeratorCount     int64 `json:"moderator_count"`
	Messages24h        int64 `json:"messages_24h"`
	WeatherRequests24h int64 `json:"weather_requests_24h"`
	LocationsSaved     int64 `json:"locations_saved"`
	ActiveAlerts       int64 `json:"active_alerts"`
}

func (s *UserService) GetUserStatistics(ctx context.Context) (*UserStatistics, error) {
	stats := &UserStatistics{}

	// Basic user counts
	s.db.WithContext(ctx).Model(&models.User{}).Count(&stats.TotalUsers)
	s.db.WithContext(ctx).Model(&models.User{}).Where("is_active = ?", true).Count(&stats.ActiveUsers)

	yesterday := time.Now().UTC().AddDate(0, 0, -1)
	s.db.WithContext(ctx).Model(&models.User{}).Where("created_at > ?", yesterday).Count(&stats.NewUsers24h)

	// Role counts
	s.db.WithContext(ctx).Model(&models.User{}).Where("role = ?", models.RoleAdmin).Count(&stats.AdminCount)
	s.db.WithContext(ctx).Model(&models.User{}).Where("role = ?", models.RoleModerator).Count(&stats.ModeratorCount)

	// Activity counts
	s.db.WithContext(ctx).Model(&models.User{}).Where("location_name != '' AND location_name IS NOT NULL").Count(&stats.LocationsSaved)
	s.db.WithContext(ctx).Model(&models.AlertConfig{}).Where("is_active = ?", true).Count(&stats.ActiveAlerts)

	// Get Redis stats
	if val, err := s.redis.Get(ctx, "stats:messages_24h").Result(); err == nil {
		if count, err := strconv.ParseInt(val, 10, 64); err == nil {
			stats.Messages24h = count
		}
	}

	if val, err := s.redis.Get(ctx, "stats:weather_requests_24h").Result(); err == nil {
		if count, err := strconv.ParseInt(val, 10, 64); err == nil {
			stats.WeatherRequests24h = count
		}
	}

	return stats, nil
}

// SetUserLocation updates the user's location
func (s *UserService) SetUserLocation(ctx context.Context, userID int64, locationName, country, city string, lat, lon float64) error {
	updates := map[string]interface{}{
		"location_name": locationName,
		"latitude":      lat,
		"longitude":     lon,
		"country":       country,
		"city":          city,
	}

	// For now, keep timezone as UTC - could be enhanced later with timezone inference
	// based on coordinates using external timezone API
	if locationName != "" {
		updates["timezone"] = "UTC" // Could be inferred from coordinates in future
	}

	err := s.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).Updates(updates).Error
	if err != nil {
		return err
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("user:%d", userID)
	s.redis.Del(ctx, cacheKey)

	return nil
}

// ClearUserLocation clears the user's location and sets timezone to UTC
func (s *UserService) ClearUserLocation(ctx context.Context, userID int64) error {
	updates := map[string]interface{}{
		"location_name": "",
		"latitude":      0,
		"longitude":     0,
		"country":       "",
		"city":          "",
		"timezone":      "UTC",
	}

	err := s.db.WithContext(ctx).Model(&models.User{}).Where("id = ?", userID).Updates(updates).Error
	if err != nil {
		return err
	}

	// Invalidate cache
	cacheKey := fmt.Sprintf("user:%d", userID)
	s.redis.Del(ctx, cacheKey)

	return nil
}

// GetUserLocation returns the user's location if set
func (s *UserService) GetUserLocation(ctx context.Context, userID int64) (string, float64, float64, error) {
	user, err := s.GetUser(ctx, userID)
	if err != nil {
		return "", 0, 0, err
	}

	if user.LocationName == "" {
		return "", 0, 0, fmt.Errorf("user has no location set")
	}

	return user.LocationName, user.Latitude, user.Longitude, nil
}

// GetUserTimezone returns the user's timezone, defaulting to UTC if location not set
func (s *UserService) GetUserTimezone(ctx context.Context, userID int64) string {
	user, err := s.GetUser(ctx, userID)
	if err != nil || user.LocationName == "" {
		return "UTC"
	}

	if user.Timezone == "" {
		return "UTC"
	}

	return user.Timezone
}

// ConvertToUserTime converts UTC time to user's local time
func (s *UserService) ConvertToUserTime(ctx context.Context, userID int64, utcTime time.Time) time.Time {
	timezone := s.GetUserTimezone(ctx, userID)

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		// Fall back to UTC if timezone is invalid
		return utcTime
	}

	return utcTime.In(loc)
}

// ConvertToUTC converts user's local time to UTC
func (s *UserService) ConvertToUTC(ctx context.Context, userID int64, localTime time.Time) time.Time {
	timezone := s.GetUserTimezone(ctx, userID)

	loc, err := time.LoadLocation(timezone)
	if err != nil {
		// Assume it's already UTC if timezone is invalid
		return localTime
	}

	// Convert to the user's timezone first, then to UTC
	return localTime.In(loc).UTC()
}
