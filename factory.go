package godaikin

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
)

// CreateDaikinDevice creates the appropriate Daikin device based on auto-detection
func CreateDaikinDevice(deviceID string, logger Logger, options ...Option) (Appliance, error) {
	config := &Config{}

	// Apply options
	for _, opt := range options {
		opt(config)
	}

	// Extract IP and port from deviceID
	deviceIP, devicePort := extractIPPort(deviceID)

	ctx := context.Background()

	// If password is provided, it's a SkyFi device
	if config.Password != "" {
		logger.Info("Detected SkyFi device", "ip", deviceIP, "password_provided", true)
		device := NewDaikinSkyFi(deviceIP, config.Password, logger)
		if devicePort != 0 && devicePort != 2000 {
			device.BaseURL = fmt.Sprintf("http://%s:%d", deviceIP, devicePort)
			logger.Debug("Using custom port for SkyFi", "port", devicePort)
		}
		err := device.Init(ctx)
		if err != nil {
			logger.Error("Failed to initialize SkyFi device", "error", err)
			return nil, fmt.Errorf("failed to initialize SkyFi device: %w", err)
		}
		logger.Info("Successfully initialized SkyFi device", "ip", deviceIP)
		return device, nil
	}

	// If key is provided, it's a BRP072C device
	if config.Key != "" {
		logger.Info("Detected BRP072C device", "ip", deviceIP, "key_provided", true)
		device := NewDaikinBRP072C(deviceIP, config.Key, config.UUID, logger)
		if devicePort != 0 && devicePort != 443 {
			device.BaseURL = fmt.Sprintf("https://%s:%d", deviceIP, devicePort)
			logger.Debug("Using custom port for BRP072C", "port", devicePort)
		}
		err := device.Init(ctx)
		if err != nil {
			logger.Error("Failed to initialize BRP072C device", "error", err)
			return nil, fmt.Errorf("failed to initialize BRP072C device: %w", err)
		}
		logger.Info("Successfully initialized BRP072C device", "ip", deviceIP)
		return device, nil
	}

	// Special case for BRP069, AirBase, and BRP firmware 2.8.0

	// First try to check if it's firmware 2.8.0
	logger.Debug("Trying connection to firmware 2.8.0", "ip", deviceIP)
	if device, err := tryBRP084Device(deviceIP, devicePort, logger); err == nil {
		logger.Info("Successfully connected to firmware 2.8.0 device", "ip", deviceIP)
		// Initialize mode to "off" if we couldn't read it
		if mode := device.GetMode(); mode == "" || mode == "unknown" {
			logger.Debug("Initializing mode to off for device with unknown mode")
			if baseDevice, ok := device.(*BaseAppliance); ok {
				baseDevice.Values.Set("mode", "off")
				baseDevice.Values.Set("pow", "0")
			}
		}
		return device, nil
	} else {
		logger.Debug("Not a firmware 2.8.0 device", "error", err)
	}

	// Try BRP069
	logger.Debug("Trying connection to BRP069", "ip", deviceIP)
	if device, err := tryBRP069Device(deviceIP, devicePort, logger); err == nil {
		logger.Info("Successfully connected to BRP069 device", "ip", deviceIP)
		return device, nil
	} else {
		logger.Debug("Falling back to AirBase", "error", err)
	}

	// Fallback to AirBase
	logger.Debug("Trying AirBase connection", "ip", deviceIP)
	device := NewDaikinAirBase(deviceIP, logger)
	if devicePort != 0 && devicePort != 80 {
		logger.Debug("Using custom port for AirBase", "port", devicePort)
		device.BaseURL = fmt.Sprintf("http://%s:%d", deviceIP, devicePort)
	}

	err := device.Init(ctx)
	if err != nil {
		logger.Error("Failed to initialize AirBase device", "error", err)
		return nil, fmt.Errorf("failed to initialize AirBase device: %w", err)
	}

	// Check if device was successfully initialized
	if mode := device.GetMode(); mode == "" {
		logger.Error("Device not supported or failed to initialize", "device_id", deviceID)
		return nil, fmt.Errorf("error creating device, %s is not supported", deviceID)
	}

	logger.Info("Successfully created Daikin device", "type", fmt.Sprintf("%T", device), "ip", deviceIP)
	return device, nil
}

// tryBRP084Device attempts to create firmware 2.8.0 device
func tryBRP084Device(deviceIP string, devicePort int, logger Logger) (Appliance, error) {
	device := NewDaikinBRP084(deviceIP, logger)

	// If we have a specific port from discovery, set it in the base_url
	if devicePort != 0 && devicePort != 80 {
		logger.Debug("Using custom port for BRP084", "port", devicePort)
		device.BaseURL = fmt.Sprintf("http://%s:%d", deviceIP, devicePort)
		device.URL = fmt.Sprintf("%s/dsiot/multireq", device.BaseURL)
	}

	ctx := context.Background()

	// Try to initialize the device by updating status
	err := device.UpdateStatus(ctx)
	if err != nil {
		return nil, fmt.Errorf("not a BRP084 device: %w", err)
	}

	if device.Values.Len() == 0 {
		return nil, fmt.Errorf("empty values from BRP084 device")
	}

	return device, nil
}

// tryBRP069Device attempts to create BRP069 device
func tryBRP069Device(deviceIP string, devicePort int, logger Logger) (Appliance, error) {
	device := NewDaikinBRP069(deviceIP, logger)

	// If we have a specific port from discovery, set it in the base_url
	if devicePort != 0 && devicePort != 80 {
		logger.Debug("Using custom port for BRP069", "port", devicePort)
		device.BaseURL = fmt.Sprintf("http://%s:%d", deviceIP, devicePort)
	}

	ctx := context.Background()

	// Try to update status with first HTTP resource
	err := device.updateStatusWithResources(ctx, []string{"common/basic_info"})
	if err != nil {
		return nil, fmt.Errorf("not a BRP069 device: %w", err)
	}

	if device.Values.Len() == 0 {
		return nil, fmt.Errorf("empty values from BRP069 device")
	}

	// Initialize the device
	err = device.Init(ctx)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize BRP069: %w", err)
	}

	return device, nil
}

// extractIPPort extracts IP address and port
func extractIPPort(deviceID string) (string, int) {
	// Check if there's a port specified in the device_id
	portRegex := regexp.MustCompile(`^(.+):(\d+)$`)
	if matches := portRegex.FindStringSubmatch(deviceID); matches != nil {
		ip := matches[1]
		port, err := strconv.Atoi(matches[2])
		if err != nil {
			return deviceID, 0
		}
		return ip, port
	}

	// TODO: Try to look up device in discovery
	// For now, just return the device_id with no port
	return deviceID, 0
}
