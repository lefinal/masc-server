package lighting

import (
	"context"
	"fmt"
	"github.com/LeFinal/masc-server/acting"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/messages"
	"github.com/LeFinal/masc-server/stores"
	"sync"
	"time"
)

type ManagerStore interface {
	// GetFixtures returns all known fixtures.
	GetFixtures() ([]stores.Fixture, error)
	// CreateFixture creates the given fixture and returns the one with the assigned
	// id.
	CreateFixture(fixture stores.Fixture) (stores.Fixture, error)
	// DeleteFixture deletes the fixture with the given id.
	DeleteFixture(fixtureID messages.FixtureID) error
	// SetFixtureName sets the name of the fixture with the given id.
	SetFixtureName(fixtureID messages.FixtureID, name string) error
	// SetFixtureOnline sets the last seen field for the fixture to the current time
	// if online and sets it to online. If it is offline, the last seen field is
	// updated again and online-state set to offline.
	SetFixtureOnline(fixtureID messages.FixtureID, isOnline bool) error
}

// Manager allows managing fixtures and their providers.
type Manager interface {
	// LoadKnownFixtures loads all known fixtures from the ManagerStore.
	LoadKnownFixtures() error
	// AcceptFixtureProvider accepts an acting.Actor with
	// acting.RoleTypeFixtureProvider and handles fixture discovery.
	AcceptFixtureProvider(ctx context.Context, actor acting.Actor) error
	// SayGoodbyeToFixtureProvider is used for when the actor should no longer be
	// available in fixtures. This is usually the case, when the actor quits.
	SayGoodbyeToFixtureProvider(actor acting.Actor) error
	// SetFixtureName sets the name of the given Fixture.
	SetFixtureName(fixture Fixture, name string) error
	// OnlineFixtures returns all fixtures that are currently online.
	OnlineFixtures() []Fixture
	// Fixtures returns all fixtures including offline ones.
	Fixtures() []Fixture
	// DeleteFixture deletes the given Fixture from the store and from the Manager.
	DeleteFixture(fixtureID messages.FixtureID) error
}

// manager is the implementation of Manager.
type manager struct {
	// store enables fixture persistence.
	store ManagerStore
	// fixtures holds all fixtures by their associated actor.
	fixtures map[messages.FixtureID]Fixture
	// m locks the manager.
	m sync.RWMutex
}

// NewManager creates a new Manager.
func NewManager(store ManagerStore) *manager {
	return &manager{
		store:    store,
		fixtures: make(map[messages.FixtureID]Fixture),
	}
}

func (manager *manager) LoadKnownFixtures() error {
	panic("implement me")
}

func (manager *manager) AcceptFixtureProvider(ctx context.Context, actor acting.Actor) error {
	// Request available fixtures.
	fixtureRes := acting.SubscribeMessageTypeFixtures(actor)
	defer acting.UnsubscribeOrLogError(fixtureRes.Newsletter)
	err := actor.Send(acting.ActorOutgoingMessage{MessageType: messages.MessageTypeGetFixtures})
	if err != nil {
		return errors.Wrap(err, "request fixtures from provider")
	}
	var messageFixtures messages.MessageFixtures
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
		err = manager.store.SetFixtureOnline(fixture.ID(), true)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("set fixture %v online", fixture.ID()))
		}
		fixture.setActor(actor)
		fixture.Reset()
		err = fixture.Apply()
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("apply initial state for fixture %v", fixture.ID()))
		}
	}
	return nil
}

// addFixturesFromProviderMessage adds the fixtures from the given
// messages.MessageFixtures to manager.fixtures. If a fixture is unknown, it
// will be created using the manager.store. Fixtures will NOT be set to online
// and will not call Fixture.Apply.
func (manager *manager) addFixturesFromProviderMessage(m messages.MessageFixtures) ([]Fixture, error) {
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
			IsOnline:   false,
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

func (manager *manager) SayGoodbyeToFixtureProvider(actor acting.Actor) error {
	manager.m.Lock()
	defer manager.m.Unlock()
	// Set all fixtures offline for the actor.
	for fixtureID, fixture := range manager.fixtures {
		fixture.setActor(nil)
		err := manager.store.SetFixtureOnline(fixtureID, false)
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("set fixture %v offline", fixtureID))
		}
	}
	return nil
}

func (manager *manager) SetFixtureName(fixture Fixture, name string) error {
	manager.m.Lock()
	defer manager.m.Unlock()
	err := manager.store.SetFixtureName(fixture.ID(), name)
	if err != nil {
		return errors.Wrap(err, "set fixture name in store")
	}
	fixture.setName(name)
	return nil
}

func (manager *manager) OnlineFixtures() []Fixture {
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

func (manager *manager) Fixtures() []Fixture {
	manager.m.RLock()
	defer manager.m.RUnlock()
	fixtures := make([]Fixture, 0)
	for _, fixture := range manager.fixtures {
		fixtures = append(fixtures, fixture)
	}
	return fixtures
}

func (manager *manager) DeleteFixture(fixtureID messages.FixtureID) error {
	manager.m.Lock()
	defer manager.m.Unlock()
	err := manager.store.DeleteFixture(fixtureID)
	if err != nil {
		return errors.Wrap(err, "delete fixture from store")
	}
	delete(manager.fixtures, fixtureID)
	return nil
}
