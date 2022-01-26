package lightswitch

import (
	"context"
	"fmt"
	"github.com/LeFinal/masc-server/acting"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/logging"
	"github.com/LeFinal/masc-server/messages"
	"github.com/LeFinal/masc-server/stores"
	"github.com/gobuffalo/nulls"
	"math/rand"
	"sync"
	"time"
)

type ManagerStore interface {
	// GetFixtures returns all known fixtures.
	GetFixtures() ([]stores.Fixture, error)
	// LightSwitches retrieves all available light switches.
	LightSwitches() ([]stores.LightSwitch, error)
	// LightSwitchByDeviceAndProviderID retrieves stores.LightSwitch by its device
	// and provider id.
	LightSwitchByDeviceAndProviderID(deviceID messages.DeviceID, providerID messages.ProviderID) (stores.LightSwitch, error)
	// UpdateLightSwitchByID updates the stores.LightSwitch with the given id.
	UpdateLightSwitchByID(id messages.LightSwitchID, payload stores.EditLightSwitch) error
	// DeleteLightSwitchByID deletes the stores.LightSwitch with the given id.
	//
	// Warning: This is only allowed when the light switch is currently NOT online!
	DeleteLightSwitchByID(id messages.LightSwitchID) error
	// CreateLightSwitch creates the given stores.LightSwitch.
	CreateLightSwitch(lightSwitch stores.LightSwitch) (stores.LightSwitch, error)
	// RefreshLastSeenForLightSwitchByID refreshes the last seen timestamp for the
	// stores.LightSwitch with the given id.
	RefreshLastSeenForLightSwitchByID(id messages.LightSwitchID) error
}

type ManagerAppliers interface {
	SwitchBaseApplier
}

type Manager interface {
	// GetFixtures returns all known fixtures.
	GetFixtures() ([]stores.Fixture, error)
	// AcceptLightSwitchProvider accepts an acting.Actor with
	// acting.RoleTypeLightSwitchProvider and handles light switch discovery.
	//
	// Warning: We expect the calling functions to hand over the acting.Actor so we
	// assume that waiting for the quit-channel is allowed. Therefore, no manual
	// unregistering needs to be done.
	AcceptLightSwitchProvider(ctx context.Context, actor acting.Actor) error
	// UpdateLightSwitchByID updates a light switch by its id.
	UpdateLightSwitchByID(id messages.LightSwitchID, payload stores.EditLightSwitch) error
	// OnlineLightSwitches returns all currently connected light switches.
	OnlineLightSwitches() []LightSwitch
	// LightSwitches returns all known light switches.
	LightSwitches() ([]messages.LightSwitch, error)
	// DeleteLightSwitchByID deletes a light switch identified by its id.
	DeleteLightSwitchByID(id messages.LightSwitchID) error
}

type manager struct {
	// store allows light switch persistence.
	store ManagerStore
	// appliers allows applying changes from light switches.
	appliers ManagerAppliers
	// activeLightSwitches holds all light switches by their id.
	activeLightSwitches map[messages.LightSwitchID]LightSwitch
	// activeLightSwitchesMutex locks activeLightSwitches.
	activeLightSwitchesMutex sync.RWMutex
}

// NewManager creates a new Manager with the given store and appliers.
func NewManager(store ManagerStore, appliers ManagerAppliers) Manager {
	return &manager{
		store:               store,
		appliers:            appliers,
		activeLightSwitches: make(map[messages.LightSwitchID]LightSwitch),
	}
}

func (m *manager) GetFixtures() ([]stores.Fixture, error) {
	return m.store.GetFixtures()
}

func (m *manager) AcceptLightSwitchProvider(ctx context.Context, actor acting.Actor) error {
	// Request available light switches.
	offersRes := acting.SubscribeMessageTypeLightSwitchOffers(actor)
	defer acting.UnsubscribeOrLogError(offersRes.Newsletter)
	err := actor.Send(acting.ActorOutgoingMessage{MessageType: messages.MessageTypeGetLightSwitchOffers})
	if err != nil {
		return errors.Wrap(err, "request light switches from provider", nil)
	}
	var messageOffers messages.MessageLightSwitchOffers
	select {
	case <-ctx.Done():
		return errors.NewContextAbortedError("request available light switches")
	case messageOffers = <-offersRes.Receive:
	}
	// Create them.
	lightSwitches, err := m.createLightSwitchesFromProviderMessage(messageOffers)
	if err != nil {
		return errors.Wrap(err, "add light switches from provider message", errors.Details{
			"message_offers": messageOffers,
		})
	}
	// Add and run them.
	m.activeLightSwitchesMutex.Lock()
	defer m.activeLightSwitchesMutex.Unlock()
	lightSwitchCtx, cancelLightSwitches := context.WithCancel(ctx)
	for _, lightSwitch := range lightSwitches {
		// Assure not already registered as overwriting would maybe lead to a memory
		// leak due to running stuff.
		if _, ok := m.activeLightSwitches[lightSwitch.ID()]; ok {
			cancelLightSwitches()
			return errors.NewInternalError("light switch already registered", errors.Details{
				"light_switch_id": lightSwitch.ID(),
			})
		}
		// Add it.
		m.activeLightSwitches[lightSwitch.ID()] = lightSwitch
		// Run it.
		go func(lightSwitch LightSwitch) {
			err := lightSwitch.run(lightSwitchCtx, actor)
			if err != nil {
				errors.Log(logging.LightSwitchLogger, errors.Wrap(err, "run light switch",
					errors.Details{
						"light_switch_id":   lightSwitch.ID(),
						"light_switch_type": lightSwitch.Type(),
					}))
			}
		}(lightSwitch)
	}
	// Okay, so every light switch should be running now. Finally, we add a listener
	// for when the actor quits so that we can cancel all light switches.
	go func() {
		select {
		case <-ctx.Done():
		case <-actor.Quit():
			cancelLightSwitches()
			m.activeLightSwitchesMutex.Lock()
			defer m.activeLightSwitchesMutex.Unlock()
			// Remove all light switches from active ones.
			for _, lightSwitch := range lightSwitches {
				delete(m.activeLightSwitches, lightSwitch.ID())
			}
		}
	}()
	return nil
}

// createLightSwitchesFromProviderMessage creates all offered and supported
// light switches from the given messages.MessageLightSwitchOffers. If they are
// unknown, new ones will be created in the ManagerStore.
func (m *manager) createLightSwitchesFromProviderMessage(message messages.MessageLightSwitchOffers) ([]LightSwitch, error) {
	m.activeLightSwitchesMutex.RLock()
	defer m.activeLightSwitchesMutex.RUnlock()
	lightSwitches := make([]LightSwitch, 0, len(message.LightSwitches))
	// Add each.
	for _, offeredLightSwitch := range message.LightSwitches {
		// Retrieve information from store regarding the light switch.
		stored, err := m.store.LightSwitchByDeviceAndProviderID(message.DeviceID, offeredLightSwitch.ProviderID)
		if err != nil {
			if e, ok := errors.Cast(err); !ok || e.Code != errors.ErrNotFound {
				return nil, errors.Wrap(err, "light switch by device and provider id", errors.Details{
					"device_id":   message.DeviceID,
					"provider_id": offeredLightSwitch.ProviderID,
				})
			}
			// Not found -> create.
			stored, err = m.store.CreateLightSwitch(stores.LightSwitch{
				Device:      message.DeviceID,
				ProviderID:  offeredLightSwitch.ProviderID,
				Name:        nulls.NewString(randomLightSwitchName()),
				Type:        offeredLightSwitch.Type,
				LastSeen:    time.Now(),
				Assignments: make([]messages.FixtureID, 0),
			})
			if err != nil {
				return nil, errors.Wrap(err, "create light switch", nil)
			}
		} else {
			// Assure type match.
			if stored.Type != offeredLightSwitch.Type {
				return nil, errors.NewInternalError("light switch type mismatch", errors.Details{
					"type_from_stored":  stored.Type,
					"type_from_offered": offeredLightSwitch.Type,
				})
			}
			// Refresh last seen.
			err = m.store.RefreshLastSeenForLightSwitchByID(stored.ID)
			if err != nil {
				return nil, errors.Wrap(err, "refresh last seen for light switch",
					errors.Details{"light_switch_id": stored.ID})
			}
		}
		// Create the light switch.
		lightSwitch, err := m.newLightSwitchByIDAndType(stored.ID, stored.Type)
		if err != nil {
			return nil, errors.Wrap(err, "new light switch by id and type", errors.Details{
				"light_switch_id":   stored.ID,
				"light_switch_type": stored.Type,
			})
		}
		lightSwitch.setAssignments(stored.Assignments)
		lightSwitches = append(lightSwitches, lightSwitch)
	}
	return lightSwitches, nil
}

// newLightSwitchByIDAndType instantiates the correct light switch with the
// given messages.LightSwitchID and messages.LightSwitchType. However, you need
// to manually call LightSwitch.run.
func (m *manager) newLightSwitchByIDAndType(id messages.LightSwitchID, switchType messages.LightSwitchType) (LightSwitch, error) {
	switch switchType {
	case messages.LightSwitchTypeHiLo:
		return newLightSwitchHiLo(id, m.appliers), nil
	default:
		return nil, errors.NewInternalError("unknown light switch type", errors.Details{
			"light_switch_type": switchType,
		})
	}
}

func (m *manager) UpdateLightSwitchByID(id messages.LightSwitchID, payload stores.EditLightSwitch) error {
	m.activeLightSwitchesMutex.RLock()
	defer m.activeLightSwitchesMutex.RUnlock()
	// Update in store.
	err := m.store.UpdateLightSwitchByID(id, payload)
	if err != nil {
		return errors.Wrap(err, "update light switch by id in store", errors.Details{
			"light_switch_id": id,
			"payload":         payload,
		})
	}
	// If online, we need to set the fields as well.
	lightSwitch, ok := m.activeLightSwitches[id]
	if !ok {
		return nil
	}
	lightSwitch.setName(payload.Name)
	lightSwitch.setAssignments(payload.Assignments)
	return nil
}

func (m *manager) OnlineLightSwitches() []LightSwitch {
	m.activeLightSwitchesMutex.RLock()
	defer m.activeLightSwitchesMutex.RUnlock()
	onlineLightSwitches := make([]LightSwitch, 0, len(m.activeLightSwitches))
	for _, lightSwitch := range m.activeLightSwitches {
		onlineLightSwitches = append(onlineLightSwitches, lightSwitch)
	}
	return onlineLightSwitches
}

func (m *manager) LightSwitches() ([]messages.LightSwitch, error) {
	m.activeLightSwitchesMutex.RLock()
	defer m.activeLightSwitchesMutex.RUnlock()
	// Retrieve from store.
	storeLightSwitches, err := m.store.LightSwitches()
	if err != nil {
		return nil, errors.Wrap(err, "light switches from store", nil)
	}
	// Map each to desired type.
	lightSwitches := make([]messages.LightSwitch, 0, len(storeLightSwitches))
	for _, storeLightSwitch := range storeLightSwitches {
		lightSwitch := messages.LightSwitch{
			ID:          storeLightSwitch.ID,
			DeviceID:    storeLightSwitch.Device,
			DeviceName:  storeLightSwitch.DeviceName,
			ProviderID:  storeLightSwitch.ProviderID,
			Type:        storeLightSwitch.Type,
			Name:        storeLightSwitch.Name,
			LastSeen:    storeLightSwitch.LastSeen,
			Assignments: storeLightSwitch.Assignments,
		}
		// Check for each one if online.
		if activeLightSwitch, ok := m.activeLightSwitches[storeLightSwitch.ID]; ok {
			lightSwitch.IsOnline = true
			lightSwitch.Features = activeLightSwitch.Features()
		}
		lightSwitches = append(lightSwitches, lightSwitch)
	}
	return lightSwitches, nil
}

func (m *manager) DeleteLightSwitchByID(id messages.LightSwitchID) error {
	m.activeLightSwitchesMutex.RLock()
	defer m.activeLightSwitchesMutex.RUnlock()
	// Assure not online.
	if _, ok := m.activeLightSwitches[id]; ok {
		return errors.NewBadRequestErr("light switch still online", nil)
	}
	// Delete from store.
	err := m.store.DeleteLightSwitchByID(id)
	if err != nil {
		return errors.Wrap(err, "delete light switch by id", errors.Details{
			"light_switch_id": id,
		})
	}
	return nil
}

func randomLightSwitchName() string {
	return fmt.Sprintf("unknown-%d", rand.Intn(9999))
}
