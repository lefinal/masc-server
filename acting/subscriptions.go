package acting

import (
	"context"
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
	// Actor is the Actor the subscription is for.
	Actor Actor
	// Subscription is the actual Subscription.
	Subscription *subscription
}

// GeneralNewsletter wraps Newsletter with the receive-channel for raw messages.
type GeneralNewsletter struct {
	Newsletter
	// Receive allows receiving messages. However, you should never read directly
	// from this but use helper functions like SubscribeMessageTypeRoleAssignments
	// because the Receive-channel will never be closed. If you still want to read
	// from it, use the context in Newsletter.Subscription in order to check if the
	// subscription is still active.
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
	// Ctx is the context of the subscription which is done when the subscription is
	// no longer active.
	Ctx context.Context
	// setInactive cancels the Ctx. This allows dropping messages to forward.
	setInactive context.CancelFunc
	// messageType is the messages.MessageType the subscription is for.
	messageType messages.MessageType
	// token is the SubscriptionToken for unsubscribing.
	token SubscriptionToken
	// out is the channel for received messages that matched the messageType. If the
	// subscription is already contained in Newsletter, do not receive manually from
	// here!
	out chan json.RawMessage
}

// Message is a container for messages to be handled by a SubscriptionManager.
type Message struct {
	// MessageType is the type of the message. This determines how to parse the
	// Content.
	MessageType messages.MessageType
	// Content is the raw message content which will be parsed based on the
	// MessageType.
	Content json.RawMessage
}

// SubscriptionManager allows managing subscriptions by using a channel that
// receives messages and forwarding them to subscriptions for the respective
// messages.MessageType. Add subscriptions via subscribeMessageType and don't
// forget to unsubscribe. If all subscriptions should be cancelled, call
// cancelAllSubscriptions. Messages are handled by calling handleMessage.
type SubscriptionManager struct {
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

// NewSubscriptionManager creates a new SubscriptionManager that is ready to
// use.
func NewSubscriptionManager() *SubscriptionManager {
	return &SubscriptionManager{
		subscriptionsByMessageType: make(map[messages.MessageType][]*subscription),
		subscriptionsByToken:       make(map[SubscriptionToken]*subscription),
	}
}

// SubscribeMessageType subscribes messages with the given messages.MessageType.
func (m *SubscriptionManager) SubscribeMessageType(messageType messages.MessageType) *subscription {
	m.subscriptionsMutex.Lock()
	defer m.subscriptionsMutex.Unlock()
	m.subscriptionCounter++
	out := make(chan json.RawMessage)
	ctx, setInactive := context.WithCancel(context.Background())
	sub := &subscription{
		Ctx:         ctx,
		setInactive: setInactive,
		messageType: messageType,
		token:       SubscriptionToken(m.subscriptionCounter),
		out:         out,
	}
	m.subscriptionsByToken[sub.token] = sub
	if _, ok := m.subscriptionsByMessageType[messageType]; !ok {
		m.subscriptionsByMessageType[messageType] = make([]*subscription, 0, 1)
	}
	m.subscriptionsByMessageType[messageType] = append(m.subscriptionsByMessageType[messageType], sub)
	return sub
}

// HandleMessage handles the passed ActorIncomingMessage and returns the number
// of subscribers the message was forwarded to.
func (m *SubscriptionManager) HandleMessage(message Message) int {
	// Handle incoming message.
	m.subscriptionsMutex.RLock()
	forwards := 0
	if subscriptions, ok := m.subscriptionsByMessageType[message.MessageType]; ok {
		// Forward to each subscriber.
		for _, s := range subscriptions {
			select {
			case <-s.Ctx.Done():
				// Subscription done. We simply drop the message.
			case s.out <- message.Content:
				forwards++
			}
		}
	}
	m.subscriptionsMutex.RUnlock()
	return forwards
}

// Unsubscribe allows unsubscribing an ongoing subscription with the given
// SubscriptionToken.
func (m *SubscriptionManager) Unsubscribe(sub *subscription) error {
	// Set subscription to inactive and then allowing a potential
	// receiving abort. We will remove it from the subscriptions list afterwards.
	m.subscriptionsMutex.RLock()
	sub.setInactive()
	m.subscriptionsMutex.RUnlock()
	// Now we wait until message handling has finished in order to remove it from
	// the list.
	m.subscriptionsMutex.Lock()
	defer m.subscriptionsMutex.Unlock()
	_, subFoundByToken := m.subscriptionsByToken[sub.token]
	if !subFoundByToken {
		return errors.Error{
			Code:    errors.ErrInternal,
			Message: fmt.Sprintf("unknown message subscription token %v", sub),
			Details: errors.Details{"token": sub},
		}
	}
	// Find position of token in subscriptions by message type.
	subsForMessageType, subscriptionsExistForMessageType := m.subscriptionsByMessageType[sub.messageType]
	if !subscriptionsExistForMessageType {
		// LOL.
		return errors.Error{
			Code:    errors.ErrInternal,
			Message: "no subscriptions for token by message type although it should exist. what?",
			Details: errors.Details{"token": sub, "alreadyFoundSubscription": sub},
		}
	}
	pos := -1
	for i, subForMessageType := range subsForMessageType {
		if subForMessageType.token == sub.token {
			pos = i
			break
		}
	}
	if pos == -1 {
		// WHAT?
		return errors.Error{
			Code: errors.ErrInternal,
			Message: fmt.Sprintf("subscription with token %v not found in subscriptions by message type %v although it should exist. why?",
				sub, sub.messageType),
			Details: errors.Details{"token": sub, "alreadyFoundSubscription": sub},
		}
	}
	// Remove from subs for message type.
	subsForMessageType[pos] = subsForMessageType[len(subsForMessageType)-1]
	m.subscriptionsByMessageType[sub.messageType] = subsForMessageType[:len(subsForMessageType)-1]
	// Remove from subs by token.
	delete(m.subscriptionsByToken, sub.token)
	return nil
}

// CancelAllSubscriptions stops the running SubscriptionManager and cancels all ongoing
// subscriptions.
func (m *SubscriptionManager) CancelAllSubscriptions() {
	m.subscriptionsMutex.Lock()
	var wg sync.WaitGroup
	for i := range m.subscriptionsByToken {
		wg.Add(1)
		go func(sub *subscription) {
			defer wg.Done()
			err := m.Unsubscribe(sub)
			if err != nil {
				errors.Log(logging.SubscriptionManagerLogger, errors.Wrap(err, "unsubscribe for cancel all", nil))
			}
		}(m.subscriptionsByToken[i])
	}
	m.subscriptionsByToken = make(map[SubscriptionToken]*subscription)
	m.subscriptionsByMessageType = make(map[messages.MessageType][]*subscription)
	m.subscriptionsMutex.Unlock()
	wg.Wait()
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
	err := newsletter.Actor.Unsubscribe(newsletter.Subscription)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("unsubscribe message type for actor %s", newsletter.Actor.ID()), nil)
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

// SubscribeNotifyForMessageType subscribes to the given messages.MessageType
// for the Actor and returns a NotifyingNewsletter with a self-closing
// receive-channel.
func SubscribeNotifyForMessageType(messageType messages.MessageType, actor Actor) NotifyingNewsletter {
	newsletter := actor.SubscribeMessageType(messageType)
	cc := make(chan struct{})
	go func() {
		defer close(cc)
		for {
			select {
			case <-newsletter.Subscription.Ctx.Done():
				return
			case <-newsletter.Receive:
				select {
				case <-newsletter.Subscription.Ctx.Done():
					return
				case cc <- struct{}{}:
				}
			}
		}
	}()
	return NotifyingNewsletter{
		Newsletter: newsletter.Newsletter,
		Receive:    cc,
	}
}

// NewsletterSetDeviceName wraps Newsletter with a self-closing receive-channel
// for messages.MessageSetDeviceName.
type NewsletterSetDeviceName struct {
	Newsletter
	Receive <-chan messages.MessageSetDeviceName
}

// SubscribeMessageTypeSetDeviceName subscribes message with
// messages.MessageTypeSetDeviceName for the given Actor.
func SubscribeMessageTypeSetDeviceName(actor Actor) NewsletterSetDeviceName {
	newsletter := actor.SubscribeMessageType(messages.MessageTypeSetDeviceName)
	cc := make(chan messages.MessageSetDeviceName)
	go func() {
		defer close(cc)
		for {
			select {
			case <-newsletter.Subscription.Ctx.Done():
				return
			case raw := <-newsletter.Receive:
				var m messages.MessageSetDeviceName
				if !decodeAsJSONOrLogSubscriptionParseError(messages.MessageTypeSetDeviceName, raw, &m) {
					continue
				}
				select {
				case <-newsletter.Subscription.Ctx.Done():
					return
				case cc <- m:
				}
			}
		}
	}()
	return NewsletterSetDeviceName{
		Newsletter: newsletter.Newsletter,
		Receive:    cc,
	}
}

// NewsletterDeleteDevice wraps Newsletter with a self-closing receive-channel
// for messages.MessageDeleteDevice.
type NewsletterDeleteDevice struct {
	Newsletter
	Receive <-chan messages.MessageDeleteDevice
}

// SubscribeMessageTypeDeleteDevice subscribes message with
// messages.MessageTypeDeleteDevice for the given Actor.
func SubscribeMessageTypeDeleteDevice(actor Actor) NewsletterDeleteDevice {
	newsletter := actor.SubscribeMessageType(messages.MessageTypeDeleteDevice)
	cc := make(chan messages.MessageDeleteDevice)
	go func() {
		defer close(cc)
		for {
			select {
			case <-newsletter.Subscription.Ctx.Done():
				return
			case raw := <-newsletter.Receive:
				var m messages.MessageDeleteDevice
				if !decodeAsJSONOrLogSubscriptionParseError(messages.MessageTypeDeleteDevice, raw, &m) {
					continue
				}
				select {
				case <-newsletter.Subscription.Ctx.Done():
					return
				case cc <- m:
				}
			}
		}
	}()
	return NewsletterDeleteDevice{
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
		defer close(cc)
		for {
			select {
			case <-newsletter.Subscription.Ctx.Done():
				return
			case _ = <-newsletter.Receive:
				select {
				case <-newsletter.Subscription.Ctx.Done():
					return
				case cc <- struct{}{}:
				}
			}
		}
	}()
	return NotifyingNewsletter{
		Newsletter: newsletter.Newsletter,
		Receive:    cc,
	}
}

// NewsletterRoleAssignments wraps Newsletter with a self-closing
// receive-channel for messages.MessageRoleAssignments.
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
		defer close(cc)
		for {
			select {
			case <-newsletter.Subscription.Ctx.Done():
				return
			case raw := <-newsletter.Receive:
				var m messages.MessageRoleAssignments
				if !decodeAsJSONOrLogSubscriptionParseError(messages.MessageTypeRoleAssignments, raw, &m) {
					continue
				}
				select {
				case <-newsletter.Subscription.Ctx.Done():
					return
				case cc <- m:
				}
			}
		}
	}()
	return NewsletterRoleAssignments{
		Newsletter: newsletter.Newsletter,
		Receive:    cc,
	}
}

// NewsletterAbortMatch wraps Newsletter with a self-closing receive-channel for
// messages.MessageRoleAssignments.
type NewsletterAbortMatch struct {
	Newsletter
	Receive <-chan struct{}
}

// SubscribeMessageTypeAbortMatch subscribes messages with
// messages.MessageTypeAbortMatch for the given Actor.
func SubscribeMessageTypeAbortMatch(actor Actor) NewsletterAbortMatch {
	newsletter := actor.SubscribeMessageType(messages.MessageTypeAbortMatch)
	cc := make(chan struct{})
	go func() {
		defer close(cc)
		for {
			select {
			case <-newsletter.Subscription.Ctx.Done():
				return
			case <-newsletter.Receive:
				select {
				case <-newsletter.Subscription.Ctx.Done():
					return
				case cc <- struct{}{}:
				}
			}
		}
	}()
	return NewsletterAbortMatch{
		Newsletter: newsletter.Newsletter,
		Receive:    cc,
	}
}

// NewsletterReadyState wraps Newsletter with a self-closing receive-channel for
// messages.MessageReadyState.
type NewsletterReadyState struct {
	Newsletter
	Receive <-chan messages.MessageReadyState
}

// SubscribeMessageTypeReadyState subscribes messages with
// messages.MessageTypeReadyState for the given Actor.
func SubscribeMessageTypeReadyState(actor Actor) NewsletterReadyState {
	newsletter := actor.SubscribeMessageType(messages.MessageTypeReadyState)
	cc := make(chan messages.MessageReadyState)
	go func() {
		defer close(cc)
		for {
			select {
			case <-newsletter.Subscription.Ctx.Done():
				return
			case raw := <-newsletter.Receive:
				var m messages.MessageReadyState
				if !decodeAsJSONOrLogSubscriptionParseError(messages.MessageTypeReadyState, raw, &m) {
					continue
				}
				select {
				case <-newsletter.Subscription.Ctx.Done():
					return
				case cc <- m:
				}
			}
		}
	}()
	return NewsletterReadyState{
		Newsletter: newsletter.Newsletter,
		Receive:    cc,
	}
}

// NewsletterPlayerJoin wraps Newsletter with a self-closing receive-channel for
// messages.MessageTypePlayerJoin.
type NewsletterPlayerJoin struct {
	Newsletter
	Receive <-chan messages.MessagePlayerJoin
}

// SubscribeMessageTypePlayerJoin subscribes messages with
// messages.MessageTypePlayerJoin for the given Actor.
func SubscribeMessageTypePlayerJoin(actor Actor) NewsletterPlayerJoin {
	newsletter := actor.SubscribeMessageType(messages.MessageTypePlayerJoin)
	cc := make(chan messages.MessagePlayerJoin)
	go func() {
		defer close(cc)
		for {
			select {
			case <-newsletter.Subscription.Ctx.Done():
				return
			case raw := <-newsletter.Receive:
				var m messages.MessagePlayerJoin
				if !decodeAsJSONOrLogSubscriptionParseError(messages.MessageTypePlayerJoin, raw, &m) {
					continue
				}
				select {
				case <-newsletter.Subscription.Ctx.Done():
					return
				case cc <- m:
				}
			}
		}
	}()
	return NewsletterPlayerJoin{
		Newsletter: newsletter.Newsletter,
		Receive:    cc,
	}
}

// NewsletterPlayerLeave wraps Newsletter with a self-closing receive-channel for
// messages.MessageTypePlayerLeave.
type NewsletterPlayerLeave struct {
	Newsletter
	Receive <-chan messages.MessagePlayerLeave
}

// SubscribeMessageTypePlayerLeave subscribes messages with
// messages.MessageTypePlayerLeave for the given Actor.
func SubscribeMessageTypePlayerLeave(actor Actor) NewsletterPlayerLeave {
	newsletter := actor.SubscribeMessageType(messages.MessageTypePlayerLeave)
	cc := make(chan messages.MessagePlayerLeave)
	go func() {
		defer close(cc)
		for {
			select {
			case <-newsletter.Subscription.Ctx.Done():
				return
			case raw := <-newsletter.Receive:
				var m messages.MessagePlayerLeave
				if !decodeAsJSONOrLogSubscriptionParseError(messages.MessageTypePlayerLeave, raw, &m) {
					continue
				}
				select {
				case <-newsletter.Subscription.Ctx.Done():
					return
				case cc <- m:
				}
			}
		}
	}()
	return NewsletterPlayerLeave{
		Newsletter: newsletter.Newsletter,
		Receive:    cc,
	}
}

// NewsletterFixtureOffers wraps Newsletter with a self-closing receive-channel
// for messages.MessageFixtureOffers.
type NewsletterFixtureOffers struct {
	Newsletter
	Receive <-chan messages.MessageFixtureOffers
}

// SubscribeMessageTypeFixtureOffers subscribes messages with
// messages.MessageTypeFixtureOffers for the given Actor.
func SubscribeMessageTypeFixtureOffers(actor Actor) NewsletterFixtureOffers {
	newsletter := actor.SubscribeMessageType(messages.MessageTypeFixtureOffers)
	cc := make(chan messages.MessageFixtureOffers)
	go func() {
		defer close(cc)
		for {
			select {
			case <-newsletter.Subscription.Ctx.Done():
				return
			case raw := <-newsletter.Receive:
				var m messages.MessageFixtureOffers
				if !decodeAsJSONOrLogSubscriptionParseError(messages.MessageTypeFixtureOffers, raw, &m) {
					continue
				}
				select {
				case <-newsletter.Subscription.Ctx.Done():
					return
				case cc <- m:
				}
			}
		}
	}()
	return NewsletterFixtureOffers{
		Newsletter: newsletter.Newsletter,
		Receive:    cc,
	}
}

// NewsletterSetFixtureName wraps Newsletter with a self-closing receive-channel
// for messages.MessageSetFixtureName.
type NewsletterSetFixtureName struct {
	Newsletter
	Receive <-chan messages.MessageSetFixtureName
}

// SubscribeMessageTypeSetFixtureName subscribes message with
// messages.MessageTypeSetFixtureName for the given Actor.
func SubscribeMessageTypeSetFixtureName(actor Actor) NewsletterSetFixtureName {
	newsletter := actor.SubscribeMessageType(messages.MessageTypeSetFixtureName)
	cc := make(chan messages.MessageSetFixtureName)
	go func() {
		defer close(cc)
		for {
			select {
			case <-newsletter.Subscription.Ctx.Done():
				return
			case raw := <-newsletter.Receive:
				var m messages.MessageSetFixtureName
				if !decodeAsJSONOrLogSubscriptionParseError(messages.MessageTypeSetFixtureName, raw, &m) {
					continue
				}
				select {
				case <-newsletter.Subscription.Ctx.Done():
					return
				case cc <- m:
				}
			}
		}
	}()
	return NewsletterSetFixtureName{
		Newsletter: newsletter.Newsletter,
		Receive:    cc,
	}
}

// NewsletterGetFixtures wraps Newsletter with a self-closing receive-channel.
type NewsletterGetFixtures struct {
	Newsletter
	Receive <-chan struct{}
}

// SubscribeMessageTypeGetFixtures subscribes messages with
// messages.MessageTypeGetFixtures for the given Actor.
func SubscribeMessageTypeGetFixtures(actor Actor) NewsletterGetFixtures {
	newsletter := actor.SubscribeMessageType(messages.MessageTypeGetFixtures)
	cc := make(chan struct{})
	go func() {
		defer close(cc)
		for {
			select {
			case <-newsletter.Subscription.Ctx.Done():
				return
			case <-newsletter.Receive:
				select {
				case <-newsletter.Subscription.Ctx.Done():
					return
				case cc <- struct{}{}:
				}
			}
		}
	}()
	return NewsletterGetFixtures{
		Newsletter: newsletter.Newsletter,
		Receive:    cc,
	}
}

// NewsletterDeleteFixture wraps Newsletter with a self-closing receive-channel
// for messages.MessageDeleteFixture.
type NewsletterDeleteFixture struct {
	Newsletter
	Receive <-chan messages.MessageDeleteFixture
}

// SubscribeMessageTypeDeleteFixture subscribes message with
// messages.MessageTypeDeleteFixture for the given Actor.
func SubscribeMessageTypeDeleteFixture(actor Actor) NewsletterDeleteFixture {
	newsletter := actor.SubscribeMessageType(messages.MessageTypeDeleteFixture)
	cc := make(chan messages.MessageDeleteFixture)
	go func() {
		defer close(cc)
		for {
			select {
			case <-newsletter.Subscription.Ctx.Done():
				return
			case raw := <-newsletter.Receive:
				var m messages.MessageDeleteFixture
				if !decodeAsJSONOrLogSubscriptionParseError(messages.MessageTypeDeleteFixture, raw, &m) {
					continue
				}
				select {
				case <-newsletter.Subscription.Ctx.Done():
					return
				case cc <- m:
				}
			}
		}
	}()
	return NewsletterDeleteFixture{
		Newsletter: newsletter.Newsletter,
		Receive:    cc,
	}
}

// NewsletterGetFixtureStates wraps Newsletter with a self-closing receive-channel.
type NewsletterGetFixtureStates struct {
	Newsletter
	Receive <-chan struct{}
}

// SubscribeMessageTypeGetFixtureStates subscribes messages with
// messages.MessageTypeGetFixtureStates for the given Actor.
func SubscribeMessageTypeGetFixtureStates(actor Actor) NewsletterGetFixtureStates {
	newsletter := actor.SubscribeMessageType(messages.MessageTypeGetFixtureStates)
	cc := make(chan struct{})
	go func() {
		defer close(cc)
		for {
			select {
			case <-newsletter.Subscription.Ctx.Done():
				return
			case <-newsletter.Receive:
				select {
				case <-newsletter.Subscription.Ctx.Done():
					return
				case cc <- struct{}{}:
				}
			}
		}
	}()
	return NewsletterGetFixtureStates{
		Newsletter: newsletter.Newsletter,
		Receive:    cc,
	}
}

// NewsletterSetFixturesEnabled wraps Newsletter with a self-closing receive-channel
// for messages.MessageSetFixturesEnabled.
type NewsletterSetFixturesEnabled struct {
	Newsletter
	Receive <-chan messages.MessageSetFixturesEnabled
}

// SubscribeMessageTypeSetFixturesEnabled subscribes message with
// messages.MessageTypeSetFixturesEnabled for the given Actor.
func SubscribeMessageTypeSetFixturesEnabled(actor Actor) NewsletterSetFixturesEnabled {
	newsletter := actor.SubscribeMessageType(messages.MessageTypeSetFixturesEnabled)
	cc := make(chan messages.MessageSetFixturesEnabled)
	go func() {
		defer close(cc)
		for {
			select {
			case <-newsletter.Subscription.Ctx.Done():
				return
			case raw := <-newsletter.Receive:
				var m messages.MessageSetFixturesEnabled
				if !decodeAsJSONOrLogSubscriptionParseError(messages.MessageTypeSetFixturesEnabled, raw, &m) {
					continue
				}
				select {
				case <-newsletter.Subscription.Ctx.Done():
					return
				case cc <- m:
				}
			}
		}
	}()
	return NewsletterSetFixturesEnabled{
		Newsletter: newsletter.Newsletter,
		Receive:    cc,
	}
}

// NewsletterSetFixturesLocating wraps Newsletter with a self-closing receive-channel
// for messages.MessageSetFixturesLocating.
type NewsletterSetFixturesLocating struct {
	Newsletter
	Receive <-chan messages.MessageSetFixturesLocating
}

// SubscribeMessageTypeSetFixturesLocating subscribes message with
// messages.MessageTypeSetFixturesLocating for the given Actor.
func SubscribeMessageTypeSetFixturesLocating(actor Actor) NewsletterSetFixturesLocating {
	newsletter := actor.SubscribeMessageType(messages.MessageTypeSetFixturesLocating)
	cc := make(chan messages.MessageSetFixturesLocating)
	go func() {
		defer close(cc)
		for {
			select {
			case <-newsletter.Subscription.Ctx.Done():
				return
			case raw := <-newsletter.Receive:
				var m messages.MessageSetFixturesLocating
				if !decodeAsJSONOrLogSubscriptionParseError(messages.MessageTypeSetFixturesLocating, raw, &m) {
					continue
				}
				select {
				case <-newsletter.Subscription.Ctx.Done():
					return
				case cc <- m:
				}
			}
		}
	}()
	return NewsletterSetFixturesLocating{
		Newsletter: newsletter.Newsletter,
		Receive:    cc,
	}
}

// NewsletterLightSwitchHiLoState wraps Newsletter with a self-closing
// receive-channel for messages.MessageLightSwitchHiLoState.
type NewsletterLightSwitchHiLoState struct {
	Newsletter
	Receive <-chan messages.MessageLightSwitchHiLoState
}

// SubscribeMessageTypeLightSwitchHiLoState subscribes messages with
// messages.SubscribeMessageTypeLightSwitchHiLoState for the given Actor.
func SubscribeMessageTypeLightSwitchHiLoState(actor Actor) NewsletterLightSwitchHiLoState {
	newsletter := actor.SubscribeMessageType(messages.MessageTypeLightSwitchHiLoState)
	cc := make(chan messages.MessageLightSwitchHiLoState)
	go func() {
		defer close(cc)
		for {
			select {
			case <-newsletter.Subscription.Ctx.Done():
				return
			case raw := <-newsletter.Receive:
				var m messages.MessageLightSwitchHiLoState
				if !decodeAsJSONOrLogSubscriptionParseError(messages.MessageTypeLightSwitchHiLoState, raw, &m) {
					continue
				}
				select {
				case <-newsletter.Subscription.Ctx.Done():
					return
				case cc <- m:
				}
			}
		}
	}()
	return NewsletterLightSwitchHiLoState{
		Newsletter: newsletter.Newsletter,
		Receive:    cc,
	}
}

// NewsletterLightSwitchOffers wraps Newsletter with a self-closing
// receive-channel for messages.MessageLightSwitchOffers.
type NewsletterLightSwitchOffers struct {
	Newsletter
	Receive <-chan messages.MessageLightSwitchOffers
}

// SubscribeMessageTypeLightSwitchOffers subscribes messages with
// messages.MessageTypeLightSwitchOffers for the given Actor.
func SubscribeMessageTypeLightSwitchOffers(actor Actor) NewsletterLightSwitchOffers {
	newsletter := actor.SubscribeMessageType(messages.MessageTypeLightSwitchOffers)
	cc := make(chan messages.MessageLightSwitchOffers)
	go func() {
		defer close(cc)
		for {
			select {
			case <-newsletter.Subscription.Ctx.Done():
				return
			case raw := <-newsletter.Receive:
				var m messages.MessageLightSwitchOffers
				if !decodeAsJSONOrLogSubscriptionParseError(messages.MessageTypeLightSwitchOffers, raw, &m) {
					continue
				}
				select {
				case <-newsletter.Subscription.Ctx.Done():
					return
				case cc <- m:
				}
			}
		}
	}()
	return NewsletterLightSwitchOffers{
		Newsletter: newsletter.Newsletter,
		Receive:    cc,
	}
}

// NewsletterUpdateLightSwitch wraps Newsletter with a self-closing
// receive-channel for messages.MessageUpdateLightSwitch.
type NewsletterUpdateLightSwitch struct {
	Newsletter
	Receive <-chan messages.MessageUpdateLightSwitch
}

// SubscribeMessageTypeUpdateLightSwitch subscribes messages with
// messages.MessageTypeUpdateLightSwitch for the given Actor.
func SubscribeMessageTypeUpdateLightSwitch(actor Actor) NewsletterUpdateLightSwitch {
	newsletter := actor.SubscribeMessageType(messages.MessageTypeUpdateLightSwitch)
	cc := make(chan messages.MessageUpdateLightSwitch)
	go func() {
		defer close(cc)
		for {
			select {
			case <-newsletter.Subscription.Ctx.Done():
				return
			case raw := <-newsletter.Receive:
				var m messages.MessageUpdateLightSwitch
				if !decodeAsJSONOrLogSubscriptionParseError(messages.MessageTypeUpdateLightSwitch, raw, &m) {
					continue
				}
				select {
				case <-newsletter.Subscription.Ctx.Done():
					return
				case cc <- m:
				}
			}
		}
	}()
	return NewsletterUpdateLightSwitch{
		Newsletter: newsletter.Newsletter,
		Receive:    cc,
	}
}

// NewsletterDeleteLightSwitch wraps Newsletter with a self-closing
// receive-channel for messages.MessageDeleteLightSwitch.
type NewsletterDeleteLightSwitch struct {
	Newsletter
	Receive <-chan messages.MessageDeleteLightSwitch
}

// SubscribeMessageTypeDeleteLightSwitch subscribes messages with
// messages.MessageTypeDeleteLightSwitch for the given Actor.
func SubscribeMessageTypeDeleteLightSwitch(actor Actor) NewsletterDeleteLightSwitch {
	newsletter := actor.SubscribeMessageType(messages.MessageTypeDeleteLightSwitch)
	cc := make(chan messages.MessageDeleteLightSwitch)
	go func() {
		defer close(cc)
		for {
			select {
			case <-newsletter.Subscription.Ctx.Done():
				return
			case raw := <-newsletter.Receive:
				var m messages.MessageDeleteLightSwitch
				if !decodeAsJSONOrLogSubscriptionParseError(messages.MessageTypeDeleteLightSwitch, raw, &m) {
					continue
				}
				select {
				case <-newsletter.Subscription.Ctx.Done():
					return
				case cc <- m:
				}
			}
		}
	}()
	return NewsletterDeleteLightSwitch{
		Newsletter: newsletter.Newsletter,
		Receive:    cc,
	}
}
