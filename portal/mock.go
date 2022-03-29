package portal

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/eclipse/paho.golang/paho"
	"github.com/lefinal/masc-server/event"
	"github.com/stretchr/testify/mock"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Stub mocks Portal.
type Stub struct {
	mock.Mock
	// logger is the logger to use when calling Logger. If not set, this will always
	// default to a nop logger.
	logger *zap.Logger
}

// Subscribe to the given Topic. Calls mock.Mock.
func (s *Stub) Subscribe(ctx context.Context, topic Topic) *Newsletter[any] {
	var newsletter *Newsletter[any]
	newsletter = s.Called(ctx, topic).Get(0).(*Newsletter[any])
	return newsletter
}

// Publish the given serializable payload to a topic. Calls mock.Mock.
func (s *Stub) Publish(ctx context.Context, topic Topic, payload interface{}) {
	s.Called(ctx, topic, payload)
}

// Logger returns the logger set for the Stub. If not set, a nop-logger will be
// returned.
func (s *Stub) Logger() *zap.Logger {
	if s.logger == nil {
		return zap.New(zapcore.NewNopCore())
	}
	return s.logger
}

// NewSelfClosingMockNewsletter returns a Newsletter that closes itself after
// the given context.Context is done. Of course manually unsubscribing is
// supported, too.
func NewSelfClosingMockNewsletter(ctx context.Context) *Newsletter[any] {
	lifetime, cancel := context.WithCancel(ctx)
	receive := make(chan event.Event[any])
	go func() {
		<-lifetime.Done()
		close(receive)
	}()
	return &Newsletter[any]{
		unregisterFn: cancel,
		Receive:      receive,
	}
}

// NewSelfClosingReceivingMockNewsletter returns a Newsletter that closes
// Newsletter.Receive after the given context.Context is done or the given
// forward channel is closed. While active, it receives from the given channel,
// marshals the event-payload into the publish-payload and forwards the result
// to the returned Newsletter.Receive.
func NewSelfClosingReceivingMockNewsletter(ctx context.Context, forward <-chan event.Event[any]) *Newsletter[any] {
	lifetime, cancel := context.WithCancel(ctx)
	receive := make(chan event.Event[any])
	go func() {
		defer close(receive)
		for {
			select {
			case <-lifetime.Done():
				return
			case e, more := <-forward:
				if !more {
					return
				}
				// Marshal.
				raw, err := json.Marshal(e.Payload)
				if err != nil {
					panic(fmt.Sprintf("marshal payload: %v", err))
				}
				if e.Publish == nil {
					e.Publish = &paho.Publish{}
				}
				e.Publish.Payload = raw
				e.Payload = event.Event[any]{}
				// Forward.
				select {
				case <-lifetime.Done():
					return
				case receive <- e:
				}
			}
		}
	}()
	return &Newsletter[any]{
		unregisterFn: cancel,
		Receive:      receive,
	}
}
