package lighting

import (
	"github.com/LeFinal/masc-server/acting"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/logging"
	"github.com/LeFinal/masc-server/messages"
	"github.com/gobuffalo/nulls"
	"go.uber.org/zap"
	"sync"
	"time"
)

// basicFixture implements Fixture for basic access.
type basicFixture struct {
	// id identifies the fixture.
	id messages.FixtureID
	// name is the assigned human-readable name.
	name nulls.String
	// deviceID is the id of the device the fixture is associated with.
	deviceID messages.DeviceID
	// providerID is the id the fixture was assigned to from the provider.
	providerID messages.ProviderID
	// actor is the acting.Actor which is used for communication.
	actor acting.Actor
	// isEnabled describes whether the fixture is currently enabled or turned off.
	isEnabled bool
	// fixtureType holds the type of the fixture.
	fixtureType messages.FixtureType
	// isLocating holds the state set via Fixture.Locate.
	isLocating bool
	// lastSeen is the timestamp for when the fixture updated its online-state.
	lastSeen time.Time
	// updateNotifier is called when updates are being applied.
	updateNotifier FixtureStateUpdateNotifier
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
		lastSeen:    time.Now(),
	}
}

// State returns the messages.MessageFixtureBasicState for the basicFixture.
func (f *basicFixture) State() interface{} {
	return f.fixtureBasicState()
}

func (f *basicFixture) fixtureBasicState() messages.MessageFixtureBasicState {
	f.m.RLock()
	defer f.m.RUnlock()
	return messages.MessageFixtureBasicState{
		Fixture:    f.providerID,
		IsEnabled:  f.isEnabled,
		IsLocating: f.isLocating,
	}
}

// sendStateUpdate is used for sending the given message when basicFixture.actor
// is set. Otherwise, an error is being returned. This also applies to errors
// while sending. This also calls the update notifier.
func (f *basicFixture) sendStateUpdate(message acting.ActorOutgoingMessage) error {
	f.m.RLock()
	defer f.m.RUnlock()
	if f.updateNotifier != nil {
		f.updateNotifier.HandleFixtureStateUpdated()
	}
	if f.actor == nil {
		logging.LightingLogger.Warn("dropping apply for fixture due to no actor being set",
			zap.Any("fixture_id", f.ID()))
		return nil
	}
	err := f.actor.Send(message)
	if err != nil {
		return errors.Wrap(err, "send", nil)
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

func (f *basicFixture) setDeviceID(deviceID messages.DeviceID) {
	f.m.Lock()
	defer f.m.Unlock()
	f.deviceID = deviceID
}

func (f *basicFixture) ProviderID() messages.ProviderID {
	f.m.RLock()
	defer f.m.RUnlock()
	return f.providerID
}

func (f *basicFixture) setProviderID(providerID messages.ProviderID) {
	f.m.Lock()
	defer f.m.Unlock()
	f.providerID = providerID
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
	if !isEnabled {
		f.isLocating = false
	}
}

func (f *basicFixture) ToggleEnabled() {
	f.m.Lock()
	defer f.m.Unlock()
	f.isEnabled = !f.isEnabled
	if !f.isEnabled {
		f.isLocating = false
	}
}

func (f *basicFixture) Reset() {
	// Nothing to do here.
}

func (f *basicFixture) Type() messages.FixtureType {
	f.m.RLock()
	defer f.m.RUnlock()
	return f.fixtureType
}

func (f *basicFixture) Name() nulls.String {
	f.m.RLock()
	defer f.m.RUnlock()
	return f.name
}

// setName sets the name of the fixture.
func (f *basicFixture) setName(name nulls.String) {
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
	return f.sendStateUpdate(acting.ActorOutgoingMessage{
		MessageType: messages.MessageTypeFixtureBasicState,
		Content:     f.fixtureBasicState(),
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

func (f *basicFixture) LastSeen() time.Time {
	f.m.RLock()
	defer f.m.RUnlock()
	return f.lastSeen
}

func (f *basicFixture) setLastSeen(lastSeen time.Time) {
	f.m.Lock()
	defer f.m.Unlock()
	f.lastSeen = lastSeen
}

func (f *basicFixture) setUpdateNotifier(notifier FixtureStateUpdateNotifier) {
	f.m.Lock()
	defer f.m.Unlock()
	f.updateNotifier = notifier
}
