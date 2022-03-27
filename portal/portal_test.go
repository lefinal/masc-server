package portal

import (
	"context"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/event"
	"github.com/eclipse/paho.golang/paho"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"runtime"
	"sync"
	"testing"
)

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

// portalStub mocks Portal.
type portalStub struct {
	mock.Mock
	// logger is the logger to use when calling Logger. If not set, this will always
	// default to a nop logger.
	logger *zap.Logger
}

func (s *portalStub) Subscribe(ctx context.Context, topic Topic) *Newsletter[any] {
	var newsletter *Newsletter[any]
	newsletter, _ = s.Called(ctx, topic).Get(0).(*Newsletter[any])
	return newsletter
}

func (s *portalStub) Publish(ctx context.Context, topic Topic, payload interface{}) {
	s.Called(ctx, topic, payload)
}

func (s *portalStub) Logger() *zap.Logger {
	if s.logger == nil {
		return zap.New(zapcore.NewNopCore())
	}
	return s.logger
}

// subscribeSuite tests Subscribe.
type subscribeSuite struct {
	suite.Suite
	portal *portalStub
}

func (suite *subscribeSuite) SetupTest() {
	suite.portal = &portalStub{}
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
}

func TestSubscribe(t *testing.T) {
	suite.Run(t, new(subscribeSuite))
}

func TestPortal_Subscribe(t *testing.T) {
	var wg sync.WaitGroup
	mqttRouter := &mqttRouterStub{}
	portal := &portal{
		logger:    zap.New(zapcore.NewNopCore()),
		router:    newRouter(zap.New(zapcore.NewNopCore()), mqttRouter),
		publisher: nil,
	}
	handlerToRun := make(chan paho.MessageHandler)
	timeout, cancel := context.WithTimeout(context.Background(), timeout)
	mqttRouter.On("RegisterHandler", "cats", mock.Anything).Run(func(args mock.Arguments) {
		select {
		case <-timeout.Done():
		case handlerToRun <- args.Get(1).(paho.MessageHandler):
		}
	})
	wg.Add(1)
	mqttRouter.On("UnregisterHandler", "cats").Run(func(_ mock.Arguments) {
		// Await unregister call.
		wg.Done()
	})
	defer mqttRouter.AssertExpectations(t)
	toPublish := &paho.Publish{}
	// Await handler for testing message forwarding.
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-timeout.Done():
		case handler := <-handlerToRun:
			handler(toPublish)
		}
	}()
	// Subscribe, await handler call and unsubscribe.
	wg.Add(1)
	go func() {
		defer wg.Done()
		newsletter := portal.Subscribe(timeout, "cats")
		select {
		case <-timeout.Done():
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
}

// publisherStub mocks publisher.
type publisherStub struct {
	mock.Mock
}

func (s *publisherStub) Publish(ctx context.Context, publish *paho.Publish) (*paho.PublishResponse, error) {
	args := s.Called(ctx, publisherStub{})
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
