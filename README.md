# GoDaikin

A Go library for controlling Daikin HVAC systems, ported from the Python [pydaikin](https://github.com/fredrike/pydaikin) library. This library provides a simple and efficient way to interact with Daikin air conditioning units over HTTP.

```bash
go get github.com/jattkaim/godaikin
```

## Quick Start

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

## Contributing

Contributions are welcome! Please feel free to submit issues and pull requests.

## License

This project is licensed under the MIT License.
