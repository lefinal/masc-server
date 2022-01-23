package messages

import (
	"github.com/gobuffalo/nulls"
	"time"
)

const (
	// MessageTypeDeleteFixture is used with MessageDeleteFixture for deleting a
	// Fixture.
	MessageTypeDeleteFixture MessageType = "delete-fixture"
	// MessageTypeFixtureBasicState is used for setting the state of
	// FixtureTypeBasic.
	MessageTypeFixtureBasicState MessageType = "fixture-basic-state"
	// MessageTypeFixtureDimmerState is used for setting the state of
	// FixtureTypeDimmer.
	MessageTypeFixtureDimmerState MessageType = "fixture-dimmer-state"
	// MessageTypeFixtureList is used with MessageFixtureList as an answer to
	// MessageTypeGetFixtures.
	MessageTypeFixtureList MessageType = "fixture-list"
	// MessageTypeFixtureStates is used for transmitting the state of all known
	// fixtures to an operator.
	MessageTypeFixtureStates MessageType = "fixture-states"
	// MessageTypeGetFixtureOffers is sent to the client for requesting fixture
	// offers.
	MessageTypeGetFixtureOffers MessageType = "get-fixture-offers"
	// MessageTypeGetFixtures is received for requesting all known fixtures.
	MessageTypeGetFixtures MessageType = "get-fixtures"
	// MessageTypeGetFixtureStates is received from fixture operators for retrieving
	// fixture states that are being returned with MessageTypeFixtureStates.
	MessageTypeGetFixtureStates MessageType = "get-fixture-states"
	// MessageTypeFixtureOffers provides all available fixtures from a fixture
	// provider. Used with MessageFixtureOffers.
	MessageTypeFixtureOffers MessageType = "fixture-offers"
	// MessageTypeSetFixtureName is used with MessageSetFixtureName for setting the
	// name of an OfferedFixture.
	MessageTypeSetFixtureName MessageType = "set-fixture-name"
	// MessageTypeSetFixturesEnabled is used with MessageSetFixturesEnabled in order
	// to set the enabled state for fixtures.
	MessageTypeSetFixturesEnabled MessageType = "set-fixtures-enabled"
	// MessageTypeSetFixturesLocating is used with MessageSetFixturesLocating in
	// order to set the enabled state for fixtures.
	MessageTypeSetFixturesLocating MessageType = "set-fixtures-locating"
)

// FixtureID identifies a lighting.Fixture.
type FixtureID int

// FixtureType is the type of Light which is used for human-readability.
type FixtureType string

const (
	// FixtureTypeBasic is used for fixtures that provide only capabilities of being
	// turned on and off as well as locating mode.
	FixtureTypeBasic FixtureType = "basic"
	// FixtureTypeDimmer is used for fixtures that add dimmer functionality to
	// FixtureTypeBasic.
	FixtureTypeDimmer FixtureType = "dimmer"
)

// FixtureFeature describes features a OfferedFixture offers.
type FixtureFeature string

const (
	// FixtureFeatureDimmer allows setting the brightness. Used with
	// lighting.DimmerFixture.
	FixtureFeatureDimmer FixtureFeature = "dimmer"
)

// MessageFixtureStatesFixture is used in MessageFixtureStates as a container
// for the actual message state. This matches most fields of
// MessageFixtureBasicState but for compatibility and reliability, they are
// treated differently.
type MessageFixtureStatesFixture struct {
	// ID is the id of the fixture.
	ID FixtureID `json:"id"`
	// FixtureType is the type of the fixture.
	FixtureType FixtureType `json:"fixture_type"`
	// DeviceID
	DeviceID DeviceID `json:"device_id"`
	// ProviderID is how the provider identifies the fixture.
	ProviderID ProviderID `json:"provider_id"`
	// Name is an optionally assigned fixture name.
	Name nulls.String `json:"name"`
	// IsOnline determines whether the fixture is online.
	IsOnline bool `json:"is_online"`
	// IsEnabled describes whether the fixture is currently enabled or turned off.
	IsEnabled bool `json:"is_enabled"`
	// IsLocating holds the state set via Fixture.Locate.
	IsLocating bool `json:"is_locating"`
	// State is the actual fixture state that is different for each FixtureType.
	State interface{} `json:"state"`
}

// MessageFixtureStates is used with MessageTypeFixtureStates.
type MessageFixtureStates struct {
	Fixtures []MessageFixtureStatesFixture `json:"fixtures"`
}

// MessageFixtureBasicState is used with MessageTypeFixtureBasicState.
type MessageFixtureBasicState struct {
	// Fixture is the id of the fixture that is being set by the fixture provider.
	Fixture ProviderID `json:"fixture"`
	// IsEnabled describes whether the fixture is currently enabled or turned off.
	IsEnabled bool `json:"is_enabled"`
	// IsLocating holds the state set via Fixture.Locate.
	IsLocating bool `json:"is_locating"`
}

// MessageFixtureDimmerState is used with MessageTypeFixtureDimmerState.
type MessageFixtureDimmerState struct {
	MessageFixtureBasicState
	// Brightness is the brightness the dimmer should use.
	Brightness float64 `json:"brightness"`
}

// OfferedFixture is an offered fixture in MessageFixtureOffers.
type OfferedFixture struct {
	// ProviderID is the id provided by the fixture provider.
	ProviderID ProviderID `json:"id"`
	// Type is the fixture type. This determines how the fixture is going to be
	// handled.
	Type FixtureType `json:"type"`
}

// MessageFixtureOffers is used with MessageTypeFixtureOffers.
type MessageFixtureOffers struct {
	// DeviceID is the regular device id. We only need it for association of already
	// known fixtures.
	DeviceID DeviceID `json:"device_id"`
	// Fixtures holds all available fixtures.
	Fixtures []OfferedFixture `json:"fixtures"`
}

// MessageSetFixtureName is used with MessageTypeSetFixtureName.
type MessageSetFixtureName struct {
	// FixtureID is the id of the fixture to set the name for.
	FixtureID FixtureID `json:"fixture_id"`
	// Name is the new name.
	Name nulls.String `json:"name,omitempty"`
}

// Fixture is the message version of lighting.Fixture and stores.Fixture.
type Fixture struct {
	// ID identifies the fixture.
	ID FixtureID `json:"id"`
	// DeviceID is the device the fixture is associated with.
	DeviceID DeviceID `json:"device_id"`
	// ProviderID is how the fixture is identified by the device itself.
	ProviderID ProviderID `json:"provider_id"`
	// IsEnabled describes whether the fixture is currently turned on.
	IsEnabled bool `json:"is_enabled"`
	// Type is the FixtureType which describes its features.
	Type FixtureType `json:"type"`
	// Name is the optionally assigned human-readable name.
	Name nulls.String `json:"name,omitempty"`
	// Features returns available features based on the Type.
	Features []FixtureFeature `json:"features"`
	// IsLocating states whether the fixture is currently in locating-mode. This is
	// not set when not online.
	IsLocating bool `json:"is_locating,omitempty"`
	// IsOnline describes whether the fixture is currently connected.
	IsOnline bool `json:"is_online"`
	// LastSeen is the last time the online-state was updated.
	LastSeen time.Time `json:"last_seen"`
}

// MessageFixtureList is used with MessageTypeFixtureList.
type MessageFixtureList struct {
	Fixtures []Fixture `json:"fixtures"`
}

// MessageDeleteFixture is used with MessageTypeDeleteFixture.
type MessageDeleteFixture struct {
	// FixtureID is the id of the device to delete.
	FixtureID FixtureID `json:"fixture_id"`
}

// MessageSetFixturesEnabledFixtureState is used in MessageSetFixturesEnabled
// for setting the enabled state of fixtures.
type MessageSetFixturesEnabledFixtureState struct {
	FixtureID FixtureID `json:"fixture_id"`
	IsEnabled bool      `json:"is_enabled"`
}

// MessageSetFixturesEnabled is used with MessageTypeSetFixturesEnabled.
type MessageSetFixturesEnabled struct {
	Fixtures []MessageSetFixturesEnabledFixtureState `json:"fixtures"`
}

// MessageSetFixturesLocatingFixtureState is used in MessageSetFixturesLocating
// for setting the locating-mode of fixtures.
type MessageSetFixturesLocatingFixtureState struct {
	FixtureID  FixtureID `json:"fixture_id"`
	IsLocating bool      `json:"is_locating"`
}

// MessageSetFixturesLocating is used with MessageTypeSetFixturesLocating.
type MessageSetFixturesLocating struct {
	Fixtures []MessageSetFixturesLocatingFixtureState `json:"fixtures"`
}
