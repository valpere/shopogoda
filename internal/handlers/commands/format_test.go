package commands

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/valpere/shopogoda/internal/services"
	"github.com/valpere/shopogoda/pkg/weather"
	"github.com/valpere/shopogoda/tests/helpers"
)

func TestFormatWeatherMessage(t *testing.T) {
	logger := helpers.NewSilentTestLogger()
	locService := services.NewLocalizationService(logger)
	svc := &services.Services{
		Localization: locService,
	}

	handler := &CommandHandler{
		services: svc,
		logger:   logger,
	}

	tests := []struct {
		name     string
		weather  *services.WeatherData
		language string
	}{
		{
			name: "Complete weather data in English",
			weather: &services.WeatherData{
				LocationName: "London",
				Location: &weather.Location{
					Name:    "London",
					Country: "GB",
				},
				Temperature: 15.5,
				Description: "clear sky",
				Humidity:    75,
				Pressure:    1013,
				WindSpeed:   5.5,
			},
			language: "en",
		},
		{
			name: "Weather with high values",
			weather: &services.WeatherData{
				LocationName: "Dubai",
				Location: &weather.Location{
					Name:    "Dubai",
					Country: "AE",
				},
				Temperature: 45.0,
				Description: "hot",
				Humidity:    90,
				Pressure:    1020,
				WindSpeed:   25.5,
			},
			language: "en",
		},
		{
			name: "Weather with low values",
			weather: &services.WeatherData{
				LocationName: "Reykjavik",
				Location: &weather.Location{
					Name:    "Reykjavik",
					Country: "IS",
				},
				Temperature: -5.0,
				Description: "snow",
				Humidity:    80,
				Pressure:    995,
				WindSpeed:   15.0,
			},
			language: "en",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.formatWeatherMessage(tt.weather, tt.language)

			// Verify message contains key information
			assert.Contains(t, result, tt.weather.LocationName)
			assert.NotEmpty(t, result)

			// Verify message contains weather data
			assert.Contains(t, result, tt.weather.Description)
		})
	}
}

func TestFormatForecastMessage(t *testing.T) {
	logger := helpers.NewSilentTestLogger()
	locService := services.NewLocalizationService(logger)
	svc := &services.Services{
		Localization: locService,
	}

	handler := &CommandHandler{
		services: svc,
		logger:   logger,
	}

	// Note: Date field needs to be time.Time, not string
	date1, _ := time.Parse("2006-01-02", "2025-01-11")
	date2, _ := time.Parse("2006-01-02", "2025-01-12")

	tests := []struct {
		name     string
		forecast *weather.ForecastData
		language string
	}{
		{
			name: "Multi-day forecast in English",
			forecast: &weather.ForecastData{
				Location: "Paris, FR",
				Forecasts: []weather.DailyForecast{
					{
						Date:        date1,
						MinTemp:     5.0,
						MaxTemp:     12.0,
						Description: "partly cloudy",
						Humidity:    70,
						WindSpeed:   8.0,
					},
					{
						Date:        date2,
						MinTemp:     7.0,
						MaxTemp:     14.0,
						Description: "sunny",
						Humidity:    65,
						WindSpeed:   6.0,
					},
				},
			},
			language: "en",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.formatForecastMessage(tt.forecast, tt.language)

			// Verify message is generated and contains forecast data
			assert.NotEmpty(t, result)
			assert.Contains(t, result, tt.forecast.Forecasts[0].Description)
		})
	}
}

func TestFormatAirQualityMessage(t *testing.T) {
	logger := helpers.NewSilentTestLogger()
	locService := services.NewLocalizationService(logger)
	svc := &services.Services{
		Localization: locService,
	}

	handler := &CommandHandler{
		services: svc,
		logger:   logger,
	}

	tests := []struct {
		name     string
		airData  *weather.AirQualityData
		language string
	}{
		{
			name: "Good air quality",
			airData: &weather.AirQualityData{
				AQI:  25,
				CO:   200.0,
				NO2:  10.0,
				O3:   50.0,
				PM25: 8.0,
				PM10: 12.0,
			},
			language: "en",
		},
		{
			name: "Hazardous air quality",
			airData: &weather.AirQualityData{
				AQI:  350,
				CO:   2000.0,
				NO2:  100.0,
				O3:   150.0,
				PM25: 250.0,
				PM10: 300.0,
			},
			language: "en",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := handler.formatAirQualityMessage(tt.airData, tt.language)

			// Verify message is generated
			assert.NotEmpty(t, result)
		})
	}
}
