package portal

import (
	"context"
	"github.com/eclipse/paho.golang/paho"
	"github.com/lefinal/masc-server/errors"
	"github.com/lefinal/masc-server/event"
	"go.uber.org/zap"
	"sync"
)

// mqttRouter abstracts paho.Router with only stuff that is needed for router.
type mqttRouter interface {
	RegisterHandler(topic string, handler paho.MessageHandler)
	UnregisterHandler(topic string)
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
		handler.subscriptionsMutex.RUnlock()
		allForwarded.Wait()
	}
}

// router is used for multiplexing MQTT subscriptions and forwarding received
// messages according to them.
type router struct {
	logger *zap.Logger
	// mqtt is the actual router that performs the matching.
	mqtt mqttRouter
	// registeredHandlers holds all handlers by subscribed topics.
	registeredHandlers map[Topic]*registeredHandler
	// registeredHandlersMutex locks registeredHandlers.
	registeredHandlersMutex sync.Mutex
}

func newRouter(logger *zap.Logger, mqtt mqttRouter) *router {
	return &router{
		logger:             logger,
		mqtt:               mqtt,
		registeredHandlers: make(map[Topic]*registeredHandler),
	}
}

// subscribe for the given Topic and forward messages to the given channel until
// the context.Context is done.
func (router *router) subscribe(lifetime context.Context, topic Topic, forward chan<- event.Event[any]) {
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
	}()
}

// unsubscribe the given subscription for the Topic. Only router should call this!
func (router *router) unsubscribe(topic Topic, sub *subscription) {
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
