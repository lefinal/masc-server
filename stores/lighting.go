package stores

import (
	"github.com/LeFinal/masc-server/messages"
	"github.com/gobuffalo/nulls"
	"time"
)

// Fixture is the store representation of lighting.Fixture.
type Fixture struct {
	// ID identifies the fixture.
	ID messages.FixtureID
	// Device is the device id of the provider which is needed in order to reuse
	// fixtures.
	Device messages.DeviceID
	// ProviderID is the id for the fixture the provider assigned to it.
	ProviderID messages.FixtureProviderFixtureID
	// Name is a human-readable description of the fixture.
	Name nulls.String
	// Type is the fixture type.
	Type messages.FixtureType
	// LastSeen is the last time the fixture online state changed.
	LastSeen time.Time
	// IsOnline is true when the fixture is currently online.
	IsOnline bool
}
