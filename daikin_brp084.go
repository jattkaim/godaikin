package godaikin

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type DaikinAttribute struct {
	Name  string
	Value interface{}
	Path  []string
	To    string
}

func (d *DaikinAttribute) Format() map[string]interface{} {
	return map[string]interface{}{
		"pn": d.Name,
		"pv": d.Value,
	}
}

type DaikinRequest struct {
	Attributes []DaikinAttribute
}

// Serialize the request to JSON payload
func (d *DaikinRequest) Serialize(payload map[string]interface{}) map[string]interface{} {
	if payload == nil {
		payload = map[string]interface{}{
			"requests": []map[string]interface{}{},
		}
	}

	requests := payload["requests"].([]map[string]interface{})

	getExistingTo := func(to string, requests []map[string]interface{}) map[string]interface{} {
		for _, request := range requests {
			if thisTo, exists := request["to"]; exists && thisTo == to {
				return request
			}
		}
		return nil
	}

	getExistingIndex := func(name string, children []map[string]interface{}) int {
		for index, child := range children {
			if pn, exists := child["pn"]; exists && pn == name {
				return index
			}
		}
		return -1
	}

	for _, attribute := range d.Attributes {
		to := getExistingTo(attribute.To, requests)
		if to == nil {
			newRequest := map[string]interface{}{
				"op": 3,
				"pc": map[string]interface{}{
					"pn":  "dgc_status",
					"pch": []map[string]interface{}{},
				},
				"to": attribute.To,
			}
			requests = append(requests, newRequest)
			to = newRequest
		}

		entry := to["pc"].(map[string]interface{})["pch"].([]map[string]interface{})
		for _, pn := range attribute.Path {
			index := getExistingIndex(pn, entry)
			if index == -1 {
				newEntry := map[string]interface{}{
					"pn":  pn,
					"pch": []map[string]interface{}{},
				}
				entry = append(entry, newEntry)
				entry = newEntry["pch"].([]map[string]interface{})
			} else {
				entry = entry[index]["pch"].([]map[string]interface{})
			}
		}
		entry = append(entry, attribute.Format())
	}

	payload["requests"] = requests
	return payload
}

// DaikinBRP084 represents a Daikin BRP device with firmware 2.8.0
type DaikinBRP084 struct {
	*BaseAppliance
	URL string
}

// NewDaikinBRP084 creates BRP084 device
func NewDaikinBRP084(deviceIP string, logger Logger) *DaikinBRP084 {
	base := NewBaseAppliance(deviceIP, logger)

	// Set translations like Python
	base.Translations = map[string]map[string]string{
		"mode": {
			"0300": "auto",
			"0200": "cool",
			"0100": "heat",
			"0000": "fan",
			"0500": "dry",
			"00":   "off",
			"01":   "on",
		},
		"f_rate": {
			"0A00": "auto",
			"0B00": "quiet",
			"0300": "1",
			"0400": "2",
			"0500": "3",
			"0600": "4",
			"0700": "5",
		},
		"f_dir": {
			"off":        "off",
			"vertical":   "vertical",
			"horizontal": "horizontal",
			"both":       "3d",
		},
	}

	// Empty info resources for BRP084
	base.InfoResources = []string{}

	return &DaikinBRP084{
		BaseAppliance: base,
		URL:           fmt.Sprintf("%s/dsiot/multireq", base.BaseURL),
	}
}

func (d *DaikinBRP084) GetDeviceType() string {
	return "BRP084"
}

// API paths following Python exactly
var API_PATHS = map[string]interface{}{
	"power": []string{
		"/dsiot/edge/adr_0100.dgc_status",
		"dgc_status",
		"e_1002",
		"e_A002",
		"p_01",
	},
	"mode": []string{
		"/dsiot/edge/adr_0100.dgc_status",
		"dgc_status",
		"e_1002",
		"e_3001",
		"p_01",
	},
	"indoor_temp": []string{
		"/dsiot/edge/adr_0100.dgc_status",
		"dgc_status",
		"e_1002",
		"e_A00B",
		"p_01",
	},
	"indoor_humidity": []string{
		"/dsiot/edge/adr_0100.dgc_status",
		"dgc_status",
		"e_1002",
		"e_A00B",
		"p_02",
	},
	"outdoor_temp": []string{
		"/dsiot/edge/adr_0200.dgc_status",
		"dgc_status",
		"e_1003",
		"e_A00D",
		"p_01",
	},
	"mac_address": []string{"/dsiot/edge.adp_i", "adp_i", "mac"},
	"temp_settings": map[string][]string{
		"cool": {
			"/dsiot/edge/adr_0100.dgc_status",
			"dgc_status",
			"e_1002",
			"e_3001",
			"p_02",
		},
		"heat": {
			"/dsiot/edge/adr_0100.dgc_status",
			"dgc_status",
			"e_1002",
			"e_3001",
			"p_03",
		},
		"auto": {
			"/dsiot/edge/adr_0100.dgc_status",
			"dgc_status",
			"e_1002",
			"e_3001",
			"p_1D",
		},
	},
	"fan_settings": map[string][]string{
		"auto": {
			"/dsiot/edge/adr_0100.dgc_status",
			"dgc_status",
			"e_1002",
			"e_3001",
			"p_26",
		},
		"cool": {
			"/dsiot/edge/adr_0100.dgc_status",
			"dgc_status",
			"e_1002",
			"e_3001",
			"p_09",
		},
		"heat": {
			"/dsiot/edge/adr_0100.dgc_status",
			"dgc_status",
			"e_1002",
			"e_3001",
			"p_0A",
		},
		"fan": {
			"/dsiot/edge/adr_0100.dgc_status",
			"dgc_status",
			"e_1002",
			"e_3001",
			"p_28",
		},
	},
	"swing_settings": map[string]map[string][]string{
		"auto": {
			"vertical": {
				"/dsiot/edge/adr_0100.dgc_status",
				"dgc_status",
				"e_1002",
				"e_3001",
				"p_20",
			},
			"horizontal": {
				"/dsiot/edge/adr_0100.dgc_status",
				"dgc_status",
				"e_1002",
				"e_3001",
				"p_21",
			},
		},
		"cool": {
			"vertical": {
				"/dsiot/edge/adr_0100.dgc_status",
				"dgc_status",
				"e_1002",
				"e_3001",
				"p_05",
			},
			"horizontal": {
				"/dsiot/edge/adr_0100.dgc_status",
				"dgc_status",
				"e_1002",
				"e_3001",
				"p_06",
			},
		},
		"heat": {
			"vertical": {
				"/dsiot/edge/adr_0100.dgc_status",
				"dgc_status",
				"e_1002",
				"e_3001",
				"p_07",
			},
			"horizontal": {
				"/dsiot/edge/adr_0100.dgc_status",
				"dgc_status",
				"e_1002",
				"e_3001",
				"p_08",
			},
		},
		"fan": {
			"vertical": {
				"/dsiot/edge/adr_0100.dgc_status",
				"dgc_status",
				"e_1002",
				"e_3001",
				"p_24",
			},
			"horizontal": {
				"/dsiot/edge/adr_0100.dgc_status",
				"dgc_status",
				"e_1002",
				"e_3001",
				"p_25",
			},
		},
		"dry": {
			"vertical": {
				"/dsiot/edge/adr_0100.dgc_status",
				"dgc_status",
				"e_1002",
				"e_3001",
				"p_22",
			},
			"horizontal": {
				"/dsiot/edge/adr_0100.dgc_status",
				"dgc_status",
				"e_1002",
				"e_3001",
				"p_23",
			},
		},
	},
	"energy": map[string][]string{
		"today_runtime": {
			"/dsiot/edge/adr_0100.i_power.week_power",
			"week_power",
			"today_runtime",
		},
		"weekly_data": {
			"/dsiot/edge/adr_0100.i_power.week_power",
			"week_power",
			"datas",
		},
	},
}

// Mode mappings
var MODE_MAP = map[string]string{
	"0300": "auto",
	"0200": "cool",
	"0100": "heat",
	"0000": "fan",
	"0500": "dry",
}

var FAN_MODE_MAP = map[string]string{
	"0A00": "auto",
	"0B00": "quiet",
	"0300": "1",
	"0400": "2",
	"0500": "3",
	"0600": "4",
	"0700": "5",
}

var REVERSE_MODE_MAP = make(map[string]string)
var REVERSE_FAN_MODE_MAP = make(map[string]string)

func init() {
	for k, v := range MODE_MAP {
		REVERSE_MODE_MAP[v] = k
	}
	for k, v := range FAN_MODE_MAP {
		REVERSE_FAN_MODE_MAP[v] = k
	}
}

const TURN_OFF_SWING_AXIS = "000000"
const TURN_ON_SWING_AXIS = "0F0000"

// Helper methods following Python exactly
func (d *DaikinBRP084) hexToTemp(value string, divisor int) float64 {
	if len(value) < 2 {
		return 0
	}
	val, err := strconv.ParseInt(value[:2], 16, 64)
	if err != nil {
		return 0
	}
	return float64(val) / float64(divisor)
}

func (d *DaikinBRP084) tempToHex(temperature float64, divisor int) string {
	return fmt.Sprintf("%02x", int(temperature*float64(divisor)))
}

func (d *DaikinBRP084) hexToInt(value string) int {
	val, err := strconv.ParseInt(value, 16, 64)
	if err != nil {
		return 0
	}
	return int(val)
}

func (d *DaikinBRP084) getPath(keys ...string) []string {
	current := API_PATHS
	for _, key := range keys {
		if next, exists := current[key]; exists {
			switch v := next.(type) {
			case []string:
				return v
			case map[string][]string:
				current = map[string]interface{}{}
				for k, val := range v {
					current[k] = val
				}
			case map[string]map[string][]string:
				current = map[string]interface{}{}
				for k, val := range v {
					current[k] = val
				}
			default:
				current = next.(map[string]interface{})
			}
		} else {
			d.Logger.Warn("Path key not found", "key", key)
			return nil
		}
	}
	return nil
}

func (d *DaikinBRP084) findValueByPN(data map[string]interface{}, fr string, keys ...string) (interface{}, error) {
	responses, exists := data["responses"].([]interface{})
	if !exists {
		return nil, fmt.Errorf("no responses found")
	}

	var targetData []interface{}
	for _, resp := range responses {
		if respMap, ok := resp.(map[string]interface{}); ok {
			if respMap["fr"] == fr {
				if pc, exists := respMap["pc"].(map[string]interface{}); exists {
					targetData = []interface{}{pc}
					break
				}
			}
		}
	}

	if len(targetData) == 0 {
		return nil, fmt.Errorf("fr %s not found", fr)
	}

	for _, key := range keys {
		found := false
		for _, dataItem := range targetData {
			if dataMap, ok := dataItem.(map[string]interface{}); ok {
				if dataMap["pn"] == key {
					if len(keys) == 1 {
						return dataMap["pv"], nil
					}
					if pch, exists := dataMap["pch"].([]interface{}); exists {
						targetData = pch
						found = true
						break
					}
				}
			}
		}
		if !found {
			return nil, fmt.Errorf("key %s not found", key)
		}
		keys = keys[1:]
	}

	return nil, fmt.Errorf("value not found")
}

func (d *DaikinBRP084) getSwingState(data map[string]interface{}) string {
	mode, _ := d.Values.Get("mode")
	if mode == "" || mode == "off" {
		return "off"
	}

	swingSettings := API_PATHS["swing_settings"].(map[string]map[string][]string)
	if modeSettings, exists := swingSettings[mode]; exists {
		verticalPath := modeSettings["vertical"]
		horizontalPath := modeSettings["horizontal"]

		verticalVal, err1 := d.findValueByPN(data, verticalPath[0], verticalPath[1:]...)
		horizontalVal, err2 := d.findValueByPN(data, horizontalPath[0], horizontalPath[1:]...)

		if err1 == nil && err2 == nil {
			verticalStr := fmt.Sprintf("%v", verticalVal)
			horizontalStr := fmt.Sprintf("%v", horizontalVal)

			vertical := len(verticalStr) > 0 && verticalStr[0] == 'F'
			horizontal := len(horizontalStr) > 0 && horizontalStr[0] == 'F'

			if horizontal && vertical {
				return "both"
			}
			if horizontal {
				return "horizontal"
			}
			if vertical {
				return "vertical"
			}
		}
	}

	return "off"
}

func (d *DaikinBRP084) Init(ctx context.Context) error {
	return d.UpdateStatus(ctx)
}

func (d *DaikinBRP084) UpdateStatus(ctx context.Context) error {
	payload := map[string]interface{}{
		"requests": []map[string]interface{}{
			{"op": 2, "to": "/dsiot/edge/adr_0100.dgc_status?filter=pv,pt,md"},
			{"op": 2, "to": "/dsiot/edge/adr_0200.dgc_status?filter=pv,pt,md"},
			{"op": 2, "to": "/dsiot/edge/adr_0100.i_power.week_power?filter=pv,pt,md"},
			{"op": 2, "to": "/dsiot/edge.adp_i"},
		},
	}

	response, err := d.getResource(ctx, "", payload)
	if err != nil {
		return fmt.Errorf("failed to communicate with device: %w", err)
	}

	responseMap, ok := response.(map[string]interface{})
	if !ok || responseMap["responses"] == nil {
		return fmt.Errorf("invalid response from device")
	}

	// Extract basic info
	macPath := d.getPath("mac_address")
	if mac, err := d.findValueByPN(responseMap, macPath[0], macPath[1:]...); err == nil {
		d.Values.Set("mac", fmt.Sprintf("%v", mac))
	}

	// Get power state
	powerPath := d.getPath("power")
	if powerVal, err := d.findValueByPN(responseMap, powerPath[0], powerPath[1:]...); err == nil {
		isOff := fmt.Sprintf("%v", powerVal) == "00"
		if isOff {
			d.Values.Set("pow", "0")
		} else {
			d.Values.Set("pow", "1")
		}
	}

	// Get mode
	modePath := d.getPath("mode")
	if modeVal, err := d.findValueByPN(responseMap, modePath[0], modePath[1:]...); err == nil {
		modeStr := fmt.Sprintf("%v", modeVal)
		if pow, _ := d.Values.Get("pow"); pow == "0" {
			d.Values.Set("mode", "off")
		} else if humanMode, exists := MODE_MAP[modeStr]; exists {
			d.Values.Set("mode", humanMode)
		}
	}

	// Get temperatures
	otempPath := d.getPath("outdoor_temp")
	if otempVal, err := d.findValueByPN(responseMap, otempPath[0], otempPath[1:]...); err == nil {
		otemp := d.hexToTemp(fmt.Sprintf("%v", otempVal), 2)
		d.Values.Set("otemp", fmt.Sprintf("%.1f", otemp))
	}

	htempPath := d.getPath("indoor_temp")
	if htempVal, err := d.findValueByPN(responseMap, htempPath[0], htempPath[1:]...); err == nil {
		htemp := d.hexToTemp(fmt.Sprintf("%v", htempVal), 1)
		d.Values.Set("htemp", fmt.Sprintf("%.1f", htemp))
	}

	// Get humidity
	humidPath := d.getPath("indoor_humidity")
	if humidVal, err := d.findValueByPN(responseMap, humidPath[0], humidPath[1:]...); err == nil {
		humid := d.hexToInt(fmt.Sprintf("%v", humidVal))
		d.Values.Set("hhum", fmt.Sprintf("%d", humid))
	} else {
		d.Values.Set("hhum", "--")
	}

	// Get target temperature
	if mode, _ := d.Values.Get("mode"); mode != "" && mode != "off" {
		tempSettings := API_PATHS["temp_settings"].(map[string][]string)
		if tempPath, exists := tempSettings[mode]; exists {
			if stempVal, err := d.findValueByPN(responseMap, tempPath[0], tempPath[1:]...); err == nil {
				stemp := d.hexToTemp(fmt.Sprintf("%v", stempVal), 2)
				d.Values.Set("stemp", fmt.Sprintf("%.1f", stemp))
			}
		}
	} else {
		d.Values.Set("stemp", "--")
	}

	// Get fan mode
	if mode, _ := d.Values.Get("mode"); mode != "" && mode != "off" {
		fanSettings := API_PATHS["fan_settings"].(map[string][]string)
		if fanPath, exists := fanSettings[mode]; exists {
			if fanVal, err := d.findValueByPN(responseMap, fanPath[0], fanPath[1:]...); err == nil {
				fanStr := fmt.Sprintf("%v", fanVal)
				if humanFan, exists := FAN_MODE_MAP[fanStr]; exists {
					d.Values.Set("f_rate", humanFan)
				} else {
					d.Values.Set("f_rate", "auto")
				}
			}
		}
	} else {
		d.Values.Set("f_rate", "auto")
	}

	// Get swing mode
	d.Values.Set("f_dir", d.getSwingState(responseMap))

	// Get energy data
	energyPaths := API_PATHS["energy"].(map[string][]string)
	if runtimePath, exists := energyPaths["today_runtime"]; exists {
		if runtimeVal, err := d.findValueByPN(responseMap, runtimePath[0], runtimePath[1:]...); err == nil {
			d.Values.Set("today_runtime", fmt.Sprintf("%v", runtimeVal))
		}
	}

	if weeklyPath, exists := energyPaths["weekly_data"]; exists {
		if weeklyVal, err := d.findValueByPN(responseMap, weeklyPath[0], weeklyPath[1:]...); err == nil {
			if weeklyList, ok := weeklyVal.([]interface{}); ok && len(weeklyList) > 0 {
				var strs []string
				for _, v := range weeklyList {
					strs = append(strs, fmt.Sprintf("%v", v))
				}
				d.Values.Set("datas", strings.Join(strs, "/"))
			}
		}
	}

	return nil
}

func (d *DaikinBRP084) getResource(ctx context.Context, path string, params interface{}) (interface{}, error) {
	d.Logger.Debug("Making BRP084 request", "url", d.URL, "path", path, "params", params)

	var jsonData []byte
	var err error
	if params != nil {
		jsonData, err = json.Marshal(params)
		if err != nil {
			return nil, err
		}
	}

	req, err := http.NewRequestWithContext(ctx, "POST", d.URL, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, err
	}

	req.Header.Set("Content-Type", "application/json")
	for key, value := range d.Headers {
		req.Header.Set(key, value)
	}

	resp, err := d.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("unexpected HTTP status: %d", resp.StatusCode)
	}

	var result interface{}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	return result, nil
}

func (d *DaikinBRP084) addRequest(requests *[]DaikinAttribute, path []string, value string) {
	attr := DaikinAttribute{
		Name:  path[len(path)-1],
		Value: value,
		Path:  path[2:4],
		To:    path[0],
	}
	*requests = append(*requests, attr)
}

func (d *DaikinBRP084) handlePowerSetting(settings map[string]string, requests *[]DaikinAttribute) {
	if mode, exists := settings["mode"]; exists {
		powerPath := d.getPath("power")
		if mode == "off" {
			d.addRequest(requests, powerPath, "00")
		} else {
			d.addRequest(requests, powerPath, "01")

			// Set mode
			if modeValue, exists := REVERSE_MODE_MAP[mode]; exists {
				modePath := d.getPath("mode")
				d.addRequest(requests, modePath, modeValue)
			}
		}
	}
}

func (d *DaikinBRP084) handleTemperatureSetting(settings map[string]string, requests *[]DaikinAttribute) {
	if stemp, exists := settings["stemp"]; exists {
		if mode, _ := d.Values.Get("mode"); mode != "" {
			tempSettings := API_PATHS["temp_settings"].(map[string][]string)
			if tempPath, exists := tempSettings[mode]; exists {
				temp, _ := strconv.ParseFloat(stemp, 64)
				tempHex := d.tempToHex(temp, 2)
				d.addRequest(requests, tempPath, tempHex)
			}
		}
	}
}

func (d *DaikinBRP084) handleFanSetting(settings map[string]string, requests *[]DaikinAttribute) {
	if fRate, exists := settings["f_rate"]; exists {
		if mode, _ := d.Values.Get("mode"); mode != "" {
			fanSettings := API_PATHS["fan_settings"].(map[string][]string)
			if fanPath, exists := fanSettings[mode]; exists {
				if fanValue, exists := REVERSE_FAN_MODE_MAP[fRate]; exists {
					d.addRequest(requests, fanPath, fanValue)
				}
			}
		}
	}
}

func (d *DaikinBRP084) handleSwingSetting(settings map[string]string, requests *[]DaikinAttribute) {
	if fDir, exists := settings["f_dir"]; exists {
		if mode, _ := d.Values.Get("mode"); mode != "" {
			swingSettings := API_PATHS["swing_settings"].(map[string]map[string][]string)
			if modeSettings, exists := swingSettings[mode]; exists {
				verticalPath := modeSettings["vertical"]
				horizontalPath := modeSettings["horizontal"]

				var verticalValue, horizontalValue string
				switch fDir {
				case "off":
					verticalValue = TURN_OFF_SWING_AXIS
					horizontalValue = TURN_OFF_SWING_AXIS
				case "vertical":
					verticalValue = TURN_ON_SWING_AXIS
					horizontalValue = TURN_OFF_SWING_AXIS
				case "horizontal":
					verticalValue = TURN_OFF_SWING_AXIS
					horizontalValue = TURN_ON_SWING_AXIS
				case "both", "3d":
					verticalValue = TURN_ON_SWING_AXIS
					horizontalValue = TURN_ON_SWING_AXIS
				}

				d.addRequest(requests, verticalPath, verticalValue)
				d.addRequest(requests, horizontalPath, horizontalValue)
			}
		}
	}
}

func (d *DaikinBRP084) updateSettings(_ context.Context, settings map[string]string) error {
	log.Printf("Updating settings: %v", settings)

	for key, value := range settings {
		if key == "mode" && value == "off" {
			d.Values.Set("pow", "0")
		} else if key == "mode" {
			d.Values.Set("pow", "1")
			d.Values.Set("mode", value)
		} else {
			d.Values.Set(key, value)
		}
	}

	return nil
}

func (d *DaikinBRP084) Set(ctx context.Context, settings map[string]string) error {
	err := d.updateSettings(ctx, settings)
	if err != nil {
		return err
	}

	var requests []DaikinAttribute

	d.handlePowerSetting(settings, &requests)
	d.handleTemperatureSetting(settings, &requests)
	d.handleFanSetting(settings, &requests)
	d.handleSwingSetting(settings, &requests)

	if len(requests) > 0 {
		request := DaikinRequest{Attributes: requests}
		requestPayload := request.Serialize(nil)
		log.Printf("Sending request: %v", requestPayload)

		response, err := d.getResource(ctx, "", requestPayload)
		if err != nil {
			return err
		}
		log.Printf("Response: %v", response)

		// Update status after setting
		return d.UpdateStatus(ctx)
	}

	return nil
}

// SetStreamer - not supported in firmware 2.8.0
func (d *DaikinBRP084) SetStreamer(ctx context.Context, mode string) error {
	log.Printf("Streamer mode not supported in firmware 2.8.0")
	return fmt.Errorf("streamer mode not supported in firmware 2.8.0")
}

// SetHoliday - not supported in firmware 2.8.0
func (d *DaikinBRP084) SetHoliday(ctx context.Context, mode string) error {
	log.Printf("Holiday mode not supported in firmware 2.8.0")
	return fmt.Errorf("holiday mode not supported in firmware 2.8.0")
}

// SetAdvancedMode - not supported in firmware 2.8.0
func (d *DaikinBRP084) SetAdvancedMode(ctx context.Context, mode, value string) error {
	log.Printf("Advanced mode not supported in firmware 2.8.0")
	return fmt.Errorf("advanced mode not supported in firmware 2.8.0")
}

// Support properties like Python
func (d *DaikinBRP084) SupportAwayMode() bool {
	return false // not supported in firmware 2.8.0
}

func (d *DaikinBRP084) SupportAdvancedModes() bool {
	return false // not supported in firmware 2.8.0
}

func (d *DaikinBRP084) SupportZoneCount() bool {
	return false // not supported in firmware 2.8.0
}

// Additional methods from Python daikin_brp084.py

// GetInsideTemperature returns current inside temperature
func (d *DaikinBRP084) GetInsideTemperature() (float64, error) {
	if htemp, exists := d.Values.Get("htemp"); exists && htemp != "" && htemp != "-" {
		if temp, err := strconv.ParseFloat(htemp, 64); err == nil {
			return temp, nil
		}
	}
	return 0, fmt.Errorf("unable to parse inside temperature")
}

// GetOutsideTemperature returns current outside temperature
func (d *DaikinBRP084) GetOutsideTemperature() (float64, error) {
	if otemp, exists := d.Values.Get("otemp"); exists && otemp != "" && otemp != "-" {
		if temp, err := strconv.ParseFloat(otemp, 64); err == nil {
			return temp, nil
		}
	}
	return 0, fmt.Errorf("unable to parse outside temperature")
}

// GetTargetTemperature returns current target temperature
func (d *DaikinBRP084) GetTargetTemperature() (float64, error) {
	if stemp, exists := d.Values.Get("stemp"); exists && stemp != "" && stemp != "--" {
		if temp, err := strconv.ParseFloat(stemp, 64); err == nil {
			return temp, nil
		}
	}
	return 0, fmt.Errorf("unable to parse target temperature")
}

// GetMode returns current operating mode
func (d *DaikinBRP084) GetMode() string {
	if pow, _ := d.Values.Get("pow"); pow == "0" {
		return "off"
	}
	if mode, exists := d.Values.Get("mode"); exists {
		return mode
	}
	return "unknown"
}

// GetPowerState returns whether device is powered on
func (d *DaikinBRP084) GetPowerState() string {
	if pow, exists := d.Values.Get("pow"); exists {
		return pow
	}
	return "0"
}

// GetFanRate returns current fan rate
func (d *DaikinBRP084) GetFanRate() string {
	if rate, exists := d.Values.Get("f_rate"); exists {
		return rate
	}
	return "unknown"
}

// GetFanDirection returns current fan direction
func (d *DaikinBRP084) GetFanDirection() string {
	if dir, exists := d.Values.Get("f_dir"); exists {
		return dir
	}
	return "unknown"
}

// Additional support methods
func (d *DaikinBRP084) SupportsFanRate() bool {
	return d.Values.Has("f_rate")
}

func (d *DaikinBRP084) SupportsSwingMode() bool {
	return d.Values.Has("f_dir")
}

func (d *DaikinBRP084) SupportsEnergyConsumption() bool {
	return d.Values.Has("datas") || d.Values.Has("today_runtime")
}

// GetMAC returns device MAC address
func (d *DaikinBRP084) GetMAC() string {
	if mac, exists := d.Values.Get("mac"); exists {
		return formatMAC(mac)
	}
	return d.DeviceIP
}
