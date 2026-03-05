package skills

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"

	"dumbclaw/config"
)

func init() {
	Register("weather", func(cfg *config.Config) Skill {
		return &WeatherSkill{}
	})
}

// WeatherSkill fetches current weather using the Open-Meteo API (no API key required).
type WeatherSkill struct{}

func (s *WeatherSkill) Name() string { return "weather" }
func (s *WeatherSkill) Description() string {
	return `Get current weather for a location. Params: {"location": "London"}`
}

func (s *WeatherSkill) Execute(params map[string]any) (string, error) {
	location, ok := params["location"].(string)
	if !ok || location == "" {
		return "", fmt.Errorf("missing required param: location")
	}

	lat, lon, resolvedName, err := geocode(location)
	if err != nil {
		return "", err
	}

	return fetchWeather(lat, lon, resolvedName)
}

// geocode converts a place name to coordinates using the Open-Meteo geocoding API.
func geocode(location string) (lat, lon float64, name string, err error) {
	apiURL := "https://geocoding-api.open-meteo.com/v1/search?name=" + url.QueryEscape(location) + "&count=1"
	resp, err := http.Get(apiURL)
	if err != nil {
		return 0, 0, "", fmt.Errorf("geocoding request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Results []struct {
			Name    string  `json:"name"`
			Country string  `json:"country"`
			Lat     float64 `json:"latitude"`
			Lon     float64 `json:"longitude"`
		} `json:"results"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return 0, 0, "", fmt.Errorf("failed to parse geocoding response: %w", err)
	}
	if len(result.Results) == 0 {
		return 0, 0, "", fmt.Errorf("location not found: %q", location)
	}

	r := result.Results[0]
	return r.Lat, r.Lon, fmt.Sprintf("%s, %s", r.Name, r.Country), nil
}

// fetchWeather retrieves current weather from Open-Meteo for the given coordinates.
func fetchWeather(lat, lon float64, name string) (string, error) {

	apiURL := fmt.Sprintf(
		"https://api.open-meteo.com/v1/forecast?latitude=%f&longitude=%f&current=temperature_2m,relative_humidity_2m,wind_speed_10m,weather_code&wind_speed_unit=mph&temperature_unit=celsius",
		lat, lon,
	)

	log.Printf("Fetching weather for %s (lat: %.2f, lon: %.2f)", name, lat, lon)

	resp, err := http.Get(apiURL)
	if err != nil {
		return "", fmt.Errorf("weather request failed: %w", err)
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	var result struct {
		Current struct {
			Temperature float64 `json:"temperature_2m"`
			Humidity    float64 `json:"relative_humidity_2m"`
			WindSpeed   float64 `json:"wind_speed_10m"`
			WeatherCode int     `json:"weather_code"`
		} `json:"current"`
	}
	if err := json.Unmarshal(body, &result); err != nil {
		return "", fmt.Errorf("failed to parse weather response: %w", err)
	}

	c := result.Current
	return fmt.Sprintf(
		"Weather in %s: %s, %.1f°C, humidity %d%%, wind %.1f mph",
		name,
		weatherDescription(c.WeatherCode),
		c.Temperature,
		int(c.Humidity),
		c.WindSpeed,
	), nil
}

// weatherDescription maps WMO weather codes to human-readable descriptions.
func weatherDescription(code int) string {
	switch {
	case code == 0:
		return "clear sky"
	case code <= 2:
		return "partly cloudy"
	case code == 3:
		return "overcast"
	case code <= 49:
		return "foggy"
	case code <= 59:
		return "drizzle"
	case code <= 69:
		return "rain"
	case code <= 79:
		return "snow"
	case code <= 84:
		return "rain showers"
	case code <= 94:
		return "thunderstorm"
	default:
		return "thunderstorm with hail"
	}
}
