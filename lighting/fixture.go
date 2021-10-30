package lighting

import (
	"github.com/LeFinal/masc-server/acting"
	"github.com/LeFinal/masc-server/logging"
	"github.com/LeFinal/masc-server/messages"
)

// Fixture is a general fixture which allows turning on and off. This is the
// most basic one.
//
// Warning: fixtures do not provide any information if the actor quit. Apply
// will return an error with errors.KindMissingActor. The reason for this is
// that lighting is not considered critical.
type Fixture interface {
	// ID returns the id of the fixture.
	ID() messages.FixtureID
	// DeviceID is the id of the device the fixture is associated with.
	DeviceID() messages.DeviceID
	// ProviderID returns the id that is being assigned to the fixture by the
	// fixture provider.
	ProviderID() messages.FixtureProviderFixtureID
	// IsEnabled describes whether the fixture is turned on or off. For setting use
	// SetEnabled.
	IsEnabled() bool
	// SetEnabled turns the fixture on or off. Any changes regarding brightness,
	// color, etc. will have no effect when the fixture is not enabled.
	SetEnabled(isEnabled bool)
	// Reset sets the state to the initial ones. Don't forget to call Apply.
	Reset()
	// Type returns the messages.FixtureType of the fixture.
	Type() messages.FixtureType
	// Name returns the assigned name.
	Name() string
	// SetName sets the Name of the fixture.
	setName(name string)
	// Features returns the features the fixture provides.
	Features() []messages.FixtureFeature
	// IsLocating describes whether the fixture is currently in locating mode. Set
	// via Locate.
	IsLocating() bool
	// SetLocating enables a special mode in which the fixture can be located by humans.
	SetLocating(isLocating bool)
	// Apply sends the status update to the actor.
	Apply() error
	// IsOnline returns true when an actor is set.
	IsOnline() bool
	// setActor sets the acting.Actor for the fixture. If the actor is set, IsOnline
	// returns true.
	setActor(actor acting.Actor)
	// actorID returns the id of the actor that was set via setActor. This should
	// only be used at one specific point where no further checks are required.
	actorID() messages.ActorID
}

// getFixtureFeatures returns the features of the given Fixture.
func getFixtureFeatures(f Fixture) []messages.FixtureFeature {
	features := make([]messages.FixtureFeature, 0)
	// Dimmer.
	if _, ok := f.(DimmerFixture); ok {
		features = append(features, messages.FixtureFeatureDimmer)
	}

	return features
}

// newFixture creates the correct Fixture for the given messages.FixtureType.
func newFixture(fixtureID messages.FixtureID, fixtureType messages.FixtureType) Fixture {
	switch fixtureType {
	case messages.FixtureTypeBasic:
		return newBasicFixture(fixtureID)
	case messages.FixtureTypeDimmer:
		return newDimmerFixture(fixtureID)
	}
	logging.LightingLogger.Warnf("unsupported fixture type %v for fixture %v. using basic one...", fixtureType, fixtureID)
	return newBasicFixture(fixtureID)
}