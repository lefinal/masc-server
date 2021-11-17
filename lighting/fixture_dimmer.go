package lighting

import (
	"github.com/LeFinal/masc-server/acting"
	"github.com/LeFinal/masc-server/messages"
)

// DimmerFixture is a Fixture that allows setting brightness. Used with
// FixtureFeatureDimmer.
type DimmerFixture interface {
	Fixture
	// SetBrightness sets the brightness of the fixture.
	SetBrightness(brightness float64)
}

// dimmerFixture is the implementation of DimmerFixture.
type dimmerFixture struct {
	basicFixture
	// brightness is the current brightness for the fixture.
	brightness float64
}

// newDimmerFixture creates a new dimmerFixture with initialised values.
func newDimmerFixture(fixtureID messages.FixtureID) *dimmerFixture {
	f := &dimmerFixture{
		basicFixture: *newBasicFixture(fixtureID),
	}
	f.fixtureType = messages.FixtureTypeDimmer
	f.Reset()
	return f
}

func (f *dimmerFixture) SetBrightness(brightness float64) {
	f.m.Lock()
	defer f.m.Unlock()
	f.brightness = brightness
}

func (f *dimmerFixture) Reset() {
	f.brightness = 100
}

func (f *dimmerFixture) Apply() error {
	return f.sendStateUpdate(acting.ActorOutgoingMessage{
		MessageType: messages.MessageTypeFixtureDimmerState,
		Content:     f.fixtureDimmerState(),
	})
}

func (f *dimmerFixture) fixtureDimmerState() messages.MessageFixtureDimmerState {
	basicState := f.fixtureBasicState()
	f.m.RLock()
	defer f.m.RUnlock()
	return messages.MessageFixtureDimmerState{
		MessageFixtureBasicState: basicState,
		Brightness:               f.brightness,
	}
}

func (f *dimmerFixture) State() interface{} {
	return f.fixtureDimmerState()
}
