package godaikin

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// DaikinSkyFi represents a Daikin SkyFi device
type DaikinSkyFi struct {
	*BaseAppliance
	Password string
}

// NewDaikinSkyFi creates SkyFi device
func NewDaikinSkyFi(deviceIP, password string, logger Logger) *DaikinSkyFi {
	base := NewBaseAppliance(deviceIP, logger)
	base.BaseURL = fmt.Sprintf("http://%s:2000", deviceIP)

	// Set translations
	base.Translations = map[string]map[string]string{
		"mode": {
			"0":  "off",
			"1":  "auto",
			"2":  "hot",
			"3":  "auto-3",
			"4":  "dry",
			"8":  "cool",
			"9":  "auto-9",
			"16": "fan",
		},
		"f_rate": {
			"1": "low",
			"2": "medium",
			"3": "high",
			"5": "low/auto",
			"6": "medium/auto",
			"7": "high/auto",
		},
	}

	base.HTTPResources = []string{"ac.cgi", "zones.cgi"}
	base.InfoResources = base.HTTPResources
	base.MaxConcurrentRequests = 1

	return &DaikinSkyFi{
		BaseAppliance: base,
		Password:      password,
	}
}

func (d *DaikinSkyFi) GetDeviceType() string {
	return "SkyFi"
}

func (d *DaikinSkyFi) Init(ctx context.Context) error {
	for _, resource := range d.HTTPResources {
		params := map[string]string{"pass": d.Password}
		data, err := d.getResource(ctx, resource, params)
		if err != nil {
			d.Logger.Warn("Failed to get resource", "resource", resource, "error", err)
			continue
		}

		skyfiData := d.parseSkyFiResponse(fmt.Sprintf("%v", data))
		d.Values.UpdateByResource(resource, skyfiData)
	}
	return nil
}

func (d *DaikinSkyFi) UpdateStatus(ctx context.Context) error {
	for _, resource := range d.InfoResources {
		if d.Values.ShouldResourceBeUpdated(resource) {
			params := map[string]string{"pass": d.Password}
			data, err := d.getResource(ctx, resource, params)
			if err != nil {
				d.Logger.Warn("Failed to get resource", "resource", resource, "error", err)
				continue
			}

			skyfiData := d.parseSkyFiResponse(fmt.Sprintf("%v", data))
			d.Values.UpdateByResource(resource, skyfiData)
		}
	}
	return nil
}

func (d *DaikinSkyFi) Set(ctx context.Context, settings map[string]string) error {
	d.Logger.Info("Updating SkyFi settings", "settings", settings)

	err := d.UpdateStatus(ctx)
	if err != nil {
		return fmt.Errorf("failed to update status: %w", err)
	}

	// Merge current_val with mapped settings
	for key, value := range settings {
		skyfiKey := d.daikinToSkyFi(key)
		daikinValue := d.reverseTranslateValue(key, value)
		d.Values.Set(skyfiKey, daikinValue)
	}

	d.Logger.Debug("Updated device values", "values", d.Values.All())

	// Handle off mode
	if mode, exists := settings["mode"]; exists && mode == "off" {
		d.Values.Set("opmode", "0")
		params := map[string]string{
			"p": d.Values.All()["opmode"],
		}
		_, err := d.getResource(ctx, "set.cgi", params)
		if err != nil {
			return fmt.Errorf("failed to turn off: %w", err)
		}
	} else {
		// Normal operation
		if _, exists := settings["mode"]; exists {
			d.Values.Set("opmode", "1")
		}

		allValues := d.Values.All()
		params := map[string]string{
			"p": allValues["opmode"],
			"t": allValues["settemp"],
			"f": allValues["fanspeed"],
			"m": allValues["acmode"],
		}

		_, err := d.getResource(ctx, "set.cgi", params)
		if err != nil {
			return fmt.Errorf("failed to set control: %w", err)
		}
	}

	return nil
}

func (d *DaikinSkyFi) parseSkyFiResponse(response string) map[string]string {
	d.Logger.Debug("Parsing SkyFi response", "response", response)

	result := make(map[string]string)
	pairs := strings.Split(response, "&")

	for _, pair := range pairs {
		if parts := strings.SplitN(pair, "=", 2); len(parts) == 2 {
			key := parts[0]
			value := parts[1]
			result[key] = value
		}
	}

	if fanflags, exists := result["fanflags"]; exists && fanflags == "3" {
		if fanspeed, exists := result["fanspeed"]; exists {
			if speed, err := strconv.Atoi(fanspeed); err == nil {
				result["fanspeed"] = strconv.Itoa(speed + 4)
			}
		}
	}

	mapped := make(map[string]string)
	for key, value := range result {
		daikinKey := d.skyfiToDaikin(key)
		mapped[daikinKey] = value
	}

	// Add original keys that weren't mapped
	for key, value := range result {
		if _, exists := mapped[key]; !exists {
			mapped[key] = value
		}
	}

	return mapped
}

func (d *DaikinSkyFi) SupportAwayMode() bool {
	return false
}

func (d *DaikinSkyFi) SupportFanRate() bool {
	return true
}

func (d *DaikinSkyFi) SupportSwingMode() bool {
	return false
}

// SKYFI_TO_DAIKIN mapping
func (d *DaikinSkyFi) skyfiToDaikin(key string) string {
	mapping := map[string]string{
		"outsidetemp": "otemp",
		"roomtemp":    "htemp",
		"settemp":     "stemp",
		"opmode":      "pow",
		"fanspeed":    "f_rate",
		"acmode":      "mode",
	}
	if daikinKey, exists := mapping[key]; exists {
		return daikinKey
	}
	return key
}

// DAIKIN_TO_SKYFI mapping
func (d *DaikinSkyFi) daikinToSkyFi(key string) string {
	mapping := map[string]string{
		"otemp":  "outsidetemp",
		"htemp":  "roomtemp",
		"stemp":  "settemp",
		"pow":    "opmode",
		"f_rate": "fanspeed",
		"mode":   "acmode",
	}
	if skyfiKey, exists := mapping[key]; exists {
		return skyfiKey
	}
	return key
}

// Zones support
func (d *DaikinSkyFi) GetZones() []map[string]interface{} {
	nz := d.Values.All()["nz"]
	if nz == "" {
		return nil
	}

	var zones []map[string]interface{}
	zoneStatus := d.representZone()

	for i, zone := range zoneStatus {
		if zone != fmt.Sprintf("Zone %d", i+1) {
			zones = append(zones, map[string]interface{}{
				"name":   zone,
				"status": string(d.representZoneOnOff()[i]),
			})
		}
	}
	return zones
}

func (d *DaikinSkyFi) representZone() []string {
	zoneVal := d.Values.All()["zone"]
	if zoneVal == "" {
		return nil
	}

	// zone is a binary representation
	zoneInt, _ := strconv.Atoi(zoneVal)
	zoneBinary := fmt.Sprintf("%08b", zoneInt+256)[3:] // Get last 8 bits

	nzStr := d.Values.All()["nz"]
	nz, _ := strconv.Atoi(nzStr)
	if nz == 0 {
		nz = 8
	}

	return strings.Split(zoneBinary[:nz], "")
}

func (d *DaikinSkyFi) representZoneOnOff() []rune {
	zoneStatus := d.representZone()
	var result []rune
	for _, status := range zoneStatus {
		result = append(result, rune(status[0]))
	}
	return result
}

func (d *DaikinSkyFi) SetZone(ctx context.Context, zoneID int, key string, value interface{}) error {
	if key != "zone_onoff" {
		return fmt.Errorf("only zone_onoff supported")
	}

	zoneID += 1 // Python uses 1-based indexing

	params := map[string]string{
		"z": strconv.Itoa(zoneID),
		"s": fmt.Sprintf("%v", value),
	}

	response, err := d.getResource(ctx, "setzone.cgi", params)
	if err != nil {
		return fmt.Errorf("failed to set zone: %w", err)
	}

	skyfiData := d.parseSkyFiResponse(fmt.Sprintf("%v", response))
	d.Values.Update(skyfiData)

	return nil
}
