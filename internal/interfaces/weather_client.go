package interfaces

import (
	"context"

	"github.com/valpere/shopogoda/pkg/weather"
)

//go:generate mockgen -source=weather_client.go -destination=../tests/mocks/weather_client_mock.go -package=mocks

// WeatherClientInterface defines the interface for weather API client
type WeatherClientInterface interface {
	GetCurrentWeather(ctx context.Context, lat, lon float64) (*weather.WeatherData, error)
	GetForecast(ctx context.Context, lat, lon float64, days int) (*weather.ForecastData, error)
	GetAirQuality(ctx context.Context, lat, lon float64) (*weather.AirQualityData, error)
}

// GeocodingClientInterface defines the interface for geocoding operations
type GeocodingClientInterface interface {
	GeocodeLocation(ctx context.Context, location string) (*weather.Location, error)
}