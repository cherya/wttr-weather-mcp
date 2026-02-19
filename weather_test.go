package main

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWeatherClientGetCurrent(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/London") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if !strings.Contains(r.URL.RawQuery, "format=") {
			t.Error("expected format parameter in query")
		}
		w.Write([]byte("London: ☀️ +20°C"))
	}))
	defer srv.Close()

	client := &WeatherClient{
		httpClient: srv.Client(),
		baseURL:    srv.URL,
	}

	result, err := client.GetCurrent("London")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "London: ☀️ +20°C" {
		t.Errorf("unexpected result: %s", result)
	}
}

func TestWeatherClientGetForecast(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.HasPrefix(r.URL.Path, "/Tokyo") {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if !strings.Contains(r.URL.RawQuery, "2") {
			t.Error("expected days parameter in query")
		}
		if !strings.Contains(r.URL.RawQuery, "lang=ru") {
			t.Error("expected lang=ru in query")
		}
		w.Write([]byte("forecast data"))
	}))
	defer srv.Close()

	client := &WeatherClient{
		httpClient: srv.Client(),
		baseURL:    srv.URL,
	}

	result, err := client.GetForecast("Tokyo", 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != "forecast data" {
		t.Errorf("unexpected result: %s", result)
	}
}

func TestWeatherClientGetDetailed(t *testing.T) {
	jsonResp := `{"current_condition":[{"temp_C":"25"}]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.URL.RawQuery, "format=j1") {
			t.Error("expected format=j1 in query")
		}
		w.Write([]byte(jsonResp))
	}))
	defer srv.Close()

	client := &WeatherClient{
		httpClient: srv.Client(),
		baseURL:    srv.URL,
	}

	result, err := client.GetDetailed("Dubai")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result != jsonResp {
		t.Errorf("unexpected result: %s", result)
	}
}

func TestWeatherClientHTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		w.Write([]byte("Unknown location"))
	}))
	defer srv.Close()

	client := &WeatherClient{
		httpClient: srv.Client(),
		baseURL:    srv.URL,
	}

	_, err := client.GetCurrent("NonexistentPlace")
	if err == nil {
		t.Fatal("expected error for 404 response")
	}
	if !strings.Contains(err.Error(), "404") {
		t.Errorf("expected 404 in error, got: %v", err)
	}
}

func TestWeatherClientLocationEncoding(t *testing.T) {
	var receivedRawURL string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedRawURL = r.RequestURI
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	client := &WeatherClient{
		httpClient: srv.Client(),
		baseURL:    srv.URL,
	}

	client.GetCurrent("New York")
	if !strings.HasPrefix(receivedRawURL, "/New%20York") {
		t.Errorf("expected URL-encoded path, got %s", receivedRawURL)
	}
}

func TestWeatherClientUserAgent(t *testing.T) {
	var receivedUA string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedUA = r.Header.Get("User-Agent")
		w.Write([]byte("ok"))
	}))
	defer srv.Close()

	client := &WeatherClient{
		httpClient: srv.Client(),
		baseURL:    srv.URL,
	}

	client.GetCurrent("London")
	if receivedUA != "wttr-weather-mcp/1.0" {
		t.Errorf("expected User-Agent wttr-weather-mcp/1.0, got %s", receivedUA)
	}
}
