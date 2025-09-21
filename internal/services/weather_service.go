package services

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/valpere/shopogoda/internal/config"
	"github.com/valpere/shopogoda/pkg/weather"
)

type WeatherService struct {
	client        *weather.Client
	geocoder      *weather.GeocodingClient
	redis         *redis.Client
	config        *config.WeatherConfig
}

func NewWeatherService(cfg *config.WeatherConfig, redis *redis.Client) *WeatherService {
	return &WeatherService{
		client:   weather.NewClient(cfg.OpenWeatherAPIKey),
		geocoder: weather.NewGeocodingClient(cfg.OpenWeatherAPIKey),
		redis:    redis,
		config:   cfg,
	}
}

func (s *WeatherService) GetCurrentWeather(ctx context.Context, lat, lon float64) (*weather.WeatherData, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("weather:current:%.4f:%.4f", lat, lon)
	cached, err := s.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var weatherData weather.WeatherData
		if err := json.Unmarshal([]byte(cached), &weatherData); err == nil {
			return &weatherData, nil
		}
	}

	// Get from API
	weatherData, err := s.client.GetCurrentWeather(ctx, lat, lon)
	if err != nil {
		return nil, fmt.Errorf("failed to get weather data: %w", err)
	}

	// Cache for 10 minutes
	weatherJSON, _ := json.Marshal(weatherData)
	s.redis.Set(ctx, cacheKey, weatherJSON, 10*time.Minute)

	return weatherData, nil
}

func (s *WeatherService) GetForecast(ctx context.Context, lat, lon float64, days int) (*weather.ForecastData, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("weather:forecast:%.4f:%.4f:%d", lat, lon, days)
	cached, err := s.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var forecastData weather.ForecastData
		if err := json.Unmarshal([]byte(cached), &forecastData); err == nil {
			return &forecastData, nil
		}
	}

	// Get from API
	forecastData, err := s.client.GetForecast(ctx, lat, lon, days)
	if err != nil {
		return nil, fmt.Errorf("failed to get forecast data: %w", err)
	}

	// Cache for 1 hour
	forecastJSON, _ := json.Marshal(forecastData)
	s.redis.Set(ctx, cacheKey, forecastJSON, time.Hour)

	return forecastData, nil
}

func (s *WeatherService) GetAirQuality(ctx context.Context, lat, lon float64) (*weather.AirQualityData, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("weather:air:%.4f:%.4f", lat, lon)
	cached, err := s.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var airData weather.AirQualityData
		if err := json.Unmarshal([]byte(cached), &airData); err == nil {
			return &airData, nil
		}
	}

	// Get from API
	airData, err := s.client.GetAirQuality(ctx, lat, lon)
	if err != nil {
		return nil, fmt.Errorf("failed to get air quality data: %w", err)
	}

	// Cache for 30 minutes
	airJSON, _ := json.Marshal(airData)
	s.redis.Set(ctx, cacheKey, airJSON, 30*time.Minute)

	return airData, nil
}

func (s *WeatherService) GeocodeLocation(ctx context.Context, locationName string) (*weather.Location, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("geocode:%s", locationName)
	cached, err := s.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var location weather.Location
		if err := json.Unmarshal([]byte(cached), &location); err == nil {
			return &location, nil
		}
	}

	// Get from API
	location, err := s.geocoder.GeocodeLocation(ctx, locationName)
	if err != nil {
		return nil, fmt.Errorf("failed to geocode location: %w", err)
	}

	// Cache for 24 hours
	locationJSON, _ := json.Marshal(location)
	s.redis.Set(ctx, cacheKey, locationJSON, 24*time.Hour)

	return location, nil
}

// WeatherData represents weather data compatible with models.WeatherData
type WeatherData struct {
	Temperature   float64   `json:"temperature"`
	Humidity      int       `json:"humidity"`
	Pressure      float64   `json:"pressure"`
	WindSpeed     float64   `json:"wind_speed"`
	WindDirection int       `json:"wind_direction"`
	Visibility    float64   `json:"visibility"`
	UVIndex       float64   `json:"uv_index"`
	Description   string    `json:"description"`
	Icon          string    `json:"icon"`
	AQI           int       `json:"aqi"`
	CO            float64   `json:"co"`
	NO2           float64   `json:"no2"`
	O3            float64   `json:"o3"`
	PM25          float64   `json:"pm25"`
	PM10          float64   `json:"pm10"`
	Timestamp     time.Time `json:"timestamp"`
}

// GetCompleteWeatherData gets both weather and air quality data
func (s *WeatherService) GetCompleteWeatherData(ctx context.Context, lat, lon float64) (*WeatherData, error) {
	weather, err := s.GetCurrentWeather(ctx, lat, lon)
	if err != nil {
		return nil, err
	}

	air, err := s.GetAirQuality(ctx, lat, lon)
	if err != nil {
		// If air quality fails, still return weather data
		air = &weather.AirQualityData{}
	}

	return &WeatherData{
		Temperature:   weather.Temperature,
		Humidity:      weather.Humidity,
		Pressure:      weather.Pressure,
		WindSpeed:     weather.WindSpeed,
		WindDirection: weather.WindDirection,
		Visibility:    weather.Visibility,
		UVIndex:       weather.UVIndex,
		Description:   weather.Description,
		Icon:          weather.Icon,
		AQI:           air.AQI,
		CO:            air.CO,
		NO2:           air.NO2,
		O3:            air.O3,
		PM25:          air.PM25,
		PM10:          air.PM10,
		Timestamp:     weather.Timestamp,
	}, nil
}