package acting

import (
	"encoding/json"
	"fmt"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/logging"
	"github.com/LeFinal/masc-server/messages"
	"github.com/LeFinal/masc-server/util"
	"sync"
)

// SubscriptionToken is used for subscriptions that were initiated via
// Actor.SubscribeMessageType. Use it for unsubscribing.
type SubscriptionToken int

// Newsletter is a container for an Actor with a created SubscriptionToken.
// This allows easy unsubscribing as the Actor is not needed everytime.
type Newsletter struct {
	// actor is the Actor the subscription is for.
	actor Actor
	// subscriptionToken is the actual SubscriptionToken.
	subscriptionToken SubscriptionToken
}

// GeneralNewsletter wraps Newsletter with the receive-channel for raw messages.
type GeneralNewsletter struct {
	Newsletter
	Receive <-chan json.RawMessage
}

// NotifyingNewsletter wraps Newsletter with a receive-channel with empty
// struct.
type NotifyingNewsletter struct {
	Newsletter
	Receive <-chan struct{}
}

// subscription is a container for subscriptions that were created via
// Actor.SubscribeMessageType.
type subscription struct {
	// messageType is the messages.MessageType the subscription is for.
	messageType messages.MessageType
	// token is the SubscriptionToken for unsubscribing.
	token SubscriptionToken
	// out is the channel for received messages that matched the messageType.
	out chan<- json.RawMessage
}

// subscriptionManager allows managing subscriptions by using a channel that
// receives messages and forwarding them to subscriptions for the respective
// messages.MessageType. Add subscriptions via subscribeMessageType and don't
// forget to unsubscribe. If all subscriptions should be cancelled, call
// cancelAllSubscriptions. Messages are handled by calling handleMessage.
type subscriptionManager struct {
	// subscriptionsByMessageType is used for providing quick access to subscribers
	// when a message is received.
	subscriptionsByMessageType map[messages.MessageType][]*subscription
	// subscriptionsByToken allows quick access when unsubscribing by providing the
	// wanted messages.MessageType in the subscription for removing from
	// subscriptionsByMessageType.
	subscriptionsByToken map[SubscriptionToken]*subscription
	// subscriptionCounter is used for generating a simple SubscriptionToken.
	subscriptionCounter int
	// subscriptionsMutex is a lock for subscriptionsByMessageType,
	// subscriptionsByToken and subscriptionCounter.
	subscriptionsMutex sync.RWMutex
}

// newSubscriptionManager creates a new subscriptionManager that is ready to
// use.
func newSubscriptionManager() *subscriptionManager {
	return &subscriptionManager{
		subscriptionsByMessageType: make(map[messages.MessageType][]*subscription),
		subscriptionsByToken:       make(map[SubscriptionToken]*subscription),
	}
}

// subscribeMessageType subscribes messages with the given messages.MessageType.
func (m *subscriptionManager) subscribeMessageType(messageType messages.MessageType) (<-chan json.RawMessage, SubscriptionToken) {
	m.subscriptionsMutex.Lock()
	m.subscriptionCounter++
	out := make(chan json.RawMessage)
	sub := subscription{
		messageType: messageType,
		token:       SubscriptionToken(m.subscriptionCounter),
		out:         out,
	}
	m.subscriptionsByToken[sub.token] = &sub
	if _, ok := m.subscriptionsByMessageType[messageType]; !ok {
		m.subscriptionsByMessageType[messageType] = make([]*subscription, 0, 1)
	}
	m.subscriptionsByMessageType[messageType] = append(m.subscriptionsByMessageType[messageType], &sub)
	m.subscriptionsMutex.Unlock()
	return out, sub.token
}

// handleMessages handles the passed ActorIncomingMessage and returns the number
// of subscribers the message was forwarded to.
func (m *subscriptionManager) handleMessage(message ActorIncomingMessage) int {
	// Handle incoming message.
	m.subscriptionsMutex.RLock()
	forwards := 0
	if subscriptions, ok := m.subscriptionsByMessageType[message.MessageType]; ok {
		// Forward to each subscriber.
		for _, s := range subscriptions {
			forwards++
			s.out <- message.Content
		}
	}
	m.subscriptionsMutex.RUnlock()
	return forwards
}

// unsubscribe allows unsubscribing an ongoing subscription with the given
// SubscriptionToken.
func (m *subscriptionManager) unsubscribe(token SubscriptionToken) error {
	m.subscriptionsMutex.Lock()
	sub, subFoundByToken := m.subscriptionsByToken[token]
	if !subFoundByToken {
		return errors.Error{
			Code:    errors.ErrInternal,
			Kind:    errors.KindUnknown,
			Message: fmt.Sprintf("unknown message subscription token %v", token),
			Details: errors.Details{"token": token},
		}
	}
	// Find position of token in subscriptions by message type.
	subsForMessageType, subscriptionsExistForMessageType := m.subscriptionsByMessageType[sub.messageType]
	if !subscriptionsExistForMessageType {
		// LOL.
		return errors.Error{
			Code:    errors.ErrInternal,
			Kind:    errors.KindUnknown,
			Message: "no subscriptions for token by message type although it should exist. what?",
			Details: errors.Details{"token": token, "alreadyFoundSubscription": sub},
		}
	}
	pos := -1
	for i, subForMessageType := range subsForMessageType {
		if subForMessageType.token == token {
			pos = i
			break
		}
	}
	if pos == -1 {
		// WHAT?
		return errors.Error{
			Code: errors.ErrInternal,
			Kind: errors.KindUnknown,
			Message: fmt.Sprintf("subscription with token %v not found in subscriptions by message type %v although it should exist. why?",
				token, sub.messageType),
			Details: errors.Details{"token": token, "alreadyFoundSubscription": sub},
		}
	}
	// Remove from subs for message type.
	subsForMessageType[pos] = subsForMessageType[len(subsForMessageType)-1]
	m.subscriptionsByMessageType[sub.messageType] = subsForMessageType[:len(subsForMessageType)-1]
	m.subscriptionsMutex.Unlock()
	return nil
}

// cancelAllSubscriptions stops the running subscriptionManager and cancels all ongoing
// subscriptions.
func (m *subscriptionManager) cancelAllSubscriptions() {
	m.subscriptionsMutex.Lock()
	for _, sub := range m.subscriptionsByToken {
		close(sub.out)
	}
	m.subscriptionsByToken = make(map[SubscriptionToken]*subscription)
	m.subscriptionsByMessageType = make(map[messages.MessageType][]*subscription)
	m.subscriptionsMutex.Unlock()
}

// decodeAsJSONOrLogSubscriptionParseError decodes the passed raw JSON message
// for the given interface. If decoding fails, an error is logged via
// logSubscriptionParseError using the passed messages.MessageType and false is
// returned. Otherwise, true is returned.
func decodeAsJSONOrLogSubscriptionParseError(messageType messages.MessageType, raw json.RawMessage, target interface{}) bool {
	err := util.DecodeAsJSON(raw, target)
	if err != nil {
		logSubscriptionParseError(messageType, err)
		return false
	}
	return true
}

// logSubscriptionParseError logs decoding errors for message subscriptions.
func logSubscriptionParseError(messageType messages.MessageType, err error) {
	logging.ActingLogger.Infof("parse subscription message for type %s: %s", messageType, err)
}

// Unsubscribe uses the wrapped Actor in the given Newsletter in order to
// unsubscribe with the contained SubscriptionToken.
func Unsubscribe(newsletter Newsletter) error {
	err := newsletter.actor.Unsubscribe(newsletter.subscriptionToken)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("unsubscribe message type for actor %s with token %v",
			newsletter.actor.ID(), newsletter.subscriptionToken))
	}
	return nil
}

// UnsubscribeOrLogError unsubscribes the given Newsletter. If unsubscribing
// fails, the error is logged to logging.ActingLogger.
func UnsubscribeOrLogError(newsletter Newsletter) {
	err := Unsubscribe(newsletter)
	if err != nil {
		errors.Log(logging.ActingLogger, err)
	}
}

// NewsletterAcceptDevice wraps Newsletter with a receive-channel for
// messages.MessageAcceptDevice.
type NewsletterAcceptDevice struct {
	Newsletter
	Receive <-chan messages.MessageAcceptDevice
}

// SubscribeMessageTypeAcceptDevice subscribes message with
// messages.MessageTypeAcceptDevice for the given Actor.
func SubscribeMessageTypeAcceptDevice(actor Actor) NewsletterAcceptDevice {
	newsletter := actor.SubscribeMessageType(messages.MessageTypeAcceptDevice)
	cc := make(chan messages.MessageAcceptDevice)
	go func() {
		for raw := range newsletter.Receive {
			var m messages.MessageAcceptDevice
			if !decodeAsJSONOrLogSubscriptionParseError(messages.MessageTypeAcceptDevice, raw, &m) {
				continue
			}
			cc <- m
		}
		close(cc)
	}()
	return NewsletterAcceptDevice{
		Newsletter: newsletter.Newsletter,
		Receive:    cc,
	}
}

// SubscribeMessageTypeGetDevices subscribes message with
// messages.MessageTypeGetDevices for the given Actor.
func SubscribeMessageTypeGetDevices(actor Actor) NotifyingNewsletter {
	newsletter := actor.SubscribeMessageType(messages.MessageTypeGetDevices)
	cc := make(chan struct{})
	go func() {
		for range newsletter.Receive {
			cc <- struct{}{}
		}
		close(cc)
	}()
	return NotifyingNewsletter{
		Newsletter: newsletter.Newsletter,
		Receive:    cc,
	}
}

// NewsletterRoleAssignments wraps Newsletter with a receive-channel for
// messages.MessageRoleAssignments.
type NewsletterRoleAssignments struct {
	Newsletter
	Receive <-chan messages.MessageRoleAssignments
}

// SubscribeMessageTypeRoleAssignments subscribes messages with
// messages.MessageTypeRoleAssignments for the given Actor.
func SubscribeMessageTypeRoleAssignments(actor Actor) NewsletterRoleAssignments {
	newsletter := actor.SubscribeMessageType(messages.MessageTypeRoleAssignments)
	cc := make(chan messages.MessageRoleAssignments)
	go func() {
		for raw := range newsletter.Receive {
			var m messages.MessageRoleAssignments
			if !decodeAsJSONOrLogSubscriptionParseError(messages.MessageTypeRoleAssignments, raw, &m) {
				continue
			}
			cc <- m
		}
		close(cc)
	}()
	return NewsletterRoleAssignments{
		Newsletter: newsletter.Newsletter,
		Receive:    cc,
	}
}
