package portal

import (
	"context"
	"github.com/eclipse/paho.golang/paho"
	"github.com/lefinal/masc-server/errors"
	"github.com/lefinal/masc-server/event"
	"go.uber.org/zap"
	"sync"
)

// mqttKiosk allows ACTUAL subscribing and unsubscribing to MQTT topics. It is
// an abstraction of autopaho.ConnectionManager for usage in portalGateway.
type mqttKiosk interface {
	Subscribe(ctx context.Context, s *paho.Subscribe) (*paho.Suback, error)
	Unsubscribe(ctx context.Context, u *paho.Unsubscribe) (*paho.Unsuback, error)
}

// mqttInboundRouter abstracts paho.Router with only stuff that is needed for
// portalGateway.
type mqttInboundRouter interface {
	RegisterHandler(topic string, handler paho.MessageHandler)
	UnregisterHandler(topic string)
}

// portalGatewayMQTTBridge acts as an mqttInboundRouter, but also performs
// actual MQTT subscriptions using mqttKiosk when registering handlers with the
// set inboundRouter.
type portalGatewayMQTTBridge struct {
	logger        *zap.Logger
	kiosk         mqttKiosk
	inboundRouter mqttInboundRouter
}

// RegisterHandler subscribes
func (b *portalGatewayMQTTBridge) RegisterHandler(topic string, handler paho.MessageHandler) {
	_, err := b.kiosk.Subscribe(context.Background(), &paho.Subscribe{
		Subscriptions: map[string]paho.SubscribeOptions{
			topic: {QoS: mqttQOS},
		},
	})
	if err != nil {
		errors.Log(b.logger, errors.Wrap(err, "subscribe with kiosk", nil))
		return
	}
	b.inboundRouter.RegisterHandler(topic, handler)
}

func (b *portalGatewayMQTTBridge) UnregisterHandler(topic string) {
	b.inboundRouter.UnregisterHandler(topic)
	_, err := b.kiosk.Unsubscribe(context.Background(), &paho.Unsubscribe{
		Topics: []string{topic},
	})
	if err != nil {
		errors.Log(b.logger, errors.Wrap(err, "unsubscribe with kiosk", nil))
		return
	}
}

// subscription is a container for the lifetime context.Context and the channel
// to forward the received paho.Publish message to.
type subscription struct {
	lifetime context.Context
	forward  chan<- event.Event[any]
}

// registeredHandler is a container for subscriptions to serve.
type registeredHandler struct {
	// subscriptions contains all active subscriptions that are served by the
	// handler.
	subscriptions map[*subscription]struct{}
	// subscriptionsMutex locks subscriptions.
	subscriptionsMutex sync.RWMutex
}

// Handler returns a paho.MessageHandler that forwards to all subscriptions for
// the handler.
func (handler *registeredHandler) Handler() paho.MessageHandler {
	return func(publish *paho.Publish) {
		// Forward to all listeners.
		var allForwarded sync.WaitGroup
		handler.subscriptionsMutex.RLock()
		for sub := range handler.subscriptions {
			allForwarded.Add(1)
			go func(sub *subscription) {
				defer allForwarded.Done()
				select {
				case <-sub.lifetime.Done():
				case sub.forward <- event.Event[any]{Publish: publish}:
				}
			}(sub)
		}
		allForwarded.Wait()
		handler.subscriptionsMutex.RUnlock()
	}
}

// portalGateway is used for multiplexing MQTT subscriptions and forwarding received
// messages according to them.
type portalGateway struct {
	logger *zap.Logger
	// mqtt is the actual portalGateway that performs the matching.
	mqtt *portalGatewayMQTTBridge
	// registeredHandlers holds all handlers by subscribed topics.
	registeredHandlers map[Topic]*registeredHandler
	// registeredHandlersMutex locks registeredHandlers.
	registeredHandlersMutex sync.Mutex
}

func newPortalGateway(logger *zap.Logger, mqtt *portalGatewayMQTTBridge) *portalGateway {
	return &portalGateway{
		logger:             logger,
		mqtt:               mqtt,
		registeredHandlers: make(map[Topic]*registeredHandler),
	}
}

// subscribe for the given Topic and forward messages to the given channel until
// the context.Context is done. After that, the channel will be closed!
func (router *portalGateway) subscribe(lifetime context.Context, topic Topic) <-chan event.Event[any] {
	router.registeredHandlersMutex.Lock()
	defer router.registeredHandlersMutex.Unlock()
	// Check if already existing.
	handlerRef, ok := router.registeredHandlers[topic]
	if !ok {
		handlerRef = &registeredHandler{subscriptions: make(map[*subscription]struct{})}
		router.registeredHandlers[topic] = handlerRef
		// Subscribe MQTT topic.
		router.mqtt.RegisterHandler(string(topic), handlerRef.Handler())
		router.logger.Debug("subscribed to topic", zap.Any("topic", topic))
	}
	// Add subscription.
	forward := make(chan event.Event[any])
	sub := &subscription{
		lifetime: lifetime,
		forward:  forward,
	}
	handlerRef.subscriptionsMutex.Lock()
	handlerRef.subscriptions[sub] = struct{}{}
	handlerRef.subscriptionsMutex.Unlock()
	// Unsubscribe when lifetime done.
	go func() {
		<-lifetime.Done()
		router.unsubscribe(topic, sub)
		handlerRef.subscriptionsMutex.Lock()
		close(forward)
		handlerRef.subscriptionsMutex.Unlock()
	}()
	return forward
}

// unsubscribe the given subscription for the Topic. Only portalGateway should call this!
func (router *portalGateway) unsubscribe(topic Topic, sub *subscription) {
	router.registeredHandlersMutex.Lock()
	defer router.registeredHandlersMutex.Unlock()
	// Get handler.
	handler, ok := router.registeredHandlers[topic]
	if !ok {
		errors.Log(router.logger, errors.NewInternalError("unsubscribe called for unknown registered handler",
			errors.Details{"topic": topic}))
		return
	}
	// Remove subscription.
	handler.subscriptionsMutex.Lock()
	defer handler.subscriptionsMutex.Unlock()
	if _, ok := handler.subscriptions[sub]; !ok {
		errors.Log(router.logger, errors.NewInternalError("unsubscribe with unknown subscription for handler",
			errors.Details{"topic": topic}))
		return
	}
	delete(handler.subscriptions, sub)
	// Check if subscriptions left as then we do not need to unregister the handler.
	if len(handler.subscriptions) > 0 {
		return
	}
	// Unregister handler.
	router.mqtt.UnregisterHandler(string(topic))
}
