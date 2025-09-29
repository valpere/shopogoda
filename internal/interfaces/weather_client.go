package interfaces

import (
	"context"

	"github.com/valpere/shopogoda/pkg/weather"
)

//go:generate mockgen -source=weather_client.go -destination=../tests/mocks/weather_client_mock.go -package=mocks

// WeatherClientInterface defines the interface for weather API client
type WeatherClientInterface interface {
	GetCurrentWeather(ctx context.Context, lat, lon float64) (*weather.CurrentWeatherResponse, error)
	GetForecast(ctx context.Context, lat, lon float64) (*weather.ForecastResponse, error)
	GetAirQuality(ctx context.Context, lat, lon float64) (*weather.AirQualityResponse, error)
	GeocodeLocation(ctx context.Context, location string) ([]weather.GeocodeResponse, error)
}

// HTTPClientInterface defines the interface for HTTP client operations
type HTTPClientInterface interface {
	Get(url string) (*weather.APIResponse, error)
}