package godaikin

import (
	"context"
	"crypto/tls"
	"fmt"
	"log/slog"
)

type ClientOption func(*DaikinClient)

type DaikinClient struct {
	logger Logger
}

func NewClient(opts ...ClientOption) *DaikinClient {
	client := &DaikinClient{
		logger: NoOpLogger{},
	}

	for _, opt := range opts {
		opt(client)
	}

	return client
}

func WithLogger(slogger *slog.Logger) ClientOption {
	return func(c *DaikinClient) {
		c.logger = NewSlogAdapter(slogger)
	}
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

func (c *DaikinClient) TestConnection(deviceIP string, options ...Option) error {
	c.logger.Info("Testing connection to Daikin device", "ip", deviceIP)

	device, err := c.Connect(deviceIP, options...)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}

	ctx := context.Background()
	c.logger.Debug("Retrieving device status for connection test")
	err = device.UpdateStatus(ctx)
	if err != nil {
		c.logger.Error("Failed to get device status", "ip", deviceIP, "error", err)
		return fmt.Errorf("failed to get device status: %w", err)
	}

	c.logger.Info("Device status retrieved",
		"power_state", device.GetPowerState(),
		"mode", device.GetMode())
	return nil
}

type Config struct {
	Password   string
	Key        string
	UUID       string
	SSLContext *tls.Config
}

type Option func(*Config)

func WithPassword(password string) Option {
	return func(c *Config) {
		c.Password = password
	}
}

func WithKey(key string) Option {
	return func(c *Config) {
		c.Key = key
	}
}

func WithUUID(uuid string) Option {
	return func(c *Config) {
		c.UUID = uuid
	}
}

func WithSSLContext(sslContext *tls.Config) Option {
	return func(c *Config) {
		c.SSLContext = sslContext
	}
}
