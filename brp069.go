package godaikin

import (
	"context"
	"fmt"
	"strconv"
	"time"
)

// DaikinBRP069 represents a Daikin BRP069[A-B]xx device
type DaikinBRP069 struct {
	*BaseAppliance
}

// NewDaikinBRP069 creates a new BRP069 device instance
func NewDaikinBRP069(deviceIP string, logger Logger) *DaikinBRP069 {
	base := NewBaseAppliance(deviceIP, logger)

	// Set device-specific translations
	base.Translations = map[string]map[string]string{
		"mode": {
			"2":  "dry",
			"3":  "cool",
			"4":  "hot",
			"6":  "fan",
			"0":  "auto",
			"1":  "auto-1",
			"7":  "auto-7",
			"10": "off",
		},
		"f_rate": {
			"A": "auto",
			"B": "silence",
			"3": "1",
			"4": "2",
			"5": "3",
			"6": "4",
			"7": "5",
		},
		"f_dir": {
			"0": "off",
			"1": "vertical",
			"2": "horizontal",
			"3": "3d",
		},
		"en_hol": {
			"0": "off",
			"1": "on",
		},
		"en_streamer": {
			"0": "off",
			"1": "on",
		},
		"adv": {
			"":      "off",
			"2":     "powerful",
			"2/13":  "powerful streamer",
			"12":    "econo",
			"12/13": "econo streamer",
			"13":    "streamer",
		},
		"spmode_kind": {
			"0": "streamer",
			"1": "powerful",
			"2": "econo",
		},
		"spmode": {
			"0": "off",
			"1": "on",
		},
	}

	// Set HTTP resources
	base.HTTPResources = []string{
		"common/basic_info",
		"common/get_remote_method",
		"aircon/get_sensor_info",
		"aircon/get_model_info",
		"aircon/get_control_info",
		"aircon/get_target",
		"aircon/get_price",
		"common/get_holiday",
		"common/get_notify",
		"aircon/get_day_power_ex",
		"aircon/get_week_power",
		"aircon/get_year_power",
		"common/get_datetime",
	}

	// Set info resources (used for regular updates)
	base.InfoResources = []string{
		"aircon/get_sensor_info",
		"aircon/get_control_info",
	}

	// BRP069 only allows 1 concurrent request
	base.MaxConcurrentRequests = 1

	return &DaikinBRP069{BaseAppliance: base}
}

func (d *DaikinBRP069) GetDeviceType() string {
	return "BRP069"
}

// Init initializes the BRP069 device
func (d *DaikinBRP069) Init(ctx context.Context) error {
	// Auto-set clock first
	if err := d.autoSetClock(ctx); err != nil {
		d.Logger.Warn("Failed to auto-set clock", "error", err)
	}

	// Update status with basic info first
	if err := d.updateStatusWithResources(ctx, []string{"common/basic_info"}); err != nil {
		return fmt.Errorf("failed to get basic info: %w", err)
	}

	// Then update with all other resources
	if err := d.updateStatusWithResources(ctx, d.HTTPResources[1:]); err != nil {
		return fmt.Errorf("failed to get device status: %w", err)
	}

	return nil
}

// UpdateStatus updates the device status using info resources
func (d *DaikinBRP069) UpdateStatus(ctx context.Context) error {
	resources := d.InfoResources

	// Add energy resources if supported
	if d.SupportsEnergyConsumption() {
		resources = append(resources, "aircon/get_day_power_ex", "aircon/get_week_power")
	}

	return d.updateStatusWithResources(ctx, resources)
}

// updateStatusWithResources updates status using specified resources
func (d *DaikinBRP069) updateStatusWithResources(ctx context.Context, resources []string) error {
	// Filter resources that need to be updated
	var resourcesToUpdate []string
	for _, resource := range resources {
		if d.Values.ShouldResourceBeUpdated(resource) {
			resourcesToUpdate = append(resourcesToUpdate, resource)
		}
	}

	if len(resourcesToUpdate) == 0 {
		return nil
	}

	d.Logger.Debug("Updating device resources", "resources", resourcesToUpdate)

	// Update each resource
	for _, resource := range resourcesToUpdate {
		data, err := d.getResource(ctx, resource, nil)
		if err != nil {
			d.Logger.Error("Error updating resource", "resource", resource, "error", err)
			continue
		}

		// Apply special parsing for BRP069 (handle swing mode from separate parameters)
		data = d.parseSpecialFields(data)

		d.Values.UpdateByResource(resource, data)
	}

	return nil
}

// parseSpecialFields handles special field parsing for BRP069
func (d *DaikinBRP069) parseSpecialFields(data map[string]string) map[string]string {
	// Handle swing mode translation from f_dir_ud and f_dir_lr to f_dir
	if udDir, hasUD := data["f_dir_ud"]; hasUD {
		if lrDir, hasLR := data["f_dir_lr"]; hasLR {
			switch {
			case udDir == "0" && lrDir == "0":
				data["f_dir"] = "0"
			case udDir == "S" && lrDir == "0":
				data["f_dir"] = "1"
			case udDir == "0" && lrDir == "S":
				data["f_dir"] = "2"
			case udDir == "S" && lrDir == "S":
				data["f_dir"] = "3"
			}
		}
	}

	return data
}

// Set sets device parameters
func (d *DaikinBRP069) Set(ctx context.Context, settings map[string]string) error {
	// Update settings first
	if err := d.updateSettings(ctx, settings); err != nil {
		return fmt.Errorf("failed to update settings: %w", err)
	}

	// Prepare parameters for the set request
	params := map[string]string{
		"mode":  d.Values.All()["mode"],
		"pow":   d.Values.All()["pow"],
		"shum":  d.Values.All()["shum"],
		"stemp": d.Values.All()["stemp"],
	}

	// Add fan rate if supported
	if d.SupportsFanRate() {
		params["f_rate"] = d.Values.All()["f_rate"]
	}

	// Add swing mode if supported
	if d.SupportsSwingMode() {
		allValues := d.Values.All()
		if _, hasUDLR := allValues["f_dir_lr"]; hasUDLR {
			// Australian Alira X uses separate parameters
			fDir := allValues["f_dir"]
			switch fDir {
			case "1", "3":
				params["f_dir_ud"] = "S"
			default:
				params["f_dir_ud"] = "0"
			}
			switch fDir {
			case "2", "3":
				params["f_dir_lr"] = "S"
			default:
				params["f_dir_lr"] = "0"
			}
		} else {
			params["f_dir"] = allValues["f_dir"]
		}
	}

	d.Logger.Info("Setting device parameters", "params", params)

	// Make the request
	_, err := d.getResource(ctx, "aircon/set_control_info", params)
	if err != nil {
		return fmt.Errorf("failed to set control info: %w", err)
	}

	return nil
}

// updateSettings updates the internal settings based on user input
func (d *DaikinBRP069) updateSettings(ctx context.Context, settings map[string]string) error {
	// Get current control info
	currentValues, err := d.getResource(ctx, "aircon/get_control_info", nil)
	if err != nil {
		return fmt.Errorf("failed to get current control info: %w", err)
	}

	// Update values with current state
	d.Values.UpdateByResource("aircon/get_control_info", currentValues)

	// Process settings
	for key, value := range settings {
		daikinValue := d.reverseTranslateValue(key, value)
		d.Values.Set(key, daikinValue)
	}

	// Handle special cases
	if mode, exists := settings["mode"]; exists {
		if mode == "off" {
			d.Values.Set("pow", "0")
			// Keep current mode when turning off
			if currentMode, exists := currentValues["mode"]; exists {
				d.Values.Set("mode", currentMode)
			}
		} else {
			d.Values.Set("pow", "1")
		}
	} else if len(settings) > 0 {
		// If any setting is provided but not mode, assume power on
		d.Values.Set("pow", "1")
	}

	// Use mode-specific settings for temperature, humidity, and fan rate
	allValues := d.Values.All()
	currentMode := allValues["mode"]

	settingsMap := map[string]string{
		"stemp":  "dt",
		"shum":   "dh",
		"f_rate": "dfr",
	}

	for setting, prefix := range settingsMap {
		if _, exists := settings[setting]; !exists {
			key := prefix + currentMode
			if value, exists := currentValues[key]; exists {
				d.Values.Set(setting, value)
			}
		}
	}

	return nil
}

// SetHoliday sets holiday/away mode
func (d *DaikinBRP069) SetHoliday(ctx context.Context, mode string) error {
	value := d.reverseTranslateValue("en_hol", mode)
	if value != "0" && value != "1" {
		return fmt.Errorf("invalid holiday mode: %s", mode)
	}

	d.Values.Set("en_hol", value)

	params := map[string]string{"en_hol": value}
	d.Logger.Info("Setting holiday mode", "mode", mode, "params", params)

	_, err := d.getResource(ctx, "common/set_holiday", params)
	if err != nil {
		return fmt.Errorf("failed to set holiday mode: %w", err)
	}

	return nil
}

// SetAdvancedMode sets advanced modes like powerful, econo, etc.
func (d *DaikinBRP069) SetAdvancedMode(ctx context.Context, mode, value string) error {
	modeValue := d.reverseTranslateValue("spmode_kind", mode)
	enableValue := d.reverseTranslateValue("spmode", value)

	if enableValue != "0" && enableValue != "1" {
		return fmt.Errorf("invalid advanced mode value: %s", value)
	}

	params := map[string]string{
		"spmode_kind": modeValue,
		"set_spmode":  enableValue,
	}

	d.Logger.Info("Setting advanced mode", "mode", mode, "value", value, "params", params)

	response, err := d.getResource(ctx, "aircon/set_special_mode", params)
	if err != nil {
		return fmt.Errorf("failed to set advanced mode: %w", err)
	}

	// Update the adv value from response
	d.Values.Update(response)
	return nil
}

// SetStreamer sets streamer mode
func (d *DaikinBRP069) SetStreamer(ctx context.Context, mode string) error {
	value := d.reverseTranslateValue("en_streamer", mode)
	if value != "0" && value != "1" {
		return fmt.Errorf("invalid streamer mode: %s", mode)
	}

	params := map[string]string{"en_streamer": value}
	d.Logger.Info("Setting streamer mode", "mode", mode, "params", params)

	response, err := d.getResource(ctx, "aircon/set_special_mode", params)
	if err != nil {
		return fmt.Errorf("failed to set streamer mode: %w", err)
	}

	// Update the adv value from response
	d.Values.Update(response)
	return nil
}

// autoSetClock tells the AC to auto-set its internal clock
func (d *DaikinBRP069) autoSetClock(ctx context.Context) error {
	_, err := d.getResource(ctx, "common/get_datetime", map[string]string{"cur": ""})
	return err
}

// SetClock sets the clock on the AC to the current time
func (d *DaikinBRP069) SetClock(ctx context.Context) error {
	now := time.Now().UTC()
	params := map[string]string{
		"date": now.Format("2006/01/02"),
		"zone": "GMT",
		"time": now.Format("15:04:05"),
	}

	_, err := d.getResource(ctx, "common/notify_date_time", params)
	if err != nil {
		return fmt.Errorf("failed to set clock: %w", err)
	}

	return nil
}

// SupportsHumidity returns whether the device has humidity sensor
func (d *DaikinBRP069) SupportsHumidity() bool {
	if humidity, err := d.parseFloat("hhum"); err == nil {
		return humidity > 0
	}
	return false
}

// GetHumidity returns the current humidity
func (d *DaikinBRP069) GetHumidity() (float64, error) {
	return d.parseFloat("hhum")
}

// GetTargetHumidity returns the target humidity
func (d *DaikinBRP069) GetTargetHumidity() (float64, error) {
	return d.parseFloat("shum")
}

// GetCompressorFrequency returns the current compressor frequency
func (d *DaikinBRP069) GetCompressorFrequency() (float64, error) {
	return d.parseFloat("cmpfreq")
}

// SupportsCompressorFrequency returns whether device supports compressor frequency reading
func (d *DaikinBRP069) SupportsCompressorFrequency() bool {
	return d.Values.Has("cmpfreq")
}

// parseFloat parses a float value from the values container
func (d *DaikinBRP069) parseFloat(key string) (float64, error) {
	if value, exists := d.Values.Get(key); exists && value != "" && value != "-" && value != "--" {
		if f, err := strconv.ParseFloat(value, 64); err == nil {
			return f, nil
		}
	}
	return 0, fmt.Errorf("unable to parse float value for key %s", key)
}
