package acting

import (
	"context"
	nativeerrors "errors"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/messages"
	"github.com/stretchr/testify/suite"
	"sync"
	"testing"
)

type MockAgencyTestSuite struct {
	suite.Suite
}

func (suite *MockAgencyTestSuite) TestNew() {
	a := NewMockAgency()
	suite.Require().NotNil(a, "should create agency")
	suite.Assert().NotNil(a.Actors, "actor list should be created")
}

func (suite *MockAgencyTestSuite) TestAddActor() {
	a := NewMockAgency()
	actor := NewMockActor("what")
	role := RoleTeamBase
	a.AddActor(actor, role)
	actorsForRole, ok := a.Actors[role]
	suite.Require().True(ok, "should have added actor")
	suite.Require().NotNil(actorsForRole, "should create actors list")
	suite.Require().Len(actorsForRole, 1, "should add one actor")
	suite.Assert().Equal(actor, actorsForRole[0], "should add same actor")
}

func (suite *MockAgencyTestSuite) TestActorByIDNotFound1() {
	a := NewMockAgency()
	_, found := a.ActorByID("hello")
	suite.Assert().False(found, "actor should not be found")
}

func (suite *MockAgencyTestSuite) TestActorByIDNotFound2() {
	a := NewMockAgency()
	a.Actors["lol"] = append(a.Actors["lol"], NewMockActor("world!"))
	_, found := a.ActorByID("hello")
	suite.Assert().False(found, "actor should not be found")
}

func (suite *MockAgencyTestSuite) TestActorByIDFound1() {
	a := NewMockAgency()
	actorID := messages.ActorID("what")
	a.Actors["lol"] = append(a.Actors["lol"], NewMockActor(actorID))
	actor, found := a.ActorByID(actorID)
	suite.Require().True(found, "actor should be found")
	suite.Assert().Equal(actorID, actor.ID(), "should be wanted actor")
}

func (suite *MockAgencyTestSuite) TestActorByIDFound2() {
	a := NewMockAgency()
	actorID := messages.ActorID("what")
	a.Actors["yo"] = append(a.Actors["yo"], NewMockActor("hello"), NewMockActor("world"))
	a.Actors["lol"] = append(a.Actors["lol"], NewMockActor("!"), NewMockActor(actorID))
	actor, found := a.ActorByID(actorID)
	suite.Require().True(found, "actor should be found")
	suite.Assert().Equal(actorID, actor.ID(), "should be wanted actor")
}

func (suite *MockAgencyTestSuite) TestRemoveActor() {
	a := NewMockAgency()
	actor := NewMockActor("hello")
	a.AddActor(actor, "catering")
	_, found := a.ActorByID(actor.ID())
	suite.Require().True(found, "should have added the actor")
	a.RemoveActor(actor)
	_, found = a.ActorByID(actor.ID())
	suite.Assert().False(found, "should have removed the actor")
}

func (suite *MockAgencyTestSuite) TestAvailableActors() {
	a := NewMockAgency()
	wantedRole := Role("speedsoft")
	a.AddActor(NewMockActor("1"), "milsim")
	a.AddActor(NewMockActor("2"), "milsim")
	hired := NewMockActor("3")
	hired.isHired = true
	a.AddActor(hired, wantedRole)
	a.AddActor(NewMockActor("4"), wantedRole)
	hired = NewMockActor("5")
	hired.isHired = true
	a.AddActor(hired, wantedRole)
	a.AddActor(NewMockActor("6"), wantedRole)

	availableActors := a.AvailableActors(wantedRole)
	suite.Require().Len(availableActors, 2, "should return correct amount of available actors")
	suite.Assert().EqualValues("4", availableActors[0].ID())
	suite.Assert().EqualValues("6", availableActors[1].ID())
}

func TestMockAgency(t *testing.T) {
	suite.Run(t, new(MockAgencyTestSuite))
}

type MockActorTestSuite struct {
	suite.Suite
}

func (suite *MockActorTestSuite) TestNew() {
	id := messages.ActorID("hello-world")
	a := NewMockActor(id)
	suite.Require().NotNil(a, "should create")
	suite.Assert().Equal(id, a.id, "id should be set")
	suite.Assert().NotNil(a.quit, "quit channel should be created")
	suite.Assert().NotNil(a.MessageCollector, "message collector should be created")
	suite.Assert().NotNil(a.incomingSM, "subscription manager should be created")
}

func (suite *MockActorTestSuite) TestQuit() {
	a := NewMockActor("what")
	ctx, cancel := context.WithTimeout(context.Background(), waitTimeout)
	go func() {
		a.PerformQuit()
	}()
	select {
	case <-ctx.Done():
		suite.Fail("timeout while waiting for quit")
		cancel()
	case <-a.Quit():
		cancel()
	}
}

func (suite *MockActorTestSuite) TestQuitCancelsNewsletters() {
	a := NewMockActor("what")
	newsletter := a.SubscribeMessageType(messages.MessageTypeHello)
	ctx, cancel := context.WithTimeout(context.Background(), waitTimeout)
	go func() {
		a.PerformQuit()
	}()
	select {
	case <-ctx.Done():
		suite.Fail("timeout while waiting for quit")
		cancel()
	case <-newsletter.Subscription.Ctx.Done():
		cancel()
	}
}

func (suite *MockActorTestSuite) TestGetID() {
	id := messages.ActorID("what")
	a := NewMockActor(id)
	suite.Equal(id, a.ID(), "id should be as set")
}

func (suite *MockActorTestSuite) TestHireWithError() {
	a := NewMockActor("")
	a.HireErr = nativeerrors.New("sad life")
	suite.Assert().Equal(a.HireErr, a.Hire(), "should fail with set error")
	suite.Assert().False(a.IsHired(), "should not be hired")
}

func (suite *MockActorTestSuite) TestHireOK() {
	a := NewMockActor("")
	err := a.Hire()
	suite.Assert().Nilf(err, "should not fail but got: %s", errors.Prettify(err))
	suite.Assert().True(a.IsHired(), "should be hired")
}

func (suite *MockActorTestSuite) TestSendWithError() {
	a := NewMockActor("")
	a.SendErr = nativeerrors.New("sad life")
	before := a.MessageCollector.OutgoingMessageCount()
	suite.Assert().Equal(a.SendErr, a.Send(ActorOutgoingMessage{}), "should fail with set error")
	suite.Assert().Equal(before, a.MessageCollector.OutgoingMessageCount(), "should not be sent")
}

func (suite *MockActorTestSuite) TestSendWithoutError() {
	a := NewMockActor("")
	before := a.MessageCollector.OutgoingMessageCount()
	err := a.Send(ActorOutgoingMessage{})
	suite.Assert().Nilf(err, "should not fail but got: %s", errors.Prettify(err))
	suite.Assert().Equal(before+1, a.MessageCollector.OutgoingMessageCount(), "should be sent")
}

func (suite *MockActorTestSuite) TestSubscribeMessageType() {
	a := NewMockActor("")
	newsletter := a.SubscribeMessageType(messages.MessageTypeGetDevices)
	ctx, cancel := context.WithTimeout(context.Background(), waitTimeout)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		select {
		case <-ctx.Done():
			suite.Fail("timeout", "message not received from newsletter")
		case <-newsletter.Receive:
			cancel()
		}
	}()
	a.HandleIncomingMessage(ActorOutgoingMessage{MessageType: messages.MessageTypeGetDevices})
	wg.Wait()
}

func TestMockActor(t *testing.T) {
	suite.Run(t, new(MockActorTestSuite))
}

type MessageCollectorGeneralTestSuite struct {
	suite.Suite
}

func (suite *MessageCollectorGeneralTestSuite) TestNew() {
	c := newMessageCollector()
	suite.Assert().NotNil(c.incoming, "incoming list should be created")
	suite.Assert().NotNil(c.outgoing, "outgoing list should be created")
}

func (suite *MessageCollectorGeneralTestSuite) TestHandleIncoming() {
	c := newMessageCollector()
	before := len(c.incoming)
	m := ActorOutgoingMessage{}
	c.handleIncoming(m)
	suite.Require().Equal(before+1, len(c.incoming), "should have added incoming message")
	suite.Assert().Equal(m, c.incoming[0], "should match incoming message")
}

func (suite *MessageCollectorGeneralTestSuite) TestHandleOutgoing() {
	c := newMessageCollector()
	before := len(c.outgoing)
	m := ActorOutgoingMessage{}
	c.handleOutgoing(m)
	suite.Require().Equal(before+1, len(c.outgoing), "should have added outgoing message")
	suite.Assert().Equal(m, c.outgoing[0], "should match outgoing message")
}

func (suite *MessageCollectorGeneralTestSuite) TestIncomingMessageCount() {
	c := newMessageCollector()
	n := 8
	for i := 0; i < n; i++ {
		c.handleIncoming(ActorOutgoingMessage{})
	}
	suite.Assert().Equal(n, c.IncomingMessageCount(), "should record all incoming messages")
}

func (suite *MessageCollectorGeneralTestSuite) TestOutgoingMessageCount() {
	c := newMessageCollector()
	n := 8
	for i := 0; i < n; i++ {
		c.handleOutgoing(ActorOutgoingMessage{})
	}
	suite.Assert().Equal(n, c.OutgoingMessageCount(), "should record all outgoing messages")
}

func TestMessageCollectorGeneral(t *testing.T) {
	suite.Run(t, new(MessageCollectorGeneralTestSuite))
}

type MessageCollectorAssureIncomingMessageTypesTestSuite struct {
	suite.Suite
	c *messageCollector
}

func (suite *MessageCollectorAssureIncomingMessageTypesTestSuite) record(messageType messages.MessageType) {
	suite.c.handleIncoming(ActorOutgoingMessage{MessageType: messageType})
}

func (suite *MessageCollectorAssureIncomingMessageTypesTestSuite) SetupTest() {
	suite.c = newMessageCollector()
}

func (suite *MessageCollectorAssureIncomingMessageTypesTestSuite) TestNotExactFail1() {
	err := suite.c.AssureIncomingMessageTypes(false, messages.MessageTypeHello)
	suite.Assert().NotNil(err, "should fail")
}

func (suite *MessageCollectorAssureIncomingMessageTypesTestSuite) TestNotExactFail2() {
	suite.record(messages.MessageTypeAcceptDevice)
	err := suite.c.AssureIncomingMessageTypes(false, messages.MessageTypeHello)
	suite.Assert().NotNil(err, "should fail")
}

func (suite *MessageCollectorAssureIncomingMessageTypesTestSuite) TestNotExactFail3() {
	suite.record(messages.MessageTypeHello)
	suite.record(messages.MessageTypeYouAreIn)
	err := suite.c.AssureIncomingMessageTypes(false, messages.MessageTypeHello,
		messages.MessageTypeYouAreIn, messages.MessageTypeError)
	suite.Assert().NotNil(err, "should fail")
}

func (suite *MessageCollectorAssureIncomingMessageTypesTestSuite) TestNotExactFail4() {
	suite.record(messages.MessageTypeHello)
	err := suite.c.AssureIncomingMessageTypes(false, messages.MessageTypeHello,
		messages.MessageTypeError)
	suite.Assert().NotNil(err, "should fail")
}

func (suite *MessageCollectorAssureIncomingMessageTypesTestSuite) TestExactFail1() {
	err := suite.c.AssureIncomingMessageTypes(true, messages.MessageTypeHello)
	suite.Assert().NotNil(err, "should fail")
}

func (suite *MessageCollectorAssureIncomingMessageTypesTestSuite) TestExactFail2() {
	suite.record(messages.MessageTypeRoleAssignments)
	suite.record(messages.MessageTypeAcceptDevice)
	suite.record(messages.MessageTypeHello)
	err := suite.c.AssureIncomingMessageTypes(true, messages.MessageTypeRoleAssignments,
		messages.MessageTypeHello)
	suite.Assert().NotNil(err, "should fail")
}

func (suite *MessageCollectorAssureIncomingMessageTypesTestSuite) TestExactFail3() {
	suite.record(messages.MessageTypeRoleAssignments)
	suite.record(messages.MessageTypeAcceptDevice)
	suite.record(messages.MessageTypeAcceptDevice)
	err := suite.c.AssureIncomingMessageTypes(true, messages.MessageTypeAcceptDevice,
		messages.MessageTypeAcceptDevice)
	suite.Assert().NotNil(err, "should fail")
}

func TestMessageCollector_AssureIncomingMessageTypes(t *testing.T) {
	suite.Run(t, new(MessageCollectorAssureIncomingMessageTypesTestSuite))
}

type MessageCollectorAssureOutgoingMessageTypesTestSuite struct {
	suite.Suite
	c *messageCollector
}

func (suite *MessageCollectorAssureOutgoingMessageTypesTestSuite) record(messageType messages.MessageType) {
	suite.c.handleOutgoing(ActorOutgoingMessage{MessageType: messageType})
}

func (suite *MessageCollectorAssureOutgoingMessageTypesTestSuite) SetupTest() {
	suite.c = newMessageCollector()
}

func (suite *MessageCollectorAssureOutgoingMessageTypesTestSuite) TestNotExactFail1() {
	err := suite.c.AssureOutgoingMessageTypes(false, messages.MessageTypeHello)
	suite.Assert().NotNil(err, "should fail")
}

func (suite *MessageCollectorAssureOutgoingMessageTypesTestSuite) TestNotExactFail2() {
	suite.record(messages.MessageTypeAcceptDevice)
	err := suite.c.AssureOutgoingMessageTypes(false, messages.MessageTypeHello)
	suite.Assert().NotNil(err, "should fail")
}

func (suite *MessageCollectorAssureOutgoingMessageTypesTestSuite) TestNotExactFail3() {
	suite.record(messages.MessageTypeHello)
	suite.record(messages.MessageTypeYouAreIn)
	err := suite.c.AssureOutgoingMessageTypes(false, messages.MessageTypeHello,
		messages.MessageTypeYouAreIn, messages.MessageTypeError)
	suite.Assert().NotNil(err, "should fail")
}

func (suite *MessageCollectorAssureOutgoingMessageTypesTestSuite) TestNotExactFail4() {
	suite.record(messages.MessageTypeHello)
	err := suite.c.AssureOutgoingMessageTypes(false, messages.MessageTypeHello,
		messages.MessageTypeError)
	suite.Assert().NotNil(err, "should fail")
}

func (suite *MessageCollectorAssureOutgoingMessageTypesTestSuite) TestExactFail1() {
	err := suite.c.AssureOutgoingMessageTypes(true, messages.MessageTypeHello)
	suite.Assert().NotNil(err, "should fail")
}

func (suite *MessageCollectorAssureOutgoingMessageTypesTestSuite) TestExactFail2() {
	suite.record(messages.MessageTypeRoleAssignments)
	suite.record(messages.MessageTypeAcceptDevice)
	suite.record(messages.MessageTypeHello)
	err := suite.c.AssureOutgoingMessageTypes(true, messages.MessageTypeRoleAssignments,
		messages.MessageTypeHello)
	suite.Assert().NotNil(err, "should fail")
}

func (suite *MessageCollectorAssureOutgoingMessageTypesTestSuite) TestExactFail3() {
	suite.record(messages.MessageTypeRoleAssignments)
	suite.record(messages.MessageTypeAcceptDevice)
	suite.record(messages.MessageTypeAcceptDevice)
	err := suite.c.AssureOutgoingMessageTypes(true, messages.MessageTypeAcceptDevice,
		messages.MessageTypeAcceptDevice)
	suite.Assert().NotNil(err, "should fail")
}

func TestMessageCollector_AssureOutgoingMessageTypes(t *testing.T) {
	suite.Run(t, new(MessageCollectorAssureOutgoingMessageTypesTestSuite))
}
