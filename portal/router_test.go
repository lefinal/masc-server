package portal

import (
	"context"
	"github.com/eclipse/paho.golang/paho"
	"github.com/lefinal/masc-server/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"runtime"
	"sync"
	"testing"
)

// mqttKioskStub mocks mqttKiosk.
type mqttKioskStub struct {
	mock.Mock
}

func (stub *mqttKioskStub) Subscribe(ctx context.Context, s *paho.Subscribe) (*paho.Suback, error) {
	args := stub.Called(ctx, s)
	var subAck *paho.Suback
	subAck, _ = args.Get(0).(*paho.Suback)
	return subAck, args.Error(1)
}

func (stub *mqttKioskStub) Unsubscribe(ctx context.Context, u *paho.Unsubscribe) (*paho.Unsuback, error) {
	args := stub.Called(ctx, u)
	var unSubAck *paho.Unsuback
	unSubAck, _ = args.Get(0).(*paho.Unsuback)
	return unSubAck, args.Error(1)
}

// mqttInboundRouterStub mocks mqttInboundRouter.
type mqttInboundRouterStub struct {
	mock.Mock
}

func (s *mqttInboundRouterStub) RegisterHandler(topic string, handler paho.MessageHandler) {
	s.Called(topic, handler)
}

func (s *mqttInboundRouterStub) UnregisterHandler(topic string) {
	s.Called(topic)
}

func TestRouter_new(t *testing.T) {
	router := newPortalGateway(zap.New(zapcore.NewNopCore()), &portalGatewayMQTTBridge{
		logger:        zap.New(zapcore.NewNopCore()),
		kiosk:         &mqttKioskStub{},
		inboundRouter: &mqttInboundRouterStub{},
	})
	assert.NotNil(t, router.registeredHandlers, "should have initialized handlers")
}

// routerSubscribe tests portalGateway.subscribe.
type routerSubscribe struct {
	suite.Suite
	router            *portalGateway
	mqttKiosk         *mqttKioskStub
	mqttInboundRouter *mqttInboundRouterStub
}

func (suite *routerSubscribe) SetupTest() {
	suite.mqttKiosk = &mqttKioskStub{}
	suite.mqttInboundRouter = &mqttInboundRouterStub{}
	suite.router = newPortalGateway(zap.New(zapcore.NewNopCore()), &portalGatewayMQTTBridge{
		logger:        zap.New(zapcore.NewNopCore()),
		kiosk:         suite.mqttKiosk,
		inboundRouter: suite.mqttInboundRouter,
	})
}

// TestNoSubs expects the router to create a handler and register in the MQTT
// router.
func (suite *routerSubscribe) TestNoSubs() {
	unregisterTimeout, cancelUnregisterTimeout := context.WithTimeout(context.Background(), timeout)
	suite.mqttKiosk.On("Subscribe", mock.Anything, mock.Anything).Return(nil, nil)
	suite.mqttKiosk.On("Unsubscribe", mock.Anything, mock.Anything).Return(nil, nil).Run(func(_ mock.Arguments) {
		cancelUnregisterTimeout()
	}).Once()
	defer suite.mqttKiosk.AssertExpectations(suite.T())
	suite.mqttInboundRouter.On("RegisterHandler", "cats", mock.Anything).Once()
	suite.mqttInboundRouter.On("UnregisterHandler", "cats").Once()
	defer suite.mqttInboundRouter.AssertExpectations(suite.T())
	// Subscribe.
	lifetime, cancel := context.WithCancel(context.Background())
	_ = suite.router.subscribe(lifetime, "cats")
	// Check if everything ok.
	suite.Require().Contains(suite.router.registeredHandlers, Topic("cats"), "should have created handler for the topic")
	handler := suite.router.registeredHandlers["cats"]
	suite.Len(handler.subscriptions, 1, "should have added subscription")
	// Cancel subscription and wait until unregistered.
	cancel()
	<-unregisterTimeout.Done()
	suite.Equal(context.Canceled, unregisterTimeout.Err(), "should not time out")
}

// TestHandlerAlreadyRegistered assures that the handler is not registered again
// for the same topic if already registered.
func (suite *routerSubscribe) TestHandlerAlreadyRegistered() {
	var wg sync.WaitGroup
	initialSubscription := &subscription{
		lifetime: nil,
		forward:  nil,
	}
	suite.router.registeredHandlers["cats"] = &registeredHandler{
		subscriptions: map[*subscription]struct{}{
			initialSubscription: {},
		},
	}
	suite.mqttInboundRouter.On("RegisterHandler").Run(func(_ mock.Arguments) {
		suite.Fail("should not call register")
	})
	// Subscribe another one.
	timeout, cancel := context.WithTimeout(context.Background(), timeout)
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer cancel()
		suite.router.subscribe(timeout, "cats")
	}()
	<-timeout.Done()
	suite.Equal(context.Canceled, timeout.Err(), "should not time out")
	wg.Wait()
}

// TestLifetimeDone assures that after the passed context to subscribe is done,
// unsubscribe is called.
func (suite *routerSubscribe) TestLifetimeDone() {
	timeout, cancel := context.WithTimeout(context.Background(), timeout)
	suite.mqttKiosk.On("Subscribe", mock.Anything, mock.Anything).Return(nil, nil)
	suite.mqttKiosk.On("Unsubscribe", mock.Anything, mock.Anything).Return(nil, nil).
		Run(func(_ mock.Arguments) {
			cancel()
		})
	defer suite.mqttKiosk.AssertExpectations(suite.T())
	suite.mqttInboundRouter.On("RegisterHandler", "cats", mock.Anything)
	suite.mqttInboundRouter.On("UnregisterHandler", "cats")
	defer suite.mqttInboundRouter.AssertExpectations(suite.T())
	// Subscribe.
	lifetime, cancelLifetime := context.WithCancel(context.Background())
	suite.router.subscribe(lifetime, "cats")
	cancelLifetime()
	// Expect unregister to have been called.
	<-timeout.Done()
	suite.Equal(context.Canceled, timeout.Err(), "should not time out")
}

func TestRouter_subscribe(t *testing.T) {
	suite.Run(t, new(routerSubscribe))
}

// routerUnsubscribe tests portalGateway.unsubscribe.
type routerUnsubscribe struct {
	suite.Suite
	router            *portalGateway
	mqttKiosk         *mqttKioskStub
	mqttInboundRouter *mqttInboundRouterStub
}

func (suite *routerUnsubscribe) SetupTest() {
	suite.mqttKiosk = &mqttKioskStub{}
	suite.mqttInboundRouter = &mqttInboundRouterStub{}
	suite.router = newPortalGateway(zap.New(zapcore.NewNopCore()), &portalGatewayMQTTBridge{
		logger:        zap.New(zapcore.NewNopCore()),
		kiosk:         suite.mqttKiosk,
		inboundRouter: suite.mqttInboundRouter,
	})
}

func (suite *routerUnsubscribe) TestUnknownTopic() {
	defer suite.mqttInboundRouter.AssertExpectations(suite.T())
	suite.NotPanics(func() {
		suite.router.unsubscribe("unknown", nil)
	}, "should not fail")
}

func (suite *routerUnsubscribe) TestUnsubscribeWithSubscriptionsLeft() {
	defer suite.mqttInboundRouter.AssertExpectations(suite.T())
	subToUnsubscribe := &subscription{
		lifetime: nil,
		forward:  nil,
	}
	suite.router.registeredHandlers["cats"] = &registeredHandler{
		subscriptions: map[*subscription]struct{}{
			&subscription{}:  {},
			&subscription{}:  {},
			subToUnsubscribe: {},
			&subscription{}:  {},
		},
	}
	// Unsubscribe.
	suite.router.unsubscribe("cats", subToUnsubscribe)
}

func TestRouter_unsubscribe(t *testing.T) {
	suite.Run(t, new(routerUnsubscribe))
}

// registeredHandlerHandlerSuite tests registeredHandler.Handler.
type registeredHandlerHandlerSuite struct {
	suite.Suite
	rh *registeredHandler
}

func (suite *registeredHandlerHandlerSuite) SetupTest() {
	suite.rh = &registeredHandler{subscriptions: make(map[*subscription]struct{})}
}

// TestNoneSubscribed asserts that even if it should not be possible to exist
// without subscriptions, we still are not crashing.
func (suite *registeredHandlerHandlerSuite) TestNoneSubscribed() {
	suite.NotPanics(func() {
		suite.rh.Handler()(&paho.Publish{})
	}, "should not fail")
}

func (suite *registeredHandlerHandlerSuite) TestSingleSub() {
	var wg sync.WaitGroup
	timeout, cancel := context.WithTimeout(context.Background(), timeout)
	subReceive := make(chan event.Event[any])
	sub := &subscription{
		lifetime: timeout,
		forward:  subReceive,
	}
	suite.rh.subscriptions[sub] = struct{}{}
	// Handle.
	wg.Add(1)
	go func() {
		defer wg.Done()
		suite.rh.Handler()(&paho.Publish{})
	}()
	// Await result.
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-timeout.Done():
			suite.Fail("timeout", "should receive from sub within timeout")
			return
		case <-subReceive:
		}
	}()
	// Await all done.
	go func() {
		wg.Wait()
		cancel()
	}()
	<-timeout.Done()
	suite.Equal(context.Canceled, timeout.Err(), "should not time out")
	wg.Wait()
}

func (suite *registeredHandlerHandlerSuite) TestMultipleSubs() {
	subCount := 32
	var wg sync.WaitGroup
	timeout, cancel := context.WithTimeout(context.Background(), timeout)
	// We use the same channel for all subscriptions.
	subsReceive := make(chan event.Event[any])
	for i := 0; i < subCount; i++ {
		suite.rh.subscriptions[&subscription{
			lifetime: timeout,
			forward:  subsReceive,
		}] = struct{}{}
	}
	// Handle.
	wg.Add(1)
	go func() {
		defer wg.Done()
		suite.rh.Handler()(&paho.Publish{})
	}()
	// Await results.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i := 0; i < subCount; i++ {
			select {
			case <-timeout.Done():
				suite.Fail("timeout", "should receive all within timeout")
				return
			case <-subsReceive:
			}
		}
	}()
	// Await all.
	go func() {
		wg.Wait()
		cancel()
	}()
	<-timeout.Done()
	suite.Equal(context.Canceled, timeout.Err(), "should not time out")
	wg.Wait()
}

// TestUnsubscribeDuringForward assures that forwarding is cancelled if the
// subscribes unsubscribes during forwarding and therefore never picks up the
// message.
func (suite *registeredHandlerHandlerSuite) TestUnsubscribeDuringForward() {
	var wg sync.WaitGroup
	subLifetime, cancelSub := context.WithCancel(context.Background())
	suite.rh.subscriptions[&subscription{
		lifetime: subLifetime,
		forward:  make(chan event.Event[any]),
	}] = struct{}{}
	// Handle.
	wg.Add(1)
	go func() {
		defer wg.Done()
		suite.rh.Handler()(&paho.Publish{})
	}()
	// This is a bit tricky, but by yielding we hope to raise chances of cancelling
	// the subscription context when we already are receiving.
	wg.Add(1)
	go func() {
		defer wg.Done()
		runtime.Gosched()
		cancelSub()
	}()
	timeout, cancel := context.WithTimeout(context.Background(), timeout)
	go func() {
		wg.Wait()
		cancel()
	}()
	<-timeout.Done()
	suite.Equal(context.Canceled, timeout.Err(), "should not time out")
	wg.Wait()
}

func TestRegisteredHandler_Handler(t *testing.T) {
	suite.Run(t, new(registeredHandlerHandlerSuite))
}
