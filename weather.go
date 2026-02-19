package main

import (
	"fmt"
	"io"
	"net/http"
	"net/url"
	"time"
)

type WeatherClient struct {
	httpClient *http.Client
	baseURL    string
}

func NewWeatherClient() *WeatherClient {
	return &WeatherClient{
		httpClient: &http.Client{Timeout: 10 * time.Second},
		baseURL:    "https://wttr.in",
	}
}

// GetCurrent returns a one-line summary of current weather.
func (c *WeatherClient) GetCurrent(location string) (string, error) {
	u := fmt.Sprintf("%s/%s?format=%%l:+%%c+%%t+(%%f)+%%h+%%w", c.baseURL, url.PathEscape(location))
	return c.fetch(u)
}

// GetForecast returns a text forecast for the given number of days.
func (c *WeatherClient) GetForecast(location string, days int) (string, error) {
	u := fmt.Sprintf("%s/%s?%d&lang=ru", c.baseURL, url.PathEscape(location), days)
	return c.fetch(u)
}

// GetDetailed returns structured JSON weather data.
func (c *WeatherClient) GetDetailed(location string) (string, error) {
	u := fmt.Sprintf("%s/%s?format=j1", c.baseURL, url.PathEscape(location))
	return c.fetch(u)
}

func (c *WeatherClient) fetch(rawURL string) (string, error) {
	req, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return "", fmt.Errorf("creating request: %w", err)
	}
	req.Header.Set("User-Agent", "wttr-weather-mcp/1.0")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("fetching weather: %w", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", fmt.Errorf("reading response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("wttr.in returned status %d: %s", resp.StatusCode, string(body))
	}

	return string(body), nil
}
