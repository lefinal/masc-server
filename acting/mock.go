package acting

import (
	"encoding/json"
	"fmt"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/messages"
	"sync"
)

// MockAgency allows simple adding and removing of actors.
type MockAgency struct {
	// Actors holds the managed Actors.
	Actors map[RoleType][]Actor
	// actorsMutex is a lock for Actors.
	actorsMutex sync.RWMutex
}

func NewMockAgency() *MockAgency {
	return &MockAgency{
		Actors: make(map[RoleType][]Actor),
	}
}

func (a *MockAgency) AddActor(actor Actor, role RoleType) {
	a.actorsMutex.Lock()
	a.Actors[role] = append(a.Actors[role], actor)
	a.actorsMutex.Unlock()
}

func (a *MockAgency) RemoveActor(actorToDelete Actor) {
	defer a.actorsMutex.Unlock()
	a.actorsMutex.Lock()
	// Find actor and delete.
	for role, actors := range a.Actors {
		for i, actor := range actors {
			if actor == actorToDelete {
				actors[i] = actors[len(actors)-1]
				a.Actors[role] = actors[:len(actors)-1]
				return
			}
		}
	}
	// Not found.
	panic("actor not found")
}

func (a *MockAgency) ActorByID(id messages.ActorID) (Actor, bool) {
	defer a.actorsMutex.RUnlock()
	a.actorsMutex.RLock()
	for _, actors := range a.Actors {
		for _, actor := range actors {
			if actor.ID() == id {
				return actor, true
			}
		}
	}
	return nil, false
}

func (a *MockAgency) AvailableActors(role RoleType) []Actor {
	defer a.actorsMutex.RUnlock()
	a.actorsMutex.RLock()
	available := make([]Actor, 0)
	for _, actor := range a.Actors[role] {
		if !actor.IsHired() {
			available = append(available, actor)
		}
	}
	return available
}

func (a *MockAgency) Open() error {
	panic("implement me")
}

func (a *MockAgency) Close() error {
	panic("implement me")
}

// MockActor is used for testing with acting.Actor by providing methods to emit
// incoming messages and recording via MessageCollector. Create one with
// NewMockActor.
type MockActor struct {
	id        messages.ActorID
	idMutex   sync.RWMutex
	name      string
	nameMutex sync.RWMutex
	quit      chan struct{}
	// HireErr is the error to return when Hire is called.
	HireErr      error
	isHired      bool
	isHiredMutex sync.RWMutex
	// FireErr is the error to return when Fire is called.
	FireErr error
	// SendErr is the error to return when Send is called.
	SendErr error
	// MessageCollector for testing message flow.
	MessageCollector *messageCollector
	// incomingSM is the acting.SubscriptionManager for incoming messages.
	incomingSM *SubscriptionManager
	// outgoingSM is the acting.SubscriptionManager for outgoing messages.
	outgoingSM *SubscriptionManager
}

func NewMockActor(id messages.ActorID) *MockActor {
	return &MockActor{
		id:               id,
		quit:             make(chan struct{}),
		MessageCollector: newMessageCollector(),
		incomingSM:       NewSubscriptionManager(),
		outgoingSM:       NewSubscriptionManager(),
	}
}

// PerformQuit makes the actor quit by sending to Quit.
func (a *MockActor) PerformQuit() {
	a.incomingSM.CancelAllSubscriptions()
	a.outgoingSM.CancelAllSubscriptions()
	a.quit <- struct{}{}
}

// HandleIncomingMessage emits an incoming message.
func (a *MockActor) HandleIncomingMessage(message ActorOutgoingMessage) {
	// Marshal message.
	rawMessage, err := json.Marshal(message.Content)
	if err != nil {
		panic(fmt.Sprintf("marshal incoming message: %s", errors.Prettify(err)))
	}
	a.MessageCollector.handleIncoming(message)
	a.incomingSM.HandleMessage(Message{
		MessageType: message.MessageType,
		Content:     rawMessage,
	})
}

func (a *MockActor) ID() messages.ActorID {
	a.idMutex.RLock()
	defer a.idMutex.RUnlock()
	return a.id
}

func (a *MockActor) Hire(displayedName string) error {
	if a.HireErr != nil {
		return a.HireErr
	}
	a.isHiredMutex.Lock()
	defer a.isHiredMutex.Unlock()
	a.isHired = true
	a.nameMutex.Lock()
	defer a.nameMutex.Unlock()
	a.name = displayedName
	return a.Send(ActorOutgoingMessage{
		MessageType: messages.MessageTypeYouAreIn,
		Content:     messages.MessageYouAreIn{ActorID: a.id},
	})
}

func (a *MockActor) Name() string {
	a.nameMutex.RLock()
	defer a.nameMutex.RUnlock()
	return a.name
}

func (a *MockActor) IsHired() bool {
	a.isHiredMutex.RLock()
	defer a.isHiredMutex.RUnlock()
	return a.isHired
}

func (a *MockActor) Fire() error {
	if a.FireErr != nil {
		return a.FireErr
	}
	a.isHiredMutex.Lock()
	defer a.isHiredMutex.Unlock()
	a.isHired = false
	return nil
}

func (a *MockActor) Send(message ActorOutgoingMessage) error {
	if a.SendErr != nil {
		return a.SendErr
	}
	a.MessageCollector.handleOutgoing(message)
	// Marshal message.
	rawMessage, err := json.Marshal(message.Content)
	if err != nil {
		return errors.Wrap(err, "marshal message content")
	}
	a.outgoingSM.HandleMessage(Message{
		MessageType: message.MessageType,
		Content:     rawMessage,
	})
	return nil
}

// SubscribeMessageType subscribes to incoming messages.
func (a *MockActor) SubscribeMessageType(messageType messages.MessageType) GeneralNewsletter {
	sub := a.incomingSM.SubscribeMessageType(messageType)
	return GeneralNewsletter{
		Newsletter: Newsletter{
			Actor:        a,
			Subscription: sub,
		},
		Receive: sub.out,
	}
}

// Unsubscribe subscribes from incoming messages.
func (a *MockActor) Unsubscribe(subscription *subscription) error {
	return a.incomingSM.Unsubscribe(subscription)
}

// SubscribeOutgoingMessageType subscribes to outgoing messages.
func (a *MockActor) SubscribeOutgoingMessageType(messageType messages.MessageType) GeneralNewsletter {
	sub := a.outgoingSM.SubscribeMessageType(messageType)
	return GeneralNewsletter{
		Newsletter: Newsletter{
			Actor:        a,
			Subscription: sub,
		},
		Receive: sub.out,
	}
}

// UnsubscribeOutgoing subscribes from outgoing messages.
func (a *MockActor) UnsubscribeOutgoing(sub *subscription) error {
	return a.outgoingSM.Unsubscribe(sub)
}

func (a *MockActor) Receive() <-chan ActorIncomingMessage {
	panic("NO!")
}

func (a *MockActor) Quit() <-chan struct{} {
	return a.quit
}

// messageCollector is used for testing message flow for actors.
type messageCollector struct {
	// incoming holds all received messages.
	incoming []ActorOutgoingMessage
	// outgoing holds all outgoing messages.
	outgoing []ActorOutgoingMessage
	// m is a lock for incoming and outgoing.
	m sync.RWMutex
}

func newMessageCollector() *messageCollector {
	return &messageCollector{
		incoming: make([]ActorOutgoingMessage, 0),
		outgoing: make([]ActorOutgoingMessage, 0),
	}
}

func (c *messageCollector) String() string {
	return fmt.Sprintf("INCOMING:\n%v\n\nOUTGOING:%v", c.incoming, c.outgoing)
}

// handleIncoming records the given incoming message.
func (c *messageCollector) handleIncoming(message ActorOutgoingMessage) {
	c.m.Lock()
	c.incoming = append(c.incoming, message)
	c.m.Unlock()
}

// handleOutgoing records the given outgoing message.
func (c *messageCollector) handleOutgoing(message ActorOutgoingMessage) {
	c.m.Lock()
	c.outgoing = append(c.outgoing, message)
	c.m.Unlock()
}

func assureMatchingMessageTypes(exact bool, expected []messages.MessageType, actual []messages.MessageType) error {
	if len(expected) == 0 {
		panic("missing message types")
	}
	if len(actual) < len(expected) {
		return fmt.Errorf("more required message types (%d) than recorded ones (%d)", len(expected), len(actual))
	}
	wantedMessageTypeIndex := 0
	for _, messageType := range actual {
		if messageType == expected[wantedMessageTypeIndex] {
			// Nice.
			wantedMessageTypeIndex++
		} else if exact {
			return fmt.Errorf("last %d message type(s) do not match recorded ones", len(expected)-wantedMessageTypeIndex)
		}
		if wantedMessageTypeIndex == len(expected) {
			// All found.
			return nil
		}
	}
	if wantedMessageTypeIndex != len(expected) {
		return fmt.Errorf("last %d message type(s) not in recorded ones", len(expected)-wantedMessageTypeIndex)
	}
	return nil
}

// AssureIncomingMessageTypes assures that the given message types are present
// in the (optionally exact) order in recorded incoming messages.
func (c *messageCollector) AssureIncomingMessageTypes(exact bool, messageTypes ...messages.MessageType) error {
	c.m.RLock()
	defer c.m.RUnlock()
	actual := make([]messages.MessageType, len(c.incoming))
	for i, message := range c.incoming {
		actual[i] = message.MessageType
	}
	return assureMatchingMessageTypes(exact, messageTypes, actual)
}

// AssureOutgoingMessageTypes assures that the given message types are present
// in the (optionally exact) order in recorded outgoing messages.
func (c *messageCollector) AssureOutgoingMessageTypes(exact bool, messageTypes ...messages.MessageType) error {
	c.m.RLock()
	defer c.m.RUnlock()
	actual := make([]messages.MessageType, len(c.outgoing))
	for i, message := range c.outgoing {
		actual[i] = message.MessageType
	}
	return assureMatchingMessageTypes(exact, messageTypes, actual)
}

// IncomingMessageCount is the number of recorded incoming messages.
func (c *messageCollector) IncomingMessageCount() int {
	return len(c.incoming)
}

// OutgoingMessageCount is the number of recorded outgoing messages.
func (c *messageCollector) OutgoingMessageCount() int {
	return len(c.outgoing)
}
