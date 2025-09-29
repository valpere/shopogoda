package helpers

import (
	"database/sql/driver"
	"fmt"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"

	"github.com/valpere/shopogoda/internal/models"
)

// MockDB represents a mocked database connection for testing
type MockDB struct {
	DB   *gorm.DB
	Mock sqlmock.Sqlmock
}

// NewMockDB creates a new mock database connection
func NewMockDB(t *testing.T) *MockDB {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)

	gormDB, err := gorm.Open(postgres.New(postgres.Config{
		Conn: db,
	}), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
		NowFunc: func() time.Time {
			return time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC)
		},
	})
	require.NoError(t, err)

	return &MockDB{
		DB:   gormDB,
		Mock: mock,
	}
}

// Close closes the mock database connection
func (m *MockDB) Close() error {
	sqlDB, err := m.DB.DB()
	if err != nil {
		return err
	}
	return sqlDB.Close()
}

// ExpectationsWereMet checks if all expected database interactions were met
func (m *MockDB) ExpectationsWereMet(t *testing.T) {
	require.NoError(t, m.Mock.ExpectationsWereMet())
}

// MockUser creates a mock user for testing
func MockUser(userID int64) *models.User {
	return &models.User{
		ID:           userID,
		FirstName:    "Test",
		LastName:     "User",
		Username:     "testuser",
		LanguageCode: "en",
		LocationName: "Test City",
		Latitude:     51.5074,
		Longitude:    -0.1278,
		Country:      "UK",
		City:         "London",
		Timezone:     "Europe/London",
		Role:         models.UserRole,
		CreatedAt:    time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:    time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

// MockWeatherData creates mock weather data for testing
func MockWeatherData(userID int64) *models.WeatherData {
	return &models.WeatherData{
		UserID:       userID,
		LocationName: "Test City",
		Temperature:  20.5,
		FeelsLike:    22.0,
		Humidity:     65,
		Pressure:     1013,
		Visibility:   10,
		UVIndex:      3,
		WindSpeed:    5.2,
		WindDirection: 180,
		Description:  "Clear sky",
		AQI:          2,
		CO:           0.3,
		NO:           0.1,
		NO2:          15.2,
		O3:           45.3,
		SO2:          2.1,
		PM25:         8.5,
		PM10:         12.3,
		NH3:          1.2,
		Timestamp:    time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC),
	}
}

// MockAlert creates a mock alert for testing
func MockAlert(userID int64) *models.EnvironmentalAlert {
	return &models.EnvironmentalAlert{
		UserID:    userID,
		Type:      models.AlertTemperature,
		Threshold: 25.0,
		Condition: "greater_than",
		Value:     26.5,
		Severity:  models.SeverityMedium,
		Title:     "High Temperature Alert",
		Description: "Temperature exceeded threshold",
		IsActive:  true,
		CreatedAt: time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
		TriggeredAt: &[]time.Time{time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)}[0],
	}
}

// MockSubscription creates a mock subscription for testing
func MockSubscription(userID int64) *models.Subscription {
	return &models.Subscription{
		UserID:      userID,
		Type:        models.SubscriptionDaily,
		Frequency:   models.FrequencyDaily,
		TimeOfDay:   "08:00",
		IsActive:    true,
		CreatedAt:   time.Date(2023, 1, 1, 0, 0, 0, 0, time.UTC),
	}
}

// AnyTime is a custom matcher for time values in SQL mocks
type AnyTime struct{}

// Match implements the driver.Valuer interface
func (a AnyTime) Match(v driver.Value) bool {
	_, ok := v.(time.Time)
	return ok
}

// AnyUUID is a custom matcher for UUID values in SQL mocks
type AnyUUID struct{}

// Match implements the driver.Valuer interface
func (a AnyUUID) Match(v driver.Value) bool {
	s, ok := v.(string)
	if !ok {
		return false
	}
	// Simple UUID format validation
	return len(s) == 36 && s[8] == '-' && s[13] == '-' && s[18] == '-' && s[23] == '-'
}

// ExpectUserCreate sets up expectations for creating a user
func (m *MockDB) ExpectUserCreate() {
	m.Mock.ExpectBegin()
	m.Mock.ExpectQuery(`INSERT INTO "users"`).
		WillReturnRows(sqlmock.NewRows([]string{"id", "created_at", "updated_at"}).
			AddRow(123, time.Now(), time.Now()))
	m.Mock.ExpectCommit()
}

// ExpectUserFind sets up expectations for finding a user
func (m *MockDB) ExpectUserFind(userID int64, user *models.User) {
	rows := sqlmock.NewRows([]string{
		"id", "first_name", "last_name", "username", "language_code",
		"location_name", "latitude", "longitude", "country", "city", "timezone",
		"role", "created_at", "updated_at",
	}).AddRow(
		user.ID, user.FirstName, user.LastName, user.Username, user.LanguageCode,
		user.LocationName, user.Latitude, user.Longitude, user.Country, user.City,
		user.Timezone, user.Role, user.CreatedAt, user.UpdatedAt,
	)

	m.Mock.ExpectQuery(`SELECT \* FROM "users" WHERE "users"\."id" = \$1`).
		WithArgs(userID).
		WillReturnRows(rows)
}

// ExpectUserUpdate sets up expectations for updating a user
func (m *MockDB) ExpectUserUpdate(userID int64) {
	m.Mock.ExpectBegin()
	m.Mock.ExpectExec(`UPDATE "users" SET`).
		WithArgs(sqlmock.AnyArg(), userID). // updated_at, id
		WillReturnResult(sqlmock.NewResult(1, 1))
	m.Mock.ExpectCommit()
}

// NewResult creates a new SQL result for testing
func NewResult(lastInsertID, rowsAffected int64) driver.Result {
	return sqlmock.NewResult(lastInsertID, rowsAffected)
}