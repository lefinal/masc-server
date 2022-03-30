package devicesvc

import (
	"context"
	"github.com/gobuffalo/nulls"
	"github.com/lefinal/masc-server/event"
	"github.com/lefinal/masc-server/portal"
	"github.com/lefinal/masc-server/store"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"go.uber.org/atomic"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"sync"
	"testing"
	"time"
)

const timeout = 3 * time.Second

// storeStub mocks Store.
type storeStub struct {
	mock.Mock
}

func (stub *storeStub) RegisterDevice(ctx context.Context, deviceID string, deviceType string) (store.Device, bool, error) {
	args := stub.Called(ctx, deviceID, deviceType)
	return args.Get(0).(store.Device), args.Bool(1), args.Error(2)
}

func (stub *storeStub) UpdateDeviceLastSeen(ctx context.Context, deviceID string) error {
	args := stub.Called(ctx, deviceID)
	return args.Error(0)
}

func TestNewDeviceService(t *testing.T) {
	logger := zap.New(zapcore.NewNopCore())
	portalStub := &portal.Stub{}
	storeStub := &storeStub{}
	s := NewDeviceService(logger, portalStub, storeStub).(*deviceService)
	require.NotNil(t, s, "should not be nil")
	assert.Equal(t, logger, s.logger, "should set correct logger")
	assert.Equal(t, portalStub, s.portal, "should set correct portal")
	assert.Equal(t, storeStub, s.store, "should set correct store")
}

// deviceServiceSuite tests deviceService
type deviceServiceSuite struct {
	suite.Suite
	portalStub *portal.Stub
	storeStub  *storeStub
	service    *deviceService
}

func (suite *deviceServiceSuite) SetupTest() {
	suite.portalStub = &portal.Stub{}
	suite.storeStub = &storeStub{}
	suite.service = NewDeviceService(zap.New(zapcore.NewNopCore()), suite.portalStub, suite.storeStub).(*deviceService)
}

// TestSubscribeReportOnRun assures that we subscribe to report topic.
func (suite *deviceServiceSuite) TestSubscribeReportOnRun() {
	var wg sync.WaitGroup
	timeout, cancel := context.WithTimeout(context.Background(), timeout)
	suite.portalStub.On("Subscribe", mock.Anything, topicReport).
		Run(func(_ mock.Arguments) { cancel() }).
		Return(portal.NewSelfClosingMockNewsletter(timeout)).Once()
	suite.portalStub.On("Subscribe", mock.Anything, mock.Anything).
		Return(portal.NewSelfClosingMockNewsletter(timeout))
	suite.portalStub.On("Publish", mock.Anything, mock.Anything, mock.Anything)
	defer suite.portalStub.AssertExpectations(suite.T())
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := suite.service.Run(timeout)
		suite.Nil(err, "should not fail")
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

// TestSubscribeDeviceOnlineOnRun assures that we subscribe to device online
// topic.
func (suite *deviceServiceSuite) TestSubscribeDeviceOnlineOnRun() {
	var wg sync.WaitGroup
	timeout, cancel := context.WithTimeout(context.Background(), timeout)
	suite.portalStub.On("Subscribe", mock.Anything, topicDeviceOnline).
		Run(func(_ mock.Arguments) { cancel() }).
		Return(portal.NewSelfClosingMockNewsletter(timeout)).Once()
	suite.portalStub.On("Subscribe", mock.Anything, mock.Anything).
		Return(portal.NewSelfClosingMockNewsletter(timeout))
	suite.portalStub.On("Publish", mock.Anything, mock.Anything, mock.Anything)
	defer suite.portalStub.AssertExpectations(suite.T())
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := suite.service.Run(timeout)
		suite.Nil(err, "should not fail")
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

// TestSubscribeDeviceOfflineOnRun assures that we subscribe to device offline
// topic.
func (suite *deviceServiceSuite) TestSubscribeDeviceOfflineOnRun() {
	var wg sync.WaitGroup
	timeout, cancel := context.WithTimeout(context.Background(), timeout)
	suite.portalStub.On("Subscribe", mock.Anything, topicDeviceOffline).
		Run(func(_ mock.Arguments) { cancel() }).
		Return(portal.NewSelfClosingMockNewsletter(timeout)).Once()
	suite.portalStub.On("Subscribe", mock.Anything, mock.Anything).
		Return(portal.NewSelfClosingMockNewsletter(timeout))
	suite.portalStub.On("Publish", mock.Anything, mock.Anything, mock.Anything)
	defer suite.portalStub.AssertExpectations(suite.T())
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := suite.service.Run(timeout)
		suite.Nil(err, "should not fail")
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

// TestRequestDeviceOnlineReportAfterRunSetup assures that we request a
// device-online-report after having made all subscriptions.
func (suite *deviceServiceSuite) TestRequestDeviceOnlineReportAfterRunSetup() {
	var wg sync.WaitGroup
	timeout, cancel := context.WithTimeout(context.Background(), timeout)
	// publishCall needs to be false when a subscribe-call is made in order to
	// assure that we are subscribed to the report topic first and are recording all
	// online devices from the online-devices-topi.
	publishCallMade := atomic.NewBool(false)
	suite.portalStub.On("Subscribe", mock.Anything, mock.Anything).
		Run(func(_ mock.Arguments) {
			suite.False(publishCallMade.Load(), "should finish subscriptions before publishing")
		}).
		Return(portal.NewSelfClosingMockNewsletter(timeout))
	suite.portalStub.On("Publish", mock.Anything, topicReport, event.EmptyEvent{}).
		Run(func(_ mock.Arguments) {
			publishCallMade.Store(true)
			cancel()
		}).Once()
	defer suite.portalStub.AssertExpectations(suite.T())
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := suite.service.Run(timeout)
		suite.Nil(err, "should not fail")
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

// TestHandleDeviceOnlineEvent tests handling of an event.DeviceOnlineEvent.
func (suite *deviceServiceSuite) TestHandleDeviceOnlineEvent() {
	var wg sync.WaitGroup
	var expectedCalls sync.WaitGroup
	timeout, cancel := context.WithTimeout(context.Background(), timeout)
	runCtx, cancelRun := context.WithCancel(timeout)
	// Setup.
	deviceOnlineReceive := make(chan event.Event[any])
	suite.portalStub.On("Subscribe", mock.Anything, topicDeviceOnline).
		Return(portal.NewSelfClosingReceivingMockNewsletter(runCtx, deviceOnlineReceive)).Once()
	suite.portalStub.On("Subscribe", mock.Anything, mock.Anything).
		Return(portal.NewSelfClosingMockNewsletter(runCtx))
	expectedCalls.Add(1)
	suite.storeStub.On("RegisterDevice", mock.Anything, "<device-id>", "<device-type>").
		Return(store.Device{
			ID:       "<device-id>",
			Type:     "<device-type>",
			LastSeen: time.UnixMicro(42),
			Name:     nulls.NewString("<device-name>"),
			Config:   nulls.ByteSlice{},
		}, false, nil).Run(func(_ mock.Arguments) { expectedCalls.Done() }).Once()
	expectedCalls.Add(1)
	suite.portalStub.On("Publish", mock.Anything, topicDeviceOnlineDetailed, event.DeviceOnlineDetailedEvent{
		DeviceID:   "<device-id>",
		DeviceType: "<device-type>",
		LastSeen:   time.UnixMicro(42),
		Name:       nulls.NewString("<device-name>"),
	}).Run(func(_ mock.Arguments) { expectedCalls.Done() }).Once()
	suite.portalStub.On("Publish", mock.Anything, mock.Anything, mock.Anything).Once()
	defer suite.storeStub.AssertExpectations(suite.T())
	defer suite.portalStub.AssertExpectations(suite.T())
	// Handle.
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := suite.service.Run(runCtx)
		suite.Nil(err, "should not fail")
	}()
	// Send online report.
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-timeout.Done():
			suite.Fail("timeout", "should have picked up event within timeout")
			return
		case deviceOnlineReceive <- event.Event[any]{
			Payload: event.DeviceOnlineEvent{
				DeviceID:   "<device-id>",
				DeviceType: "<device-type>",
			},
		}:
		}
	}()
	// Await all.
	go func() {
		expectedCalls.Wait()
		cancelRun()
		wg.Wait()
		cancel()
	}()
	<-timeout.Done()
	suite.Equal(context.Canceled, timeout.Err(), "should not time out")
}

// TestHandleDeviceOfflineEvent assures that an event.DeviceOfflineEvent is
// handled correctly.
func (suite *deviceServiceSuite) TestHandleDeviceOfflineEvent() {
	var wg sync.WaitGroup
	var expectedCalls sync.WaitGroup
	timeout, cancel := context.WithTimeout(context.Background(), timeout)
	runCtx, cancelRun := context.WithCancel(timeout)
	// Setup.
	deviceOfflineReceive := make(chan event.Event[any])
	suite.portalStub.On("Subscribe", mock.Anything, topicDeviceOffline).
		Return(portal.NewSelfClosingReceivingMockNewsletter(runCtx, deviceOfflineReceive)).Once()
	suite.portalStub.On("Subscribe", mock.Anything, mock.Anything).
		Return(portal.NewSelfClosingMockNewsletter(runCtx))
	suite.portalStub.On("Publish", mock.Anything, mock.Anything, mock.Anything).Once()
	expectedCalls.Add(1)
	suite.storeStub.On("UpdateDeviceLastSeen", mock.Anything, "<device-id>").
		Return(nil).Run(func(_ mock.Arguments) { expectedCalls.Done() }).Once()
	defer suite.storeStub.AssertExpectations(suite.T())
	defer suite.portalStub.AssertExpectations(suite.T())
	// Handle.
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := suite.service.Run(runCtx)
		suite.Nil(err, "should not fail")
	}()
	// Send online report.
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-timeout.Done():
			suite.Fail("timeout", "should have picked up event within timeout")
			return
		case deviceOfflineReceive <- event.Event[any]{
			Payload: event.DeviceOfflineEvent{
				DeviceID: "<device-id>",
			},
		}:
		}
	}()
	// Await all.
	go func() {
		expectedCalls.Wait()
		cancelRun()
		wg.Wait()
		cancel()
	}()
	<-timeout.Done()
	suite.Equal(context.Canceled, timeout.Err(), "should not time out")
}

// TestHandleReportOnlineEvent assures that a report-online event is handled
// correctly.
func (suite *deviceServiceSuite) TestHandleReportOnlineEvent() {
	var wg sync.WaitGroup
	var expectedCalls sync.WaitGroup
	timeout, cancel := context.WithTimeout(context.Background(), timeout)
	runCtx, cancelRun := context.WithCancel(timeout)
	// Setup.
	reportReceive := make(chan event.Event[any])
	suite.portalStub.On("Subscribe", mock.Anything, topicReport).
		Return(portal.NewSelfClosingReceivingMockNewsletter(runCtx, reportReceive)).Once()
	suite.portalStub.On("Subscribe", mock.Anything, mock.Anything).
		Return(portal.NewSelfClosingMockNewsletter(runCtx))
	expectedCalls.Add(1)
	suite.portalStub.On("Publish", mock.Anything, topicDeviceOnline, event.DeviceOnlineEvent{
		DeviceID:   deviceID,
		DeviceType: "masc-server",
	}).Run(func(_ mock.Arguments) { expectedCalls.Done() }).Once()
	suite.portalStub.On("Publish", mock.Anything, mock.Anything, mock.Anything).Once()
	defer suite.storeStub.AssertExpectations(suite.T())
	defer suite.portalStub.AssertExpectations(suite.T())
	// Handle.
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := suite.service.Run(runCtx)
		suite.Nil(err, "should not fail")
	}()
	// Send online report.
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-timeout.Done():
			suite.Fail("timeout", "should have picked up event within timeout")
			return
		case reportReceive <- event.Event[any]{
			Payload: event.Event[any]{},
		}:
		}
	}()
	// Await all.
	go func() {
		expectedCalls.Wait()
		cancelRun()
		wg.Wait()
		cancel()
	}()
	<-timeout.Done()
	suite.Equal(context.Canceled, timeout.Err(), "should not time out")
}

func TestDeviceService(t *testing.T) {
	suite.Run(t, new(deviceServiceSuite))
}
