package services

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/valpere/shopogoda/internal/models"
	"github.com/valpere/shopogoda/tests/helpers"
)

func TestUserService_CreateUser(t *testing.T) {
	// Setup
	mockDB := helpers.NewMockDB(t)
	defer mockDB.Close()

	logger := helpers.NewSilentTestLogger()
	service := NewUserService(mockDB.DB, logger)

	t.Run("successful user creation", func(t *testing.T) {
		user := helpers.MockUser(123)

		// Mock database expectations
		mockDB.ExpectUserCreate()

		err := service.CreateUser(context.Background(), user)

		assert.NoError(t, err)
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("database error", func(t *testing.T) {
		user := helpers.MockUser(123)

		// Mock database error
		mockDB.Mock.ExpectBegin()
		mockDB.Mock.ExpectQuery(`INSERT INTO "users"`).
			WillReturnError(errors.New("constraint violation"))
		mockDB.Mock.ExpectRollback()

		err := service.CreateUser(context.Background(), user)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "constraint violation")
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("duplicate user", func(t *testing.T) {
		user := helpers.MockUser(123)

		// Mock duplicate key error
		mockDB.Mock.ExpectBegin()
		mockDB.Mock.ExpectQuery(`INSERT INTO "users"`).
			WillReturnError(errors.New("duplicate key value violates unique constraint"))
		mockDB.Mock.ExpectRollback()

		err := service.CreateUser(context.Background(), user)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "duplicate key")
		mockDB.ExpectationsWereMet(t)
	})
}

func TestUserService_GetUser(t *testing.T) {
	// Setup
	mockDB := helpers.NewMockDB(t)
	defer mockDB.Close()

	logger := helpers.NewSilentTestLogger()
	service := NewUserService(mockDB.DB, logger)

	t.Run("successful user retrieval", func(t *testing.T) {
		userID := int64(123)
		expectedUser := helpers.MockUser(userID)

		mockDB.ExpectUserFind(userID, expectedUser)

		user, err := service.GetUser(context.Background(), userID)

		assert.NoError(t, err)
		assert.NotNil(t, user)
		assert.Equal(t, userID, user.ID)
		assert.Equal(t, expectedUser.FirstName, user.FirstName)
		assert.Equal(t, expectedUser.LocationName, user.LocationName)
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("user not found", func(t *testing.T) {
		userID := int64(999)

		// Mock no rows found
		mockDB.Mock.ExpectQuery(`SELECT \* FROM "users" WHERE "users"\."id" = \$1`).
			WithArgs(userID).
			WillReturnError(errors.New("record not found"))

		user, err := service.GetUser(context.Background(), userID)

		assert.Error(t, err)
		assert.Nil(t, user)
		assert.Contains(t, err.Error(), "record not found")
		mockDB.ExpectationsWereMet(t)
	})
}

func TestUserService_SetUserLocation(t *testing.T) {
	// Setup
	mockDB := helpers.NewMockDB(t)
	defer mockDB.Close()

	logger := helpers.NewSilentTestLogger()
	service := NewUserService(mockDB.DB, logger)

	t.Run("successful location update", func(t *testing.T) {
		userID := int64(123)
		locationName := "Paris"
		country := "France"
		city := "Paris"
		lat, lon := 48.8566, 2.3522

		// Mock finding the user first
		user := helpers.MockUser(userID)
		mockDB.ExpectUserFind(userID, user)

		// Mock location update
		mockDB.ExpectUserUpdate(userID)

		err := service.SetUserLocation(context.Background(), userID, locationName, country, city, lat, lon)

		assert.NoError(t, err)
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("user not found", func(t *testing.T) {
		userID := int64(999)

		// Mock user not found
		mockDB.Mock.ExpectQuery(`SELECT \* FROM "users" WHERE "users"\."id" = \$1`).
			WithArgs(userID).
			WillReturnError(errors.New("record not found"))

		err := service.SetUserLocation(context.Background(), userID, "Paris", "France", "Paris", 48.8566, 2.3522)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "record not found")
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("update fails", func(t *testing.T) {
		userID := int64(123)
		user := helpers.MockUser(userID)

		// Mock finding the user
		mockDB.ExpectUserFind(userID, user)

		// Mock update failure
		mockDB.Mock.ExpectBegin()
		mockDB.Mock.ExpectExec(`UPDATE "users" SET`).
			WillReturnError(errors.New("database error"))
		mockDB.Mock.ExpectRollback()

		err := service.SetUserLocation(context.Background(), userID, "Paris", "France", "Paris", 48.8566, 2.3522)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "database error")
		mockDB.ExpectationsWereMet(t)
	})
}

func TestUserService_GetUserLocation(t *testing.T) {
	// Setup
	mockDB := helpers.NewMockDB(t)
	defer mockDB.Close()

	logger := helpers.NewSilentTestLogger()
	service := NewUserService(mockDB.DB, logger)

	t.Run("user with location", func(t *testing.T) {
		userID := int64(123)
		user := helpers.MockUser(userID)
		user.LocationName = "London"
		user.Latitude = 51.5074
		user.Longitude = -0.1278

		mockDB.ExpectUserFind(userID, user)

		locationName, lat, lon, err := service.GetUserLocation(context.Background(), userID)

		assert.NoError(t, err)
		assert.Equal(t, "London", locationName)
		assert.Equal(t, 51.5074, lat)
		assert.Equal(t, -0.1278, lon)
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("user without location", func(t *testing.T) {
		userID := int64(123)
		user := helpers.MockUser(userID)
		user.LocationName = ""
		user.Latitude = 0
		user.Longitude = 0

		mockDB.ExpectUserFind(userID, user)

		locationName, lat, lon, err := service.GetUserLocation(context.Background(), userID)

		assert.Error(t, err)
		assert.Empty(t, locationName)
		assert.Equal(t, float64(0), lat)
		assert.Equal(t, float64(0), lon)
		assert.Contains(t, err.Error(), "no location set")
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("user not found", func(t *testing.T) {
		userID := int64(999)

		mockDB.Mock.ExpectQuery(`SELECT \* FROM "users" WHERE "users"\."id" = \$1`).
			WithArgs(userID).
			WillReturnError(errors.New("record not found"))

		locationName, lat, lon, err := service.GetUserLocation(context.Background(), userID)

		assert.Error(t, err)
		assert.Empty(t, locationName)
		assert.Equal(t, float64(0), lat)
		assert.Equal(t, float64(0), lon)
		mockDB.ExpectationsWereMet(t)
	})
}

func TestUserService_SetUserLanguage(t *testing.T) {
	// Setup
	mockDB := helpers.NewMockDB(t)
	defer mockDB.Close()

	logger := helpers.NewSilentTestLogger()
	service := NewUserService(mockDB.DB, logger)

	t.Run("successful language update", func(t *testing.T) {
		userID := int64(123)
		language := "de"

		user := helpers.MockUser(userID)
		mockDB.ExpectUserFind(userID, user)
		mockDB.ExpectUserUpdate(userID)

		err := service.SetUserLanguage(context.Background(), userID, language)

		assert.NoError(t, err)
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("invalid language code", func(t *testing.T) {
		userID := int64(123)
		language := "invalid"

		err := service.SetUserLanguage(context.Background(), userID, language)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "unsupported language")
	})
}

func TestUserService_GetUserLanguage(t *testing.T) {
	// Setup
	mockDB := helpers.NewMockDB(t)
	defer mockDB.Close()

	logger := helpers.NewSilentTestLogger()
	service := NewUserService(mockDB.DB, logger)

	t.Run("user with language", func(t *testing.T) {
		userID := int64(123)
		user := helpers.MockUser(userID)
		user.LanguageCode = "de"

		mockDB.ExpectUserFind(userID, user)

		language := service.GetUserLanguage(context.Background(), userID)

		assert.Equal(t, "de", language)
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("user not found - fallback to English", func(t *testing.T) {
		userID := int64(999)

		mockDB.Mock.ExpectQuery(`SELECT \* FROM "users" WHERE "users"\."id" = \$1`).
			WithArgs(userID).
			WillReturnError(errors.New("record not found"))

		language := service.GetUserLanguage(context.Background(), userID)

		assert.Equal(t, "en", language) // Should fallback to English
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("user with empty language - fallback to English", func(t *testing.T) {
		userID := int64(123)
		user := helpers.MockUser(userID)
		user.LanguageCode = ""

		mockDB.ExpectUserFind(userID, user)

		language := service.GetUserLanguage(context.Background(), userID)

		assert.Equal(t, "en", language) // Should fallback to English
		mockDB.ExpectationsWereMet(t)
	})
}

func TestUserService_TimezoneOperations(t *testing.T) {
	// Setup
	mockDB := helpers.NewMockDB(t)
	defer mockDB.Close()

	logger := helpers.NewSilentTestLogger()
	service := NewUserService(mockDB.DB, logger)

	t.Run("timezone conversion", func(t *testing.T) {
		userID := int64(123)
		user := helpers.MockUser(userID)
		user.Timezone = "America/New_York"

		mockDB.ExpectUserFind(userID, user)

		utcTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
		userTime := service.ConvertToUserTime(context.Background(), userID, utcTime)

		// America/New_York is UTC-5 in winter
		expected := time.Date(2023, 1, 1, 7, 0, 0, 0, time.UTC)
		assert.Equal(t, expected.Hour(), userTime.Hour())
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("invalid timezone - fallback to UTC", func(t *testing.T) {
		userID := int64(123)
		user := helpers.MockUser(userID)
		user.Timezone = "Invalid/Timezone"

		mockDB.ExpectUserFind(userID, user)

		utcTime := time.Date(2023, 1, 1, 12, 0, 0, 0, time.UTC)
		userTime := service.ConvertToUserTime(context.Background(), userID, utcTime)

		// Should fallback to UTC (no conversion)
		assert.Equal(t, utcTime, userTime)
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("get user timezone", func(t *testing.T) {
		userID := int64(123)
		user := helpers.MockUser(userID)
		user.Timezone = "Europe/Berlin"

		mockDB.ExpectUserFind(userID, user)

		timezone := service.GetUserTimezone(context.Background(), userID)

		assert.Equal(t, "Europe/Berlin", timezone)
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("user not found - fallback to UTC", func(t *testing.T) {
		userID := int64(999)

		mockDB.Mock.ExpectQuery(`SELECT \* FROM "users" WHERE "users"\."id" = \$1`).
			WithArgs(userID).
			WillReturnError(errors.New("record not found"))

		timezone := service.GetUserTimezone(context.Background(), userID)

		assert.Equal(t, "UTC", timezone) // Should fallback to UTC
		mockDB.ExpectationsWereMet(t)
	})
}

func TestUserService_GetActiveUsersWithLocations(t *testing.T) {
	// Setup
	mockDB := helpers.NewMockDB(t)
	defer mockDB.Close()

	logger := helpers.NewSilentTestLogger()
	service := NewUserService(mockDB.DB, logger)

	t.Run("successful retrieval", func(t *testing.T) {
		// Mock multiple users with locations
		users := []*models.User{
			helpers.MockUser(123),
			helpers.MockUser(456),
		}

		rows := mockDB.Mock.NewRows([]string{
			"id", "first_name", "last_name", "username", "language_code",
			"location_name", "latitude", "longitude", "country", "city", "timezone",
			"role", "created_at", "updated_at",
		})

		for _, user := range users {
			rows.AddRow(
				user.ID, user.FirstName, user.LastName, user.Username, user.LanguageCode,
				user.LocationName, user.Latitude, user.Longitude, user.Country, user.City,
				user.Timezone, user.Role, user.CreatedAt, user.UpdatedAt,
			)
		}

		mockDB.Mock.ExpectQuery(`SELECT \* FROM "users" WHERE location_name != '' AND location_name IS NOT NULL`).
			WillReturnRows(rows)

		result, err := service.GetActiveUsersWithLocations(context.Background())

		assert.NoError(t, err)
		assert.Len(t, result, 2)
		assert.Equal(t, int64(123), result[0].ID)
		assert.Equal(t, int64(456), result[1].ID)
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("no users found", func(t *testing.T) {
		// Mock empty result
		mockDB.Mock.ExpectQuery(`SELECT \* FROM "users" WHERE location_name != '' AND location_name IS NOT NULL`).
			WillReturnRows(mockDB.Mock.NewRows([]string{
				"id", "first_name", "last_name", "username", "language_code",
				"location_name", "latitude", "longitude", "country", "city", "timezone",
				"role", "created_at", "updated_at",
			}))

		result, err := service.GetActiveUsersWithLocations(context.Background())

		assert.NoError(t, err)
		assert.Len(t, result, 0)
		mockDB.ExpectationsWereMet(t)
	})

	t.Run("database error", func(t *testing.T) {
		mockDB.Mock.ExpectQuery(`SELECT \* FROM "users" WHERE location_name != '' AND location_name IS NOT NULL`).
			WillReturnError(errors.New("database connection error"))

		result, err := service.GetActiveUsersWithLocations(context.Background())

		assert.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "database connection error")
		mockDB.ExpectationsWereMet(t)
	})
}