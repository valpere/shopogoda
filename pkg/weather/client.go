package weather

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"time"
)

// Client represents a weather API client
type Client struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// WeatherData represents current weather information
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
	Timestamp     time.Time `json:"timestamp"`
}

// ForecastData represents weather forecast
type ForecastData struct {
	Location  string          `json:"location"`
	Forecasts []DailyForecast `json:"forecasts"`
}

// DailyForecast represents a single day forecast
type DailyForecast struct {
	Date        time.Time `json:"date"`
	MinTemp     float64   `json:"min_temp"`
	MaxTemp     float64   `json:"max_temp"`
	Description string    `json:"description"`
	Icon        string    `json:"icon"`
	Humidity    int       `json:"humidity"`
	WindSpeed   float64   `json:"wind_speed"`
}

// AirQualityData represents air quality information
type AirQualityData struct {
	AQI       int       `json:"aqi"`
	CO        float64   `json:"co"`
	NO2       float64   `json:"no2"`
	O3        float64   `json:"o3"`
	PM25      float64   `json:"pm25"`
	PM10      float64   `json:"pm10"`
	Timestamp time.Time `json:"timestamp"`
}

// Location represents geographic coordinates
type Location struct {
	Latitude   float64           `json:"latitude"`
	Longitude  float64           `json:"longitude"`
	Name       string            `json:"name"`
	Country    string            `json:"country"`
	City       string            `json:"city"`
	LocalNames map[string]string `json:"local_names,omitempty"`
}

// NewClient creates a new weather API client
func NewClient(apiKey string) *Client {
	return &Client{
		apiKey:  apiKey,
		baseURL: "https://api.openweathermap.org",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GetCurrentWeather retrieves current weather for a location
func (c *Client) GetCurrentWeather(ctx context.Context, lat, lon float64) (*WeatherData, error) {
	url := fmt.Sprintf("%s/data/2.5/weather?lat=%.6f&lon=%.6f&appid=%s&units=metric",
		c.baseURL, lat, lon, c.apiKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req) // nosec G704
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var apiResponse struct {
		Main struct {
			Temp     float64 `json:"temp"`
			Humidity int     `json:"humidity"`
			Pressure float64 `json:"pressure"`
		} `json:"main"`
		Wind struct {
			Speed float64 `json:"speed"`
			Deg   int     `json:"deg"`
		} `json:"wind"`
		Weather []struct {
			Main        string `json:"main"`
			Description string `json:"description"`
			Icon        string `json:"icon"`
		} `json:"weather"`
		Visibility int `json:"visibility"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	weather := &WeatherData{
		Temperature:   apiResponse.Main.Temp,
		Humidity:      apiResponse.Main.Humidity,
		Pressure:      apiResponse.Main.Pressure,
		WindSpeed:     apiResponse.Wind.Speed * 3.6, // Convert m/s to km/h
		WindDirection: apiResponse.Wind.Deg,
		Visibility:    float64(apiResponse.Visibility) / 1000, // Convert m to km
		Timestamp:     time.Now(),
	}

	if len(apiResponse.Weather) > 0 {
		weather.Description = apiResponse.Weather[0].Description
		weather.Icon = apiResponse.Weather[0].Icon
	}

	return weather, nil
}

// GetForecast retrieves weather forecast for a location
func (c *Client) GetForecast(ctx context.Context, lat, lon float64, days int) (*ForecastData, error) {
	url := fmt.Sprintf("%s/data/2.5/forecast?lat=%.6f&lon=%.6f&appid=%s&units=metric",
		c.baseURL, lat, lon, c.apiKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req) // nosec G704
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var apiResponse struct {
		List []struct {
			Dt   int64 `json:"dt"`
			Main struct {
				Temp    float64 `json:"temp"`
				TempMin float64 `json:"temp_min"`
				TempMax float64 `json:"temp_max"`
			} `json:"main"`
			Weather []struct {
				Description string `json:"description"`
				Icon        string `json:"icon"`
			} `json:"weather"`
		} `json:"list"`
		City struct {
			Name    string `json:"name"`
			Country string `json:"country"`
		} `json:"city"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	forecast := &ForecastData{
		Location:  fmt.Sprintf("%s, %s", apiResponse.City.Name, apiResponse.City.Country),
		Forecasts: make([]DailyForecast, 0),
	}

	// Group forecasts by day and take the first 'days' entries
	daysSeen := make(map[string]bool)
	for _, item := range apiResponse.List {
		if len(forecast.Forecasts) >= days {
			break
		}

		date := time.Unix(item.Dt, 0).UTC().Truncate(24 * time.Hour)
		dateKey := date.Format("2006-01-02")

		if !daysSeen[dateKey] {
			daysSeen[dateKey] = true

			daily := DailyForecast{
				Date:    date,
				MinTemp: item.Main.TempMin,
				MaxTemp: item.Main.TempMax,
			}

			if len(item.Weather) > 0 {
				daily.Description = item.Weather[0].Description
				daily.Icon = item.Weather[0].Icon
			}

			forecast.Forecasts = append(forecast.Forecasts, daily)
		}
	}

	return forecast, nil
}

// GetAirQuality retrieves air quality data for a location
func (c *Client) GetAirQuality(ctx context.Context, lat, lon float64) (*AirQualityData, error) {
	url := fmt.Sprintf("%s/data/2.5/air_pollution?lat=%.6f&lon=%.6f&appid=%s",
		c.baseURL, lat, lon, c.apiKey)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req) // nosec G704
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var apiResponse struct {
		List []struct {
			Main struct {
				AQI int `json:"aqi"`
			} `json:"main"`
			Components struct {
				CO   float64 `json:"co"`
				NO2  float64 `json:"no2"`
				O3   float64 `json:"o3"`
				PM25 float64 `json:"pm2_5"`
				PM10 float64 `json:"pm10"`
			} `json:"components"`
		} `json:"list"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(apiResponse.List) == 0 {
		return nil, fmt.Errorf("no air quality data available")
	}

	data := apiResponse.List[0]
	return &AirQualityData{
		AQI:       data.Main.AQI,
		CO:        data.Components.CO,
		NO2:       data.Components.NO2,
		O3:        data.Components.O3,
		PM25:      data.Components.PM25,
		PM10:      data.Components.PM10,
		Timestamp: time.Now(),
	}, nil
}

// GeocodingClient handles location-related requests
type GeocodingClient struct {
	apiKey     string
	baseURL    string
	httpClient *http.Client
}

// NewGeocodingClient creates a new geocoding client
func NewGeocodingClient(apiKey string) *GeocodingClient {
	return &GeocodingClient{
		apiKey:  apiKey,
		baseURL: "https://api.openweathermap.org",
		httpClient: &http.Client{
			Timeout: 10 * time.Second,
		},
	}
}

// GeocodeLocation converts location name to coordinates
func (c *GeocodingClient) GeocodeLocation(ctx context.Context, locationName string) (*Location, error) {
	// URL encode the location name to properly handle non-English characters
	encodedLocation := url.QueryEscape(locationName)
	requestURL := fmt.Sprintf("%s/geo/1.0/direct?q=%s&limit=1&appid=%s",
		c.baseURL, encodedLocation, c.apiKey)

	req, err := http.NewRequestWithContext(ctx, "GET", requestURL, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req) // nosec G704
	if err != nil {
		return nil, fmt.Errorf("failed to make request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("API request failed with status: %d", resp.StatusCode)
	}

	var apiResponse []struct {
		Name       string            `json:"name"`
		Country    string            `json:"country"`
		State      string            `json:"state"`
		Lat        float64           `json:"lat"`
		Lon        float64           `json:"lon"`
		LocalNames map[string]string `json:"local_names"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	if len(apiResponse) == 0 {
		return nil, fmt.Errorf("location not found")
	}

	result := apiResponse[0]
	return &Location{
		Latitude:   result.Lat,
		Longitude:  result.Lon,
		Name:       result.Name,
		Country:    result.Country,
		City:       result.Name,
		LocalNames: result.LocalNames,
	}, nil
}
