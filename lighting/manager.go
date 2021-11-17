package lighting

import (
	"context"
	"fmt"
	"github.com/LeFinal/masc-server/acting"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/logging"
	"github.com/LeFinal/masc-server/messages"
	"github.com/LeFinal/masc-server/stores"
	"github.com/gobuffalo/nulls"
	"sort"
	"sync"
	"time"
)

var fixtureStateBroadcastBufferDuration = 100 * time.Millisecond

type ManagerStore interface {
	// GetFixtures returns all known fixtures.
	GetFixtures() ([]stores.Fixture, error)
	// CreateFixture creates the given fixture and returns the one with the assigned
	// id.
	CreateFixture(fixture stores.Fixture) (stores.Fixture, error)
	// DeleteFixture deletes the fixture with the given id.
	DeleteFixture(fixtureID messages.FixtureID) error
	// SetFixtureName sets the name of the fixture with the given id.
	SetFixtureName(fixtureID messages.FixtureID, name nulls.String) error
	// RefreshLastSeenForFixture sets the last seen field for the fixture to the
	// current time.
	RefreshLastSeenForFixture(fixtureID messages.FixtureID) error
}

// Manager allows managing fixtures and their providers.
type Manager interface {
	// Run runs the Manager.
	Run(ctx context.Context)
	// LoadKnownFixtures loads all known fixtures from the ManagerStore.
	LoadKnownFixtures() error
	// AcceptFixtureProvider accepts an acting.Actor with
	// acting.RoleTypeFixtureProvider and handles fixture discovery.
	AcceptFixtureProvider(ctx context.Context, actor acting.Actor) error
	// SayGoodbyeToFixtureProvider is used for when the actor should no longer be
	// available in fixtures. This is usually the case, when the actor quits.
	SayGoodbyeToFixtureProvider(actor acting.Actor) error
	// SetFixtureName sets the name of the given Fixture.
	SetFixtureName(fixtureID messages.FixtureID, name nulls.String) error
	// OnlineFixtures returns all fixtures that are currently online.
	OnlineFixtures() []Fixture
	// Fixtures returns all fixtures including offline ones.
	Fixtures() []Fixture
	// DeleteFixture deletes the given Fixture from the store and from the Manager.
	DeleteFixture(fixtureID messages.FixtureID) error
	// GetFixtureStatesBroadcast is the channel for fixture state updates are being
	// read from.
	GetFixtureStatesBroadcast() <-chan messages.MessageFixtureStates
	// FixtureStates returns all fixture states. Usually, important for fixture
	// operators.
	FixtureStates() messages.MessageFixtureStates
}

// StoredManager is the implementation of Manager.
type StoredManager struct {
	// store enables fixture persistence.
	store ManagerStore
	// fixtures holds all fixtures by their associated actor.
	fixtures map[messages.FixtureID]Fixture
	// fixtureStateBroadcaster is the broadcaster for publishing fixture states.
	fixtureStateBroadcaster *fixtureStateBroadcaster
	// m locks the StoredManager.
	m sync.RWMutex
}

// NewStoredManager creates a new Manager.
func NewStoredManager(store ManagerStore) *StoredManager {
	m := &StoredManager{
		store:    store,
		fixtures: make(map[messages.FixtureID]Fixture),
	}
	m.fixtureStateBroadcaster = newFixtureStateBroadcaster(m, fixtureStateBroadcastBufferDuration)
	return m
}

func (manager *StoredManager) Run(ctx context.Context) {
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		manager.fixtureStateBroadcaster.run(ctx)
	}()
	logging.LightingLogger.Info("lighting manager running")
	<-ctx.Done()
	wg.Wait()
	logging.LightingLogger.Info("lighting manager shut down.")
}

func (manager *StoredManager) LoadKnownFixtures() error {
	manager.m.Lock()
	defer manager.m.Unlock()
	fixtures, err := manager.store.GetFixtures()
	if err != nil {
		return errors.Wrap(err, "retrieve fixtures from store")
	}
	for _, storeFixture := range fixtures {
		f := newFixture(storeFixture.ID, storeFixture.Type)
		// Apply fields from store fixture.
		if storeFixture.Name.Valid {
			f.setName(storeFixture.Name)
		}
		f.setDeviceID(storeFixture.Device)
		f.setProviderID(storeFixture.ProviderID)
		f.setLastSeen(storeFixture.LastSeen)
		f.setUpdateNotifier(manager.fixtureStateBroadcaster)
		manager.fixtures[storeFixture.ID] = f
	}
	return nil
}

func (manager *StoredManager) AcceptFixtureProvider(ctx context.Context, actor acting.Actor) error {
	// Request available fixtures.
	fixtureRes := acting.SubscribeMessageTypeFixtureOffers(actor)
	defer acting.UnsubscribeOrLogError(fixtureRes.Newsletter)
	err := actor.Send(acting.ActorOutgoingMessage{MessageType: messages.MessageTypeGetFixtureOffers})
	if err != nil {
		return errors.Wrap(err, "request fixtures from provider")
	}
	var messageFixtures messages.MessageFixtureOffers
	select {
	case <-ctx.Done():
		return errors.NewContextAbortedError("request available fixtures")
	case messageFixtures = <-fixtureRes.Receive:
	}
	// Add them.
	acceptedFixtures, err := manager.addFixturesFromProviderMessage(messageFixtures)
	if err != nil {
		return errors.Wrap(err, "add fixtures from provider message")
	}
	// Set fixtures to online, initialise and apply.
	manager.m.Lock()
	defer manager.m.Unlock()
	for _, fixture := range acceptedFixtures {
		err = manager.store.RefreshLastSeenForFixture(fixture.ID())
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("set fixture %v online", fixture.ID()))
		}
		fixture.setActor(actor)
		fixture.setUpdateNotifier(manager.fixtureStateBroadcaster)
		fixture.Reset()
		err = fixture.Apply()
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("apply initial state for fixture %v", fixture.ID()))
		}
	}
	return nil
}

// addFixturesFromProviderMessage adds the fixtures from the given
// messages.MessageFixtureOffers to StoredManager.fixtures. If a fixture is unknown, it
// will be created using the StoredManager.store. Fixtures will NOT be set to online
// and will not call Fixture.Apply.
func (manager *StoredManager) addFixturesFromProviderMessage(m messages.MessageFixtureOffers) ([]Fixture, error) {
	manager.m.Lock()
	defer manager.m.Unlock()
	fixtures := make([]Fixture, 0, len(m.Fixtures))
	// Iterate over all fixtures.
addFixtures:
	for _, fixtureToAdd := range m.Fixtures {
		// Check if already existing.
		for _, knownFixture := range manager.fixtures {
			if knownFixture.DeviceID() == m.DeviceID && knownFixture.ProviderID() == fixtureToAdd.ProviderID {
				// Fixture is known, yay. But is the type correct?
				if knownFixture.Type() != fixtureToAdd.Type {
					// Shit.
					return nil, errors.Error{
						Code: errors.ErrInternal,
						Kind: errors.KindFixtureTypeConflict,
						Message: fmt.Sprintf("fixture %v from device %v already known with type %v but now is %v",
							fixtureToAdd.ProviderID, m.DeviceID, knownFixture.Type(), fixtureToAdd.Type),
						Details: errors.Details{
							"knownType": knownFixture.Type(),
							"newType":   fixtureToAdd.Type,
						},
					}
				}
				// We set the fixture later to online, because errors might still occur.
				fixtures = append(fixtures, knownFixture)
				continue addFixtures
			}
		}
		// Fixture unknown -> create.
		created, err := manager.store.CreateFixture(stores.Fixture{
			Device:     m.DeviceID,
			ProviderID: fixtureToAdd.ProviderID,
			LastSeen:   time.Now(),
			Type:       fixtureToAdd.Type,
		})
		if err != nil {
			return nil, errors.Wrap(err, fmt.Sprintf("create fixture %v from device %v",
				fixtureToAdd.ProviderID, m.DeviceID))
		}
		newFixture := newFixture(created.ID, created.Type)
		manager.fixtures[newFixture.ID()] = newFixture
		fixtures = append(fixtures, newFixture)
	}
	return fixtures, nil
}

func (manager *StoredManager) SayGoodbyeToFixtureProvider(actor acting.Actor) error {
	manager.m.Lock()
	defer manager.m.Unlock()
	// Set all fixtures offline for the actor.
	for fixtureID, fixture := range manager.fixtures {
		if fixture.actorID() != actor.ID() {
			continue
		}
		fixture.setActor(nil)
		err := manager.store.RefreshLastSeenForFixture(fixtureID)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("set fixture %v offline", fixtureID))
		}
	}
	return nil
}

func (manager *StoredManager) SetFixtureName(fixtureID messages.FixtureID, name nulls.String) error {
	manager.m.Lock()
	defer manager.m.Unlock()
	// Search for fixture in online ones.
	fixture, ok := manager.fixtures[fixtureID]
	if !ok {
		return errors.NewResourceNotFoundError(fmt.Sprintf("fixture %v not found", fixtureID),
			errors.Details{"fixture": fixtureID, "name": name})
	}
	err := manager.store.SetFixtureName(fixtureID, name)
	if err != nil {
		return errors.Wrap(err, "set fixture name in store")
	}
	fixture.setName(name)
	return nil
}

func (manager *StoredManager) OnlineFixtures() []Fixture {
	manager.m.RLock()
	defer manager.m.RUnlock()
	onlineFixtures := make([]Fixture, 0)
	for _, fixture := range manager.Fixtures() {
		if fixture.IsOnline() {
			onlineFixtures = append(onlineFixtures, fixture)
		}
	}
	return onlineFixtures
}

func (manager *StoredManager) Fixtures() []Fixture {
	manager.m.RLock()
	defer manager.m.RUnlock()
	fixtures := make([]Fixture, 0)
	for _, fixture := range manager.fixtures {
		fixtures = append(fixtures, fixture)
	}
	// Sort by last seen desc.
	sort.Slice(fixtures, func(i, j int) bool {
		return fixtures[i].LastSeen().After(fixtures[j].LastSeen())
	})
	return fixtures
}

func (manager *StoredManager) DeleteFixture(fixtureID messages.FixtureID) error {
	manager.m.Lock()
	defer manager.m.Unlock()
	err := manager.store.DeleteFixture(fixtureID)
	if err != nil {
		return errors.Wrap(err, "delete fixture from store")
	}
	delete(manager.fixtures, fixtureID)
	return nil
}

func (manager *StoredManager) GetFixtureStatesBroadcast() <-chan messages.MessageFixtureStates {
	return manager.fixtureStateBroadcaster.statesBroadcast
}

func (manager *StoredManager) FixtureStates() messages.MessageFixtureStates {
	fixtures := manager.Fixtures()
	messageFixtures := make([]messages.MessageFixtureStatesFixture, 0, len(fixtures))
	for _, fixture := range fixtures {
		messageFixtures = append(messageFixtures, messages.MessageFixtureStatesFixture{
			ID:          fixture.ID(),
			FixtureType: fixture.Type(),
			DeviceID:    fixture.DeviceID(),
			ProviderID:  fixture.ProviderID(),
			Name:        fixture.Name(),
			IsOnline:    fixture.IsOnline(),
			IsEnabled:   fixture.IsEnabled(),
			IsLocating:  fixture.IsLocating(),
			State:       fixture.State(),
		})
	}
	return messages.MessageFixtureStates{
		Fixtures: messageFixtures,
	}
}
