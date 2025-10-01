package weather

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewClient(t *testing.T) {
	apiKey := "test_api_key"
	client := NewClient(apiKey)

	assert.NotNil(t, client)
	assert.Equal(t, apiKey, client.apiKey)
	assert.Equal(t, "https://api.openweathermap.org", client.baseURL)
	assert.NotNil(t, client.httpClient)
	assert.Equal(t, 10*time.Second, client.httpClient.Timeout)
}

func TestClient_GetCurrentWeather_Success(t *testing.T) {
	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/data/2.5/weather")
		assert.Contains(t, r.URL.Query().Get("lat"), "40.7")
		assert.Contains(t, r.URL.Query().Get("lon"), "-74.0")
		assert.Equal(t, "test_key", r.URL.Query().Get("appid"))

		response := map[string]interface{}{
			"main": map[string]interface{}{
				"temp":     15.5,
				"humidity": 65,
				"pressure": 1013.0,
			},
			"wind": map[string]interface{}{
				"speed": 5.5,
				"deg":   180,
			},
			"weather": []map[string]interface{}{
				{
					"main":        "Clouds",
					"description": "scattered clouds",
					"icon":        "03d",
				},
			},
			"visibility": 10000,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient("test_key")
	client.baseURL = server.URL

	weather, err := client.GetCurrentWeather(context.Background(), 40.7128, -74.0060)

	require.NoError(t, err)
	require.NotNil(t, weather)
	assert.Equal(t, 15.5, weather.Temperature)
	assert.Equal(t, 65, weather.Humidity)
	assert.Equal(t, 1013.0, weather.Pressure)
	assert.InDelta(t, 19.8, weather.WindSpeed, 0.1) // 5.5 m/s * 3.6 = 19.8 km/h
	assert.Equal(t, 180, weather.WindDirection)
	assert.Equal(t, 10.0, weather.Visibility)
	assert.Equal(t, "scattered clouds", weather.Description)
	assert.Equal(t, "03d", weather.Icon)
	assert.False(t, weather.Timestamp.IsZero())
}

func TestClient_GetCurrentWeather_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer server.Close()

	client := NewClient("test_key")
	client.baseURL = server.URL

	weather, err := client.GetCurrentWeather(context.Background(), 40.7128, -74.0060)

	assert.Error(t, err)
	assert.Nil(t, weather)
	assert.Contains(t, err.Error(), "API request failed with status: 404")
}

func TestClient_GetCurrentWeather_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("invalid json"))
	}))
	defer server.Close()

	client := NewClient("test_key")
	client.baseURL = server.URL

	weather, err := client.GetCurrentWeather(context.Background(), 40.7128, -74.0060)

	assert.Error(t, err)
	assert.Nil(t, weather)
	assert.Contains(t, err.Error(), "failed to decode response")
}

func TestClient_GetCurrentWeather_NoWeatherData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"main": map[string]interface{}{
				"temp":     15.5,
				"humidity": 65,
				"pressure": 1013.0,
			},
			"wind": map[string]interface{}{
				"speed": 5.5,
				"deg":   180,
			},
			"weather":    []map[string]interface{}{}, // Empty weather array
			"visibility": 10000,
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient("test_key")
	client.baseURL = server.URL

	weather, err := client.GetCurrentWeather(context.Background(), 40.7128, -74.0060)

	require.NoError(t, err)
	require.NotNil(t, weather)
	assert.Equal(t, "", weather.Description)
	assert.Equal(t, "", weather.Icon)
}

func TestClient_GetForecast_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/data/2.5/forecast")

		now := time.Now().Unix()
		response := map[string]interface{}{
			"list": []map[string]interface{}{
				{
					"dt": now,
					"main": map[string]interface{}{
						"temp":     15.5,
						"temp_min": 12.0,
						"temp_max": 18.0,
					},
					"weather": []map[string]interface{}{
						{
							"description": "clear sky",
							"icon":        "01d",
						},
					},
				},
				{
					"dt": now + 86400, // Next day
					"main": map[string]interface{}{
						"temp":     16.5,
						"temp_min": 13.0,
						"temp_max": 19.0,
					},
					"weather": []map[string]interface{}{
						{
							"description": "few clouds",
							"icon":        "02d",
						},
					},
				},
			},
			"city": map[string]interface{}{
				"name":    "New York",
				"country": "US",
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient("test_key")
	client.baseURL = server.URL

	forecast, err := client.GetForecast(context.Background(), 40.7128, -74.0060, 5)

	require.NoError(t, err)
	require.NotNil(t, forecast)
	assert.Equal(t, "New York, US", forecast.Location)
	assert.Len(t, forecast.Forecasts, 2)

	assert.Equal(t, 12.0, forecast.Forecasts[0].MinTemp)
	assert.Equal(t, 18.0, forecast.Forecasts[0].MaxTemp)
	assert.Equal(t, "clear sky", forecast.Forecasts[0].Description)
	assert.Equal(t, "01d", forecast.Forecasts[0].Icon)
}

func TestClient_GetForecast_LimitDays(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		now := time.Now().Unix()
		response := map[string]interface{}{
			"list": []map[string]interface{}{
				{"dt": now, "main": map[string]interface{}{"temp": 15.0, "temp_min": 12.0, "temp_max": 18.0}, "weather": []map[string]interface{}{{"description": "day 1", "icon": "01d"}}},
				{"dt": now + 86400, "main": map[string]interface{}{"temp": 16.0, "temp_min": 13.0, "temp_max": 19.0}, "weather": []map[string]interface{}{{"description": "day 2", "icon": "02d"}}},
				{"dt": now + 172800, "main": map[string]interface{}{"temp": 17.0, "temp_min": 14.0, "temp_max": 20.0}, "weather": []map[string]interface{}{{"description": "day 3", "icon": "03d"}}},
				{"dt": now + 259200, "main": map[string]interface{}{"temp": 18.0, "temp_min": 15.0, "temp_max": 21.0}, "weather": []map[string]interface{}{{"description": "day 4", "icon": "04d"}}},
			},
			"city": map[string]interface{}{"name": "Test", "country": "US"},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient("test_key")
	client.baseURL = server.URL

	// Request only 2 days
	forecast, err := client.GetForecast(context.Background(), 40.7128, -74.0060, 2)

	require.NoError(t, err)
	assert.Len(t, forecast.Forecasts, 2) // Should limit to 2 days
}

func TestClient_GetForecast_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewClient("test_key")
	client.baseURL = server.URL

	forecast, err := client.GetForecast(context.Background(), 40.7128, -74.0060, 5)

	assert.Error(t, err)
	assert.Nil(t, forecast)
	assert.Contains(t, err.Error(), "API request failed with status: 401")
}

func TestClient_GetAirQuality_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/data/2.5/air_pollution")

		response := map[string]interface{}{
			"list": []map[string]interface{}{
				{
					"main": map[string]interface{}{
						"aqi": 3,
					},
					"components": map[string]interface{}{
						"co":    250.5,
						"no2":   30.5,
						"o3":    50.5,
						"pm2_5": 15.5,
						"pm10":  20.5,
					},
				},
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient("test_key")
	client.baseURL = server.URL

	airQuality, err := client.GetAirQuality(context.Background(), 40.7128, -74.0060)

	require.NoError(t, err)
	require.NotNil(t, airQuality)
	assert.Equal(t, 3, airQuality.AQI)
	assert.Equal(t, 250.5, airQuality.CO)
	assert.Equal(t, 30.5, airQuality.NO2)
	assert.Equal(t, 50.5, airQuality.O3)
	assert.Equal(t, 15.5, airQuality.PM25)
	assert.Equal(t, 20.5, airQuality.PM10)
	assert.False(t, airQuality.Timestamp.IsZero())
}

func TestClient_GetAirQuality_NoData(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := map[string]interface{}{
			"list": []map[string]interface{}{}, // Empty list
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient("test_key")
	client.baseURL = server.URL

	airQuality, err := client.GetAirQuality(context.Background(), 40.7128, -74.0060)

	assert.Error(t, err)
	assert.Nil(t, airQuality)
	assert.Contains(t, err.Error(), "no air quality data available")
}

func TestClient_GetAirQuality_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient("test_key")
	client.baseURL = server.URL

	airQuality, err := client.GetAirQuality(context.Background(), 40.7128, -74.0060)

	assert.Error(t, err)
	assert.Nil(t, airQuality)
	assert.Contains(t, err.Error(), "API request failed with status: 500")
}

func TestNewGeocodingClient(t *testing.T) {
	apiKey := "test_api_key"
	client := NewGeocodingClient(apiKey)

	assert.NotNil(t, client)
	assert.Equal(t, apiKey, client.apiKey)
	assert.Equal(t, "https://api.openweathermap.org", client.baseURL)
	assert.NotNil(t, client.httpClient)
	assert.Equal(t, 10*time.Second, client.httpClient.Timeout)
}

func TestGeocodingClient_GeocodeLocation_Success(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.Path, "/geo/1.0/direct")
		assert.Contains(t, r.URL.Query().Get("q"), "London")

		response := []map[string]interface{}{
			{
				"name":    "London",
				"country": "GB",
				"state":   "England",
				"lat":     51.5074,
				"lon":     -0.1278,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewGeocodingClient("test_key")
	client.baseURL = server.URL

	location, err := client.GeocodeLocation(context.Background(), "London")

	require.NoError(t, err)
	require.NotNil(t, location)
	assert.Equal(t, "London", location.Name)
	assert.Equal(t, "GB", location.Country)
	assert.Equal(t, "London", location.City)
	assert.Equal(t, 51.5074, location.Latitude)
	assert.Equal(t, -0.1278, location.Longitude)
}

func TestGeocodingClient_GeocodeLocation_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		response := []map[string]interface{}{} // Empty array

		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewGeocodingClient("test_key")
	client.baseURL = server.URL

	location, err := client.GeocodeLocation(context.Background(), "NonexistentCity")

	assert.Error(t, err)
	assert.Nil(t, location)
	assert.Contains(t, err.Error(), "location not found")
}

func TestGeocodingClient_GeocodeLocation_APIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusForbidden)
	}))
	defer server.Close()

	client := NewGeocodingClient("test_key")
	client.baseURL = server.URL

	location, err := client.GeocodeLocation(context.Background(), "London")

	assert.Error(t, err)
	assert.Nil(t, location)
	assert.Contains(t, err.Error(), "API request failed with status: 403")
}

func TestGeocodingClient_GeocodeLocation_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte("not valid json"))
	}))
	defer server.Close()

	client := NewGeocodingClient("test_key")
	client.baseURL = server.URL

	location, err := client.GeocodeLocation(context.Background(), "London")

	assert.Error(t, err)
	assert.Nil(t, location)
	assert.Contains(t, err.Error(), "failed to decode response")
}

func TestClient_ContextCancellation(t *testing.T) {
	// Create a server that delays response
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(100 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient("test_key")
	client.baseURL = server.URL

	// Create a context that cancels immediately
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := client.GetCurrentWeather(ctx, 40.7128, -74.0060)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "context canceled")
}

// Test request creation errors by using invalid base URLs
func TestClient_GetCurrentWeather_RequestCreationError(t *testing.T) {
	client := NewClient("test_key")
	client.baseURL = "ht tp://invalid url with spaces"

	_, err := client.GetCurrentWeather(context.Background(), 40.7128, -74.0060)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create request")
}

func TestClient_GetForecast_RequestCreationError(t *testing.T) {
	client := NewClient("test_key")
	client.baseURL = "ht tp://invalid url with spaces"

	_, err := client.GetForecast(context.Background(), 40.7128, -74.0060, 5)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create request")
}

func TestClient_GetAirQuality_RequestCreationError(t *testing.T) {
	client := NewClient("test_key")
	client.baseURL = "ht tp://invalid url with spaces"

	_, err := client.GetAirQuality(context.Background(), 40.7128, -74.0060)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create request")
}

func TestGeocodingClient_GeocodeLocation_RequestCreationError(t *testing.T) {
	client := NewGeocodingClient("test_key")
	client.baseURL = "ht tp://invalid url with spaces"

	_, err := client.GeocodeLocation(context.Background(), "London")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to create request")
}

// Test httpClient.Do() errors by using a closed server
func TestClient_GetForecast_NetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	serverURL := server.URL
	server.Close() // Close immediately to trigger network error

	client := NewClient("test_key")
	client.baseURL = serverURL

	_, err := client.GetForecast(context.Background(), 40.7128, -74.0060, 5)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to make request")
}

func TestClient_GetAirQuality_NetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	serverURL := server.URL
	server.Close() // Close immediately to trigger network error

	client := NewClient("test_key")
	client.baseURL = serverURL

	_, err := client.GetAirQuality(context.Background(), 40.7128, -74.0060)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to make request")
}

func TestGeocodingClient_GeocodeLocation_NetworkError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	serverURL := server.URL
	server.Close() // Close immediately to trigger network error

	client := NewGeocodingClient("test_key")
	client.baseURL = serverURL

	_, err := client.GeocodeLocation(context.Background(), "London")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to make request")
}
