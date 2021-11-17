package lighting

import (
	"fmt"
	"github.com/LeFinal/masc-server/acting"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/messages"
	"github.com/LeFinal/masc-server/stores"
	"github.com/gobuffalo/nulls"
	"github.com/google/uuid"
	"sync"
	"time"
)

type MockFixtureProps struct {
	id          messages.FixtureID
	deviceID    messages.DeviceID
	providerID  messages.FixtureProviderFixtureID
	isEnabled   bool
	fixtureType messages.FixtureType
	name        nulls.String
	isLocating  bool
	actor       acting.Actor
	features    []messages.FixtureFeature
}

type MockFixture struct {
	id          messages.FixtureID
	deviceID    messages.DeviceID
	providerID  messages.FixtureProviderFixtureID
	isEnabled   bool
	fixtureType messages.FixtureType
	name        nulls.String
	isLocating  bool
	actor       acting.Actor
	// features are returned by Features.
	features []messages.FixtureFeature
	// lastReset is the last time Reset was called.
	lastReset time.Time
	// lastApply is the last time Apply was called.
	lastApply time.Time
	lastSeen  time.Time
	// m locks all properties from MockFixture.
	m sync.RWMutex
	// ApplyErr for Apply.
	ApplyErr error
}

// NewMockFixture creates a new MockFixture with the given initial values.
func NewMockFixture(props MockFixtureProps) *MockFixture {
	return &MockFixture{
		id:          props.id,
		deviceID:    props.deviceID,
		providerID:  props.providerID,
		isEnabled:   props.isEnabled,
		fixtureType: props.fixtureType,
		name:        props.name,
		isLocating:  props.isLocating,
		actor:       props.actor,
	}
}

func (f *MockFixture) ID() messages.FixtureID {
	f.m.RLock()
	defer f.m.RUnlock()
	return f.id
}

func (f *MockFixture) DeviceID() messages.DeviceID {
	f.m.RLock()
	defer f.m.RUnlock()
	return f.deviceID
}

func (f *MockFixture) setDeviceID(deviceID messages.DeviceID) {
	f.m.Lock()
	defer f.m.Unlock()
	f.deviceID = deviceID
}

func (f *MockFixture) ProviderID() messages.FixtureProviderFixtureID {
	f.m.RLock()
	defer f.m.RUnlock()
	return f.providerID
}

func (f *MockFixture) setProviderID(providerID messages.FixtureProviderFixtureID) {
	f.m.Lock()
	defer f.m.Unlock()
	f.providerID = providerID
}

func (f *MockFixture) IsEnabled() bool {
	f.m.RLock()
	defer f.m.RUnlock()
	return f.isEnabled
}

func (f *MockFixture) SetEnabled(isEnabled bool) {
	f.m.Lock()
	defer f.m.Unlock()
	f.isEnabled = isEnabled
}

func (f *MockFixture) Reset() {
	f.m.Lock()
	defer f.m.Unlock()
	f.lastReset = time.Now()
}

func (f *MockFixture) Type() messages.FixtureType {
	f.m.RLock()
	defer f.m.RUnlock()
	return f.fixtureType
}

func (f *MockFixture) Name() nulls.String {
	f.m.RLock()
	defer f.m.RUnlock()
	return f.name
}

func (f *MockFixture) setName(name nulls.String) {
	f.m.Lock()
	defer f.m.Unlock()
	f.name = name
}

func (f *MockFixture) setFeatures(features ...messages.FixtureFeature) {
	f.features = features
}

func (f *MockFixture) Features() []messages.FixtureFeature {
	f.m.RLock()
	defer f.m.RUnlock()
	features := make([]messages.FixtureFeature, 0, len(f.features))
	for _, feature := range f.features {
		features = append(features, feature)
	}
	return features
}

func (f *MockFixture) IsLocating() bool {
	f.m.RLock()
	defer f.m.RUnlock()
	return f.isLocating
}

func (f *MockFixture) SetLocating(isLocating bool) {
	f.m.Lock()
	defer f.m.Unlock()
	f.isLocating = isLocating
}

func (f *MockFixture) Apply() error {
	if f.ApplyErr != nil {
		return f.ApplyErr
	}
	f.m.Lock()
	defer f.m.Unlock()
	f.lastApply = time.Now()
	return nil
}

func (f *MockFixture) IsOnline() bool {
	f.m.RLock()
	defer f.m.RUnlock()
	return f.actor != nil
}

func (f *MockFixture) setActor(actor acting.Actor) {
	f.m.Lock()
	defer f.m.Unlock()
	f.actor = actor
}

func (f *MockFixture) actorID() messages.ActorID {
	f.m.RLock()
	defer f.m.RUnlock()
	if f.actor == nil {
		panic("no actor set")
	}
	return f.actor.ID()
}

func (f *MockFixture) LastSeen() time.Time {
	f.m.RLock()
	defer f.m.RUnlock()
	return f.lastSeen
}

func (f *MockFixture) setLastSeen(lastSeen time.Time) {
	f.m.Lock()
	defer f.m.Unlock()
	f.lastSeen = lastSeen
}

func (f *MockFixture) State() interface{} {
	// TODO
	return nil
}

func (f *MockFixture) setUpdateNotifier(notifier FixtureStateUpdateNotifier) {
	// TODO
}

// MockManagerStore mocks ManagerStore.
type MockManagerStore struct {
	// fixtures holds all registered fixtures.
	fixtures map[messages.FixtureID]stores.Fixture
	// m locks the whole MockManagerStore.
	m sync.RWMutex
	// GetFixturesErr for GetFixtures.
	GetFixturesErr error
	// CreateFixtureErr for CreateFixture.
	CreateFixtureErr error
	// DeleteFixtureErr for DeleteFixture.
	DeleteFixtureErr error
	// SetFixtureNameErr for SetFixtureName.
	SetFixtureNameErr error
	// SetFixtureOnlineErr for SetFixtureOnlineErr.
	SetFixtureOnlineErr error
}

// NewMockManagerStore creates a new MockManagerStore with the given initial
// fixtures.
func NewMockManagerStore(fixtures ...stores.Fixture) *MockManagerStore {
	manager := &MockManagerStore{
		fixtures: make(map[messages.FixtureID]stores.Fixture),
	}
	for _, fixture := range fixtures {
		if _, ok := manager.fixtures[fixture.ID]; ok {
			panic(fmt.Sprintf("duplicate fixture id %v while creating mock manager store", fixture.ID))
		}
		manager.fixtures[fixture.ID] = fixture
	}
	return manager
}

func (manager *MockManagerStore) GetFixtures() ([]stores.Fixture, error) {
	if manager.GetFixturesErr != nil {
		return nil, manager.GetFixturesErr
	}
	manager.m.RLock()
	defer manager.m.RUnlock()
	fixtures := make([]stores.Fixture, 0, len(manager.fixtures))
	for _, fixture := range manager.fixtures {
		fixtures = append(fixtures, fixture)
	}
	return fixtures, nil
}

func (manager *MockManagerStore) CreateFixture(fixture stores.Fixture) (stores.Fixture, error) {
	if manager.CreateFixtureErr != nil {
		return stores.Fixture{}, manager.CreateFixtureErr
	}
	manager.m.Lock()
	defer manager.m.Unlock()
	id := messages.FixtureID(uuid.New().ID())
	fixture.ID = id
	manager.fixtures[id] = fixture
	return fixture, nil
}

func (manager *MockManagerStore) DeleteFixture(fixtureID messages.FixtureID) error {
	if manager.DeleteFixtureErr != nil {
		return manager.DeleteFixtureErr
	}
	manager.m.Lock()
	defer manager.m.Unlock()
	if _, ok := manager.fixtures[fixtureID]; !ok {
		return errors.NewResourceNotFoundError(fmt.Sprintf("fixture %v not found", fixtureID), errors.Details{})
	}
	delete(manager.fixtures, fixtureID)
	return nil
}

func (manager *MockManagerStore) SetFixtureName(fixtureID messages.FixtureID, name nulls.String) error {
	if manager.SetFixtureNameErr != nil {
		return manager.SetFixtureNameErr
	}
	manager.m.Lock()
	defer manager.m.Unlock()
	if fixture, ok := manager.fixtures[fixtureID]; !ok {
		return errors.NewResourceNotFoundError(fmt.Sprintf("fixture %v not found", fixtureID), errors.Details{})
	} else {
		fixture.Name = name
		manager.fixtures[fixtureID] = fixture
	}
	return nil
}

func (manager *MockManagerStore) RefreshLastSeenForFixture(fixtureID messages.FixtureID) error {
	if manager.SetFixtureOnlineErr != nil {
		return manager.SetFixtureOnlineErr
	}
	manager.m.Lock()
	defer manager.m.Unlock()
	if fixture, ok := manager.fixtures[fixtureID]; !ok {
		return errors.NewResourceNotFoundError(fmt.Sprintf("fixture %v not found", fixtureID), errors.Details{})
	} else {
		fixture.LastSeen = time.Now()
		manager.fixtures[fixtureID] = fixture
	}
	return nil
}
