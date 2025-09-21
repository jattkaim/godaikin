package godaikin

import (
	"context"
	"crypto/tls"
	"fmt"
	"net/http"
)

type DaikinBRP072C struct {
	*DaikinBRP069
	Key  string
	UUID string
}

// NewDaikinBRP072C creates BRP072C device
func NewDaikinBRP072C(deviceIP, key, uuid string, logger Logger) *DaikinBRP072C {
	brp069 := NewDaikinBRP069(deviceIP, logger)
	brp069.BaseURL = fmt.Sprintf("https://%s", deviceIP)

	brp069.HTTPClient.Transport = &http.Transport{
		TLSClientConfig: &tls.Config{
			InsecureSkipVerify: true,
		},
	}

	if uuid == "" {
		uuid = "pydaikin00000000000000000000000000000000"
	}

	brp069.Headers["X-Daikin-uuid"] = uuid

	return &DaikinBRP072C{
		DaikinBRP069: brp069,
		Key:          key,
		UUID:         uuid,
	}
}

func (d *DaikinBRP072C) GetDeviceType() string {
	return "BRP072C"
}

func (d *DaikinBRP072C) Init(ctx context.Context) error {
	_, err := d.getResource(ctx, "common/register_terminal", map[string]string{"key": d.Key})
	if err != nil {
		return fmt.Errorf("failed to register terminal: %w", err)
	}

	return d.DaikinBRP069.Init(ctx)
}

// Override getResource to use the proper base appliance method
func (d *DaikinBRP072C) getResource(ctx context.Context, path string, params map[string]string) (map[string]string, error) {
	return d.BaseAppliance.getResource(ctx, path, params)
}

// All other methods inherit from BRP069 automatically through embedded struct
// This includes: UpdateStatus, Set, SetHoliday, SetAdvancedMode, SetStreamer, etc.

// Support properties inherited from BRP069 through embedded struct
// These methods are automatically inherited through the *DaikinBRP069 embedded field
// No need to explicitly define them - Go handles this automatically
