# GoDaikin

A Go library for controlling Daikin HVAC systems, ported from the Python pydaikin library. This library provides a simple and efficient way to interact with Daikin air conditioning units over HTTP.

## Features

- **Multiple Device Support**: Works with various Daikin WiFi modules including:
  - BRP069Axx/BRP069Bxx/BRP072Axx (standard HTTP API)
  - BRP15B61 (AirBase)
  - BRP072B/Cxx (HTTPS with key authentication)
  - BRP084 devices with firmware version 2.8.0
  - SKYFi (password-based authentication)

- **Auto-Detection**: Automatically detects device type and uses appropriate communication protocol
- **Thread-Safe**: Built with Go's concurrency patterns in mind
- **Simple API**: Easy-to-use interface for common operations
- **Comprehensive Status**: Get detailed information about temperature, mode, fan settings, and more
- **Flexible Logging**: Silent by default with optional structured logging support

## Installation

```bash
go get github.com/godaikin
```

## Logging

The `godaikin` library uses dependency injection for logging, giving you full control over how logging is handled. By default, the library is **silent** - it won't produce any log output unless you explicitly provide a logger.

### Logging Options

1. **Silent Mode (Default)**: No logging output
2. **Built-in slog**: Use Go's standard `log/slog` package
3. **Custom Logger**: Implement the `Logger` interface with your preferred logging library

### Logger Interface

```go
type Logger interface {
    Debug(msg string, args ...any)
    Info(msg string, args ...any)
    Warn(msg string, args ...any)
    Error(msg string, args ...any)
}
```

## Quick Start

### Basic Connection Test (Silent Mode)

```go
package main

import (
    "fmt"
    "os"
    "github.com/godaikin"
)

func main() {
    // Silent by default - no logging output
    client := godaikin.NewClient(nil)
    
    device, err := client.Connect("192.168.1.100")
    if err != nil {
        fmt.Printf("Error: %v\n", err)
        os.Exit(1)
    }
    
    fmt.Printf("Power: %t\n", device.GetPowerState())
    fmt.Printf("Mode: %s\n", device.GetMode())
}
```

### With Structured Logging (slog)

```go
package main

import (
    "log/slog"
    "os"
    "github.com/godaikin"
)

func main() {
    // Create a structured logger
    logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
        Level: slog.LevelInfo, // Set to LevelDebug for more details
    }))
    
    // Use the logger with your client
    client := godaikin.NewClientWithSlog(logger)
    
    device, err := client.Connect("192.168.1.100")
    if err != nil {
        logger.Error("Connection failed", "error", err)
        os.Exit(1)
    }
    
    // The library will now log connection details, API calls, etc.
    fmt.Printf("Power: %t\n", device.GetPowerState())
}
```

### With Custom Logger (e.g., logrus, zap)

```go
package main

import (
    "github.com/sirupsen/logrus"
    "github.com/godaikin"
)

// LogrusAdapter adapts logrus to godaikin's Logger interface
type LogrusAdapter struct {
    logger *logrus.Logger
}

func (l *LogrusAdapter) Debug(msg string, args ...any) { l.logger.Debug(msg, args) }
func (l *LogrusAdapter) Info(msg string, args ...any)  { l.logger.Info(msg, args) }
func (l *LogrusAdapter) Warn(msg string, args ...any)  { l.logger.Warn(msg, args) }
func (l *LogrusAdapter) Error(msg string, args ...any) { l.logger.Error(msg, args) }

func main() {
    logrusLogger := logrus.New()
    logrusLogger.SetLevel(logrus.InfoLevel)
    
    adapter := &LogrusAdapter{logger: logrusLogger}
    client := godaikin.NewClient(adapter)
    
    device, err := client.Connect("192.168.1.100")
    // ... rest of your code
}
```

## Device Types and Authentication

### Standard Devices (BRP069)
```go
client := godaikin.NewClient(nil)
device, err := client.Connect("192.168.1.100")
```

### SkyFi Devices (Password Authentication)
```go
client := godaikin.NewClient(nil)
device, err := client.Connect("192.168.1.100", 
    godaikin.WithPassword("your_password"))
```

### BRP072C Devices (Key + UUID Authentication)
```go
client := godaikin.NewClient(nil)
device, err := client.Connect("192.168.1.100",
    godaikin.WithKey("your_key"),
    godaikin.WithUUID("your_uuid"))
```

## Basic Operations

### Get Device Status
```go
fmt.Printf("Device Type: %s\n", device.GetDeviceType())
fmt.Printf("IP Address: %s\n", device.GetDeviceIP())
fmt.Printf("MAC Address: %s\n", device.GetMAC())
fmt.Printf("Power State: %t\n", device.GetPowerState())
fmt.Printf("Mode: %s\n", device.GetMode())

if temp, err := device.GetInsideTemperature(); err == nil {
    fmt.Printf("Inside Temperature: %.1f°C\n", temp)
}

if temp, err := device.GetTargetTemperature(); err == nil {
    fmt.Printf("Target Temperature: %.1f°C\n", temp)
}
```

### Control Device
```go
ctx := context.Background()

// Set temperature
err := device.Set(ctx, map[string]string{
    "stemp": "24.0",  // Set target temperature to 24°C
})

// Change mode
err = device.Set(ctx, map[string]string{
    "mode": "cool",   // Set to cooling mode
})

// Turn off
err = device.Set(ctx, map[string]string{
    "pow": "0",       // Power off
})
```

### Advanced Features (if supported)
```go
// Holiday mode (if supported)
if device.SupportsAwayMode() {
    err := device.SetHoliday(ctx, "on")
}

// Streamer mode (if supported)
err := device.SetStreamer(ctx, "on")

// Check feature support
fmt.Printf("Supports Fan Rate: %t\n", device.SupportsFanRate())
fmt.Printf("Supports Swing Mode: %t\n", device.SupportsSwingMode())
fmt.Printf("Supports Away Mode: %t\n", device.SupportsAwayMode())
```

## Logging Levels

When using structured logging, the library uses different log levels appropriately:

- **Debug**: Detailed technical information (HTTP requests, response parsing, device detection)
- **Info**: Important operations (connections, device control, status updates)
- **Warn**: Recoverable issues (failed resource requests, missing features)
- **Error**: Critical failures (connection failures, authentication errors)

### Example Log Output with Debug Level

```
time=2024-01-15T10:30:45.123Z level=INFO msg="Connecting to Daikin device" ip=192.168.1.100
time=2024-01-15T10:30:45.124Z level=DEBUG msg="Trying connection to firmware 2.8.0" ip=192.168.1.100
time=2024-01-15T10:30:45.150Z level=DEBUG msg="Making HTTP request" url="http://192.168.1.100/dsiot/multireq"
time=2024-01-15T10:30:45.250Z level=INFO msg="Successfully connected to firmware 2.8.0 device" ip=192.168.1.100
time=2024-01-15T10:30:45.251Z level=INFO msg="Successfully connected to device" ip=192.168.1.100 type=BRP084
```

## Testing

Run the example:
```bash
cd examples
go run main.go 192.168.1.100 --verbose
```

## Error Handling

The library provides specific error types for different failure scenarios:

```go
device, err := client.Connect("192.168.1.100")
if err != nil {
    switch {
    case errors.Is(err, godaikin.ErrConnection):
        log.Printf("Network/connection error: %v", err)
    case errors.Is(err, godaikin.ErrAuthentication):
        log.Printf("Authentication failed: %v", err)
    default:
        log.Printf("Other error: %v", err)
    }
}
```

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

## License

This project is licensed under the MIT License.