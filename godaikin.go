package godaikin

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
)

type DaikinClient struct {
	logger Logger
}

// NewClient creates a new client with a custom logger (silent by default)
func NewClient(logger Logger) *DaikinClient {
	if logger == nil {
		logger = NoOpLogger{}
	}
	return &DaikinClient{
		logger: logger,
	}
}

// NewClientWithSlog creates a client with slog
func NewClientWithSlog(slogger *slog.Logger) *DaikinClient {
	if slogger == nil {
		slogger = slog.Default()
	}
	return NewClient(NewSlogAdapter(slogger))
}

func (c *DaikinClient) Connect(deviceIP string, options ...Option) (Appliance, error) {
	c.logger.Info("Connecting to Daikin device", "ip", deviceIP)
	device, err := CreateDaikinDevice(deviceIP, c.logger, options...)
	if err != nil {
		c.logger.Error("Failed to connect to device", "ip", deviceIP, "error", err)
		return nil, err
	}
	c.logger.Info("Successfully connected to device", "ip", deviceIP, "type", device.GetDeviceType())
	return device, nil
}

func (c *DaikinClient) ConnectAndStatus(deviceIP string, options ...Option) (map[string]interface{}, error) {
	c.logger.Info("Connecting and retrieving status", "ip", deviceIP)
	device, err := c.Connect(deviceIP, options...)
	if err != nil {
		return nil, fmt.Errorf("failed to connect to device: %w", err)
	}

	ctx := context.Background()
	c.logger.Debug("Updating device status")
	err = device.UpdateStatus(ctx)
	if err != nil {
		c.logger.Error("Failed to update device status", "ip", deviceIP, "error", err)
		return nil, fmt.Errorf("failed to update device status: %w", err)
	}

	status := make(map[string]interface{})

	status["ip"] = device.GetDeviceIP()
	status["mac"] = device.GetMAC()
	status["power"] = device.GetPowerState()
	status["mode"] = device.GetMode()

	if temp, err := device.GetInsideTemperature(); err == nil {
		status["inside_temperature"] = temp
	}

	if temp, err := device.GetOutsideTemperature(); err == nil {
		status["outside_temperature"] = temp
	}

	if temp, err := device.GetTargetTemperature(); err == nil {
		status["target_temperature"] = temp
	}

	if device.SupportsFanRate() {
		status["fan_rate"] = device.GetFanRate()
	}

	if device.SupportsSwingMode() {
		status["fan_direction"] = device.GetFanDirection()
	}

	support := make(map[string]bool)
	support["fan_rate"] = device.SupportsFanRate()
	support["swing_mode"] = device.SupportsSwingMode()
	support["away_mode"] = device.SupportsAwayMode()
	support["advanced_modes"] = device.SupportsAdvancedModes()
	support["energy_consumption"] = device.SupportsEnergyConsumption()
	status["supports"] = support

	return status, nil
}

// TestConnection tests the connection to a Daikin device without modifying settings
func (c *DaikinClient) TestConnection(deviceIP string, options ...Option) error {
	c.logger.Info("Testing connection to Daikin device", "ip", deviceIP)

	device, err := c.Connect(deviceIP, options...)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}

	c.logger.Info("Successfully connected to device",
		"ip", device.GetDeviceIP(),
		"mac", device.GetMAC())

	ctx := context.Background()
	c.logger.Debug("Retrieving device status for connection test")
	err = device.UpdateStatus(ctx)
	if err != nil {
		c.logger.Error("Failed to get device status", "ip", deviceIP, "error", err)
		return fmt.Errorf("failed to get device status: %w", err)
	}

	// Log basic status
	c.logger.Info("Device status retrieved",
		"power_state", device.GetPowerState(),
		"mode", device.GetMode())

	if temp, err := device.GetInsideTemperature(); err == nil {
		c.logger.Debug("Inside temperature", "temperature", temp, "unit", "celsius")
	}

	if temp, err := device.GetOutsideTemperature(); err == nil {
		c.logger.Debug("Outside temperature", "temperature", temp, "unit", "celsius")
	}

	if temp, err := device.GetTargetTemperature(); err == nil {
		c.logger.Debug("Target temperature", "temperature", temp, "unit", "celsius")
	}

	if device.SupportsFanRate() {
		c.logger.Debug("Fan rate", "rate", device.GetFanRate())
	}

	if device.SupportsSwingMode() {
		c.logger.Debug("Fan direction", "direction", device.GetFanDirection())
	}

	c.logger.Info("Connection test completed successfully", "ip", deviceIP)
	return nil
}

func GetDeviceStatus(deviceIP string, options ...Option) (map[string]interface{}, error) {
	client := NewClient(nil) // silent by default
	return client.ConnectAndStatus(deviceIP, options...)
}

type Config struct {
	Password   string
	Key        string
	UUID       string
	SSLContext *tls.Config
}

type Option func(*Config)

// WithPassword sets the password for SkyFi devices
func WithPassword(password string) Option {
	return func(c *Config) {
		c.Password = password
	}
}

// WithKey sets the key for BRP072C devices
func WithKey(key string) Option {
	return func(c *Config) {
		c.Key = key
	}
}

// WithUUID sets the UUID for BRP072C devices
func WithUUID(uuid string) Option {
	return func(c *Config) {
		c.UUID = uuid
	}
}

// WithSSLContext sets the SSL context for HTTPS devices
func WithSSLContext(sslContext *tls.Config) Option {
	return func(c *Config) {
		c.SSLContext = sslContext
	}
}

func TestDeviceConnection(deviceIP string, options ...Option) error {
	client := NewClient(nil) // silent by default
	return client.TestConnection(deviceIP, options...)
}

func SetDeviceMode(deviceIP, mode string, options ...Option) error {
	client := NewClient(nil) // silent by default
	device, err := client.Connect(deviceIP, options...)
	if err != nil {
		return err
	}

	ctx := context.Background()
	settings := map[string]string{"mode": mode}
	return device.Set(ctx, settings)
}

func SetDeviceTemperature(deviceIP string, temperature float64, options ...Option) error {
	client := NewClient(nil) // silent by default
	device, err := client.Connect(deviceIP, options...)
	if err != nil {
		return err
	}

	ctx := context.Background()
	settings := map[string]string{"stemp": fmt.Sprintf("%.1f", temperature)}
	return device.Set(ctx, settings)
}
