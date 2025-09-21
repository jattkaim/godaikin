package main

import (
	"fmt"
	"log/slog"
	"os"

	"github.com/jattkaim/godaikin"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run main.go <ip> [--verbose]")
		fmt.Println("Examples:")
		fmt.Println("  go run main.go 192.168.1.100          # Silent mode")
		fmt.Println("  go run main.go 192.168.1.100 --verbose # With detailed logging")
		os.Exit(1)
	}

	var client *godaikin.DaikinClient

	// Check if verbose logging is requested
	verbose := len(os.Args) > 2 && os.Args[2] == "--verbose"

	if verbose {
		// Create client with verbose logging
		logger := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
			Level: slog.LevelDebug,
		}))
		client = godaikin.NewClientWithSlog(logger)
		fmt.Println("🔍 Verbose logging enabled")
	} else {
		// Create silent client (default)
		client = godaikin.NewClient(nil)
	}

	fmt.Printf("🏠 Connecting to Daikin device at %s...\n", os.Args[1])
	device, err := client.Connect(os.Args[1])
	if err != nil {
		fmt.Printf("❌ Error: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✅ Connected to %s device\n", device.GetDeviceType())
	fmt.Printf("📍 IP: %s\n", device.GetDeviceIP())
	fmt.Printf("🔧 MAC: %s\n", device.GetMAC())
	fmt.Printf("⚡ Power: %t\n", device.GetPowerState())
	fmt.Printf("🎛️  Mode: %s\n", device.GetMode())

	if temp, err := device.GetTargetTemperature(); err == nil {
		fmt.Printf("🎯 Target: %.1f°C\n", temp)
	}

	if temp, err := device.GetInsideTemperature(); err == nil {
		fmt.Printf("🌡️  Inside: %.1f°C\n", temp)
	}

	if temp, err := device.GetOutsideTemperature(); err == nil {
		fmt.Printf("🌡️  Outside: %.1f°C\n", temp)
	}

	if device.SupportsFanRate() {
		fmt.Printf("💨 Fan Rate: %s\n", device.GetFanRate())
	}

	if device.SupportsSwingMode() {
		fmt.Printf("🌀 Fan Direction: %s\n", device.GetFanDirection())
	}
}
