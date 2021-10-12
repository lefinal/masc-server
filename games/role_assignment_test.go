package games

import (
	"context"
	nativeerrors "errors"
	"github.com/LeFinal/masc-server/acting"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/messages"
	"github.com/gobuffalo/nulls"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
	"sync"
	"testing"
)

type CastingAddRequestTestSuite struct {
	suite.Suite
	agency *acting.MockAgency
	c      *casting
}

func (suite *CastingAddRequestTestSuite) SetupTest() {
	suite.agency = acting.NewMockAgency()
	suite.c = NewCasting(suite.agency)
}

func (suite *CastingAddRequestTestSuite) TestWhileAssigning() {
	suite.c.state = castingStateCasting
	err := suite.c.AddRequest(ActorRequest{})
	suite.Assert().NotNil(err, "should fail")
}

func (suite *CastingAddRequestTestSuite) TestWhenDone() {
	suite.c.state = castingStateDone
	err := suite.c.AddRequest(ActorRequest{})
	suite.Assert().NotNil(err, "should fail")
}

func (suite *CastingAddRequestTestSuite) TestInvalidMax() {
	r := ActorRequest{Key: "hello-world", Max: nulls.NewUInt32(0)}
	err := suite.c.AddRequest(r)
	suite.Assert().NotNil(err, "should fail")
}

func (suite *CastingAddRequestTestSuite) TestInvalidMinMax() {
	r := ActorRequest{Key: "hello-world", Min: nulls.NewUInt32(3), Max: nulls.NewUInt32(2)}
	err := suite.c.AddRequest(r)
	suite.Assert().NotNil(err, "should fail")
}

func (suite *CastingAddRequestTestSuite) TestAlreadyExistsForKey() {
	r := ActorRequest{Key: "hello-world"}
	err := suite.c.AddRequest(r)
	suite.Require().Nilf(err, "first add request should not fail but got: %s", errors.Prettify(err))
	err = suite.c.AddRequest(r)
	suite.Assert().NotNil(err, "should fail")
}

func (suite *CastingAddRequestTestSuite) TestOK() {
	r := ActorRequest{Key: CastingKey("hello-world")}
	err := suite.c.AddRequest(r)
	suite.Assert().Nilf(err, "should not fail but got: %s", errors.Prettify(err))
}

func TestCasting_AddRequest(t *testing.T) {
	suite.Run(t, new(CastingAddRequestTestSuite))
}

func TestNewCasting(t *testing.T) {
	c := NewCasting(acting.NewMockAgency())
	require.NotNil(t, c, "casting should exist")
	assert.NotNil(t, c.agency, "agency should be set")
	assert.NotNil(t, c.requests, "requests list should be created")
}

type CastingPerformAndHireTestSuite struct {
	suite.Suite
	agency *acting.MockAgency
	jury   *acting.MockActor
	c      *casting
}

func (suite *CastingPerformAndHireTestSuite) SetupTest() {
	suite.agency = acting.NewMockAgency()
	suite.c = NewCasting(suite.agency)
	suite.jury = acting.NewMockActor("jury")
	err := suite.jury.Hire()
	suite.Require().Nilf(err, "hire jury should not fail but got: %s", errors.Prettify(err))
}

func (suite *CastingPerformAndHireTestSuite) TestWhileAssigning() {
	suite.c.state = castingStateCasting
	err := suite.c.PerformAndHire(context.Background(), nil)
	suite.Assert().NotNil(err, "should fail")
}

func (suite *CastingPerformAndHireTestSuite) TestWhenDone() {
	suite.c.state = castingStateDone
	err := suite.c.PerformAndHire(context.Background(), nil)
	suite.Assert().NotNil(err, "should fail")
}

func (suite *CastingPerformAndHireTestSuite) TestSendRequestFail() {
	suite.jury.SendErr = nativeerrors.New("sad life")
	ctx, cancel := context.WithTimeout(context.Background(), waitTimeout)
	performCtx, cancelPerform := context.WithCancel(context.Background())
	result := make(chan error)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		result <- suite.c.PerformAndHire(performCtx, suite.jury)
	}()
	// Wait for result.
	select {
	case <-ctx.Done():
		suite.Fail("timeout", "timeout while waiting for result")
	case err := <-result:
		suite.Assert().NotNil(err, "should fail")
	}
	cancel()
	cancelPerform()
	wg.Wait()
}

func (suite *CastingPerformAndHireTestSuite) TestCancelWhileWaiting() {
	ctx, cancel := context.WithTimeout(context.Background(), waitTimeout)
	performCtx, cancelPerform := context.WithCancel(context.Background())
	result := make(chan error)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		result <- suite.c.PerformAndHire(performCtx, suite.jury)
	}()
	cancelPerform()
	// Wait for result.
	select {
	case <-ctx.Done():
		suite.Fail("timeout", "timeout while waiting for finish")
	case err := <-result:
		suite.Assert().NotNil(err, "should fail")
	}
	cancel()
	wg.Wait()
}

func (suite *CastingPerformAndHireTestSuite) TestJuryQuitWhileWaiting() {
	ctx, cancel := context.WithTimeout(context.Background(), waitTimeout)
	performCtx, cancelPerform := context.WithCancel(context.Background())
	result := make(chan error)
	requestNewsletter := suite.jury.SubscribeOutgoingMessageType(messages.MessageTypeRequestRoleAssignments)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		result <- suite.c.PerformAndHire(performCtx, suite.jury)
	}()
	// Wait for request message.
	select {
	case <-ctx.Done():
		suite.Fail("timeout", "timeout while waiting for request message")
	case <-requestNewsletter.Receive:
		// Now we quit.
		go suite.jury.PerformQuit()
		<-suite.jury.Quit()
	}
	// Wait for result.
	select {
	case <-ctx.Done():
		suite.Fail("timeout", "timeout while waiting for finish")
	case err := <-result:
		suite.Assert().NotNil(err, "should fail")
	}
	cancel()
	cancelPerform()
	wg.Wait()
}

func (suite *CastingPerformAndHireTestSuite) TestUnknownActor() {
	ctx, cancel := context.WithTimeout(context.Background(), waitTimeout)
	defer cancel()
	performCtx, cancelPerform := context.WithCancel(context.Background())
	defer cancelPerform()
	err := suite.c.AddRequest(ActorRequest{Key: "hello-word", Role: "hello-world"})

	// Setup handlers.
	requestNewsletter := suite.jury.SubscribeOutgoingMessageType(messages.MessageTypeRequestRoleAssignments)
	errorNewsletter := suite.jury.SubscribeOutgoingMessageType(messages.MessageTypeError)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Handle request message.
		select {
		case <-ctx.Done():
			suite.Fail("timeout", "timeout while waiting for request message for jury")
		case <-requestNewsletter.Receive:
			err = suite.jury.UnsubscribeOutgoing(requestNewsletter.Subscription)
			suite.Require().Nilf(err, "unsubscribe request newsletter for jury should not fail but got: %s", errors.Prettify(err))
			// Send invalid assignments.
			suite.jury.HandleIncomingMessage(acting.ActorOutgoingMessage{
				MessageType: messages.MessageTypeRoleAssignments,
				Content: messages.MessageRoleAssignments{
					Assignments: []messages.RoleAssignment{
						{Key: "hello-world", ActorID: "unknown-actor-id"},
					},
				},
			})
		}
		// Handle error message.
		select {
		case <-ctx.Done():
			suite.Fail("timeout", "timeout while waiting for error message for jury")
		case <-errorNewsletter.Receive:
			err = suite.jury.UnsubscribeOutgoing(errorNewsletter.Subscription)
			suite.Require().Nilf(err, "unsubscribe error newsletter for jury should not fail but got: %s", errors.Prettify(err))
			// Done.
			cancelPerform()
		}
	}()
	// Perform.
	result := make(chan error)
	wg.Add(1)
	go func() {
		defer wg.Done()
		result <- suite.c.PerformAndHire(performCtx, suite.jury)
	}()
	select {
	case <-ctx.Done():
		suite.Fail("timeout", "timeout while waiting for result")
	case err := <-result:
		suite.Assert().NotNil(err, "should fail")
	}
	wg.Wait()
}

func (suite *CastingPerformAndHireTestSuite) TestRequestNotSatisfied1() {
	suite.agency.AddActor(acting.NewMockActor("actor-0"), "hello-world")
	suite.agency.AddActor(acting.NewMockActor("actor-1"), "hello-world")
	err := suite.c.AddRequest(ActorRequest{Key: "hello-world", Role: "hello-world", Min: nulls.NewUInt32(2)})
	suite.Require().Nilf(err, "add request should not fail but got: %s", errors.Prettify(err))
	suite.assurePerformAndHireInvalidAssignments([]messages.RoleAssignment{
		{Key: "hello-world", ActorID: "actor-0"},
		{Key: "forget-me", ActorID: "actor-1"},
	})
}

func (suite *CastingPerformAndHireTestSuite) TestRequestNotSatisfied2() {
	suite.agency.AddActor(acting.NewMockActor("actor-0"), "hello-world")
	suite.agency.AddActor(acting.NewMockActor("actor-1"), "hello-world")
	err := suite.c.AddRequest(ActorRequest{Key: "hello-world", Role: "hello-world", Min: nulls.NewUInt32(1), Max: nulls.NewUInt32(1)})
	suite.Require().Nilf(err, "add request should not fail but got: %s", errors.Prettify(err))
	suite.assurePerformAndHireInvalidAssignments([]messages.RoleAssignment{
		{Key: "hello-world", ActorID: "actor-0"},
		{Key: "hello-world", ActorID: "actor-1"},
	})
}

func (suite *CastingPerformAndHireTestSuite) TestRequestNotSatisfied3() {
	suite.agency.AddActor(acting.NewMockActor("actor-0"), "hello-world")
	suite.agency.AddActor(acting.NewMockActor("actor-1"), "hello-world")
	err := suite.c.AddRequest(ActorRequest{Key: "hello-world", Role: "hello-world", Min: nulls.NewUInt32(1)})
	suite.Require().Nilf(err, "add request should not fail but got: %s", errors.Prettify(err))
	err = suite.c.AddRequest(ActorRequest{Key: "i-love-cookies", Role: "i-love-cookies", Min: nulls.NewUInt32(1)})
	suite.Require().Nilf(err, "add request should not fail but got: %s", errors.Prettify(err))
	suite.assurePerformAndHireInvalidAssignments([]messages.RoleAssignment{
		{Key: "hello-world", ActorID: "actor-0"},
		{Key: "forget-me", ActorID: "actor-1"},
	})
}

func (suite *CastingPerformAndHireTestSuite) TestHireFail1() {
	actor := acting.NewMockActor("actor-0")
	actor.HireErr = nativeerrors.New("sad life")
	suite.agency.AddActor(actor, "hello-world")
	err := suite.c.AddRequest(ActorRequest{Key: "hello-world", Role: "hello-world"})
	suite.Require().Nilf(err, "add request should not fail but got: %s", errors.Prettify(err))
	suite.assurePerformAndHireInvalidAssignments([]messages.RoleAssignment{
		{Key: "hello-world", ActorID: "actor-0"},
	})
}

// We need to manually test hiring errors because assignments are a map, and we
// iterate over it when setting winners.

func (suite *CastingPerformAndHireTestSuite) TestHireWinnersFailFireAllNotNeeded() {
	actorWithHireFail := acting.NewMockActor("actor-1")
	actorWithHireFail.HireErr = nativeerrors.New("sad life")
	suite.c.winners = map[CastingKey][]acting.Actor{
		"hello-world": {acting.NewMockActor("actor-0"), actorWithHireFail},
	}
	badRequestErr, internalErr := suite.c.hireWinners()
	suite.Assert().NotNil(badRequestErr, "bad request error should not be nil")
	suite.Assert().Nilf(internalErr, "internal error should be nil but got: %s", errors.Prettify(internalErr))
}

func (suite *CastingPerformAndHireTestSuite) TestHireWinnersFailFireAllOK() {
	actorWithFireFail := acting.NewMockActor("actor-0")
	actorWithFireFail.HireErr = nativeerrors.New("sad life")
	suite.c.winners = map[CastingKey][]acting.Actor{
		"hello-world": {actorWithFireFail, acting.NewMockActor("actor-1")},
	}
	badRequestErr, internalErr := suite.c.hireWinners()
	suite.Assert().NotNil(badRequestErr, "bad request error should not be nil")
	suite.Assert().Nilf(internalErr, "internal error should be nil but got: %s", errors.Prettify(internalErr))
}

func (suite *CastingPerformAndHireTestSuite) TestHireWinnersFailFireAllFail() {
	actorWithFireFail := acting.NewMockActor("actor-1")
	actorWithFireFail.FireErr = nativeerrors.New("sad life")
	actorWithHireFail := acting.NewMockActor("actor-2")
	actorWithHireFail.HireErr = nativeerrors.New("sad life")
	suite.c.winners = map[CastingKey][]acting.Actor{
		"hello-world": {actorWithFireFail, actorWithHireFail},
	}
	badRequestErr, internalErr := suite.c.hireWinners()
	suite.Assert().Nilf(badRequestErr, "bad request error should be nil but got: %s", errors.Prettify(badRequestErr))
	suite.Assert().NotNil(internalErr, "internal error should not be nil")
}

func (suite *CastingPerformAndHireTestSuite) assurePerformAndHireInvalidAssignments(assignments []messages.RoleAssignment) {
	ctx, cancel := context.WithTimeout(context.Background(), waitTimeout)
	defer cancel()
	performCtx, cancelPerform := context.WithCancel(context.Background())
	defer cancelPerform()

	// Setup handlers.
	requestNewsletter := suite.jury.SubscribeOutgoingMessageType(messages.MessageTypeRequestRoleAssignments)
	errorNewsletter := suite.jury.SubscribeOutgoingMessageType(messages.MessageTypeError)
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		// Handle request message.
		select {
		case <-ctx.Done():
			suite.Fail("timeout", "timeout while waiting for request message for jury")
		case <-requestNewsletter.Receive:
			err := suite.jury.UnsubscribeOutgoing(requestNewsletter.Subscription)
			suite.Require().Nilf(err, "unsubscribe request newsletter for jury should not fail but got: %s", errors.Prettify(err))
			// Send invalid assignments.
			suite.jury.HandleIncomingMessage(acting.ActorOutgoingMessage{
				MessageType: messages.MessageTypeRoleAssignments,
				Content: messages.MessageRoleAssignments{
					Assignments: assignments,
				},
			})
		}
		// Handle error message.
		select {
		case <-ctx.Done():
			suite.Fail("timeout", "timeout while waiting for error message for jury")
		case <-errorNewsletter.Receive:
			err := suite.jury.UnsubscribeOutgoing(errorNewsletter.Subscription)
			suite.Require().Nilf(err, "unsubscribe error newsletter for jury should not fail but got: %s", errors.Prettify(err))
			// Done.
			cancelPerform()
		}
	}()
	// Perform.
	result := make(chan error)
	wg.Add(1)
	go func() {
		defer wg.Done()
		result <- suite.c.PerformAndHire(performCtx, suite.jury)
	}()
	select {
	case <-ctx.Done():
		suite.Fail("timeout", "timeout while waiting for result")
	case err := <-result:
		suite.Assert().NotNil(err, "should fail")
	}
	wg.Wait()
}

func (suite *CastingPerformAndHireTestSuite) TestRepeatRequestAfterInvalidAssignments() {
	ctx, cancel := context.WithTimeout(context.Background(), waitTimeout)
	defer cancel()
	performCtx, cancelPerform := context.WithCancel(context.Background())
	defer cancelPerform()
	err := suite.c.AddRequest(ActorRequest{Key: "hello-word", Role: "hello-world", Min: nulls.NewUInt32(1)})

	// Setup handlers.
	requestNewsletter := suite.jury.SubscribeOutgoingMessageType(messages.MessageTypeRequestRoleAssignments)
	errorNewsletter := suite.jury.SubscribeOutgoingMessageType(messages.MessageTypeError)
	repeat := 4
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		defer func() {
			err = suite.jury.UnsubscribeOutgoing(requestNewsletter.Subscription)
			suite.Require().Nilf(err, "unsubscribe request newsletter for jury should not fail but got: %s", errors.Prettify(err))
			err = suite.jury.UnsubscribeOutgoing(errorNewsletter.Subscription)
			suite.Require().Nilf(err, "unsubscribe error newsletter for jury should not fail but got: %s", errors.Prettify(err))
		}()
		for i := 0; i < repeat; i++ {
			// Handle request message.
			select {
			case <-ctx.Done():
				suite.Fail("timeout", "timeout while waiting for request message for jury")
				return
			case <-requestNewsletter.Receive:
				// Send invalid assignments.
				suite.jury.HandleIncomingMessage(acting.ActorOutgoingMessage{
					MessageType: messages.MessageTypeRoleAssignments,
					Content: messages.MessageRoleAssignments{
						Assignments: []messages.RoleAssignment{},
					},
				})
			}
			// Handle error message.
			select {
			case <-ctx.Done():
				suite.Fail("timeout", "timeout while waiting for error message for jury")
				return
			case <-errorNewsletter.Receive:
			}
		}
		cancelPerform()
	}()
	// Perform.
	result := make(chan error)
	wg.Add(1)
	go func() {
		defer wg.Done()
		result <- suite.c.PerformAndHire(performCtx, suite.jury)
	}()
	select {
	case <-ctx.Done():
		suite.Fail("timeout", "timeout while waiting for result")
		cancelPerform()
	case err := <-result:
		suite.Assert().NotNil(err, "should fail")
	}
	wg.Wait()
}

func TestCasting_PerformAndHire(t *testing.T) {
	suite.Run(t, new(CastingPerformAndHireTestSuite))
}

type CastingGetWinnersTestSuite struct {
	suite.Suite
	c *casting
}

func (suite *CastingGetWinnersTestSuite) SetupTest() {
	suite.c = NewCasting(acting.NewMockAgency())
	suite.c.state = castingStateDone
	suite.c.winners = map[CastingKey][]acting.Actor{
		"hello-world":    {acting.NewMockActor("0"), acting.NewMockActor("1")},
		"i-love-cookies": {acting.NewMockActor("2")},
	}
}

func (suite *CastingGetWinnersTestSuite) TestWhenReady() {
	suite.c.state = castingStateReady
	_, err := suite.c.GetWinners("hello-world")
	suite.Assert().NotNil(err, "should fail")
}

func (suite *CastingGetWinnersTestSuite) TestWhileAssigning() {
	suite.c.state = castingStateCasting
	_, err := suite.c.GetWinners("hello-world")
	suite.Assert().NotNil(err, "should fail")
}

func (suite *CastingGetWinnersTestSuite) TestUnknownCastingKey() {
	_, err := suite.c.GetWinners("sad-life")
	suite.Assert().NotNil(err, "should fail")
}

func (suite *CastingGetWinnersTestSuite) TestOK1() {
	winners, err := suite.c.GetWinners("hello-world")
	suite.Require().Nilf(err, "should not fail but got: %s", errors.Prettify(err))
	suite.Assert().Len(winners, 2, "should return correct amount of winners")
}

func (suite *CastingGetWinnersTestSuite) TestOK2() {
	winners, err := suite.c.GetWinners("i-love-cookies")
	suite.Require().Nilf(err, "should not fail but got: %s", errors.Prettify(err))
	suite.Assert().Len(winners, 1, "should return correct amount of winners")
}

func TestCasting_GetWinners(t *testing.T) {
	suite.Run(t, new(CastingGetWinnersTestSuite))
}

type CastingGetWinnerTestSuite struct {
	suite.Suite
	c *casting
}

func (suite *CastingGetWinnerTestSuite) SetupTest() {
	suite.c = NewCasting(acting.NewMockAgency())
	suite.c.state = castingStateDone
	suite.c.winners = map[CastingKey][]acting.Actor{
		"hello-world":    {acting.NewMockActor("0"), acting.NewMockActor("1")},
		"i-love-cookies": {acting.NewMockActor("2")},
	}
}

func (suite *CastingGetWinnerTestSuite) TestWhenReady() {
	suite.c.state = castingStateReady
	_, err := suite.c.GetWinner("hello-world")
	suite.Assert().NotNil(err, "should fail")
}

func (suite *CastingGetWinnerTestSuite) TestWhileAssigning() {
	suite.c.state = castingStateCasting
	_, err := suite.c.GetWinner("hello-world")
	suite.Assert().NotNil(err, "should fail")
}

func (suite *CastingGetWinnerTestSuite) TestUnknownCastingKey() {
	_, err := suite.c.GetWinner("sad-life")
	suite.Assert().NotNil(err, "should fail")
}

func (suite *CastingGetWinnerTestSuite) TestMoreThanOne() {
	_, err := suite.c.GetWinner("hello-world")
	suite.Require().NotNil(err, "should fail")
}

func (suite *CastingGetWinnerTestSuite) TestOK() {
	actor, err := suite.c.GetWinner("i-love-cookies")
	suite.Require().Nilf(err, "should not fail but got: %s", errors.Prettify(err))
	suite.Assert().NotNil(actor, 1, "should return correct amount of winners")
}

func TestCasting_GetWinner(t *testing.T) {
	suite.Run(t, new(CastingGetWinnerTestSuite))
}
