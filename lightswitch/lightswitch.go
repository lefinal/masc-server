package lightswitch

import (
	"context"
	"github.com/LeFinal/masc-server/acting"
	"github.com/LeFinal/masc-server/messages"
	"github.com/gobuffalo/nulls"
	"sync"
	"time"
)

type LightSwitch interface {
	// ID returns the id of the light switch.
	ID() messages.LightSwitchID
	// DeviceID is the id of the device the light switch is associated with.
	DeviceID() messages.DeviceID
	// setDeviceID sets the device id that can be retrieved via DeviceID.
	setDeviceID(deviceID messages.DeviceID)
	// ProviderID returns the id that is being assigned to the light switch by the
	// fixture provider.
	ProviderID() messages.ProviderID
	// SetProviderID sets the provider id that can be retrieved via ProviderID.
	setProviderID(providerID messages.ProviderID)
	// Type returns the messages.LightSwitchType of the light switch.
	Type() messages.LightSwitchType
	// Name returns the assigned name.
	Name() nulls.String
	// SetName sets the Name of the light switch.
	setName(name nulls.String)
	// Features returns the features the light switch provides.
	Features() []messages.LightSwitchFeature
	// IsOnline returns true when running.
	IsOnline() bool
	// LastSeen returns the timestamp when the light switch last updated its
	// online-state.
	LastSeen() time.Time
	// setLastSeen sets the timestamp that can be retrieved via LastSeen.
	setLastSeen(lastSeen time.Time)
	// Assignments returns all assigned fixtures.
	Assignments() []messages.FixtureID
	// setAssignments sets the Assignments.
	setAssignments(assignments []messages.FixtureID)
	// run the light switch. Don't forget to set the running state!
	run(ctx context.Context, actor acting.Actor) error
}

type lightSwitchBase struct {
	// id identifies the light switch.
	id messages.LightSwitchID
	// name is the assigned human-readable name.
	name nulls.String
	// deviceID is the id of the device the light switch is associated with.
	deviceID messages.DeviceID
	// providerID is the id the fixture was assigned to from the provider.
	providerID messages.ProviderID
	// actor is the acting.Actor which is used for communication.
	actor acting.Actor
	// lastSeen is the timestamp for when the light switch updated its online-state.
	lastSeen time.Time
	// assignments holds all assigned fixtures.
	assignments []messages.FixtureID
	// isRunning holds the running state for the light switch.
	isRunning bool
	// m locks all properties including the ones from compositing.
	m sync.RWMutex
}

func newLightSwitchBase(lightSwitchID messages.LightSwitchID) lightSwitchBase {
	return lightSwitchBase{
		id:          lightSwitchID,
		lastSeen:    time.Now(),
		assignments: make([]messages.FixtureID, 0),
		isRunning:   false,
	}
}

func (ls *lightSwitchBase) ID() messages.LightSwitchID {
	ls.m.RLock()
	defer ls.m.RUnlock()
	return ls.id
}

func (ls *lightSwitchBase) DeviceID() messages.DeviceID {
	ls.m.RLock()
	defer ls.m.RUnlock()
	return ls.deviceID
}

func (ls *lightSwitchBase) setDeviceID(deviceID messages.DeviceID) {
	ls.m.Lock()
	defer ls.m.Unlock()
	ls.deviceID = deviceID
}

func (ls *lightSwitchBase) ProviderID() messages.ProviderID {
	ls.m.RLock()
	defer ls.m.RUnlock()
	return ls.providerID
}

func (ls *lightSwitchBase) setProviderID(providerID messages.ProviderID) {
	ls.m.Lock()
	defer ls.m.Unlock()
	ls.providerID = providerID
}

func (ls *lightSwitchBase) Name() nulls.String {
	ls.m.RLock()
	defer ls.m.RUnlock()
	return ls.name
}

func (ls *lightSwitchBase) setName(name nulls.String) {
	ls.m.Lock()
	defer ls.m.Unlock()
	ls.name = name
}

func (ls *lightSwitchBase) IsOnline() bool {
	ls.m.RLock()
	defer ls.m.RUnlock()
	return ls.isRunning
}

func (ls *lightSwitchBase) LastSeen() time.Time {
	ls.m.RLock()
	defer ls.m.RUnlock()
	return ls.lastSeen
}

func (ls *lightSwitchBase) setLastSeen(lastSeen time.Time) {
	ls.m.Lock()
	defer ls.m.Unlock()
	ls.lastSeen = lastSeen
}

func (ls *lightSwitchBase) setRunning(isRunning bool) {
	ls.m.Lock()
	defer ls.m.Unlock()
	ls.isRunning = isRunning
}

func (ls *lightSwitchBase) Assignments() []messages.FixtureID {
	ls.m.RLock()
	defer ls.m.RUnlock()
	return ls.assignments
}

func (ls *lightSwitchBase) setAssignments(assignments []messages.FixtureID) {
	ls.m.Lock()
	defer ls.m.Unlock()
	ls.assignments = assignments
	// Catch stupid coding mistakes.
	if ls.assignments == nil {
		ls.assignments = make([]messages.FixtureID, 0)
	}
}
