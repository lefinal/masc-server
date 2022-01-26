package messages

import (
	"github.com/gobuffalo/nulls"
	"time"
)

const (
	// MessageTypeDeleteLightSwitch is used with MessageDeleteLightSwitch in order
	// to remove a light switch.
	MessageTypeDeleteLightSwitch MessageType = "delete-light-switch"
	// MessageTypeGetLightSwitches is received from the client when he wants to
	// retrieve all light switches. MASC should respond with a
	// MessageTypeLightSwitchList.
	MessageTypeGetLightSwitches MessageType = "get-light-switches"
	// MessageTypeGetLightSwitchOffers is sent to the client for requesting
	// available light switches.
	MessageTypeGetLightSwitchOffers MessageType = "get-light-switch-offers"
	// MessageTypeLightSwitchHiLoState is used by clients for notifying of a
	// LightSwitchTypeHiLo state.
	MessageTypeLightSwitchHiLoState MessageType = "light-switch-hi-lo-state"
	// MessageTypeLightSwitchList is used with MessageLightSwitchList sent to the
	// client when he requests a light switch list.
	MessageTypeLightSwitchList MessageType = "light-switch-list"
	// MessageTypeLightSwitchOffers is received from light switch providers when
	// requesting all available light switches using
	// MessageTypeGetLightSwitchOffers.
	MessageTypeLightSwitchOffers MessageType = "light-switch-offers"
	// MessageTypeUpdateLightSwitch is used with MessageUpdateLightSwitch in order
	// to set the name and assignments for an OfferedLightSwitch.
	MessageTypeUpdateLightSwitch MessageType = "update-light-switch"
)

// LightSwitchID identifies a lightswitch.LightSwitch.
type LightSwitchID int

// LightSwitchType is the type of light switch.
type LightSwitchType string

const (
	// LightSwitchTypeHiLo is a light switch that toggles the enabled state by
	// sending high/low states.
	LightSwitchTypeHiLo LightSwitchType = "hi-lo"
)

// LightSwitchFeature is used for when light switches offer features like
// illumination of enabled-states.
type LightSwitchFeature string

// MessageLightSwitchBase is the base structure for messages regarding light
// switches.
type MessageLightSwitchBase struct {
	// LightSwitch is the id of the switch that is being addressed for/by the
	// provider.
	LightSwitch ProviderID `json:"light_switch"`
}

// MessageLightSwitchHiLoState is used with MessageTypeLightSwitchHiLoState.
type MessageLightSwitchHiLoState struct {
	// IsHigh describes whether we are currently high. If set to false, we are low.
	IsHigh bool `json:"is_high"`
}

// OfferedLightSwitch is used with MessageLightSwitchOffers.
type OfferedLightSwitch struct {
	// ProviderID is the id provided by the light switch provider.
	ProviderID ProviderID `json:"provider_id"`
	// Type is the light switch type. This determines how the light switch is going
	// to be handled.
	Type LightSwitchType `json:"type"`
}

// MessageLightSwitchOffers is used with MessageTypeLightSwitchOffers.
type MessageLightSwitchOffers struct {
	// DeviceID is the regular device id. We only need it for association of already
	// known light switches.
	DeviceID DeviceID `json:"device_id"`
	// LightSwitches holds all available light switches.
	LightSwitches []OfferedLightSwitch `json:"light_switches"`
}

// MessageUpdateLightSwitch is used with MessageTypeUpdateLightSwitch.
type MessageUpdateLightSwitch struct {
	// LightSwitchID is the id of the light switch to update.
	LightSwitchID LightSwitchID `json:"light_switch_id"`
	// Name is the new name.
	Name nulls.String `json:"name,omitempty"`
	// Assignments are the assigned fixtures.
	Assignments []FixtureID `json:"assignments"`
}

// LightSwitch is the message representation of lightswitch.LightSwitch and
// stores.LightSwitch.
type LightSwitch struct {
	// ID identifies the light switch.
	ID LightSwitchID `json:"id"`
	// DeviceID is the device the light switch is associated with.
	DeviceID DeviceID `json:"device_id"`
	// DeviceName holds the optional human-readable device name.
	DeviceName nulls.String `json:"device_name"`
	// ProviderID is how the light switch is identified by the device itself.
	ProviderID ProviderID `json:"provider_id"`
	// Type is the LightSwitchType which describes its features.
	Type LightSwitchType `json:"type"`
	// Name is the optionally assigned human-readable name.
	Name nulls.String `json:"name,omitempty"`
	// Features returns available features based on the Type.
	Features []LightSwitchFeature `json:"features"`
	// IsOnline describes whether the light switch is currently connected.
	IsOnline bool `json:"is_online"`
	// LastSeen is the last time the online-state was updated.
	LastSeen time.Time `json:"last_seen"`
	// Assignments are the assigned fixtures.
	Assignments []FixtureID `json:"assignments"`
}

// MessageLightSwitchList is used with MessageTypeLightSwitchList.
type MessageLightSwitchList struct {
	LightSwitches []LightSwitch `json:"light_switches"`
}

// MessageDeleteLightSwitch is used with MessageTypeDeleteLightSwitch.
type MessageDeleteLightSwitch struct {
	// LightSwitchID is the id of the light switch to delete.
	LightSwitchID LightSwitchID `json:"light_switch_id"`
}
