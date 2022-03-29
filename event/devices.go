package event

import (
	"github.com/gobuffalo/nulls"
	"time"
)

// DeviceOnlineEvent is used for when a device announces its presence.
type DeviceOnlineEvent struct {
	// DeviceID is the id of the device it assigned itself.
	DeviceID string `json:"device_id"`
	// DeviceType is of which type the device describes itself.
	DeviceType string `json:"device_type"`
}

// DeviceOfflineEvent is used when a device goes offline. Most times this should
// be used as LWT.
type DeviceOfflineEvent struct {
	// DeviceID is the id of the device that went offline.
	DeviceID string `json:"device_id"`
}

// DeviceOnlineDetailedEvent is used for mapping online devices announcements to
// detailed one based on stored information.
type DeviceOnlineDetailedEvent struct {
	// DeviceID is the id of the device it assigned itself.
	DeviceID string `json:"device_id"`
	// DeviceType is of which type the device describes itself.
	DeviceType string `json:"device_type"`
	// LastSeen is the last time the device updated its online state.
	LastSeen time.Time `json:"last_seen"`
	// Name is an optional human-readable name for the device.
	Name nulls.String `json:"name"`
}
