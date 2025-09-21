package godaikin

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
)

// DaikinAirBase represents a Daikin AirBase device (BRP15B61)
type DaikinAirBase struct {
	*BaseAppliance
}

// NewDaikinAirBase creates a new AirBase device instance
func NewDaikinAirBase(deviceIP string, logger Logger) *DaikinAirBase {
	base := NewBaseAppliance(deviceIP, logger)

	base.Translations = map[string]map[string]string{
		"mode": {
			"0": "fan",
			"1": "hot",
			"2": "cool",
			"3": "auto",
			"7": "dry",
		},
		"f_rate": {
			"0":  "auto",
			"1":  "low",
			"3":  "mid",
			"5":  "high",
			"1a": "low/auto",
			"3a": "mid/auto",
			"5a": "high/auto",
		},
	}

	// HTTP_RESOURCES exactly
	base.HTTPResources = []string{
		"common/basic_info",
		"aircon/get_control_info",
		"aircon/get_model_info",
		"aircon/get_sensor_info",
		"aircon/get_zone_setting",
	}

	// INFO_RESOURCES
	base.InfoResources = []string{
		"aircon/get_sensor_info",
		"aircon/get_control_info",
		"aircon/get_zone_setting",
	}

	return &DaikinAirBase{BaseAppliance: base}
}

func (d *DaikinAirBase) GetDeviceType() string {
	return "AirBase"
}

func (d *DaikinAirBase) parseResponse(data map[string]string) map[string]string {
	if fAuto, exists := data["f_auto"]; exists && fAuto == "1" {
		if fRate, exists := data["f_rate"]; exists {
			data["f_rate"] = fRate + "a"
		}
	}
	return data
}

func (d *DaikinAirBase) Init(ctx context.Context) error {
	for _, resource := range d.HTTPResources {
		skyfiResource := "skyfi/" + resource
		data, err := d.getResource(ctx, skyfiResource, nil)
		if err != nil {
			d.Logger.Warn("Failed to get resource", "resource", skyfiResource, "error", err)
			continue
		}

		data = d.parseResponse(data)
		d.Values.UpdateByResource(skyfiResource, data)
	}

	// only set if they don't exist
	if !d.Values.Has("htemp") {
		d.Values.Set("htemp", "-")
	}
	if !d.Values.Has("otemp") {
		d.Values.Set("otemp", "-")
	}
	if !d.Values.Has("shum") {
		d.Values.Set("shum", "--")
	}

	if model, exists := d.Values.Get("model"); exists && model == "NOTSUPPORT" {
		d.Values.Set("model", "Airbase BRP15B61")
	}

	return nil
}

func (d *DaikinAirBase) UpdateStatus(ctx context.Context) error {
	// Use skyfi/ prefix for info resources
	for _, resource := range d.InfoResources {
		skyfiResource := "skyfi/" + resource
		if d.Values.ShouldResourceBeUpdated(skyfiResource) {
			data, err := d.getResource(ctx, skyfiResource, nil)
			if err != nil {
				d.Logger.Warn("Failed to get resource", "resource", skyfiResource, "error", err)
				continue
			}

			// Parse special fields
			data = d.parseResponse(data)
			d.Values.UpdateByResource(skyfiResource, data)
		}
	}
	return nil
}

func (d *DaikinAirBase) Set(ctx context.Context, settings map[string]string) error {
	err := d.updateSettings(ctx, settings)
	if err != nil {
		return fmt.Errorf("failed to update settings: %w", err)
	}

	allValues := d.Values.All()

	// Set default f_airside
	if _, exists := allValues["f_airside"]; !exists {
		d.Values.Set("f_airside", "0")
		allValues = d.Values.All()
	}

	// Handle f_auto
	fAuto := "0"
	fRate := allValues["f_rate"]
	if len(fRate) > 0 && fRate[len(fRate)-1:] == "a" {
		fAuto = "1"
		fRate = fRate[:len(fRate)-1]
	}

	// Prepare parameters exactly
	params := map[string]string{
		"f_airside": allValues["f_airside"],
		"f_auto":    fAuto,
		"f_dir":     allValues["f_dir"],
		"f_rate":    fRate,
		"lpw":       "",
		"mode":      allValues["mode"],
		"pow":       allValues["pow"],
		"shum":      allValues["shum"],
		"stemp":     allValues["stemp"],
	}

	d.Logger.Info("Setting AirBase parameters", "params", params)
	_, err = d.getResource(ctx, "skyfi/aircon/set_control_info", params)
	if err != nil {
		return fmt.Errorf("failed to set control info: %w", err)
	}

	return nil
}

func (d *DaikinAirBase) updateSettings(ctx context.Context, settings map[string]string) error {
	currentValues, err := d.getResource(ctx, "skyfi/aircon/get_control_info", nil)
	if err != nil {
		return fmt.Errorf("failed to get current control info: %w", err)
	}

	d.Values.UpdateByResource("skyfi/aircon/get_control_info", currentValues)

	// Process settings
	for key, value := range settings {
		daikinValue := d.reverseTranslateValue(key, value)
		d.Values.Set(key, daikinValue)
	}

	// Handle special cases
	if mode, exists := settings["mode"]; exists {
		if mode == "off" {
			d.Values.Set("pow", "0")
			if currentMode, exists := currentValues["mode"]; exists {
				d.Values.Set("mode", currentMode)
			}
		} else {
			d.Values.Set("pow", "1")
		}
	} else if len(settings) > 0 {
		d.Values.Set("pow", "1")
	}

	return nil
}

func (d *DaikinAirBase) SupportAwayMode() bool {
	return false
}

func (d *DaikinAirBase) SupportSwingMode() bool {
	return false
}

func (d *DaikinAirBase) SupportZoneTemperature() bool {
	return d.Values.Has("lztemp_c") && d.Values.Has("lztemp_h")
}

func (d *DaikinAirBase) SupportZoneCount() bool {
	return d.Values.Has("en_zone")
}

func (d *DaikinAirBase) GetSupportedFanRates() []string {
	fanRates := []string{"Auto", "Low", "Mid", "High"}

	if fRateSteps, _ := d.Values.Get("frate_steps"); fRateSteps == "2" {
		if enFRateAuto, _ := d.Values.Get("en_frate_auto"); enFRateAuto == "0" {
			return []string{"Low", "High"}
		}
		return []string{"Auto", "High", "Low/Auto", "High/Auto"}
	}

	if enFRateAuto, _ := d.Values.Get("en_frate_auto"); enFRateAuto == "0" {
		return fanRates[1:4] // Skip auto
	}

	return fanRates
}

func (d *DaikinAirBase) Represent(key string) (string, interface{}) {
	k, val := d.representBase(key)

	if key == "zone_name" || key == "zone_onoff" || key == "lztemp_c" || key == "lztemp_h" {
		if value, exists := d.Values.Get(key); exists {
			decoded, _ := url.QueryUnescape(value)
			return k, strings.Split(decoded, ";")
		}
	}

	return k, val
}

func (d *DaikinAirBase) representBase(key string) (string, interface{}) {
	k := key

	// Get the raw value
	val, exists := d.Values.Get(key)
	if !exists {
		return k, ""
	}

	// Handle mode with power off like Python
	if key == "mode" {
		if pow, _ := d.Values.Get("pow"); pow == "0" {
			return k, "off"
		}
	}

	// Handle MAC address formatting
	if key == "mac" {
		return k, formatMAC(val)
	}

	// Translate the value using translations
	translated := d.translateValue(key, val)
	return k, translated
}

// GetZones returns list of zones
func (d *DaikinAirBase) GetZones() []map[string]interface{} {
	zoneNameVal, hasZoneName := d.Values.Get("zone_name")
	if !hasZoneName || zoneNameVal == "" {
		return nil
	}

	_, zoneNames := d.Represent("zone_name")
	zoneNameList := zoneNames.([]string)

	enabledZones := len(zoneNameList)
	if d.SupportZoneCount() {
		if zoneCountStr, _ := d.Values.Get("zone_count"); zoneCountStr != "" {
			if count, err := strconv.Atoi(zoneCountStr); err == nil {
				enabledZones = count
			}
		}
	}

	_, zoneOnOff := d.Represent("zone_onoff")
	zoneOnOffList := zoneOnOff.([]string)

	// Limit zones to enabled count
	if enabledZones < len(zoneNameList) {
		zoneNameList = zoneNameList[:enabledZones]
	}

	var zones []map[string]interface{}

	if d.SupportZoneTemperature() {
		mode, _ := d.Values.Get("mode")
		if mode == "3" { // auto mode
			if operate, exists := d.Values.Get("operate"); exists {
				mode = operate
			}
		}

		var zoneTemp []string
		if mode == "1" { // heat
			_, temp := d.Represent("lztemp_h")
			zoneTemp = temp.([]string)
		} else if mode == "2" { // cool
			_, temp := d.Represent("lztemp_c")
			zoneTemp = temp.([]string)
		} else {
			stemp, _ := d.Values.Get("stemp")
			zoneTemp = make([]string, len(zoneNameList))
			for i := range zoneTemp {
				zoneTemp[i] = stemp
			}
		}

		for i, name := range zoneNameList {
			temp := 0.0
			if i < len(zoneTemp) {
				temp, _ = strconv.ParseFloat(zoneTemp[i], 64)
			}

			onOff := "0"
			if i < len(zoneOnOffList) {
				onOff = zoneOnOffList[i]
			}

			zones = append(zones, map[string]interface{}{
				"name":        strings.Trim(name, " +,"),
				"status":      onOff,
				"temperature": temp,
			})
		}
	} else {
		for i, name := range zoneNameList {
			onOff := "0"
			if i < len(zoneOnOffList) {
				onOff = zoneOnOffList[i]
			}

			zones = append(zones, map[string]interface{}{
				"name":        strings.Trim(name, " +,"),
				"status":      onOff,
				"temperature": 0.0,
			})
		}
	}

	return zones
}

// SetZone sets zone status
func (d *DaikinAirBase) SetZone(ctx context.Context, zoneID int, key string, value interface{}) error {
	// Get current zone settings
	currentState, err := d.getResource(ctx, "skyfi/aircon/get_zone_setting", nil)
	if err != nil {
		return fmt.Errorf("failed to get current zone settings: %w", err)
	}

	d.Values.UpdateByResource("skyfi/aircon/get_zone_setting", currentState)

	targetKey := key
	if key == "lztemp" {
		mode, _ := d.Values.Get("mode")
		if mode == "3" { // auto mode
			if operate, exists := d.Values.Get("operate"); exists {
				mode = operate
			}
		}

		if mode == "1" {
			targetKey = "lztemp_h"
		} else if mode == "2" {
			targetKey = "lztemp_c"
		}
	}

	if _, exists := currentState[targetKey]; !exists {
		return fmt.Errorf("key %s not found in current state", targetKey)
	}

	// Get current group and update the specific zone
	_, currentGroup := d.Represent(targetKey)
	currentGroupList := currentGroup.([]string)

	if zoneID >= len(currentGroupList) {
		return fmt.Errorf("zone ID %d out of range", zoneID)
	}

	currentGroupList[zoneID] = fmt.Sprintf("%v", value)

	// URL encode the updated group
	encoded := url.QueryEscape(strings.Join(currentGroupList, ";"))
	d.Values.Set(targetKey, strings.ToLower(encoded))

	// Prepare parameters for set request
	params := map[string]string{
		"zone_name":  currentState["zone_name"],
		"zone_onoff": d.Values.All()[targetKey],
	}

	if d.SupportZoneTemperature() {
		params["lztemp_c"] = d.Values.All()["lztemp_c"]
		params["lztemp_h"] = d.Values.All()["lztemp_h"]
	}

	// Build query string manually to avoid double encoding
	var queryParts []string
	for k, v := range params {
		queryParts = append(queryParts, fmt.Sprintf("%s=%s", k, v))
	}
	path := "skyfi/aircon/set_zone_setting?" + strings.Join(queryParts, "&")

	d.Logger.Info("Updating zone setting", "zone_id", zoneID, "key", key, "value", value)
	_, err = d.getResource(ctx, path, nil)
	if err != nil {
		return fmt.Errorf("failed to set zone setting: %w", err)
	}

	return nil
}
