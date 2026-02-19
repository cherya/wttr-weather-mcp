package main

import (
	"encoding/json"
	"fmt"
	"testing"
)

// mockWeather implements WeatherService for testing.
type mockWeather struct {
	currentResult  string
	forecastResult string
	detailedResult string
	err            error
	lastLocation   string
	lastDays       int
}

func (m *mockWeather) GetCurrent(location string) (string, error) {
	m.lastLocation = location
	return m.currentResult, m.err
}

func (m *mockWeather) GetForecast(location string, days int) (string, error) {
	m.lastLocation = location
	m.lastDays = days
	return m.forecastResult, m.err
}

func (m *mockWeather) GetDetailed(location string) (string, error) {
	m.lastLocation = location
	return m.detailedResult, m.err
}

func makeRequest(method string, id interface{}, params interface{}) JSONRPCRequest {
	var raw json.RawMessage
	if params != nil {
		raw, _ = json.Marshal(params)
	}
	return JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  raw,
	}
}

func TestHandleInitialize(t *testing.T) {
	s := &Server{weather: &mockWeather{}}
	req := makeRequest("initialize", 1, nil)
	resp := s.handleRequest(req)

	if resp == nil {
		t.Fatal("expected response, got nil")
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}
	if result["protocolVersion"] != "2024-11-05" {
		t.Errorf("expected protocol version 2024-11-05, got %v", result["protocolVersion"])
	}
}

func TestHandleInitialized(t *testing.T) {
	s := &Server{weather: &mockWeather{}}
	req := makeRequest("initialized", nil, nil)
	resp := s.handleRequest(req)

	if resp != nil {
		t.Fatal("expected nil response for initialized notification")
	}
}

func TestHandleToolsList(t *testing.T) {
	s := &Server{weather: &mockWeather{}}
	req := makeRequest("tools/list", 1, nil)
	resp := s.handleRequest(req)

	if resp == nil {
		t.Fatal("expected response, got nil")
	}

	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}

	tools, ok := result["tools"].([]map[string]interface{})
	if !ok {
		t.Fatal("tools is not a slice")
	}

	if len(tools) != 3 {
		t.Fatalf("expected 3 tools, got %d", len(tools))
	}

	names := map[string]bool{}
	for _, tool := range tools {
		names[tool["name"].(string)] = true
	}

	for _, expected := range []string{"get_current_weather", "get_forecast", "get_weather_detailed"} {
		if !names[expected] {
			t.Errorf("missing tool: %s", expected)
		}
	}
}

func TestToolsListLocationRequired(t *testing.T) {
	s := &Server{weather: &mockWeather{}}
	req := makeRequest("tools/list", 1, nil)
	resp := s.handleRequest(req)

	result := resp.Result.(map[string]interface{})
	tools := result["tools"].([]map[string]interface{})

	for _, tool := range tools {
		schema := tool["inputSchema"].(map[string]interface{})
		required, ok := schema["required"].([]string)
		if !ok {
			t.Errorf("tool %s: required is not []string", tool["name"])
			continue
		}

		found := false
		for _, r := range required {
			if r == "location" {
				found = true
				break
			}
		}
		if !found {
			t.Errorf("tool %s: location should be required", tool["name"])
		}
	}
}

func TestCallGetCurrent(t *testing.T) {
	mock := &mockWeather{currentResult: "London: ☀️ +20°C (19°C) 45% ↑5km/h"}
	s := &Server{weather: mock}

	params := map[string]interface{}{
		"name":      "get_current_weather",
		"arguments": map[string]string{"location": "London"},
	}
	req := makeRequest("tools/call", 1, params)
	resp := s.handleRequest(req)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	if mock.lastLocation != "London" {
		t.Errorf("expected location London, got %s", mock.lastLocation)
	}

	assertSuccessText(t, resp, "London: ☀️ +20°C (19°C) 45% ↑5km/h")
}

func TestCallGetCurrentMissingLocation(t *testing.T) {
	s := &Server{weather: &mockWeather{}}

	params := map[string]interface{}{
		"name":      "get_current_weather",
		"arguments": map[string]string{},
	}
	req := makeRequest("tools/call", 1, params)
	resp := s.handleRequest(req)

	if resp.Error == nil {
		t.Fatal("expected error for missing location")
	}
	if resp.Error.Code != -32602 {
		t.Errorf("expected error code -32602, got %d", resp.Error.Code)
	}
}

func TestCallGetForecast(t *testing.T) {
	mock := &mockWeather{forecastResult: "forecast data"}
	s := &Server{weather: mock}

	params := map[string]interface{}{
		"name":      "get_forecast",
		"arguments": map[string]interface{}{"location": "Tokyo", "days": 2},
	}
	req := makeRequest("tools/call", 1, params)
	resp := s.handleRequest(req)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	if mock.lastLocation != "Tokyo" {
		t.Errorf("expected location Tokyo, got %s", mock.lastLocation)
	}
	if mock.lastDays != 2 {
		t.Errorf("expected 2 days, got %d", mock.lastDays)
	}

	assertSuccessText(t, resp, "forecast data")
}

func TestCallGetForecastDefaultDays(t *testing.T) {
	mock := &mockWeather{forecastResult: "forecast"}
	s := &Server{weather: mock}

	params := map[string]interface{}{
		"name":      "get_forecast",
		"arguments": map[string]string{"location": "Berlin"},
	}
	req := makeRequest("tools/call", 1, params)
	s.handleRequest(req)

	if mock.lastDays != 3 {
		t.Errorf("expected default 3 days, got %d", mock.lastDays)
	}
}

func TestCallGetForecastInvalidDays(t *testing.T) {
	mock := &mockWeather{forecastResult: "forecast"}
	s := &Server{weather: mock}

	for _, days := range []int{0, -1, 5, 100} {
		params := map[string]interface{}{
			"name":      "get_forecast",
			"arguments": map[string]interface{}{"location": "Paris", "days": days},
		}
		req := makeRequest("tools/call", 1, params)
		s.handleRequest(req)

		if mock.lastDays != 3 {
			t.Errorf("days=%d: expected clamped to 3, got %d", days, mock.lastDays)
		}
	}
}

func TestCallGetDetailed(t *testing.T) {
	mock := &mockWeather{detailedResult: `{"current_condition": [{"temp_C": "25"}]}`}
	s := &Server{weather: mock}

	params := map[string]interface{}{
		"name":      "get_weather_detailed",
		"arguments": map[string]string{"location": "Dubai"},
	}
	req := makeRequest("tools/call", 1, params)
	resp := s.handleRequest(req)

	if resp.Error != nil {
		t.Fatalf("unexpected error: %v", resp.Error)
	}

	if mock.lastLocation != "Dubai" {
		t.Errorf("expected location Dubai, got %s", mock.lastLocation)
	}
}

func TestCallWeatherError(t *testing.T) {
	mock := &mockWeather{err: fmt.Errorf("network timeout")}
	s := &Server{weather: mock}

	params := map[string]interface{}{
		"name":      "get_current_weather",
		"arguments": map[string]string{"location": "Nowhere"},
	}
	req := makeRequest("tools/call", 1, params)
	resp := s.handleRequest(req)

	if resp.Error != nil {
		t.Fatal("weather errors should be returned as isError result, not RPC error")
	}

	result := resp.Result.(map[string]interface{})
	isError, ok := result["isError"]
	if !ok || isError != true {
		t.Error("expected isError: true in result")
	}
}

func TestUnknownTool(t *testing.T) {
	s := &Server{weather: &mockWeather{}}

	params := map[string]interface{}{
		"name":      "nonexistent_tool",
		"arguments": map[string]string{},
	}
	req := makeRequest("tools/call", 1, params)
	resp := s.handleRequest(req)

	if resp.Error == nil {
		t.Fatal("expected error for unknown tool")
	}
}

func TestUnknownMethod(t *testing.T) {
	s := &Server{weather: &mockWeather{}}
	req := makeRequest("unknown/method", 1, nil)
	resp := s.handleRequest(req)

	if resp.Error == nil {
		t.Fatal("expected error for unknown method")
	}
	if resp.Error.Code != -32601 {
		t.Errorf("expected code -32601, got %d", resp.Error.Code)
	}
}

func assertSuccessText(t *testing.T, resp *JSONRPCResponse, expected string) {
	t.Helper()
	result, ok := resp.Result.(map[string]interface{})
	if !ok {
		t.Fatal("result is not a map")
	}
	content, ok := result["content"].([]map[string]string)
	if !ok {
		t.Fatal("content is not []map[string]string")
	}
	if len(content) == 0 {
		t.Fatal("content is empty")
	}
	if content[0]["text"] != expected {
		t.Errorf("expected text %q, got %q", expected, content[0]["text"])
	}
}
