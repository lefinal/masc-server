package lighting

import (
	"github.com/LeFinal/masc-server/acting"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/messages"
	"sync"
)

// basicFixture implements Fixture for basic access.
type basicFixture struct {
	// id identifies the fixture.
	id messages.FixtureID
	// name is the assigned human-readable name.
	name string
	// deviceID is the id of the device the fixture is assocaited with.
	deviceID messages.DeviceID
	// providerID is the id the fixture was assigned to from the provider.
	providerID messages.FixtureProviderFixtureID
	// actor is the acting.Actor which is used for communication.
	actor acting.Actor
	// isEnabled describes whether the fixture is currently enabled or turned off.
	isEnabled bool
	// fixtureType holds the type of the fixture.
	fixtureType messages.FixtureType
	// isLocating holds the state set via Fixture.Locate.
	isLocating bool
	// m locks all properties including the ones from compositing.
	m sync.RWMutex
}

// newBasicFixture creates a basicFixture with initial values.
func newBasicFixture(fixtureID messages.FixtureID) *basicFixture {
	return &basicFixture{
		id:          fixtureID,
		isEnabled:   false,
		fixtureType: messages.FixtureTypeBasic,
		isLocating:  false,
	}
}

// buildBasicGetStateMessage returns the messages.MessageFixtureBasicSetState for the basicFixture.
//
// Warning: This does not lock basicFixture.m!
func (f *basicFixture) buildBasicGetStateMessage() messages.MessageFixtureBasicSetState {
	return messages.MessageFixtureBasicSetState{
		Fixture:    f.providerID,
		IsEnabled:  f.isEnabled,
		IsLocating: f.isLocating,
	}
}

// sendStateUpdate is used for sending the given message when basicFixture.actor
// is set. Otherwise, an error is being returned. This also applied to errors
// while sending.
func (f *basicFixture) sendStateUpdate(message acting.ActorOutgoingMessage) error {
	f.m.RLock()
	defer f.m.RUnlock()
	if f.actor == nil {
		return errors.Error{
			Code:    errors.ErrCommunication,
			Kind:    errors.KindMissingActor,
			Message: "no actor provided",
		}
	}
	err := f.actor.Send(message)
	if err != nil {
		return errors.Wrap(err, "send")
	}
	return nil
}

func (f *basicFixture) ID() messages.FixtureID {
	f.m.RLock()
	defer f.m.RUnlock()
	return f.id
}

func (f *basicFixture) DeviceID() messages.DeviceID {
	f.m.RLock()
	defer f.m.RUnlock()
	return f.deviceID
}

func (f *basicFixture) ProviderID() messages.FixtureProviderFixtureID {
	f.m.RLock()
	defer f.m.RUnlock()
	return f.providerID
}

func (f *basicFixture) IsEnabled() bool {
	f.m.RLock()
	defer f.m.RUnlock()
	return f.isEnabled
}

func (f *basicFixture) SetEnabled(isEnabled bool) {
	f.m.Lock()
	defer f.m.Unlock()
	f.isEnabled = isEnabled
}

func (f *basicFixture) Reset() {
	// Nothing to do here.
}

func (f *basicFixture) Type() messages.FixtureType {
	f.m.RLock()
	defer f.m.RUnlock()
	return f.fixtureType
}

func (f *basicFixture) Name() string {
	f.m.RLock()
	defer f.m.RUnlock()
	return f.name
}

// setName sets the name of the fixture.
func (f *basicFixture) setName(name string) {
	f.m.Lock()
	defer f.m.Unlock()
	f.name = name
}

func (f *basicFixture) IsOnline() bool {
	f.m.RLock()
	defer f.m.RUnlock()
	return f.actor != nil
}

func (f *basicFixture) setActor(actor acting.Actor) {
	f.m.Lock()
	defer f.m.Unlock()
	f.actor = actor
}

func (f *basicFixture) Features() []messages.FixtureFeature {
	return getFixtureFeatures(f)
}

func (f *basicFixture) IsLocating() bool {
	f.m.RLock()
	defer f.m.RUnlock()
	return f.isLocating
}

func (f *basicFixture) SetLocating(isLocating bool) {
	f.m.Lock()
	defer f.m.Unlock()
	f.isLocating = isLocating
}

func (f *basicFixture) Apply() error {
	f.m.RLock()
	defer f.m.RUnlock()
	return f.sendStateUpdate(acting.ActorOutgoingMessage{
		MessageType: messages.MessageTypeFixtureBasicSetState,
		Content:     f.buildBasicGetStateMessage(),
	})
}

func (f *basicFixture) actorID() messages.ActorID {
	f.m.RLock()
	defer f.m.RUnlock()
	if f.actor == nil {
		return "sad life"
	}
	return f.actor.ID()
}
