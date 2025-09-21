package services

import (
    "context"
    "time"

    "github.com/google/uuid"
    "github.com/redis/go-redis/v9"
    "gorm.io/gorm"

    "github.com/valpere/shopogoda/internal/models"
)

// Additional UserService methods
func (s *UserService) GetActiveUsers(ctx context.Context) ([]models.User, error) {
    var users []models.User
    err := s.db.WithContext(ctx).
        Where("is_active = ?", true).
        Find(&users).Error
    return users, err
}

type UserStatistics struct {
    TotalUsers        int64 `json:"total_users"`
    ActiveUsers       int64 `json:"active_users"`
    NewUsers24h       int64 `json:"new_users_24h"`
    AdminCount        int64 `json:"admin_count"`
    ModeratorCount    int64 `json:"moderator_count"`
    Messages24h       int64 `json:"messages_24h"`
    WeatherRequests24h int64 `json:"weather_requests_24h"`
    LocationsSaved    int64 `json:"locations_saved"`
    ActiveAlerts      int64 `json:"active_alerts"`
}

func (s *UserService) GetUserStatistics(ctx context.Context) (*UserStatistics, error) {
    stats := &UserStatistics{}

    // Basic user counts
    s.db.WithContext(ctx).Model(&models.User{}).Count(&stats.TotalUsers)
    s.db.WithContext(ctx).Model(&models.User{}).Where("is_active = ?", true).Count(&stats.ActiveUsers)

    yesterday := time.Now().AddDate(0, 0, -1)
    s.db.WithContext(ctx).Model(&models.User{}).Where("created_at > ?", yesterday).Count(&stats.NewUsers24h)

    // Role counts
    s.db.WithContext(ctx).Model(&models.User{}).Where("role = ?", models.RoleAdmin).Count(&stats.AdminCount)
    s.db.WithContext(ctx).Model(&models.User{}).Where("role = ?", models.RoleModerator).Count(&stats.ModeratorCount)

    // Activity counts
    s.db.WithContext(ctx).Model(&models.Location{}).Where("is_active = ?", true).Count(&stats.LocationsSaved)
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

// Additional AlertService methods
func (s *AlertService) DeleteAlert(ctx context.Context, userID int64, alertID uuid.UUID) error {
    return s.db.WithContext(ctx).
        Model(&models.AlertConfig{}).
        Where("id = ? AND user_id = ?", alertID, userID).
        Update("is_active", false).Error
}

func (s *AlertService) UpdateAlert(ctx context.Context, userID int64, alertID uuid.UUID, updates map[string]interface{}) error {
    updates["updated_at"] = time.Now()

    return s.db.WithContext(ctx).
        Model(&models.AlertConfig{}).
        Where("id = ? AND user_id = ?", alertID, userID).
        Updates(updates).Error
}

func (s *AlertService) GetAlert(ctx context.Context, userID int64, alertID uuid.UUID) (*models.AlertConfig, error) {
    var alert models.AlertConfig
    err := s.db.WithContext(ctx).
        Preload("Location").
        Where("id = ? AND user_id = ?", alertID, userID).
        First(&alert).Error

    if err != nil {
        return nil, err
    }
    return &alert, nil
}
