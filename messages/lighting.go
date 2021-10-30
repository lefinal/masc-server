package messages

// FixtureID identifies a lighting.Fixture.
type FixtureID int

// FixtureProviderFixtureID is the id of a fixture that is being set by the
// fixture provider and remembered.
type FixtureProviderFixtureID string

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

// FixtureFeature describes features a Fixture offers.
type FixtureFeature string

const (
	// FixtureFeatureDimmer allows setting the brightness. Used with
	// lighting.DimmerFixture.
	FixtureFeatureDimmer FixtureFeature = "dimmer"
)

// MessageSetFixtureEnabled is used with MessageTypeSetFixtureEnabled.
type MessageSetFixtureEnabled struct {
	// IsEnabled describes whether the fixture should be enabled or disabled.
	IsEnabled bool `json:"is_enabled"`
}

// MessageSetFixtureLocating is used with MessageTypeSetFixtureLocating.
type MessageSetFixtureLocating struct {
	// IsLocating describes whether the fixture should be in locating mode or not.
	IsLocating bool `json:"is_enabled"`
}

// MessageSetFixtureBrightness is used with MessageTypeSetFixtureBrightness.
type MessageSetFixtureBrightness struct {
	// Brightness is the target brightness.
	Brightness float64 `json:"brightness"`
}

// MessageFixtureBasicSetState is used with MessageTypeFixtureBasicSetState.
type MessageFixtureBasicSetState struct {
	// Fixture is the id of the fixture that is being set by the fixture provider.
	Fixture FixtureProviderFixtureID `json:"fixture"`
	// IsEnabled describes whether the fixture is currently enabled or turned off.
	IsEnabled bool `json:"is_enabled"`
	// IsLocating holds the state set via Fixture.Locate.
	IsLocating bool `json:"is_locating"`
}

// MessageFixtureDimmerSetState is used with MessageTypeFixtureDimmerSetState.
type MessageFixtureDimmerSetState struct {
	MessageFixtureBasicSetState
	// Brightness is the brightness the dimmer should use.
	Brightness float64 `json:"brightness"`
}

type Fixture struct {
	// ProviderID is the id provided by the fixture provider.
	ProviderID FixtureProviderFixtureID `json:"id"`
	// Type is the fixture type. This determines how the fixture is going to be
	// handled.
	Type FixtureType `json:"type"`
}

// MessageFixtures is used with MessageTypeFixtures.
type MessageFixtures struct {
	// DeviceID is the regular device id. We only need it for association of already
	// known fixtures.
	DeviceID DeviceID `json:"device_id"`
	// Fixtures holds all available fixtures.
	Fixtures []Fixture `json:"fixtures"`
}
