package portal

import (
	"context"
	"github.com/eclipse/paho.golang/paho"
	"github.com/lefinal/masc-server/errors"
	"github.com/lefinal/masc-server/event"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"runtime"
	"sync"
	"testing"
	"time"
)

var timeout = 3 * time.Second

func TestNewsletter_Unsubscribe(t *testing.T) {
	var wg sync.WaitGroup
	timeout, cancel := context.WithTimeout(context.Background(), timeout)
	n := &Newsletter[any]{
		unregisterFn: cancel,
	}
	wg.Add(1)
	go func() {
		defer wg.Done()
		n.Unsubscribe()
	}()
	<-timeout.Done()
	assert.Equal(t, context.Canceled, timeout.Err(), "should not time out")
}

// subscribeSuite tests Subscribe.
type subscribeSuite struct {
	suite.Suite
	portal *Stub
}

func (suite *subscribeSuite) SetupTest() {
	suite.portal = &Stub{}
}

// TestParse assures that parsing into the wanted type does work as expected.
func (suite *subscribeSuite) TestParse() {
	type myStruct struct {
		A int  `json:"a"`
		B bool `json:"b"`
	}
	var wg sync.WaitGroup
	fromPortal := make(chan event.Event[any])
	newsletterFromPortal := &Newsletter[any]{
		unregisterFn: func() {
			suite.Fail("unsubscribed", "should not unsubscribe")
		},
		Receive: fromPortal,
	}
	suite.portal.On("Subscribe", mock.Anything, Topic("cats")).Return(newsletterFromPortal)
	defer suite.portal.AssertExpectations(suite.T())
	// Subscribe.
	timeout, cancel := context.WithTimeout(context.Background(), timeout)
	newsletter := Subscribe[myStruct](timeout, suite.portal, "cats")
	// Publish raw result.
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-timeout.Done():
			suite.Fail("timeout", "published result should be picked up within timeout")
			return
		case fromPortal <- event.Event[any]{
			Publish: &paho.Publish{
				Payload: []byte(`{"a": 123, "b":  true}`),
			},
		}:
		}
	}()
	// Await result.
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-timeout.Done():
			suite.Fail("timeout", "newsletter should be received within timeout")
			return
		case got := <-newsletter.Receive:
			suite.Equal(myStruct{
				A: 123,
				B: true,
			}, got.Payload, "should match expected payload")
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

// TestAutoClose makes sure that the Newsletter.Receive channel from the
// returned Newsletter is closed when the subscription using Portal.Subscribe is
// done.
func (suite *subscribeSuite) TestAutoClose() {
	var wg sync.WaitGroup
	receiveFromPortal := make(chan event.Event[any])
	suite.portal.On("Subscribe", mock.Anything, Topic("cats")).Return(&Newsletter[any]{
		unregisterFn: func() {
			suite.Fail("unregistered", "should not unregister")
		},
		Receive: receiveFromPortal,
	})
	defer suite.portal.AssertExpectations(suite.T())
	timeout, cancel := context.WithTimeout(context.Background(), timeout)
	// Subscribe.
	newsletter := Subscribe[any](timeout, suite.portal, "cats")
	// Await closed.
	wg.Add(1)
	go func() {
		defer wg.Done()
		for {
			select {
			case <-timeout.Done():
				suite.Fail("timeout", "newsletter should be received within timeout")
				return
			case _, more := <-newsletter.Receive:
				suite.False(more, "should read no values from channel")
				return
			}
		}
	}()
	// Close.
	wg.Add(1)
	go func() {
		defer wg.Done()
		runtime.Gosched()
		close(receiveFromPortal)
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

func TestSubscribe(t *testing.T) {
	suite.Run(t, new(subscribeSuite))
}

func TestPortal_Subscribe(t *testing.T) {
	var wg sync.WaitGroup
	mqttKiosk := &mqttKioskStub{}
	mqttInboundRouter := &mqttInboundRouterStub{}
	portal := &portal{
		logger: zap.New(zapcore.NewNopCore()),
		router: newPortalGateway(zap.New(zapcore.NewNopCore()), &portalGatewayMQTTBridge{
			logger:        zap.New(zapcore.NewNopCore()),
			kiosk:         mqttKiosk,
			inboundRouter: mqttInboundRouter,
		}),
		publisher: nil,
	}
	handlerToRun := make(chan paho.MessageHandler)
	timeout, cancel := context.WithTimeout(context.Background(), timeout)
	mqttKiosk.On("Subscribe", mock.Anything, mock.Anything).Return(nil, nil)
	wg.Add(1) // Wait until unsubscribed.
	mqttKiosk.On("Unsubscribe", mock.Anything, mock.Anything).Return(nil, nil).
		Run(func(_ mock.Arguments) {
			// Await unregister call.
			wg.Done()
		}).Once()
	defer mqttKiosk.AssertExpectations(t)
	wg.Add(1)
	mqttInboundRouter.On("RegisterHandler", "cats", mock.Anything).Run(func(args mock.Arguments) {
		go func() {
			defer wg.Done()
			select {
			case <-timeout.Done():
				assert.Fail(t, "timeout", "registered handler should be picked up within timeout")
				return
			case handlerToRun <- args.Get(1).(paho.MessageHandler):
			}
		}()
	})
	mqttInboundRouter.On("UnregisterHandler", "cats")
	defer mqttInboundRouter.AssertExpectations(t)
	toPublish := &paho.Publish{}
	// We need this subscribed blocker because handler registration happens before
	// the actual subscription is added to the handler. Therefore, we call the
	// handler too early when the subscription is not even introduced to the handler
	// yet.
	subscribedBlocker := make(chan struct{})
	// Await handler for testing message forwarding.
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-timeout.Done():
			assert.Fail(t, "subscribed blocker should unblock within timeout")
			return
		case <-subscribedBlocker:
		}
		select {
		case <-timeout.Done():
			assert.Fail(t, "handler to run should be received within timeout")
			return
		case handler := <-handlerToRun:
			handler(toPublish)
		}
	}()
	// Subscribe, await handler call and unsubscribe.
	wg.Add(1)
	go func() {
		defer wg.Done()
		newsletter := portal.Subscribe(timeout, "cats")
		close(subscribedBlocker)
		select {
		case <-timeout.Done():
			assert.Fail(t, "newsletter should be received within timeout")
			return
		case got := <-newsletter.Receive:
			assert.Equal(t, toPublish, got.Publish, "should match expected publish")
		}
		// Unsubscribe.
		newsletter.Unsubscribe()
	}()
	// Await all.
	go func() {
		wg.Wait()
		cancel()
	}()
	<-timeout.Done()
	assert.Equal(t, context.Canceled, timeout.Err(), "should not time out")
	wg.Wait()
}

// publisherStub mocks publisher.
type publisherStub struct {
	mock.Mock
}

func (s *publisherStub) Publish(ctx context.Context, publish *paho.Publish) (*paho.PublishResponse, error) {
	args := s.Called(ctx, publish)
	var res *paho.PublishResponse
	res, _ = args.Get(0).(*paho.PublishResponse)
	return res, args.Error(1)
}

// portalPublishSuite tests portal.Publish.
type portalPublishSuite struct {
	suite.Suite
	publisher *publisherStub
	portal    *portal
}

func (suite *portalPublishSuite) SetupTest() {
	suite.publisher = &publisherStub{}
	suite.portal = &portal{
		logger:    zap.New(zapcore.NewNopCore()),
		publisher: suite.publisher,
	}
}

func (suite *portalPublishSuite) TestMarshalFail() {
	var wg sync.WaitGroup
	// Create self reference.
	type myStruct struct {
		Ref *myStruct `json:"ref"`
	}
	selfRef := myStruct{}
	selfRef.Ref = &selfRef
	defer suite.publisher.AssertExpectations(suite.T())
	timeout, cancel := context.WithTimeout(context.Background(), timeout)
	// Publish.
	wg.Add(1)
	go func() {
		defer wg.Done()
		suite.NotPanics(func() {
			suite.portal.Publish(timeout, "cats", selfRef)
		})
	}()
	// Await done.
	go func() {
		wg.Wait()
		cancel()
	}()
	<-timeout.Done()
	suite.Equal(context.Canceled, timeout.Err(), "should not time out")
}

func (suite *portalPublishSuite) TestPublishFail() {
	var wg sync.WaitGroup
	suite.publisher.On("Publish", mock.Anything, mock.Anything).
		Return(nil, errors.NewInternalError("sad life", nil))
	defer suite.publisher.AssertExpectations(suite.T())
	timeout, cancel := context.WithTimeout(context.Background(), timeout)
	// Publish.
	wg.Add(1)
	go func() {
		defer wg.Done()
		suite.NotPanics(func() {
			suite.portal.Publish(timeout, "cats", 123)
		})
	}()
	// Await done.
	go func() {
		wg.Wait()
		cancel()
	}()
	<-timeout.Done()
	suite.Equal(context.Canceled, timeout.Err(), "should not time out")
}

func (suite *portalPublishSuite) TestOK() {
	var wg sync.WaitGroup
	suite.publisher.On("Publish", mock.Anything, mock.Anything).
		Return(&paho.PublishResponse{}, nil)
	defer suite.publisher.AssertExpectations(suite.T())
	timeout, cancel := context.WithTimeout(context.Background(), timeout)
	// Publish.
	wg.Add(1)
	go func() {
		defer wg.Done()
		suite.NotPanics(func() {
			suite.portal.Publish(timeout, "cats", 123)
		})
	}()
	// Await done.
	go func() {
		wg.Wait()
		cancel()
	}()
	<-timeout.Done()
	suite.Equal(context.Canceled, timeout.Err(), "should not time out")
}

func TestPortal_Publish(t *testing.T) {
	suite.Run(t, new(portalPublishSuite))
}
