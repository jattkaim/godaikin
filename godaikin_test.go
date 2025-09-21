package godaikin

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseResponse(t *testing.T) {
	tests := []struct {
		name           string
		responseBody   string
		expectedResult map[string]string
		expectError    bool
	}{
		{
			name:         "successful response",
			responseBody: "ret=OK,type=aircon,reg=eu,dst=1,ver=1_2_54",
			expectedResult: map[string]string{
				"type": "aircon",
				"reg":  "eu",
				"dst":  "1",
				"ver":  "1_2_54",
			},
			expectError: false,
		},
		{
			name:         "response with URL encoded name",
			responseBody: "ret=OK,name=%4e%6f%74%74%65,type=aircon",
			expectedResult: map[string]string{
				"name": "Notte",
				"type": "aircon",
			},
			expectError: false,
		},
		{
			name:           "failed response",
			responseBody:   "ret=KO,type=aircon,reg=eu,dst=1",
			expectedResult: map[string]string{},
			expectError:    false,
		},
		{
			name:         "missing ret field",
			responseBody: "type=aircon,reg=eu,dst=1",
			expectError:  true,
		},
		{
			name:         "empty response",
			responseBody: "",
			expectError:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseResponse(tt.responseBody)

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedResult, result)
			}
		})
	}
}

func TestValues(t *testing.T) {
	values := NewValues()

	// Test basic set/get operations
	values.Set("test_key", "test_value")
	value, exists := values.Get("test_key")
	assert.True(t, exists)
	assert.Equal(t, "test_value", value)

	// Test non-existent key
	_, exists = values.Get("non_existent")
	assert.False(t, exists)

	// Test Has method
	assert.True(t, values.Has("test_key"))
	assert.False(t, values.Has("non_existent"))

	// Test Keys method
	keys := values.Keys()
	assert.Contains(t, keys, "test_key")
	assert.Len(t, keys, 1)

	// Test All method
	all := values.All()
	assert.Equal(t, map[string]string{"test_key": "test_value"}, all)

	// Test Delete
	values.Delete("test_key")
	assert.False(t, values.Has("test_key"))
	assert.Equal(t, 0, values.Len())
}

func TestValuesUpdateByResource(t *testing.T) {
	values := NewValues()

	// Update with resource data
	resourceData := map[string]string{
		"mode":  "cool",
		"stemp": "25.0",
		"pow":   "1",
	}
	values.UpdateByResource("test_resource", resourceData)

	// Verify data was updated
	assert.True(t, values.Has("mode"))
	assert.True(t, values.Has("stemp"))
	assert.True(t, values.Has("pow"))

	// Verify resource tracking
	assert.False(t, values.ShouldResourceBeUpdated("test_resource"))

	// Access a value to mark resource for update
	values.Get("mode")
	assert.True(t, values.ShouldResourceBeUpdated("test_resource"))
}

func TestBaseApplianceTranslations(t *testing.T) {
	base := NewBaseAppliance("192.168.1.1", nil)

	// Set up test translations
	base.Translations = map[string]map[string]string{
		"mode": {
			"3": "cool",
			"4": "hot",
			"0": "auto",
		},
		"f_rate": {
			"A": "auto",
			"3": "low",
			"5": "high",
		},
	}

	// Test translation
	assert.Equal(t, "cool", base.translateValue("mode", "3"))
	assert.Equal(t, "auto", base.translateValue("f_rate", "A"))
	assert.Equal(t, "unknown", base.translateValue("mode", "unknown"))

	// Test reverse translation
	assert.Equal(t, "3", base.reverseTranslateValue("mode", "cool"))
	assert.Equal(t, "A", base.reverseTranslateValue("f_rate", "auto"))
	assert.Equal(t, "unknown", base.reverseTranslateValue("mode", "unknown"))
}

func TestBaseApplianceProperties(t *testing.T) {
	base := NewBaseAppliance("192.168.1.1", nil)

	// Test device IP
	assert.Equal(t, "192.168.1.1", base.GetDeviceIP())

	// Test MAC formatting
	base.Values.Set("mac", "112233445566")
	assert.Equal(t, "11:22:33:44:55:66", base.GetMAC())

	// Test MAC fallback to IP
	base.Values.Delete("mac")
	assert.Equal(t, "192.168.1.1", base.GetMAC())

	// Test power state
	base.Values.Set("pow", "1")
	assert.True(t, base.GetPowerState())

	base.Values.Set("pow", "0")
	assert.False(t, base.GetPowerState())

	// Test mode with power off
	base.Values.Set("mode", "3")
	assert.Equal(t, "off", base.GetMode())

	// Test mode with power on
	base.Values.Set("pow", "1")
	base.Translations = map[string]map[string]string{
		"mode": {"3": "cool"},
	}
	assert.Equal(t, "cool", base.GetMode())
}

func TestBaseApplianceTemperatures(t *testing.T) {
	base := NewBaseAppliance("192.168.1.1", nil)

	// Test valid temperature
	base.Values.Set("htemp", "25.5")
	temp, err := base.GetInsideTemperature()
	assert.NoError(t, err)
	assert.Equal(t, 25.5, temp)

	// Test invalid temperature
	base.Values.Set("htemp", "-")
	_, err = base.GetInsideTemperature()
	assert.Error(t, err)

	// Test missing temperature
	base.Values.Delete("htemp")
	_, err = base.GetInsideTemperature()
	assert.Error(t, err)
}

func TestBaseApplianceSupport(t *testing.T) {
	base := NewBaseAppliance("192.168.1.1", nil)

	// Test support methods with no values
	assert.False(t, base.SupportsFanRate())
	assert.False(t, base.SupportsSwingMode())
	assert.False(t, base.SupportsAwayMode())
	assert.False(t, base.SupportsAdvancedModes())
	assert.False(t, base.SupportsEnergyConsumption())

	// Test support methods with values
	base.Values.Set("f_rate", "A")
	base.Values.Set("f_dir", "0")
	base.Values.Set("en_hol", "0")
	base.Values.Set("adv", "")
	base.Values.Set("datas", "100/200/300")

	assert.True(t, base.SupportsFanRate())
	assert.True(t, base.SupportsSwingMode())
	assert.True(t, base.SupportsAwayMode())
	assert.True(t, base.SupportsAdvancedModes())
	assert.True(t, base.SupportsEnergyConsumption())
}

func TestDaikinBRP069Creation(t *testing.T) {
	device := NewDaikinBRP069("192.168.1.1", nil)

	assert.NotNil(t, device)
	assert.Equal(t, "192.168.1.1", device.GetDeviceIP())
	assert.Equal(t, "http://192.168.1.1", device.BaseURL)
	assert.Equal(t, 1, device.MaxConcurrentRequests)

	// Test translations are set
	assert.Contains(t, device.Translations, "mode")
	assert.Contains(t, device.Translations, "f_rate")
	assert.Contains(t, device.Translations, "f_dir")

	// Test mode translations
	assert.Equal(t, "cool", device.translateValue("mode", "3"))
	assert.Equal(t, "hot", device.translateValue("mode", "4"))
	assert.Equal(t, "dry", device.translateValue("mode", "2"))
}

func TestDaikinBRP069SpecialFields(t *testing.T) {
	device := NewDaikinBRP069("192.168.1.1", nil)

	tests := []struct {
		name     string
		input    map[string]string
		expected string
	}{
		{
			name:     "both directions off",
			input:    map[string]string{"f_dir_ud": "0", "f_dir_lr": "0"},
			expected: "0",
		},
		{
			name:     "vertical swing only",
			input:    map[string]string{"f_dir_ud": "S", "f_dir_lr": "0"},
			expected: "1",
		},
		{
			name:     "horizontal swing only",
			input:    map[string]string{"f_dir_ud": "0", "f_dir_lr": "S"},
			expected: "2",
		},
		{
			name:     "3d swing",
			input:    map[string]string{"f_dir_ud": "S", "f_dir_lr": "S"},
			expected: "3",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := device.parseSpecialFields(tt.input)
			assert.Equal(t, tt.expected, result["f_dir"])
		})
	}
}

func TestExtractIPPort(t *testing.T) {
	tests := []struct {
		name         string
		deviceID     string
		expectedIP   string
		expectedPort int
	}{
		{
			name:         "IP only",
			deviceID:     "192.168.1.1",
			expectedIP:   "192.168.1.1",
			expectedPort: 0,
		},
		{
			name:         "IP with port",
			deviceID:     "192.168.1.1:8080",
			expectedIP:   "192.168.1.1",
			expectedPort: 8080,
		},
		{
			name:         "hostname",
			deviceID:     "daikin-ac",
			expectedIP:   "daikin-ac",
			expectedPort: 0,
		},
		{
			name:         "hostname with port",
			deviceID:     "daikin-ac:30050",
			expectedIP:   "daikin-ac",
			expectedPort: 30050,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ip, port := extractIPPort(tt.deviceID)
			assert.Equal(t, tt.expectedIP, ip)
			assert.Equal(t, tt.expectedPort, port)
		})
	}
}

func TestFormatMAC(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "valid MAC",
			input:    "112233445566",
			expected: "11:22:33:44:55:66",
		},
		{
			name:     "invalid length",
			input:    "1122334455",
			expected: "1122334455",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := formatMAC(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestNewClient(t *testing.T) {
	client := NewClient(nil)
	assert.NotNil(t, client)
}

func TestFactoryOptions(t *testing.T) {
	config := &Config{}

	// Test password option
	WithPassword("testpass")(config)
	assert.Equal(t, "testpass", config.Password)

	// Test key option
	WithKey("testkey")(config)
	assert.Equal(t, "testkey", config.Key)

	// Test UUID option
	WithUUID("testuuid")(config)
	assert.Equal(t, "testuuid", config.UUID)
}

// Integration tests would require actual device connections
func TestDaikinErrors(t *testing.T) {
	// Test DaikinError
	err := NewDaikinError("test message", nil)
	assert.Equal(t, "daikin error: test message", err.Error())

	// Test DaikinError with wrapped error
	wrappedErr := NewDaikinError("inner error", nil)
	err = NewDaikinError("outer error", wrappedErr)
	assert.Contains(t, err.Error(), "outer error")
	assert.Contains(t, err.Error(), "inner error")

	// Test ConnectionError
	connErr := NewConnectionError("connection failed", nil)
	assert.Contains(t, connErr.Error(), "connection failed")

	// Test AuthenticationError
	authErr := NewAuthenticationError("auth failed", nil)
	assert.Contains(t, authErr.Error(), "auth failed")

	// Test ParseError
	parseErr := NewParseError("parse failed", nil)
	assert.Contains(t, parseErr.Error(), "parse failed")
}

// Mock tests for methods that require HTTP calls
func TestBaseApplianceDefaultMethods(t *testing.T) {
	base := NewBaseAppliance("192.168.1.1", nil)
	ctx := context.Background()

	// Test default implementations return errors
	err := base.Init(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Init method must be implemented")

	err = base.UpdateStatus(ctx)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "UpdateStatus method must be implemented")

	err = base.Set(ctx, map[string]string{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "Set method must be implemented")

	err = base.SetHoliday(ctx, "on")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SetHoliday not supported")

	err = base.SetStreamer(ctx, "on")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SetStreamer not supported")

	err = base.SetAdvancedMode(ctx, "powerful", "on")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "SetAdvancedMode not supported")
}
