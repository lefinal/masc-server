package acting

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/gatekeeping"
	"github.com/LeFinal/masc-server/logging"
	"github.com/LeFinal/masc-server/messages"
	"github.com/LeFinal/masc-server/util"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"sync"
)

// RoleType is a set of abilities that a device can provide and so provide
// a certain functionality.
type RoleType string

const (
	// RoleTypeDeviceManager manages devices. This also includes setting up new devices.
	//
	// Warning: This is the only RoleType that is managed within an Agency, because
	// it needs to be able to accept new devices.
	RoleTypeDeviceManager RoleType = "device-manager"
	// RoleTypeFixtureManager sets up and manages fixtures.
	RoleTypeFixtureManager RoleType = "fixture-manager"
	// RoleTypeFixtureOperator controls fixtures.
	RoleTypeFixtureOperator RoleType = "fixture-operator"
	// RoleTypeFixtureProvider is used for provided fixtures.
	RoleTypeFixtureProvider RoleType = "fixture-provider"
	// RoleTypeGameMaster sets up and controls matches.
	RoleTypeGameMaster RoleType = "game-master"
	// RoleTypeGlobalMonitor receives global status information. Allows no
	// interaction.
	RoleTypeGlobalMonitor RoleType = "global-spectator"
	// RoleTypeMatchMonitor received status information for a specific match. Allows
	// no interaction.
	RoleTypeMatchMonitor RoleType = "match-monitor"
	// RoleTypeLightSwitchManager is used for managing and assigning light switches.
	RoleTypeLightSwitchManager RoleType = "light-switch-manager"
	// RoleTypeLightSwitchProvider provides light switches.
	RoleTypeLightSwitchProvider RoleType = "light-switch-provider"
	// RoleTypeTeamBase allows managing a team. Mostly used for devices that are
	// located in team bases. Also used in-game.
	RoleTypeTeamBase RoleType = "team-base"
	// RoleTypeTeamBaseMonitor receives status information for team bases. Allows no
	// interaction.
	RoleTypeTeamBaseMonitor RoleType = "team-base-spectator"
)

// getRole returns the RoleType matching the given one. If it is unknown, false
// will be returned.
func getRole(role messages.Role) (RoleType, bool) {
	r := RoleType(role)
	switch r {
	case RoleTypeDeviceManager,
		RoleTypeFixtureManager,
		RoleTypeFixtureOperator,
		RoleTypeFixtureProvider,
		RoleTypeGameMaster,
		RoleTypeGlobalMonitor,
		RoleTypeTeamBase,
		RoleTypeTeamBaseMonitor,
		RoleTypeLightSwitchManager,
		RoleTypeLightSwitchProvider:
		return r, true
	}
	return "", false
}

// ActorNewsletterRecipient is used for handling a new Actor for an Agency.
type ActorNewsletterRecipient interface {
	// HandleNewActor is called in a new goroutine when a new Actor is welcomed to
	// the Agency.
	HandleNewActor(actor Actor, role RoleType)
}

// Agency manages actors.
type Agency interface {
	// ActorByID retrieves an Actor by its Actor.ID. Returns false when the Actor is
	// not found.
	ActorByID(id messages.ActorID) (Actor, bool)
	// AvailableActors available actors for the given role that are currently not
	// hired.
	AvailableActors(role RoleType) []Actor
	// Open opens the Agency.
	Open() error
	// Close closes the agency.
	Close() error
	// SubscribeNewActors adds the given ActorNewsletterRecipient to the list of
	// subscribers for when a new Actor is welcomed. Don't forget to call
	// UnsubscribeNewActors!
	SubscribeNewActors(recipient ActorNewsletterRecipient)
	// UnsubscribeNewActors unsubscribes a subscription that was originally set up
	// using SubscribeNewActors.
	UnsubscribeNewActors(recipient ActorNewsletterRecipient)
}

// ActorOutgoingMessage is a message that is sent from an Actor.
type ActorOutgoingMessage struct {
	// MessageType is the type of the message which is added to the final message
	// later.
	MessageType messages.MessageType
	// Content is the message content that is being encoded as JSON.
	Content interface{}
}

// ActorIncomingMessage is a message that is received by an Actor.
type ActorIncomingMessage Message

// Actor performs a certain RoleType after being hired and therefore allows sending
// and receiving messages.
type Actor interface {
	// ID return the ID of the actor. This is only used for verbose information.
	ID() messages.ActorID
	// Hire hires the actor for the given RoleType. You must call Fire when he is no
	// longer needed!
	Hire(displayedName string) error
	// Name is a human-readable name. It is related to the role and provides a
	// human-readable description of what the Actor is currently doing.
	Name() string
	// IsHired describes whether the actor is currently hired.
	IsHired() bool
	// Fire fires the actor. Lol.
	Fire() error
	// Send sends the given message to the actor.
	Send(message ActorOutgoingMessage) error
	// SubscribeMessageType subscribes to all messages with the given
	// messages.MessageType by returning Newsletter. Remember to unsubscribe via
	// Unsubscribe using the created Newsletter. However, you should be aware, that
	// the channel is NOT self-closing. If you want that, use SubscribeMessageType.
	SubscribeMessageType(messageType messages.MessageType) GeneralNewsletter
	// Unsubscribe make the actor remove an existing subscription for the passed
	// subscription.
	Unsubscribe(subscription *subscription) error
	// Receive returns the channel for receiving actor messages. However, you should
	// be careful to use it in production. In most cases, a SubscriptionManager
	// handles receiving messages.
	Receive() <-chan ActorIncomingMessage
	// Quit returns the channel for when the actor decides to quit. The channel
	// provides a context.CancelFunc that needs to be called when we are sure that
	// no more writes are being performed and all pumps can be shut down.
	Quit() <-chan struct{}
}

// netActor is the net version of Actor.
type netActor struct {
	// id allows identifying the netActor.
	id messages.ActorID
	// name is a human-readable description of what the actor is currently
	// doing.
	name string
	// send is the channel for outgoing messages. These will be handled by
	// netActorDevice.
	send chan netActorDeviceOutgoingMessage
	// receiveC is the channel for incoming messages. These will already be routed
	// by netActorDevice and really belong to this actor.
	receiveC chan ActorIncomingMessage
	// isHired describes whether the actor is currently hired.
	isHired bool
	// role holds the RoleType the actor is playing.
	role RoleType
	// quit is the channel that is used for when the actor quits.
	quit chan struct{}
	// hireMutex is a mutex for allowing concurrent access to isHired, name
	// and role.
	hireMutex sync.RWMutex
	// subscriptionManager allows easy managing of calls to
	// Actor.SubscribeMessageType and Actor.Unsubscribe. Manager will be created
	// when Hire is called.
	subscriptionManager *SubscriptionManager
}

func (a *netActor) ID() messages.ActorID {
	return a.id
}

func (a *netActor) Hire(displayedName string) error {
	a.hireMutex.Lock()
	defer a.hireMutex.Unlock()
	// Check if already hired.
	if a.isHired {
		return errors.Error{
			Code:    errors.ErrInternal,
			Message: fmt.Sprintf("actor %s is already hired", a.id),
			Details: errors.Details{"actorID": a.id},
		}
	}
	// Create subscription manager.
	a.subscriptionManager = NewSubscriptionManager()
	// Notify actor that he was lucky.
	a.forceSend(ActorOutgoingMessage{
		MessageType: messages.MessageTypeYouAreIn,
		Content: messages.MessageYouAreIn{
			ActorID: a.id,
			Role:    messages.Role(a.role),
		},
	})
	a.isHired = true
	a.name = displayedName
	return nil
}

func (a *netActor) Name() string {
	a.hireMutex.RLock()
	defer a.hireMutex.RUnlock()
	return a.name
}

func (a *netActor) IsHired() bool {
	defer a.hireMutex.RUnlock()
	a.hireMutex.RLock()
	return a.isHired
}

func (a *netActor) Fire() error {
	a.hireMutex.Lock()
	defer a.hireMutex.Unlock()
	// Check if already fired or not even hired.
	if !a.isHired {
		return errors.Error{
			Code:    errors.ErrInternal,
			Message: fmt.Sprintf("actor %s is not even hired", a.id),
			Details: errors.Details{"actorID": a.id},
		}
	}
	// Notify actor.
	a.forceSend(ActorOutgoingMessage{
		MessageType: messages.MessageTypeFired,
	})
	a.subscriptionManager.CancelAllSubscriptions()
	a.isHired = false
	return nil
}

// forceSend sends the given message without checking if the actor is hired.
// This used for hiring and firing the actor.
func (a *netActor) forceSend(message ActorOutgoingMessage) {
	// Pass to actor device.
	a.send <- netActorDeviceOutgoingMessage{
		actorID: a.id,
		message: message,
	}
}

func (a *netActor) Send(message ActorOutgoingMessage) error {
	a.hireMutex.RLock()
	defer a.hireMutex.RUnlock()
	// Check if actor is allowed to send this message.
	if !a.isHired {
		return errors.Error{
			Code:    errors.ErrInternal,
			Message: "actor must not send when not hired or hiring",
			Details: errors.Details{"id": a.id, "message": message},
		}
	}
	a.forceSend(message)
	return nil
}

func (a *netActor) SubscribeMessageType(messageType messages.MessageType) GeneralNewsletter {
	subscription := a.subscriptionManager.SubscribeMessageType(messageType)
	return GeneralNewsletter{
		Newsletter: Newsletter{
			Actor:        a,
			Subscription: subscription,
		},
		Receive: subscription.out,
	}
}

func (a *netActor) Unsubscribe(subscription *subscription) error {
	return a.subscriptionManager.Unsubscribe(subscription)
}

func (a *netActor) Receive() <-chan ActorIncomingMessage {
	return a.receiveC
}

func (a *netActor) Quit() <-chan struct{} {
	return a.quit
}

// handleIncomingMessage handles an incoming message, lol.
func (a *netActor) handleIncomingMessage(message ActorIncomingMessage) {
	if a.subscriptionManager.HandleMessage(Message(message)) == 0 {
		// No subscribers -> we consider this a forbidden message, because no one wants
		// to hear it.
		SendForbiddenMessageTypeErrToActorOrLogError(logging.ActingLogger, a, message)
	}
}

// netActorDeviceOutgoingMessage is a container for an ActorIncomingMessage and
// the id of the Actor that wants to send the message. This allows using a
// single channel for sending messages to a device.
type netActorDeviceOutgoingMessage struct {
	// actorID is the id of the netActor which will be added as field to the final
	// message container that is passed to the device.
	actorID messages.ActorID
	// message is the actual message.
	message ActorOutgoingMessage
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
	send chan netActorDeviceOutgoingMessage
	// shutdown is called when the routers should shut down.
	shutdownRouter context.CancelFunc
	// routers holds all running routers.
	routers sync.WaitGroup
}

type actorWithRole struct {
	actor Actor
	role  RoleType
}

// boot sets up all fields and fires up the routers. Make sure that the device
// is set. Returns a list of added actors.
func (ad *netActorDevice) boot() ([]actorWithRole, error) {
	// Check each role of the device for being known so that if one is unknown, we
	// do not need to rollback.
	roles := make([]RoleType, len(ad.device.Roles))
	for i, role := range ad.device.Roles {
		r, knownRole := getRole(role)
		if !knownRole {
			return nil, errors.Error{
				Code:    errors.ErrBadRequest,
				Message: fmt.Sprintf("unknown role: %s", role),
				Details: errors.Details{"role": role},
			}
		}
		roles[i] = r
	}
	ad.actors = make(map[messages.ActorID]*netActor)
	ad.send = make(chan netActorDeviceOutgoingMessage)
	createdActors := make([]actorWithRole, 0, len(roles))
	// Add actors for each role the device offers.
	for _, role := range roles {
		// Create new actor.
		created := &netActor{
			id:       messages.ActorID(uuid.New().String()),
			role:     role,
			send:     ad.send,
			receiveC: make(chan ActorIncomingMessage),
			isHired:  false,
			quit:     make(chan struct{}),
		}
		// Append to own actor list.
		ad.actors[created.id] = created
		createdActors = append(createdActors, actorWithRole{
			actor: created,
			role:  role,
		})
	}
	// Start router.
	routerCtx, shutdownRouter := context.WithCancel(context.Background())
	ad.shutdownRouter = shutdownRouter
	ad.routers.Add(2)
	go ad.routeIncoming(routerCtx)
	go ad.routeOutgoing(routerCtx)
	logging.ActingLogger.Infof("net actor device %s with roles %s ready.", ad.device.ID, ad.device.Roles)
	return createdActors, nil
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
				close(actor.receiveC)
			}
			ad.actorsMutex.Unlock()
			return
		case message := <-ad.device.Receive:
			// Route to target actor.
			ad.actorsMutex.RLock()
			actor, ok := ad.actors[message.ActorID]
			if !ok {
				// Actor not found. This is a bad request!
				ad.send <- netActorDeviceOutgoingMessage{
					message: ActorErrorMessageFromError(errors.Error{
						Code:    errors.ErrBadRequest,
						Message: fmt.Sprintf("unknown actor: %s", message.ActorID),
						Details: errors.Details{"id": message.ActorID},
					}),
				}
			} else {
				// Actor found -> pass message.
				actor.handleIncomingMessage(ActorIncomingMessage{
					MessageType: message.MessageType,
					Content:     message.Content,
				})
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
			// Encode message content as JSON.
			encodedContent, err := util.EncodeAsJSON(message.message.Content)
			if err != nil {
				// Meh. We can simply log this error.
				errors.Log(logging.ActingLogger, errors.Wrap(errors.Wrap(err, "encode message content as json", nil), "route outgoing actor message", nil))
				return
			}
			// Route to device.
			ad.device.Send <- messages.MessageContainer{
				MessageType: message.message.MessageType,
				DeviceID:    ad.device.ID,
				ActorID:     message.actorID,
				Content:     encodedContent,
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
			actor.hireMutex.Lock()
			actor.isHired = false
			actor.hireMutex.Unlock()
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
	// newActorSubscribers is the collection of subscribers for when a new Actor is
	// welcomed. Subscriptions are added via SubscribeNewActors.
	newActorSubscribers map[ActorNewsletterRecipient]struct{}
}

func NewProtectedAgency(gatekeeper gatekeeping.Gatekeeper) *ProtectedAgency {
	return &ProtectedAgency{
		actorDevices:        make(map[messages.DeviceID]*netActorDevice),
		gatekeeper:          gatekeeper,
		newActorSubscribers: make(map[ActorNewsletterRecipient]struct{}),
	}
}

func (a *ProtectedAgency) WelcomeDevice(device *gatekeeping.Device) error {
	defer a.m.Unlock()
	a.m.Lock()
	// Create new actor device.
	ad := &netActorDevice{
		device: device,
	}
	// Boot.
	newActors, err := ad.boot()
	if err != nil {
		return errors.Wrap(err, "boot actor device", nil)
	}
	// Only add if successfully booted.
	a.actorDevices[device.ID] = ad
	// Satisfy actor subscriptions.
	for _, actor := range newActors {
		for recipient := range a.newActorSubscribers {
			go recipient.HandleNewActor(actor.actor, actor.role)
		}
	}
	return nil
}

func (a *ProtectedAgency) SayGoodbyeToDevice(deviceID messages.DeviceID) error {
	defer a.m.Unlock()
	a.m.Lock()
	device, ok := a.actorDevices[deviceID]
	if !ok {
		// Device may not have registered successfully due to bad request, etc.
		return nil
	}
	device.shutdown()
	// Remove from known actor devices.
	delete(a.actorDevices, deviceID)
	return nil
}

func (a *ProtectedAgency) ActorByID(id messages.ActorID) (Actor, bool) {
	for _, device := range a.actorDevices {
		for actorID, actor := range device.actors {
			if actorID == id {
				return actor, true
			}
		}
	}
	// Not found.
	return nil, false
}

func (a *ProtectedAgency) AvailableActors(role RoleType) []Actor {
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
	logging.ActingLogger.Info("agency open for actors")
	return nil
}

func (a *ProtectedAgency) Close() error {
	// Shutdown all devices.
	for _, device := range a.actorDevices {
		device.shutdown()
	}
	logging.ActingLogger.Info("agency closed")
	return nil
}

func (a *ProtectedAgency) SubscribeNewActors(recipient ActorNewsletterRecipient) {
	a.m.Lock()
	defer a.m.Unlock()
	if _, ok := a.newActorSubscribers[recipient]; ok {
		logging.ActingLogger.Warn("duplicate new actor subscribe")
		return
	}
	a.newActorSubscribers[recipient] = struct{}{}
}

func (a *ProtectedAgency) UnsubscribeNewActors(recipient ActorNewsletterRecipient) {
	a.m.Lock()
	defer a.m.Unlock()
	if _, ok := a.newActorSubscribers[recipient]; !ok {
		logging.ActingLogger.Error("unsubscribe for new actors although not subscribed")
		return
	}
	delete(a.newActorSubscribers, recipient)
}

// ActorErrorMessageFromError creates a new ActorOutgoingMessage for the given
// actor and with passed error.
func ActorErrorMessageFromError(err error) ActorOutgoingMessage {
	return ActorOutgoingMessage{
		MessageType: messages.MessageTypeError,
		Content:     messages.MessageErrorFromError(err),
	}
}

// SendForbiddenMessageTypeErrToActorOrLogError does everything the function
// name already includes lol.
func SendForbiddenMessageTypeErrToActorOrLogError(logger *logrus.Entry, a Actor, message ActorIncomingMessage) {
	LogErrorAndSendOrLog(logger, a, NewForbiddenMessageError(message.MessageType, message.Content))
}

// LogErrorAndSendOrLog logs the given error and then sends it to the given
// Actor. If that fails, the send error is logged, too.
func LogErrorAndSendOrLog(logger *logrus.Entry, a Actor, e error) {
	errors.Log(logger, e)
	SendOrLogError(a, ActorErrorMessageFromError(e))
}

// SendOrLogError sends the message to the given Actor and logs the error if
// delivery failed.
func SendOrLogError(a Actor, message ActorOutgoingMessage) {
	err := a.Send(message)
	if err != nil {
		errors.Log(logging.CommunicationFailLogger, errors.Wrap(err, "send message", nil))
	}
}

// SendOKOrLogError sends a message with messages.MessageTypeOK to the given
// Actor and logs the error if delivery failed.
func SendOKOrLogError(a Actor) {
	err := a.Send(ActorOutgoingMessage{MessageType: messages.MessageTypeOK})
	if err != nil {
		errors.Log(logging.CommunicationFailLogger, errors.Wrap(err, "send ok message", nil))
	}
}

// NewForbiddenMessageError creates a new ErrProtocolViolation error with kind
// KindForbiddenMessage.
func NewForbiddenMessageError(messageType messages.MessageType, content json.RawMessage) error {
	return errors.Error{
		Code:    errors.ErrProtocolViolation,
		Message: fmt.Sprintf("forbidden message type: %s", messageType),
		Details: errors.Details{
			"message_type":    messageType,
			"message_content": string(content),
		},
	}
}

// FireAllActors fires all passed actors.
func FireAllActors(actors []Actor) error {
	for i, actor := range actors {
		err := actor.Fire()
		if err != nil {
			return errors.Wrap(err, fmt.Sprintf("fire actor %d of %d", i, len(actors)), nil)
		}
	}
	return nil
}

// ActorRepresentation uses the Actor id and name for creating a representation
// that can be used by other actors as well.
type ActorRepresentation struct {
	// ID is the Actor.ID.
	ID messages.ActorID
	// Name is the Actor.Name.
	Name string
}

func (r *ActorRepresentation) Message() messages.ActorRepresentation {
	return messages.ActorRepresentation{
		ID:   r.ID,
		Name: r.Name,
	}
}

// ActorRepresentationFromActor creates an ActorRepresentation from the given
// Actor.
func ActorRepresentationFromActor(actor Actor) ActorRepresentation {
	return ActorRepresentation{
		ID:   actor.ID(),
		Name: actor.Name(),
	}
}
