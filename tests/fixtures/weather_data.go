package fixtures

import (
	"encoding/json"
	"time"
)

// OpenWeatherCurrentResponse represents a mock OpenWeatherMap current weather response
type OpenWeatherCurrentResponse struct {
	Weather []struct {
		Main        string `json:"main"`
		Description string `json:"description"`
	} `json:"weather"`
	Main struct {
		Temp      float64 `json:"temp"`
		FeelsLike float64 `json:"feels_like"`
		Pressure  int     `json:"pressure"`
		Humidity  int     `json:"humidity"`
	} `json:"main"`
	Wind struct {
		Speed float64 `json:"speed"`
		Deg   int     `json:"deg"`
	} `json:"wind"`
	Visibility int    `json:"visibility"`
	Name       string `json:"name"`
	Sys        struct {
		Country string `json:"country"`
	} `json:"sys"`
}

// OpenWeatherForecastResponse represents a mock OpenWeatherMap forecast response
type OpenWeatherForecastResponse struct {
	List []struct {
		Dt   int64 `json:"dt"`
		Main struct {
			Temp      float64 `json:"temp"`
			FeelsLike float64 `json:"feels_like"`
			Pressure  int     `json:"pressure"`
			Humidity  int     `json:"humidity"`
		} `json:"main"`
		Weather []struct {
			Main        string `json:"main"`
			Description string `json:"description"`
		} `json:"weather"`
		Wind struct {
			Speed float64 `json:"speed"`
			Deg   int     `json:"deg"`
		} `json:"wind"`
	} `json:"list"`
	City struct {
		Name    string `json:"name"`
		Country string `json:"country"`
	} `json:"city"`
}

// AirQualityResponse represents a mock air quality response
type AirQualityResponse struct {
	List []struct {
		Main struct {
			AQI int `json:"aqi"`
		} `json:"main"`
		Components struct {
			CO   float64 `json:"co"`
			NO   float64 `json:"no"`
			NO2  float64 `json:"no2"`
			O3   float64 `json:"o3"`
			SO2  float64 `json:"so2"`
			PM25 float64 `json:"pm2_5"`
			PM10 float64 `json:"pm10"`
			NH3  float64 `json:"nh3"`
		} `json:"components"`
	} `json:"list"`
}

// GeocodeResponse represents a mock geocoding response
type GeocodeResponse []struct {
	Name    string  `json:"name"`
	Lat     float64 `json:"lat"`
	Lon     float64 `json:"lon"`
	Country string  `json:"country"`
	State   string  `json:"state"`
}

// GetMockWeatherResponse returns a mock current weather response
func GetMockWeatherResponse() string {
	response := OpenWeatherCurrentResponse{
		Weather: []struct {
			Main        string `json:"main"`
			Description string `json:"description"`
		}{
			{Main: "Clear", Description: "clear sky"},
		},
		Main: struct {
			Temp      float64 `json:"temp"`
			FeelsLike float64 `json:"feels_like"`
			Pressure  int     `json:"pressure"`
			Humidity  int     `json:"humidity"`
		}{
			Temp:      20.5,
			FeelsLike: 22.0,
			Pressure:  1013,
			Humidity:  65,
		},
		Wind: struct {
			Speed float64 `json:"speed"`
			Deg   int     `json:"deg"`
		}{
			Speed: 5.2,
			Deg:   180,
		},
		Visibility: 10000,
		Name:       "London",
		Sys: struct {
			Country string `json:"country"`
		}{
			Country: "GB",
		},
	}

	data, _ := json.Marshal(response)
	return string(data)
}

// GetMockForecastResponse returns a mock forecast response
func GetMockForecastResponse() string {
	now := time.Now()
	response := OpenWeatherForecastResponse{
		List: []struct {
			Dt   int64 `json:"dt"`
			Main struct {
				Temp      float64 `json:"temp"`
				FeelsLike float64 `json:"feels_like"`
				Pressure  int     `json:"pressure"`
				Humidity  int     `json:"humidity"`
			} `json:"main"`
			Weather []struct {
				Main        string `json:"main"`
				Description string `json:"description"`
			} `json:"weather"`
			Wind struct {
				Speed float64 `json:"speed"`
				Deg   int     `json:"deg"`
			} `json:"wind"`
		}{
			{
				Dt: now.Unix(),
				Main: struct {
					Temp      float64 `json:"temp"`
					FeelsLike float64 `json:"feels_like"`
					Pressure  int     `json:"pressure"`
					Humidity  int     `json:"humidity"`
				}{
					Temp:      18.5,
					FeelsLike: 19.0,
					Pressure:  1015,
					Humidity:  70,
				},
				Weather: []struct {
					Main        string `json:"main"`
					Description string `json:"description"`
				}{
					{Main: "Clouds", Description: "few clouds"},
				},
				Wind: struct {
					Speed float64 `json:"speed"`
					Deg   int     `json:"deg"`
				}{
					Speed: 3.8,
					Deg:   200,
				},
			},
		},
		City: struct {
			Name    string `json:"name"`
			Country string `json:"country"`
		}{
			Name:    "London",
			Country: "GB",
		},
	}

	data, _ := json.Marshal(response)
	return string(data)
}

// GetMockAirQualityResponse returns a mock air quality response
func GetMockAirQualityResponse() string {
	response := AirQualityResponse{
		List: []struct {
			Main struct {
				AQI int `json:"aqi"`
			} `json:"main"`
			Components struct {
				CO   float64 `json:"co"`
				NO   float64 `json:"no"`
				NO2  float64 `json:"no2"`
				O3   float64 `json:"o3"`
				SO2  float64 `json:"so2"`
				PM25 float64 `json:"pm2_5"`
				PM10 float64 `json:"pm10"`
				NH3  float64 `json:"nh3"`
			} `json:"components"`
		}{
			{
				Main: struct {
					AQI int `json:"aqi"`
				}{
					AQI: 2,
				},
				Components: struct {
					CO   float64 `json:"co"`
					NO   float64 `json:"no"`
					NO2  float64 `json:"no2"`
					O3   float64 `json:"o3"`
					SO2  float64 `json:"so2"`
					PM25 float64 `json:"pm2_5"`
					PM10 float64 `json:"pm10"`
					NH3  float64 `json:"nh3"`
				}{
					CO:   0.3,
					NO:   0.1,
					NO2:  15.2,
					O3:   45.3,
					SO2:  2.1,
					PM25: 8.5,
					PM10: 12.3,
					NH3:  1.2,
				},
			},
		},
	}

	data, _ := json.Marshal(response)
	return string(data)
}

// GetMockGeocodeResponse returns a mock geocoding response
func GetMockGeocodeResponse() string {
	response := GeocodeResponse{
		{
			Name:    "London",
			Lat:     51.5074,
			Lon:     -0.1278,
			Country: "GB",
			State:   "England",
		},
	}

	data, _ := json.Marshal(response)
	return string(data)
}

// GetInvalidJSONResponse returns an invalid JSON response for error testing
func GetInvalidJSONResponse() string {
	return `{"invalid": json}`
}

// GetEmptyResponse returns an empty response
func GetEmptyResponse() string {
	return `{}`
}

// GetErrorResponse returns an error response from the API
func GetErrorResponse() string {
	return `{"cod": 404, "message": "city not found"}`
}
