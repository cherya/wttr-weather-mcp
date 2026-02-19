# wttr-weather-mcp

An MCP (Model Context Protocol) server that provides weather data via [wttr.in](https://github.com/chubin/wttr.in) — a console-oriented weather forecast service that supports multiple output formats.

wttr.in fetches data from the WorldWeatherOnline API and presents it as plain text, ANSI art, or structured JSON. This MCP server wraps the wttr.in HTTP API into three tools accessible over the MCP stdio protocol.

## Tools

- **get_current_weather** — one-line summary of current conditions (temperature, feels like, humidity, wind)
- **get_forecast** — text forecast for 1-3 days with ASCII art
- **get_weather_detailed** — structured JSON weather data (temperature, humidity, wind, UV index, etc.)

All tools require a `location` parameter (city name, e.g. "London", "New York", "Tokyo").

## Installation

```bash
go install github.com/cherya/wttr-weather-mcp@latest
```

The binary will be placed in `$GOPATH/bin/` (usually `~/go/bin/`).

## Build from source

```bash
git clone https://github.com/cherya/wttr-weather-mcp.git
cd wttr-weather-mcp
go build -o wttr-weather-mcp .
```

## Run tests

```bash
go test -v ./...
```

## Usage with Claude Code

Add to your MCP settings:

```json
{
  "mcpServers": {
    "wttr-weather": {
      "command": "wttr-weather-mcp"
    }
  }
}
```

Make sure `$GOPATH/bin` is in your `PATH`, or use the full path to the binary.

## Dependencies

None beyond the Go standard library.
