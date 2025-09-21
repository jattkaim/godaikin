package godaikin

import (
	"context"
	"fmt"
	"net/http"
	"strconv"
	"time"
)

// Appliance represents a Daikin HVAC appliance
type Appliance interface {
	Init(ctx context.Context) error
	UpdateStatus(ctx context.Context) error
	Set(ctx context.Context, settings map[string]string) error

	GetValues() *Values
	GetDeviceIP() string
	GetDeviceType() string
	GetMAC() string

	GetInsideTemperature() (float64, error)
	GetOutsideTemperature() (float64, error)
	GetTargetTemperature() (float64, error)

	GetMode() string
	GetPowerState() bool
	GetFanRate() string
	GetFanDirection() string

	SupportsFanRate() bool
	SupportsSwingMode() bool
	SupportsAwayMode() bool
	SupportsAdvancedModes() bool
	SupportsEnergyConsumption() bool

	SetHoliday(ctx context.Context, mode string) error
	SetStreamer(ctx context.Context, mode string) error
	SetAdvancedMode(ctx context.Context, mode, value string) error
}

// BaseAppliance provides common functionality for all Daikin devices
type BaseAppliance struct {
	DeviceIP   string
	BaseURL    string
	Values     *Values
	HTTPClient *http.Client
	Headers    map[string]string
	Logger     Logger

	// Translations for converting between Daikin values and human-readable values
	Translations map[string]map[string]string

	HTTPResources []string
	InfoResources []string

	MaxConcurrentRequests int
}

func NewBaseAppliance(deviceIP string, logger Logger) *BaseAppliance {
	if logger == nil {
		logger = NoOpLogger{}
	}
	return &BaseAppliance{
		DeviceIP:              deviceIP,
		BaseURL:               fmt.Sprintf("http://%s", deviceIP),
		Values:                NewValues(),
		HTTPClient:            &http.Client{Timeout: 30 * time.Second},
		Headers:               make(map[string]string),
		Logger:                logger,
		Translations:          make(map[string]map[string]string),
		MaxConcurrentRequests: 4,
	}
}

func (b *BaseAppliance) GetValues() *Values {
	return b.Values
}

func (b *BaseAppliance) GetDeviceIP() string {
	return b.DeviceIP
}

func (b *BaseAppliance) GetDeviceType() string {
	return "BaseAppliance"
}

func (b *BaseAppliance) GetMAC() string {
	if mac, exists := b.Values.Get("mac"); exists {
		return formatMAC(mac)
	}
	return b.DeviceIP
}

func (b *BaseAppliance) GetInsideTemperature() (float64, error) {
	return b.parseFloat("htemp")
}

func (b *BaseAppliance) GetOutsideTemperature() (float64, error) {
	return b.parseFloat("otemp")
}

func (b *BaseAppliance) GetTargetTemperature() (float64, error) {
	return b.parseFloat("stemp")
}

func (b *BaseAppliance) GetMode() string {
	// Check if device is powered off first
	if pow, exists := b.Values.Get("pow"); exists && pow == "0" {
		return "off"
	}

	if mode, exists := b.Values.Get("mode"); exists {
		return b.translateValue("mode", mode)
	}
	return "unknown"
}

func (b *BaseAppliance) GetPowerState() bool {
	if pow, exists := b.Values.Get("pow"); exists {
		return pow == "1"
	}
	return false
}

func (b *BaseAppliance) GetFanRate() string {
	if rate, exists := b.Values.Get("f_rate"); exists {
		return b.translateValue("f_rate", rate)
	}
	return "unknown"
}

func (b *BaseAppliance) GetFanDirection() string {
	if dir, exists := b.Values.Get("f_dir"); exists {
		return b.translateValue("f_dir", dir)
	}
	return "unknown"
}

func (b *BaseAppliance) SupportsFanRate() bool {
	return b.Values.Has("f_rate")
}

func (b *BaseAppliance) SupportsSwingMode() bool {
	return b.Values.Has("f_dir")
}

func (b *BaseAppliance) SupportsAwayMode() bool {
	return b.Values.Has("en_hol")
}

func (b *BaseAppliance) SupportsAdvancedModes() bool {
	return b.Values.Has("adv")
}

func (b *BaseAppliance) SupportsEnergyConsumption() bool {
	// Check if we have energy consumption data
	return b.Values.Has("datas") || b.Values.Has("curr_day_cool") || b.Values.Has("curr_day_heat")
}

func (b *BaseAppliance) parseFloat(key string) (float64, error) {
	if value, exists := b.Values.Get(key); exists && value != "" && value != "-" && value != "--" {
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f, nil
		}
	}
	return 0, fmt.Errorf("unable to parse float value for key %s", key)
}

func (b *BaseAppliance) translateValue(dimension, value string) string {
	if translations, exists := b.Translations[dimension]; exists {
		if translated, exists := translations[value]; exists {
			return translated
		}
	}
	return value
}

func (b *BaseAppliance) reverseTranslateValue(dimension, value string) string {
	if translations, exists := b.Translations[dimension]; exists {
		for daikinValue, humanValue := range translations {
			if humanValue == value {
				return daikinValue
			}
		}
	}
	return value
}

func formatMAC(mac string) string {
	if len(mac) != 12 {
		return mac
	}

	result := ""
	for i := 0; i < len(mac); i += 2 {
		if i > 0 {
			result += ":"
		}
		result += mac[i : i+2]
	}
	return result
}

func (b *BaseAppliance) getResource(ctx context.Context, path string, params map[string]string) (map[string]string, error) {
	url := fmt.Sprintf("%s/%s", b.BaseURL, path)

	b.Logger.Debug("Making HTTP request", "url", url, "params", params)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		b.Logger.Error("Failed to create HTTP request", "url", url, "error", err)
		return nil, NewConnectionError("failed to create request", err)
	}

	for key, value := range b.Headers {
		req.Header.Set(key, value)
	}

	if params != nil {
		q := req.URL.Query()
		for key, value := range params {
			q.Add(key, value)
		}
		req.URL.RawQuery = q.Encode()
	}

	resp, err := b.HTTPClient.Do(req)
	if err != nil {
		b.Logger.Error("HTTP request failed", "url", url, "error", err)
		return nil, NewConnectionError("failed to make request", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusForbidden {
		b.Logger.Warn("HTTP 403 Forbidden response", "url", url)
		return nil, NewAuthenticationError("HTTP 403 Forbidden", nil)
	}

	if resp.StatusCode == http.StatusNotFound {
		b.Logger.Debug("HTTP 404 Not Found response", "url", url)
		return make(map[string]string), nil
	}

	if resp.StatusCode != http.StatusOK {
		b.Logger.Error("Unexpected HTTP status", "url", url, "status", resp.StatusCode)
		return nil, NewConnectionError(fmt.Sprintf("unexpected HTTP status: %d", resp.StatusCode), nil)
	}

	body := make([]byte, 4096) // Reasonable buffer size for Daikin responses
	n, err := resp.Body.Read(body)
	if err != nil && n == 0 {
		b.Logger.Error("Failed to read response body", "url", url, "error", err)
		return nil, NewConnectionError("failed to read response body", err)
	}

	b.Logger.Debug("HTTP response received", "url", url, "bytes", n, "status", resp.StatusCode)
	return parseResponse(string(body[:n]))
}

func (b *BaseAppliance) Init(ctx context.Context) error {
	return fmt.Errorf("Init method must be implemented by specific device type")
}

func (b *BaseAppliance) UpdateStatus(ctx context.Context) error {
	return fmt.Errorf("UpdateStatus method must be implemented by specific device type")
}

func (b *BaseAppliance) Set(ctx context.Context, settings map[string]string) error {
	return fmt.Errorf("Set method must be implemented by specific device type")
}

func (b *BaseAppliance) SetHoliday(ctx context.Context, mode string) error {
	return fmt.Errorf("SetHoliday not supported by this device type")
}

func (b *BaseAppliance) SetStreamer(ctx context.Context, mode string) error {
	return fmt.Errorf("SetStreamer not supported by this device type")
}

func (b *BaseAppliance) SetAdvancedMode(ctx context.Context, mode, value string) error {
	return fmt.Errorf("SetAdvancedMode not supported by this device type")
}
