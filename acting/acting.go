package acting

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/gatekeeping"
	"github.com/LeFinal/masc-server/logging"
	"github.com/LeFinal/masc-server/messages"
	"github.com/google/uuid"
	"sync"
)

// Role is a set of abilities that a device can provide and so provide
// a certain functionality.
type Role string

const (
	// RoleDeviceManager manages devices. This also includes setting up new devices.
	RoleDeviceManager Role = "device-manager"
	// RoleGameMaster sets up and controls matches.
	RoleGameMaster Role = "game-master"
	// RoleTeamBase allows managing a team. Mostly used for devices that are located
	// in team bases. Also used in-game.
	RoleTeamBase Role = "team-base"
	// RoleTeamBaseMonitor receives status information for team bases. Allows no
	// interaction.
	RoleTeamBaseMonitor Role = "team-base-spectator"
	// RoleGlobalMonitor receives global status information. Allows no interaction.
	RoleGlobalMonitor Role = "global-spectator"
)

// getRole returns the Role matching the given one. If it is unknown, false will be returned.
func getRole(role string) (Role, bool) {
	r := Role(role)
	switch r {
	case RoleDeviceManager,
		RoleGameMaster,
		RoleTeamBase,
		RoleTeamBaseMonitor,
		RoleGlobalMonitor:
		return r, true
	}
	return "", false
}

// Agency manages actors.
type Agency interface {
	// AvailableActors available actors for the given role that are currently not hired.
	AvailableActors(role Role) []Actor
	// Open opens the Agency.
	Open() error
	// Close closes the agency.
	Close() error
}

// ActorMessage is any messages that is sent or received from an Actor.
type ActorMessage struct {
	// MessageType is the type of the message. This determines how to parse the
	// Content.
	MessageType messages.MessageType
	// Content is the raw message content which will be parsed based on the
	// MessageType.
	Content json.RawMessage
}

// Actor performs a certain Role after being hired and therefore allows sending
// and receiving messages.
type Actor interface {
	// Hire hires the actor for the given Role.
	Hire() error
	// IsHired describes whether the actor is currently hired.
	IsHired() bool
	// Fire fires the actor. Lol.
	Fire() error
	// Send sends the given message to the actor.
	Send(message ActorMessage) error
	// Receive returns the channel for receiving actor messages.
	Receive() <-chan ActorMessage
	// Quit returns the channel for when the actor decides to quit.
	Quit() <-chan struct{}
}

// netActor is the net version of Actor.
type netActor struct {
	// id allows identifying the netActor.
	id messages.ActorID
	// send is the channel for outgoing messages. These will be handled by netActorDevice.
	send chan netActorDeviceMessage
	// receive is the channel for incoming messages. These will already be routed by
	// netActorDevice and really belong to this actor.
	receive chan ActorMessage
	// isHired describes whether the actor is currently hired.
	isHired bool
	// role holds the Role the actor is playing.
	role Role
	// quit is the channel that is used for when the actor quits.
	quit chan struct{}
	// hireMutex is a mutex for allowing concurrent access to isHired and role.
	hireMutex sync.RWMutex
}

func (a *netActor) Hire() error {
	defer a.hireMutex.Unlock()
	a.hireMutex.Lock()
	// Check if already hired.
	if a.isHired {
		return errors.Error{
			Code:    errors.ErrInternal,
			Kind:    errors.KindActorAlreadyHired,
			Message: fmt.Sprintf("actor %s is already hired", a.id),
			Details: errors.Details{"actorID": a.id},
		}
	}
	a.isHired = true
	return nil
}

func (a *netActor) IsHired() bool {
	defer a.hireMutex.RUnlock()
	a.hireMutex.RLock()
	return a.isHired
}

func (a *netActor) Fire() error {
	defer a.hireMutex.Unlock()
	a.hireMutex.Lock()
	// Check if already fired or not even hired.
	if !a.isHired {
		return errors.Error{
			Code:    errors.ErrInternal,
			Kind:    errors.KindActorNotHired,
			Message: fmt.Sprintf("actor %s is not even hired", a.id),
			Details: errors.Details{"actorID": a.id},
		}
	}
	a.isHired = false
	a.hireMutex.Unlock()
	return nil
}

func (a *netActor) Send(message ActorMessage) error {
	// Check if hired.
	if !a.isHired {
		return errors.Error{
			Code:    errors.ErrInternal,
			Kind:    errors.KindActorNotHired,
			Message: "actor must not send when not hired",
			Details: errors.Details{"id": a.id, "message": message},
		}
	}
	// Pass to actor device.
	a.send <- netActorDeviceMessage{
		actorID: a.id,
		message: message,
	}
	return nil
}

func (a *netActor) Receive() <-chan ActorMessage {
	return a.receive
}

func (a *netActor) Quit() <-chan struct{} {
	return a.quit
}

// netActorDeviceMessage is a container for an ActorMessage and the id of the
// Actor that wants to send the message. This allows using a single channel for
// sending messages to a device.
type netActorDeviceMessage struct {
	// actorID is the id of the netActor which will be added as field to the final
	// message container that is passed to the device.
	actorID messages.ActorID
	// message is the actual message.
	message ActorMessage
}

// netActorDevice allows a device to have multiple actors. This handles incoming
// messages and routes them to the correct actor.
type netActorDevice struct {
	// device is the device that receives and sends messages.
	device *gatekeeping.Device
	// actors are the actors that are performing with the device. Messages will be
	// routed to and from them.
	actors map[messages.ActorID]*netActor
	// actorsMutex is the rw mutex for actors.
	actorsMutex sync.RWMutex
	// send is the channel for outgoing messages that will be passed to the device
	// by the netActorDevice. This means that we do not care about them.
	send chan netActorDeviceMessage
	// shutdown is called when the routers should shut down.
	shutdownRouter context.CancelFunc
	// routers holds all running routers.
	routers sync.WaitGroup
}

// boot sets up all fields and fires up the routers. Make sure that the device is set.
func (ad *netActorDevice) boot() error {
	// Check each role of the device for being known so that if one is unknown, we
	// do not need to rollback.
	roles := make([]Role, len(ad.device.Roles))
	for i, role := range ad.device.Roles {
		r, knownRole := getRole(role)
		if !knownRole {
			return errors.Error{
				Code:    errors.ErrBadRequest,
				Kind:    errors.KindUnknownRole,
				Message: fmt.Sprintf("unknown role: %s", role),
				Details: errors.Details{"role": role},
			}
		}
		roles[i] = r
	}
	ad.actors = make(map[messages.ActorID]*netActor)
	ad.send = make(chan netActorDeviceMessage)
	// Add actors for each role the device offers.
	for _, role := range roles {
		// Create new actor.
		created := &netActor{
			id:      messages.ActorID(uuid.New()),
			role:    role,
			send:    ad.send,
			receive: make(chan ActorMessage),
			isHired: false,
			quit:    make(chan struct{}),
		}
		// Append to own actor list.
		ad.actors[created.id] = created
	}
	// Start router.
	ctx, cancel := context.WithCancel(context.Background())
	ad.shutdownRouter = cancel
	ad.routers.Add(2)
	go ad.routeIncoming(ctx)
	go ad.routeOutgoing(ctx)
	logging.ActingLogger.Infof("net actor device %s with roles %s ready.", ad.device.ID, ad.device.Roles)
	return nil
}

// routeIncoming performs routing for incoming messages to the correct actor. When
// stopped, it also closes the receive-channels for the actors.
func (ad *netActorDevice) routeIncoming(ctx context.Context) {
	defer ad.routers.Done()
	for {
		select {
		case <-ctx.Done():
			// Close all receive channels for actors.
			ad.actorsMutex.Lock()
			for _, actor := range ad.actors {
				close(actor.receive)
			}
			ad.actorsMutex.Unlock()
			return
		case message := <-ad.device.Receive:
			// Route to target actor.
			ad.actorsMutex.RLock()
			actor, ok := ad.actors[message.ActorID]
			if !ok {
				// Actor not found. This is a bad request!
				errMessage, err := encodeAsJSON(messages.MessageErrorFromError(errors.Error{
					Code:    errors.ErrBadRequest,
					Kind:    errors.KindUnknownActor,
					Message: fmt.Sprintf("unknown actor: %s", message.ActorID),
					Details: errors.Details{"id": message.ActorID},
				}))
				if err != nil {
					ad.actorsMutex.RUnlock()
					errors.Log(logging.ActingLogger, errors.Wrap(err, "encode unknown actor error message as json"))
					continue
				}
				ad.send <- netActorDeviceMessage{
					message: ActorMessage{
						MessageType: messages.MessageTypeError,
						Content:     errMessage,
					},
				}
			} else {
				// Actor found -> pass message.
				actor.receive <- ActorMessage{
					MessageType: message.MessageType,
					Content:     message.Content,
				}
			}
			ad.actorsMutex.RUnlock()
		}
	}
}

// routeOutgoing performs routing for outgoing messages to the device. It serves
// as a simple write pump.
func (ad *netActorDevice) routeOutgoing(ctx context.Context) {
	defer ad.routers.Done()
	for {
		select {
		case <-ctx.Done():
			return
		case message := <-ad.send:
			// Route to device.
			ad.device.Send <- messages.MessageContainer{
				MessageType: message.message.MessageType,
				DeviceID:    ad.device.ID,
				ActorID:     message.actorID,
				Content:     message.message.Content,
			}
		}
	}
}

// shutdown closes the receive-channels for the actors and stops the router.
func (ad *netActorDevice) shutdown() {
	// Tell each hired actor to quit.
	ad.actorsMutex.Lock()
	for _, actor := range ad.actors {
		if actor.isHired {
			actor.quit <- struct{}{}
		}
	}
	ad.actorsMutex.Unlock()
	ad.shutdownRouter()
	ad.routers.Wait()
	logging.ActingLogger.Infof("net actor device %v shut down.", ad.device.ID)
}

// ProtectedAgency is an Agency that uses the gatekeeping.Gatekeeper for
// communication to the outer world.
type ProtectedAgency struct {
	// actorDevices holds all actor devices that manage message flow.
	actorDevices map[messages.DeviceID]*netActorDevice
	// m is a lock for actors and actorDevices.
	m sync.RWMutex
	// gatekeeper protects the agency.
	gatekeeper gatekeeping.Gatekeeper
}

func (a *ProtectedAgency) WelcomeDevice(device *gatekeeping.Device) error {
	defer a.m.Unlock()
	a.m.Lock()
	// Create new actor device.
	ad := &netActorDevice{
		device: device,
	}
	// Boot.
	err := ad.boot()
	if err != nil {
		return errors.Wrap(err, "boot actor device")
	}
	// Only add if successfully booted.
	a.actorDevices[device.ID] = ad
	return nil
}

func (a *ProtectedAgency) SayGoodbyeToDevice(deviceID messages.DeviceID) error {
	defer a.m.Unlock()
	a.m.Lock()
	device, ok := a.actorDevices[deviceID]
	if !ok {
		return errors.Error{
			Code:    errors.ErrInternal,
			Kind:    errors.KindUnknownDevice,
			Message: fmt.Sprintf("unknown device: %v", deviceID),
			Details: errors.Details{"deviceID": deviceID},
		}
	}
	device.shutdown()
	// Remove from known actor devices.
	delete(a.actorDevices, deviceID)
	return nil
}

func (a *ProtectedAgency) AvailableActors(role Role) []Actor {
	defer a.m.RUnlock()
	a.m.RLock()
	availableActors := make([]Actor, 0)
	// Iterate over all devices, then their actors.
	for _, device := range a.actorDevices {
		for _, actor := range device.actors {
			if actor.isHired || actor.role != role {
				continue
			}
			availableActors = append(availableActors, actor)
		}
	}
	return availableActors
}

func (a *ProtectedAgency) Open() error {
	a.actorDevices = make(map[messages.DeviceID]*netActorDevice)
	// Protect self by gatekeeper.
	err := a.gatekeeper.WakeUpAndProtect(a)
	if err != nil {
		return errors.Wrap(err, "wake up gatekeeper")
	}
	return nil
}

func (a *ProtectedAgency) Close() error {
	// Shutdown all devices.
	for _, device := range a.actorDevices {
		device.shutdown()
	}
	// Retire gatekeeper.
	err := a.gatekeeper.Retire()
	if err != nil {
		return errors.Wrap(err, "retire gatekeeper")
	}
	return nil
}

// encodeAsJSON serves as a wrapper for the standard encoding and error stuff.
func encodeAsJSON(content interface{}) (json.RawMessage, error) {
	j, err := json.Marshal(content)
	if err != nil {
		return json.RawMessage{}, errors.Error{
			Code:    errors.ErrInternal,
			Kind:    errors.KindEncodeJSON,
			Err:     err,
			Details: errors.Details{"content": content},
		}
	}
	return j, nil
}
