package services

import (
	"context"
	"testing"

	"github.com/valpere/shopogoda/internal/models"
	"github.com/valpere/shopogoda/tests/helpers"
)

// BenchmarkWeatherService_GeocodeLocation benchmarks the geocoding functionality
func BenchmarkWeatherService_GeocodeLocation(b *testing.B) {
	mockDB := helpers.NewMockDB(b)
	defer mockDB.Close()

	mockRedis := helpers.NewMockRedis()
	defer mockRedis.Close()

	logger := helpers.NewSilentTestLogger()
	config := helpers.GetTestConfig()
	service := NewWeatherService(mockDB.DB, mockRedis.Client, config, logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GeocodeLocation(context.Background(), "London")
	}
}

// BenchmarkWeatherService_GetCurrentWeather benchmarks current weather retrieval
func BenchmarkWeatherService_GetCurrentWeather(b *testing.B) {
	mockDB := helpers.NewMockDB(b)
	defer mockDB.Close()

	mockRedis := helpers.NewMockRedis()
	defer mockRedis.Close()

	logger := helpers.NewSilentTestLogger()
	config := helpers.GetTestConfig()
	service := NewWeatherService(mockDB.DB, mockRedis.Client, config, logger)

	lat, lon := 51.5074, -0.1278

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GetCurrentWeather(context.Background(), lat, lon)
	}
}

// BenchmarkUserService_CreateUser benchmarks user creation
func BenchmarkUserService_CreateUser(b *testing.B) {
	mockDB := helpers.NewMockDB(b)
	defer mockDB.Close()

	logger := helpers.NewSilentTestLogger()
	service := NewUserService(mockDB.DB, logger)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		user := helpers.MockUser(int64(i + 1000))
		_ = service.CreateUser(context.Background(), user)
	}
}

// BenchmarkUserService_GetUser benchmarks user retrieval
func BenchmarkUserService_GetUser(b *testing.B) {
	mockDB := helpers.NewMockDB(b)
	defer mockDB.Close()

	logger := helpers.NewSilentTestLogger()
	service := NewUserService(mockDB.DB, logger)

	userID := int64(123)
	user := helpers.MockUser(userID)

	// Setup expectations for each benchmark iteration
	for i := 0; i < b.N; i++ {
		mockDB.ExpectUserFind(userID, user)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GetUser(context.Background(), userID)
	}
}

// BenchmarkAlertService_CheckAlertsForUser benchmarks alert checking
func BenchmarkAlertService_CheckAlertsForUser(b *testing.B) {
	mockDB := helpers.NewMockDB(b)
	defer mockDB.Close()

	logger := helpers.NewSilentTestLogger()
	service := NewAlertService(mockDB.DB, logger)

	userID := int64(123)
	weatherData := helpers.MockWeatherData(userID)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.CheckAlertsForUser(context.Background(), userID, weatherData)
	}
}

// BenchmarkEnvironmentalAlert_IsTriggered benchmarks alert triggering logic
func BenchmarkEnvironmentalAlert_IsTriggered(b *testing.B) {
	alert := &models.EnvironmentalAlert{
		Type:      models.AlertTemperature,
		Threshold: 25.0,
		Condition: "greater_than",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = alert.IsTriggered(26.5)
	}
}

// BenchmarkUser_GetDisplayName benchmarks user display name generation
func BenchmarkUser_GetDisplayName(b *testing.B) {
	user := helpers.MockUser(123)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = user.GetDisplayName()
	}
}

// BenchmarkUser_HasLocation benchmarks location validation
func BenchmarkUser_HasLocation(b *testing.B) {
	user := helpers.MockUser(123)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = user.HasLocation()
	}
}

// BenchmarkWeatherData_IsRecent benchmarks weather data recency check
func BenchmarkWeatherData_IsRecent(b *testing.B) {
	weatherData := helpers.MockWeatherData(123)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = weatherData.IsRecent()
	}
}

// BenchmarkConcurrentUserOperations benchmarks concurrent user operations
func BenchmarkConcurrentUserOperations(b *testing.B) {
	mockDB := helpers.NewMockDB(b)
	defer mockDB.Close()

	logger := helpers.NewSilentTestLogger()
	service := NewUserService(mockDB.DB, logger)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		userID := int64(1)
		for pb.Next() {
			user := helpers.MockUser(userID)
			mockDB.ExpectUserFind(userID, user)
			_, _ = service.GetUser(context.Background(), userID)
			userID++
		}
	})
}

// BenchmarkConcurrentWeatherRequests benchmarks concurrent weather requests
func BenchmarkConcurrentWeatherRequests(b *testing.B) {
	mockDB := helpers.NewMockDB(b)
	defer mockDB.Close()

	mockRedis := helpers.NewMockRedis()
	defer mockRedis.Close()

	logger := helpers.NewSilentTestLogger()
	config := helpers.GetTestConfig()
	service := NewWeatherService(mockDB.DB, mockRedis.Client, config, logger)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = service.GeocodeLocation(context.Background(), "London")
		}
	})
}

// BenchmarkMemoryAllocations benchmarks memory allocations during operations
func BenchmarkMemoryAllocations(b *testing.B) {
	b.ReportAllocs()

	mockDB := helpers.NewMockDB(b)
	defer mockDB.Close()

	logger := helpers.NewSilentTestLogger()
	service := NewUserService(mockDB.DB, logger)

	userID := int64(123)
	user := helpers.MockUser(userID)

	for i := 0; i < b.N; i++ {
		mockDB.ExpectUserFind(userID, user)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = service.GetUser(context.Background(), userID)
	}
}