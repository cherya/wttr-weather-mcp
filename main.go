package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
)

const (
	serverName    = "wttr-weather"
	serverVersion = "1.0.0"

	toolGetCurrent  = "get_current_weather"
	toolGetForecast = "get_forecast"
	toolGetDetailed = "get_weather_detailed"
)

type JSONRPCRequest struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      interface{}     `json:"id,omitempty"`
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"`
	Result  interface{} `json:"result,omitempty"`
	Error   *RPCError   `json:"error,omitempty"`
}

type RPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

type WeatherService interface {
	GetCurrent(location string) (string, error)
	GetForecast(location string, days int) (string, error)
	GetDetailed(location string) (string, error)
}

type Server struct {
	weather WeatherService
}

func main() {
	server := &Server{weather: NewWeatherClient()}
	server.run()
}

func (s *Server) run() {
	scanner := bufio.NewScanner(os.Stdin)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var req JSONRPCRequest
		if err := json.Unmarshal(line, &req); err != nil {
			s.sendError(nil, -32700, "Parse error", err.Error())
			continue
		}

		response := s.handleRequest(req)
		if response != nil {
			s.sendResponse(response)
		}
	}
}

func (s *Server) sendResponse(resp *JSONRPCResponse) {
	data, _ := json.Marshal(resp)
	fmt.Println(string(data))
}

func (s *Server) sendError(id interface{}, code int, message string, data interface{}) {
	resp := &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
			Data:    data,
		},
	}
	s.sendResponse(resp)
}

func (s *Server) handleRequest(req JSONRPCRequest) *JSONRPCResponse {
	switch req.Method {
	case "initialize":
		return s.handleInitialize(req)
	case "initialized":
		return nil
	case "tools/list":
		return s.handleToolsList(req)
	case "tools/call":
		return s.handleToolsCall(req)
	default:
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    -32601,
				Message: "Method not found",
			},
		}
	}
}

func (s *Server) handleInitialize(req JSONRPCRequest) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"serverInfo": map[string]string{
				"name":    serverName,
				"version": serverVersion,
			},
			"capabilities": map[string]interface{}{
				"tools": map[string]interface{}{},
			},
		},
	}
}

func (s *Server) handleToolsList(req JSONRPCRequest) *JSONRPCResponse {
	tools := []map[string]interface{}{
		{
			"name":        toolGetCurrent,
			"description": "Get current weather conditions for a location (one-line summary)",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"location": map[string]interface{}{
						"type":        "string",
						"description": "City or location name (e.g. \"London\", \"New York\", \"Tokyo\")",
					},
				},
				"required": []string{"location"},
			},
		},
		{
			"name":        toolGetForecast,
			"description": "Get weather forecast for a location (text format with ASCII art)",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"location": map[string]interface{}{
						"type":        "string",
						"description": "City or location name (e.g. \"London\", \"New York\", \"Tokyo\")",
					},
					"days": map[string]interface{}{
						"type":        "integer",
						"description": "Number of forecast days (1-3, default: 3)",
						"default":     3,
						"minimum":     1,
						"maximum":     3,
					},
				},
				"required": []string{"location"},
			},
		},
		{
			"name":        toolGetDetailed,
			"description": "Get detailed weather data in JSON format (temperature, humidity, wind, UV index, etc.)",
			"inputSchema": map[string]interface{}{
				"type": "object",
				"properties": map[string]interface{}{
					"location": map[string]interface{}{
						"type":        "string",
						"description": "City or location name (e.g. \"London\", \"New York\", \"Tokyo\")",
					},
				},
				"required": []string{"location"},
			},
		},
	}

	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result: map[string]interface{}{
			"tools": tools,
		},
	}
}

func (s *Server) handleToolsCall(req JSONRPCRequest) *JSONRPCResponse {
	var params struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}

	if err := json.Unmarshal(req.Params, &params); err != nil {
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    -32602,
				Message: "Invalid params",
				Data:    err.Error(),
			},
		}
	}

	switch params.Name {
	case toolGetCurrent:
		return s.callGetCurrent(req.ID, params.Arguments)
	case toolGetForecast:
		return s.callGetForecast(req.ID, params.Arguments)
	case toolGetDetailed:
		return s.callGetDetailed(req.ID, params.Arguments)
	default:
		return &JSONRPCResponse{
			JSONRPC: "2.0",
			ID:      req.ID,
			Error: &RPCError{
				Code:    -32602,
				Message: "Unknown tool: " + params.Name,
			},
		}
	}
}

func (s *Server) callGetCurrent(id interface{}, args json.RawMessage) *JSONRPCResponse {
	var input struct {
		Location string `json:"location"`
	}

	if err := json.Unmarshal(args, &input); err != nil {
		return s.paramError(id, "Invalid arguments", err.Error())
	}

	if input.Location == "" {
		return s.paramError(id, "location is required", nil)
	}

	result, err := s.weather.GetCurrent(input.Location)
	if err != nil {
		return s.errorResponse(id, err)
	}

	return s.successResponse(id, result)
}

func (s *Server) callGetForecast(id interface{}, args json.RawMessage) *JSONRPCResponse {
	var input struct {
		Location string `json:"location"`
		Days     int    `json:"days"`
	}
	input.Days = 3

	if err := json.Unmarshal(args, &input); err != nil {
		return s.paramError(id, "Invalid arguments", err.Error())
	}

	if input.Location == "" {
		return s.paramError(id, "location is required", nil)
	}

	if input.Days < 1 || input.Days > 3 {
		input.Days = 3
	}

	result, err := s.weather.GetForecast(input.Location, input.Days)
	if err != nil {
		return s.errorResponse(id, err)
	}

	return s.successResponse(id, result)
}

func (s *Server) callGetDetailed(id interface{}, args json.RawMessage) *JSONRPCResponse {
	var input struct {
		Location string `json:"location"`
	}

	if err := json.Unmarshal(args, &input); err != nil {
		return s.paramError(id, "Invalid arguments", err.Error())
	}

	if input.Location == "" {
		return s.paramError(id, "location is required", nil)
	}

	result, err := s.weather.GetDetailed(input.Location)
	if err != nil {
		return s.errorResponse(id, err)
	}

	return s.successResponse(id, result)
}

func (s *Server) successResponse(id interface{}, text string) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]interface{}{
			"content": []map[string]string{
				{"type": "text", "text": text},
			},
		},
	}
}

func (s *Server) errorResponse(id interface{}, err error) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Result: map[string]interface{}{
			"content": []map[string]string{
				{"type": "text", "text": fmt.Sprintf("Error: %v", err)},
			},
			"isError": true,
		},
	}
}

func (s *Server) paramError(id interface{}, message string, data interface{}) *JSONRPCResponse {
	return &JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    -32602,
			Message: message,
			Data:    data,
		},
	}
}
