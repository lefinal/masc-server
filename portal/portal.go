package portal

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/event"
	"github.com/eclipse/paho.golang/autopaho"
	"github.com/eclipse/paho.golang/paho"
	"go.uber.org/zap"
	"net/url"
	"time"
)

const mqttClientID = "masc-server"
const baseTopic = "lefinal/masc"
const mqttKeepAlive = 8

const mqttQOS = 0

// Topic is an MQTT topic.
type Topic string

// Config is the config for the Bridge.
type Config struct {
	// MQTTAddr is the address where the MQTT-server is found.
	MQTTAddr string
}

// Newsletter is used with Portal.Subscribe in order to subscribe to topics.
type Newsletter[payloadT any] struct {
	unregisterFn func()
	// Receive receives when a new message for the subscribed topic was received.
	// When the Newsletter is unsubscribed, the Receive-channel will be closed.
	Receive <-chan event.Event[payloadT]
}

func (sub *Newsletter[payload]) Unsubscribe() {
	sub.unregisterFn()
}

// publisher is used for publishing MQTT events.
type publisher interface {
	Publish(ctx context.Context, publish *paho.Publish) (*paho.PublishResponse, error)
}

// Base is a wrapper for all connection related stuff for a Portal. Using the
// Base, you only need to Open the Base and then use portals via NewPortal.
type Base interface {
	// Open the connection. Stays opened until the given context.Context is done.
	Open(ctx context.Context) error
	// NewPortal creates a new Portal that uses the connection from the Base.
	NewPortal(name string) Portal
}

type basePortal struct {
	logger *zap.Logger
	config Config
	// brokerURL is the URL of the MQTT broker.
	brokerURL *url.URL
	// router is responsible for registering subscription requests as well as
	// multiplexing and forwarding messages.
	router *router
	// publisher is used for publishing MQTT messages.
	publisher publisher
}

type Portal interface {
	// Subscribe returns a Newsletter for the given Topic.
	Subscribe(ctx context.Context, topic Topic) *Newsletter[any]
	// Publish the given payload to the Topic. It will catch any errors during
	// publishing and log them using the Logger.
	Publish(ctx context.Context, topic Topic, payload interface{})
	// Logger is needed in order to provide error logging for Subscribe as generics
	// are not supported for methods in Go 1.18.
	Logger() *zap.Logger
}

// NewBase creates a Base with the given Config. Open it with Base.Open.
func NewBase(logger *zap.Logger, config Config) (Base, error) {
	// Parse URL.
	brokerURL, err := url.Parse(config.MQTTAddr)
	if err != nil {
		return nil, errors.NewInternalErrorFromErr(err, "invalid mqtt addr", errors.Details{"was": config.MQTTAddr})
	}
	return &basePortal{
		logger:    logger,
		config:    config,
		brokerURL: brokerURL,
	}, nil
}

// Open the base portal and keep the connection to the MQTT server until the
// given context.Context is done.
func (p *basePortal) Open(ctx context.Context) error {
	// Establish MQTT connection.
	mqttRouter := paho.NewStandardRouter()
	p.router = newRouter(p.logger, mqttRouter)
	conn, err := autopaho.NewConnection(ctx, p.genClientConfig(mqttRouter))
	if err != nil {
		return errors.NewInternalErrorFromErr(err, "create mqtt server connection failed", nil)
	}
	p.publisher = conn
	// Wait until we are done.
	<-ctx.Done()
	// Shutdown MQTT connection.
	disconnectTimeout, cancelDisconnectTimeout := context.WithTimeout(context.Background(), 3*time.Second)
	err = conn.Disconnect(disconnectTimeout)
	cancelDisconnectTimeout()
	if err != nil {
		return errors.NewInternalErrorFromErr(err, "disconnect from mqtt server failed", nil)
	}
	return nil
}

// genClientConfig generates the autopaho.ClientConfig that is ready to launch
// and will use the given paho.Router.
func (p *basePortal) genClientConfig(router paho.Router) autopaho.ClientConfig {
	return autopaho.ClientConfig{
		BrokerUrls: []*url.URL{p.brokerURL},
		KeepAlive:  mqttKeepAlive,
		OnConnectionUp: func(_ *autopaho.ConnectionManager, _ *paho.Connack) {
			p.logger.Info("mqtt server connection established")
		},
		OnConnectError: func(err error) {
			errors.Log(p.logger, errors.Error{
				Code:    errors.ErrCommunication,
				Err:     err,
				Message: "mqtt server connection failed",
			})
		},
		ClientConfig: paho.ClientConfig{
			ClientID: mqttClientID,
			Router:   router,
			OnServerDisconnect: func(disconnect *paho.Disconnect) {
				reason := string(disconnect.ReasonCode)
				if disconnect.Properties != nil {
					reason = disconnect.Properties.ReasonString
				}
				errors.Log(p.logger, errors.Error{
					Code:    errors.ErrCommunication,
					Message: fmt.Sprintf("mqtt server requested disconnect: %s", reason),
				})
			},
			OnClientError: func(err error) {
				errors.Log(p.logger, errors.Error{
					Code:    errors.ErrCommunication,
					Err:     err,
					Message: "mqtt server connection client error",
				})
			},
		},
	}
}

// NewPortal creates a new Portal that can be used to subscribe to topics and
// events.
func (p *basePortal) NewPortal(name string) Portal {
	return &portal{
		logger: p.logger.Named(name),
		router: p.router,
	}
}

// Subscribe to the given Portal for the Topic. The returned Newsletter contains
// an already unmarshalled payload. Messages that fail to unmarshal, are
// dropped. However, the error is logged to Portal.Logger.
func Subscribe[payloadT any](ctx context.Context, portal Portal, topic Topic) *Newsletter[payloadT] {
	rawSub := portal.Subscribe(ctx, topic)
	receiveParsed := make(chan event.Event[payloadT])
	go func() {
		defer close(receiveParsed)
		for e := range rawSub.Receive {
			// Parse payload.
			var payload payloadT
			err := json.Unmarshal(e.Publish.Payload, &payload)
			if err != nil {
				errors.Log(portal.Logger(), errors.NewInternalErrorFromErr(err, "parse payload failed", errors.Details{
					"topic":   e.Publish.Topic,
					"payload": string(e.Publish.Payload),
				}))
				continue
			}
			// Forward
			select {
			case <-ctx.Done():
				return
			case receiveParsed <- event.Event[payloadT]{
				Publish: e.Publish,
				Payload: payload,
			}:
			}
		}
	}()
	return &Newsletter[payloadT]{
		unregisterFn: rawSub.unregisterFn,
		Receive:      receiveParsed,
	}
}

// portal provides a higher-level API for Base that makes it easier to conduct
// tests, etc.
type portal struct {
	logger *zap.Logger
	// router is used for subscribing to MQTT topics via Subscribe.
	router *router
	// publisher is used for publishing MQTT messages via Publish.
	publisher publisher
}

// Subscribe for the given Topic using the portal's router.
func (p *portal) Subscribe(ctx context.Context, topic Topic) *Newsletter[any] {
	subLifetime, cancelSub := context.WithCancel(ctx)
	forward := make(chan event.Event[any])
	p.router.subscribe(subLifetime, topic, forward)
	return &Newsletter[any]{
		unregisterFn: cancelSub,
		Receive:      forward,
	}
}

// Publish the given payload to the Topic using the portal's router.
func (p *portal) Publish(ctx context.Context, topic Topic, payload interface{}) {
	// Marshal payload.
	payloadRaw, err := json.Marshal(payload)
	if err != nil {
		errors.Log(p.logger, errors.NewInternalErrorFromErr(err, "marshal payload for publishing", errors.Details{
			"topic": topic,
		}))
		return
	}
	// Publish.
	_, err = p.publisher.Publish(ctx, &paho.Publish{
		Topic:   string(topic),
		Payload: payloadRaw,
	})
	if err != nil {
		errors.Log(p.logger, errors.NewInternalErrorFromErr(err, "publish message failed", errors.Details{
			"topic":   topic,
			"payload": payload,
		}))
		return
	}
}

// Logger returns the portal's logger which is needed because of missing
// features regarding generics in Go 1.18.
func (p *portal) Logger() *zap.Logger {
	return p.logger
}
