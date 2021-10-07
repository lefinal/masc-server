package acting

import (
	"context"
	"encoding/json"
	nativeerrors "errors"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/gatekeeping"
	"github.com/LeFinal/masc-server/messages"
	"github.com/google/uuid"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/suite"
	"sync"
	"testing"
	"time"
)

const waitTimeout = time.Duration(3) * time.Second

// expectOutgoingMessageTypes starts a new goroutine that reads from the given
// channel and makes sure that the passed message types occur in the defined
// order. It will also add on the given sync.WaitGroup so don't do that
// yourself!
func expectOutgoingMessageTypes(suite suite.Suite, ctx context.Context, wg *sync.WaitGroup, c chan ActorOutgoingMessage, types ...messages.MessageType) {
	wg.Add(1)
	go func() {
		defer wg.Done()
		for i, messageType := range types {
			select {
			case <-ctx.Done():
				suite.Failf("timeout", "timeout while waiting for expected message %d/%d", i+1, len(types))
				return
			case message := <-c:
				suite.Equal(messageType, message.MessageType, "message type of message %d/%d should match expected", i+1, len(types))
			}
		}
	}()
}

type getRoleTestSuite struct {
	suite.Suite
}

func (suite *getRoleTestSuite) TestKnown() {
	role, found := getRole(string(RoleTeamBase))
	suite.Require().True(found, "role should be found")
	suite.Assert().Equal(RoleTeamBase, role, "role should match expected")
}

func (suite *getRoleTestSuite) TestUnknown() {
	_, found := getRole(string("unknown-role"))
	suite.Assert().False(found, "role should no be found")
}

func Test_getRole(t *testing.T) {
	suite.Run(t, new(getRoleTestSuite))
}

type netActorDeviceTestSuite struct {
	suite.Suite
	device        *gatekeeping.Device
	deviceSend    chan messages.MessageContainer
	deviceReceive chan messages.MessageContainer
	actorDevice   *netActorDevice
}

func (suite *netActorDeviceTestSuite) shutdownActorDeviceAndCatchActorQuits() {
	var wg sync.WaitGroup
	for _, actor := range suite.actorDevice.actors {
		wg.Add(1)
		go func(actor Actor) {
			defer wg.Done()
			<-actor.Quit()
		}(actor)
	}
	suite.actorDevice.shutdown()
	wg.Wait()
}

func (suite *netActorDeviceTestSuite) SetupTest() {
	suite.deviceSend = make(chan messages.MessageContainer)
	suite.deviceReceive = make(chan messages.MessageContainer)
	suite.device = &gatekeeping.Device{
		ID:      messages.DeviceID(uuid.New().String()),
		Name:    "Test device",
		Roles:   []string{string(RoleGameMaster), string(RoleTeamBase), string(RoleTeamBaseMonitor)},
		Send:    suite.deviceSend,
		Receive: suite.deviceReceive,
	}
	suite.actorDevice = &netActorDevice{device: suite.device}
}

func (suite *netActorDeviceTestSuite) TestActorCreation() {
	err := suite.actorDevice.boot()
	suite.Require().Nilf(err, "boot should not fail but got: %s", errors.Prettify(err))
	suite.Assert().Len(suite.actorDevice.actors, len(suite.device.Roles), "should create all actors")
	suite.actorDevice.shutdown()
}

func (suite *netActorDeviceTestSuite) TestActorCreationFail() {
	// Overwrite device roles with bs.
	suite.device.Roles = []string{"unknown-role"}
	err := suite.actorDevice.boot()
	suite.Require().NotNil(err, "boot should fail but got")
}

func (suite *netActorDeviceTestSuite) TestRoutingReceive() {
	err := suite.actorDevice.boot()
	suite.Require().Nilf(err, "boot should not fail but got: %s", errors.Prettify(err))
	// Test for each actor.
	var wg sync.WaitGroup
	for actorID, actor := range suite.actorDevice.actors {
		wg.Add(1)
		// Expect the actor to receive a message (will be sent right after).
		ctx, cancel := context.WithTimeout(context.Background(), waitTimeout)
		go func(actorID messages.ActorID, actor Actor) {
			defer wg.Done()
			select {
			case <-ctx.Done():
				// We do not add buffer to receive channel, so we take the message here.
				<-suite.deviceReceive
				suite.Failf("actor %s did not receive message", string(actorID))
			case message := <-actor.Receive():
				// Assure correct message.
				cancel()
				suite.Assert().Equal(messages.MessageTypeHello, message.MessageType, "message type should match expected")
			}
		}(actorID, actor)
		// Send message to device with actor id set.
		suite.deviceReceive <- messages.MessageContainer{
			MessageType: messages.MessageTypeHello,
			DeviceID:    suite.device.ID,
			ActorID:     actorID,
			Content:     json.RawMessage{},
		}
	}
	// Wait until completion.
	wg.Wait()
	suite.actorDevice.shutdown()
}

func (suite *netActorDeviceTestSuite) TestRoutingSend() {
	err := suite.actorDevice.boot()
	suite.Require().Nilf(err, "boot should not fail but got: %s", errors.Prettify(err))
	// Test for each actor.
	var wg sync.WaitGroup
	for actorID, actor := range suite.actorDevice.actors {
		wg.Add(1)
		// Expect the device to send a message (will be created right after).
		ctx, cancel := context.WithTimeout(context.Background(), waitTimeout)
		go func(actorID messages.ActorID) {
			defer wg.Done()
			for i := 0; i < 2; i++ {
				select {
				case <-ctx.Done():
					// We do not add buffer to receive channel, so we take the message here.
					<-suite.deviceSend
					suite.Fail("device did not send message")
				case message := <-suite.deviceSend:
					// Throw first message away as this will be the you-are-in message.
					if message.MessageType == messages.MessageTypeYouAreIn {
						continue
					}
					// Assure correct message.
					cancel()
					suite.Assert().Equal(messages.MessageTypeGetDevices, message.MessageType, "message type should match expected")
				}
			}
		}(actorID)
		// Hire actor.
		err = actor.Hire()
		suite.Require().Nilf(err, "hire actor should not fail but got: %s", errors.Prettify(err))
		// Send message to device with actor id set.
		err = actor.Send(ActorOutgoingMessage{
			MessageType: messages.MessageTypeGetDevices,
			Content:     struct{}{},
		})
		suite.Require().Nilf(err, "send should not fail but got: %s", errors.Prettify(err))
	}
	// Wait until completion.
	wg.Wait()
	suite.shutdownActorDeviceAndCatchActorQuits()
}

func Test_netActorDevice(t *testing.T) {
	suite.Run(t, new(netActorDeviceTestSuite))
}

type ProtectedAgencyTestSuite struct {
	suite.Suite
	device     *gatekeeping.Device
	agency     *ProtectedAgency
	gatekeeper *mockGatekeeper
}

type mockGatekeeper struct {
	mock.Mock
}

func (gk *mockGatekeeper) WakeUpAndProtect(protected gatekeeping.Protected) error {
	args := gk.Called(protected)
	return args.Error(0)
}

func (gk *mockGatekeeper) Retire() error {
	args := gk.Called()
	return args.Error(0)
}

func (gk *mockGatekeeper) GetDevices() []*gatekeeping.Device {
	args := gk.Called()
	return args.Get(0).([]*gatekeeping.Device)
}

func (gk *mockGatekeeper) AcceptDevice(deviceID messages.DeviceID, name string) error {
	args := gk.Called(deviceID, name)
	return args.Error(0)
}

func (suite *ProtectedAgencyTestSuite) SetupTest() {
	suite.device = &gatekeeping.Device{
		ID:      messages.DeviceID(uuid.New().String()),
		Name:    "Test device",
		Roles:   []string{string(RoleGameMaster), string(RoleTeamBase), string(RoleTeamBaseMonitor)},
		Send:    make(chan messages.MessageContainer),
		Receive: make(chan messages.MessageContainer),
	}
	suite.gatekeeper = new(mockGatekeeper)
	suite.agency = &ProtectedAgency{
		gatekeeper: suite.gatekeeper,
	}
}

type ProtectedAgencyOpenTestSuite struct {
	ProtectedAgencyTestSuite
}

func (suite *ProtectedAgencyOpenTestSuite) TestWakeUpGatekeeperFail() {
	suite.gatekeeper.On("WakeUpAndProtect", suite.agency).Return(nativeerrors.New("ERROR"))
	err := suite.agency.Open()
	suite.Assert().NotNil(err, "open should fail")
	suite.gatekeeper.AssertExpectations(suite.T())
}

func (suite *ProtectedAgencyOpenTestSuite) TestOK() {
	suite.gatekeeper.On("WakeUpAndProtect", suite.agency).Return(nil)
	suite.gatekeeper.On("Retire").Return(nil)

	err := suite.agency.Open()
	suite.Assert().Nilf(err, "open should not fail but got: %s", errors.Prettify(err))

	err = suite.agency.Close()
	suite.Assert().Nilf(err, "close should not fail but got: %s", errors.Prettify(err))
	suite.gatekeeper.AssertExpectations(suite.T())
}

func TestProtectedAgency_Open(t *testing.T) {
	suite.Run(t, new(ProtectedAgencyOpenTestSuite))
}

type ProtectedAgencyWelcomeDeviceTestSuite struct {
	ProtectedAgencyTestSuite
}

func (suite *ProtectedAgencyWelcomeDeviceTestSuite) TestUnknownRoles() {
	suite.gatekeeper.On("WakeUpAndProtect", mock.Anything).Return(nil)
	suite.gatekeeper.On("Retire").Return(nil)
	suite.device.Roles = []string{"unknown-role"}
	err := suite.agency.Open()
	suite.Require().Nilf(err, "open should not fail but got: %s", errors.Prettify(err))

	err = suite.agency.WelcomeDevice(suite.device)
	suite.Assert().NotNil(err, "welcome should fail")

	err = suite.agency.Close()
	suite.Assert().Nilf(err, "close should not fail but got: %s", errors.Prettify(err))
}

func (suite *ProtectedAgencyWelcomeDeviceTestSuite) TestOK() {
	suite.gatekeeper.On("WakeUpAndProtect", mock.Anything).Return(nil)
	suite.gatekeeper.On("Retire").Return(nil)
	err := suite.agency.Open()
	suite.Require().Nilf(err, "open should not fail but got: %s", errors.Prettify(err))

	err = suite.agency.WelcomeDevice(suite.device)
	suite.Require().Nilf(err, "welcome should not fail but got: %s", errors.Prettify(err))
	// Check added device.
	_, found := suite.agency.actorDevices[suite.device.ID]
	suite.Assert().True(found, "should set created actor device to agency's actor devices")

	err = suite.agency.Close()
	suite.Assert().Nilf(err, "close should not fail but got: %s", errors.Prettify(err))
}

func TestProtectedAgency_WelcomeDevice(t *testing.T) {
	suite.Run(t, new(ProtectedAgencyWelcomeDeviceTestSuite))
}

type ProtectedAgencySayGoodbyeToDeviceTestSuite struct {
	ProtectedAgencyTestSuite
}

func (suite *ProtectedAgencySayGoodbyeToDeviceTestSuite) SetupTest() {
	suite.ProtectedAgencyTestSuite.SetupTest()
	// Open agency.
	suite.gatekeeper.On("WakeUpAndProtect", mock.Anything).Return(nil)
	suite.gatekeeper.On("Retire").Return(nil)
	err := suite.agency.Open()
	suite.Require().Nilf(err, "open should not fail but got: %s", errors.Prettify(err))
	err = suite.agency.WelcomeDevice(suite.device)
	suite.Require().Nilf(err, "welcome should not fail but got: %s", errors.Prettify(err))
}

func (suite ProtectedAgencySayGoodbyeToDeviceTestSuite) AfterTest() {
	err := suite.agency.Close()
	suite.Assert().Nilf(err, "close should not fail but got: %s", errors.Prettify(err))
}

func (suite *ProtectedAgencySayGoodbyeToDeviceTestSuite) TestUnknownDevice() {
	err := suite.agency.SayGoodbyeToDevice(messages.DeviceID(uuid.New().String()))
	suite.Assert().NotNil(err, "should fail because of unknown device")
}

func (suite *ProtectedAgencySayGoodbyeToDeviceTestSuite) TestOK() {
	// Say goodbye to registered device.
	err := suite.agency.SayGoodbyeToDevice(suite.device.ID)
	suite.Assert().Nilf(err, "goodbye should not fail but got: %s", errors.Prettify(err))
	// Check if unset.
	_, found := suite.agency.actorDevices[suite.device.ID]
	suite.Assert().False(found, "device should be removed from agency actor devices")
}

func TestProtectedAgency_SayGoodbyeToDevice(t *testing.T) {
	suite.Run(t, new(ProtectedAgencySayGoodbyeToDeviceTestSuite))
}

type ProtectedAgencyAvailableActorsTestSuite struct {
	ProtectedAgencyTestSuite
}

func (suite *ProtectedAgencyAvailableActorsTestSuite) SetupTest() {
	suite.ProtectedAgencyTestSuite.SetupTest()
	// Open agency.
	suite.gatekeeper.On("WakeUpAndProtect", mock.Anything).Return(nil)
	suite.gatekeeper.On("Retire").Return(nil)
	err := suite.agency.Open()
	suite.Require().Nilf(err, "open should not fail but got: %s", errors.Prettify(err))
	err = suite.agency.WelcomeDevice(suite.device)
	suite.Require().Nilf(err, "welcome should not fail but got: %s", errors.Prettify(err))
}

func (suite *ProtectedAgencyTestSuite) AfterTest() {
	err := suite.agency.Close()
	suite.Assert().Nilf(err, "close should not fail but got: %s", errors.Prettify(err))
}

func (suite *ProtectedAgencyAvailableActorsTestSuite) TestNoneAvailable() {
	// Get available actors.
	actors := suite.agency.AvailableActors(RoleGlobalMonitor) // None set.
	suite.Assert().Empty(actors, "should return no actors")
}

func (suite *ProtectedAgencyAvailableActorsTestSuite) TestOK() {
	// Get available actors.
	actors := suite.agency.AvailableActors(RoleGameMaster) // None set.
	suite.Assert().Len(actors, 1, "should return correct actor count")
}

func TestProtectedAgency_AvailableActors(t *testing.T) {
	suite.Run(t, new(ProtectedAgencyAvailableActorsTestSuite))
}

type NetActorTestSuite struct {
	suite.Suite
	a          *netActor
	fromActor  chan netActorDeviceOutgoingMessage
	toActor    chan ActorIncomingMessage
	actorQuits chan struct{}
}

func (suite *NetActorTestSuite) SetupTest() {
	suite.fromActor = make(chan netActorDeviceOutgoingMessage)
	suite.toActor = make(chan ActorIncomingMessage)
	suite.actorQuits = make(chan struct{})
	suite.a = &netActor{
		id:        messages.ActorID(uuid.New().String()),
		send:      suite.fromActor,
		receive:   suite.toActor,
		isHired:   false,
		quit:      suite.actorQuits,
		hireMutex: sync.RWMutex{},
	}
}

func (suite *NetActorTestSuite) TestID() {
	suite.Assert().Equal(suite.a.id, suite.a.ID(), "id should be as set")
}

func (suite *NetActorTestSuite) TestHireAlreadyHired() {
	suite.a.isHired = true
	err := suite.a.Hire()
	suite.Assert().NotNil(err, "hire should fail because already hired")
}

func (suite *NetActorTestSuite) TestHireOK() {
	ctx, cancel := context.WithTimeout(context.Background(), waitTimeout)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-ctx.Done():
			suite.Fail("no you-are-in message received")
		case message := <-suite.fromActor:
			cancel()
			suite.Assert().Equal(messages.MessageTypeYouAreIn, message.message.MessageType)
			cMessageContent := message.message.Content.(messages.MessageYouAreIn)
			suite.Assert().Equal(suite.a.id, cMessageContent.ActorID, "should have actor id in message content")
			suite.Assert().Equal(string(suite.a.role), cMessageContent.Role, "should have role in message content")
		}
	}()
	err := suite.a.Hire()
	suite.Assert().Nil(err, "hire should not fail")
	wg.Wait()
}

func (suite *NetActorTestSuite) TestFireNotHired() {
	suite.a.isHired = false
	err := suite.a.Fire()
	suite.Assert().NotNil(err, "fire should fail because not hired")
}

func (suite *NetActorTestSuite) TestFireOK() {
	suite.a.isHired = true
	ctx, cancel := context.WithTimeout(context.Background(), waitTimeout)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-ctx.Done():
			suite.Fail("no fired message received")
		case message := <-suite.fromActor:
			cancel()
			suite.Assert().Equal(messages.MessageTypeFired, message.message.MessageType)
		}
	}()
	err := suite.a.Fire()
	suite.Assert().Nil(err, "fire should not fail")
	suite.Assert().False(suite.a.isHired, "should not be hired anymore")
	wg.Wait()
}

func (suite *NetActorTestSuite) TestSendNotHired() {
	err := suite.a.Send(ActorOutgoingMessage{
		MessageType: messages.MessageTypeHello,
		Content:     struct{}{},
	})
	suite.Assert().NotNil(err, "send should fail")
}

func (suite *NetActorTestSuite) TestSendNotHiredHire() {
	ctx, cancel := context.WithTimeout(context.Background(), waitTimeout)
	var wg sync.WaitGroup
	c := make(chan ActorOutgoingMessage)
	go func() {
		for message := range suite.fromActor {
			c <- message.message
		}
	}()
	expectOutgoingMessageTypes(suite.Suite, ctx, &wg, c, messages.MessageTypeYouAreIn)
	err := suite.a.Send(ActorOutgoingMessage{
		MessageType: messages.MessageTypeYouAreIn,
		Content:     struct{}{},
	})
	suite.Assert().Nil(err, "send should not fail")
	wg.Wait()
	cancel()
	close(c)
}

func (suite *NetActorTestSuite) TestSendFiredNotHired() {
	err := suite.a.Send(ActorOutgoingMessage{
		MessageType: messages.MessageTypeFired,
		Content:     struct{}{},
	})
	suite.Assert().NotNil(err, "send should fail")
}

func TestNetActor(t *testing.T) {
	suite.Run(t, new(NetActorTestSuite))
}
