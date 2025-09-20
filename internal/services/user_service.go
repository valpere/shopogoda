package services

import (
    "context"
    "encoding/json"
    "fmt"
    "time"

    "github.com/PaulSonOfLars/gotgbot/v2"
    "github.com/redis/go-redis/v9"
    "gorm.io/gorm"

    "github.com/valpere/enterprise-weather-bot/internal/models"
)

type UserService struct {
    db    *gorm.DB
    redis *redis.Client
}

type SystemStats struct {
    TotalUsers           int     `json:"total_users"`
    ActiveUsers          int     `json:"active_users"`
    NewUsers24h          int     `json:"new_users_24h"`
    TotalLocations       int     `json:"total_locations"`
    ActiveMonitoring     int     `json:"active_monitoring"`
    ActiveSubscriptions  int     `json:"active_subscriptions"`
    AlertsConfigured     int     `json:"alerts_configured"`
    MessagesSent24h      int     `json:"messages_sent_24h"`
    WeatherRequests24h   int     `json:"weather_requests_24h"`
    CacheHitRate         float64 `json:"cache_hit_rate"`
    AvgResponseTime      int     `json:"avg_response_time"`
    Uptime               float64 `json:"uptime"`
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

    // Use GORM's upsert functionality
    result := s.db.WithContext(ctx).Clauses().Create(user)
    if result.Error != nil {
        // If user exists, update the information
        result = s.db.WithContext(ctx).Model(user).Where("id = ?", user.ID).Updates(map[string]interface{}{
            "username":   user.Username,
            "first_name": user.FirstName,
            "last_name":  user.LastName,
            "language":   user.Language,
            "updated_at": time.Now(),
        })
    }

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

    yesterday := time.Now().AddDate(0, 0, -1)
    s.db.WithContext(ctx).Model(&models.User{}).Where("created_at > ?", yesterday).Count(&stats.NewUsers24h)

    // Get location statistics
