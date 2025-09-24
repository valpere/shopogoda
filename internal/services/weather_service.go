package services

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog"

	"github.com/valpere/shopogoda/internal/config"
	"github.com/valpere/shopogoda/internal/models"
	"github.com/valpere/shopogoda/pkg/weather"
)

// NominatimAddress represents the address structure returned by Nominatim API
type NominatimAddress struct {
	City          string `json:"city"`
	Town          string `json:"town"`
	Village       string `json:"village"`
	Hamlet        string `json:"hamlet"`
	County        string `json:"county"`
	State         string `json:"state"`
	Country       string `json:"country"`
	Suburb        string `json:"suburb"`
	Neighbourhood string `json:"neighbourhood"`
}

type WeatherService struct {
	client        *weather.Client
	geocoder      *weather.GeocodingClient
	redis         *redis.Client
	config        *config.WeatherConfig
	logger        *zerolog.Logger
}

func NewWeatherService(cfg *config.WeatherConfig, redis *redis.Client, logger *zerolog.Logger) *WeatherService {
	return &WeatherService{
		client:   weather.NewClient(cfg.OpenWeatherAPIKey),
		geocoder: weather.NewGeocodingClient(cfg.OpenWeatherAPIKey),
		redis:    redis,
		config:   cfg,
		logger:   logger,
	}
}

// getUserAgent safely returns the UserAgent from config with fallback to default
func (s *WeatherService) getUserAgent() string {
	if s.config != nil && s.config.UserAgent != "" {
		return s.config.UserAgent
	}
	return "ShoPogoda-Weather-Bot/1.0 (contact@shopogoda.bot)"
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
	// Normalize location name for consistent caching
	normalizedName := strings.ToLower(strings.TrimSpace(locationName))
	if normalizedName == "" {
		return nil, fmt.Errorf("location name cannot be empty")
	}

	// Try cache first
	cacheKey := fmt.Sprintf("geocode:%s", normalizedName)
	cached, err := s.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		var location weather.Location
		if err := json.Unmarshal([]byte(cached), &location); err == nil {
			return &location, nil
		}
	}

	// Try OpenWeatherMap API first if geocoder is available
	if s.geocoder != nil {
		location, err := s.geocoder.GeocodeLocation(ctx, locationName)
		if err == nil {
			// Cache for 24 hours
			if err := s.cacheLocation(ctx, cacheKey, location); err != nil {
				s.logger.Error().
					Err(err).
					Str("location", locationName).
					Str("service", "openweathermap").
					Msg("Failed to cache geocoding result")
			}
			return location, nil
		}
		s.logger.Debug().
			Err(err).
			Str("location", locationName).
			Msg("OpenWeatherMap geocoding failed, trying Nominatim")
	}

	// Try Nominatim as fallback
	nominatimLocation, err := s.geocodeWithNominatim(ctx, locationName)
	if err == nil {
		// Cache successful Nominatim results for 24 hours
		if err := s.cacheLocation(ctx, cacheKey, nominatimLocation); err != nil {
			s.logger.Error().
				Err(err).
				Str("location", locationName).
				Str("service", "nominatim").
				Msg("Failed to cache geocoding result")
		}
		return nominatimLocation, nil
	}

	return nil, fmt.Errorf("location '%s' not found - please check the spelling or try a major city name", locationName)
}

// cacheLocation is a helper method to cache location data
func (s *WeatherService) cacheLocation(ctx context.Context, cacheKey string, location *weather.Location) error {
	locationJSON, err := json.Marshal(location)
	if err != nil {
		return fmt.Errorf("failed to marshal location for caching: %w", err)
	}

	return s.redis.Set(ctx, cacheKey, locationJSON, 24*time.Hour).Err()
}

// GetCurrentWeatherByCoords gets weather data by coordinates (alias for GetCompleteWeatherData)
func (s *WeatherService) GetCurrentWeatherByCoords(ctx context.Context, lat, lon float64) (*WeatherData, error) {
	return s.GetCompleteWeatherData(ctx, lat, lon)
}

// GetCurrentWeatherByLocation gets weather data by location name
func (s *WeatherService) GetCurrentWeatherByLocation(ctx context.Context, locationName string) (*WeatherData, error) {
	// First geocode the location name to get coordinates
	location, err := s.GeocodeLocation(ctx, locationName)
	if err != nil {
		return nil, fmt.Errorf("failed to geocode location: %w", err)
	}

	// Get weather data using coordinates
	weatherData, err := s.GetCompleteWeatherData(ctx, location.Latitude, location.Longitude)
	if err != nil {
		return nil, err
	}

	// Set the location name
	weatherData.LocationName = location.Name

	return weatherData, nil
}

// GetLocationName returns a formatted location name from coordinates (reverse geocoding)
func (s *WeatherService) GetLocationName(ctx context.Context, lat, lon float64) (string, error) {
	// Try cache first
	cacheKey := fmt.Sprintf("reverse_geocode:%.4f:%.4f", lat, lon)
	cached, err := s.redis.Get(ctx, cacheKey).Result()
	if err == nil {
		return cached, nil
	}

	// Try reverse geocoding with Nominatim
	locationName, err := s.reverseGeocodeWithNominatim(ctx, lat, lon)
	if err != nil {
		s.logger.Warn().Err(err).Float64("lat", lat).Float64("lon", lon).Msg("Reverse geocoding failed")
		// Fallback to coordinate format
		return fmt.Sprintf("Location (%.4f, %.4f)", lat, lon), nil
	}

	// Cache for 24 hours
	s.redis.Set(ctx, cacheKey, locationName, 24*time.Hour)

	return locationName, nil
}

// ToModelWeatherData converts service WeatherData to models.WeatherData
func (wd *WeatherData) ToModelWeatherData() *models.WeatherData {
	return &models.WeatherData{
		Temperature: wd.Temperature,
		Humidity:    wd.Humidity,
		Pressure:    wd.Pressure,
		WindSpeed:   wd.WindSpeed,
		WindDegree:  wd.WindDirection,
		Visibility:  wd.Visibility,
		UVIndex:     wd.UVIndex,
		Description: wd.Description,
		Icon:        wd.Icon,
		AQI:         wd.AQI,
		CO:          wd.CO,
		NO2:         wd.NO2,
		O3:          wd.O3,
		PM25:        wd.PM25,
		PM10:        wd.PM10,
		Timestamp:   wd.Timestamp,
	}
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
	LocationName  string    `json:"location_name"`
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
	weatherData, err := s.GetCurrentWeather(ctx, lat, lon)
	if err != nil {
		return nil, err
	}

	air, err := s.GetAirQuality(ctx, lat, lon)
	if err != nil {
		// If air quality fails, still return weather data with empty air quality
		air = &weather.AirQualityData{
			AQI: 0,
			CO:  0,
			NO2: 0,
			O3:  0,
			PM25: 0,
			PM10: 0,
		}
	}

	return &WeatherData{
		Temperature:   weatherData.Temperature,
		Humidity:      weatherData.Humidity,
		Pressure:      weatherData.Pressure,
		WindSpeed:     weatherData.WindSpeed,
		WindDirection: weatherData.WindDirection,
		Visibility:    weatherData.Visibility,
		UVIndex:       weatherData.UVIndex,
		Description:   weatherData.Description,
		Icon:          weatherData.Icon,
		LocationName:  weatherData.LocationName,
		AQI:           air.AQI,
		CO:            air.CO,
		NO2:           air.NO2,
		O3:            air.O3,
		PM25:          air.PM25,
		PM10:          air.PM10,
		Timestamp:     weatherData.Timestamp,
	}, nil
}

// geocodeWithNominatim uses OpenStreetMap's Nominatim service as fallback geocoding
func (s *WeatherService) geocodeWithNominatim(ctx context.Context, locationName string) (*weather.Location, error) {
	requestURL := fmt.Sprintf("https://nominatim.openstreetmap.org/search?q=%s&format=json&limit=1",
		url.QueryEscape(locationName))

	req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create Nominatim request: %w", err)
	}

	// Set User-Agent as required by Nominatim usage policy
	req.Header.Set("User-Agent", s.getUserAgent())

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to make Nominatim request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("Nominatim API request failed with status: %d", resp.StatusCode)
	}

	var apiResponse []struct {
		DisplayName string `json:"display_name"`
		Lat         string `json:"lat"`
		Lon         string `json:"lon"`
		Type        string `json:"type"`
		Class       string `json:"class"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode Nominatim response: %w", err)
	}

	if len(apiResponse) == 0 {
		return nil, fmt.Errorf("location not found in Nominatim")
	}

	result := apiResponse[0]

	// Parse coordinates
	lat, err := strconv.ParseFloat(result.Lat, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid latitude from Nominatim: %w", err)
	}

	lon, err := strconv.ParseFloat(result.Lon, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid longitude from Nominatim: %w", err)
	}

	// Extract city and country from display_name
	parts := strings.Split(result.DisplayName, ", ")
	var city, country string
	if len(parts) > 0 {
		city = parts[0]
	}
	if len(parts) > 1 {
		country = parts[len(parts)-1]
	}

	return &weather.Location{
		Latitude:  lat,
		Longitude: lon,
		Name:      city,
		Country:   country,
		City:      city,
	}, nil
}

// reverseGeocodeWithNominatim performs reverse geocoding using Nominatim
func (s *WeatherService) reverseGeocodeWithNominatim(ctx context.Context, lat, lon float64) (string, error) {
	url := fmt.Sprintf("https://nominatim.openstreetmap.org/reverse?lat=%.6f&lon=%.6f&format=json&addressdetails=1",
		lat, lon)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return "", fmt.Errorf("failed to create Nominatim reverse request: %w", err)
	}

	// Set User-Agent as required by Nominatim usage policy
	req.Header.Set("User-Agent", s.getUserAgent())

	client := &http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to make Nominatim reverse request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("Nominatim reverse API request failed with status: %d", resp.StatusCode)
	}

	var result struct {
		DisplayName string           `json:"display_name"`
		Address     NominatimAddress `json:"address"`
		Lat         string           `json:"lat"`
		Lon         string           `json:"lon"`
		Type        string           `json:"type"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", fmt.Errorf("failed to decode Nominatim reverse response: %w", err)
	}

	// Determine the most appropriate location name and format
	return s.formatLocationFromAddress(result.Address, lat, lon), nil
}

// formatLocationFromAddress formats location name with smart logic for exact vs nearby matches
func (s *WeatherService) formatLocationFromAddress(address NominatimAddress, lat, lon float64) string {

	// Prioritize location names from most specific to general
	var locationName string
	var isExact bool

	if address.City != "" {
		locationName = address.City
		isExact = true // Cities are usually exact matches
	} else if address.Town != "" {
		locationName = address.Town
		isExact = true // Towns are usually exact matches
	} else if address.Village != "" {
		locationName = address.Village
		isExact = true // Villages are usually exact matches
	} else if address.Suburb != "" {
		locationName = address.Suburb
		isExact = false // Suburbs indicate "near" a larger city
	} else if address.Neighbourhood != "" {
		locationName = address.Neighbourhood
		isExact = false // Neighbourhoods indicate "near" a larger area
	} else if address.County != "" {
		locationName = address.County
		isExact = false // Counties are large areas, so "near"
	} else if address.State != "" {
		locationName = address.State
		isExact = false // States are very large, so "near"
	} else {
		// Fallback to coordinate format
		return fmt.Sprintf("Location (%.4f, %.4f)", lat, lon)
	}

	// Format the final location string
	coords := fmt.Sprintf("(%.4f, %.4f)", lat, lon)

	if isExact {
		return fmt.Sprintf("%s %s", locationName, coords)
	} else {
		return fmt.Sprintf("near %s %s", locationName, coords)
	}
}