package games

import (
	"context"
	"fmt"
	"github.com/LeFinal/masc-server/acting"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/messages"
	"github.com/stretchr/testify/suite"
	"sync"
	"testing"
)

type ReadyAwaiterTestSuite struct {
	suite.Suite
	ctx               context.Context
	cancel            context.CancelFunc
	readyStateUpdates chan ReadyStateUpdate
}

func (suite *ReadyAwaiterTestSuite) eatReadyStateUpdates(ctx context.Context) chan ReadyStateUpdate {
	c := make(chan ReadyStateUpdate)
	go func() {
		for {
			select {
			case <-ctx.Done():
				return
			case <-c:
			}
		}
	}()
	return c
}

func (suite *ReadyAwaiterTestSuite) SetupTest() {
	ctx, cancel := context.WithCancel(context.Background())
	suite.ctx = ctx
	suite.cancel = cancel
	suite.readyStateUpdates = make(chan ReadyStateUpdate)
}

func (suite *ReadyAwaiterTestSuite) TestNoActors() {
	err := RequestAndAwaitReady(suite.ctx, suite.readyStateUpdates)
	suite.Assert().Nilf(err, "should not fail but got: %s", errors.Prettify(err))
}

func (suite *ReadyAwaiterTestSuite) TestSingleActor() {
	a := acting.NewMockActor("garden")
	_, _ = a.Hire("listen")

	newsletter := a.SubscribeOutgoingMessageType(messages.MessageTypeAreYouReady)
	go func() {
		<-newsletter.Receive
		a.HandleIncomingMessage(acting.ActorOutgoingMessage{
			MessageType: messages.MessageTypeReadyState,
			Content:     messages.MessageReadyState{IsReady: true},
		})
	}()
	err := RequestAndAwaitReady(suite.ctx, suite.eatReadyStateUpdates(suite.ctx), a)
	suite.Require().Nilf(err, "should not fail but got: %s", errors.Prettify(err))
	suite.cancel()
	suite.Assert().Nil(a.MessageCollector.AssureOutgoingMessageTypes(false, messages.MessageTypeAreYouReady,
		messages.MessageTypeReadyAccepted), "should have sent expected messages")
}

func (suite *ReadyAwaiterTestSuite) TestMultiActorAllReady() {
	actorCount := 8
	actors := make([]acting.Actor, 0, actorCount)
	for i := 0; i < actorCount; i++ {
		a := acting.NewMockActor(messages.ActorID(fmt.Sprintf("slight-%d", i)))
		_, _ = a.Hire(fmt.Sprintf("slighter-%d", i))
		newsletter := a.SubscribeOutgoingMessageType(messages.MessageTypeAreYouReady)
		go func() {
			<-newsletter.Receive
			a.HandleIncomingMessage(acting.ActorOutgoingMessage{
				MessageType: messages.MessageTypeReadyState,
				Content:     messages.MessageReadyState{IsReady: true},
			})
		}()
		actors = append(actors, a)
	}

	err := RequestAndAwaitReady(suite.ctx, suite.eatReadyStateUpdates(suite.ctx), actors...)
	suite.Require().Nilf(err, "should not fail but got: %s", errors.Prettify(err))
	suite.cancel()
}

func (suite *ReadyAwaiterTestSuite) TestMultiActorStateChanging() {
	changingActorCount := 16
	changeTimeCount := 32
	changingActors := make([]acting.Actor, 0, changingActorCount)
	var changing sync.WaitGroup
	for i := 0; i < changingActorCount; i++ {
		a := acting.NewMockActor(messages.ActorID(fmt.Sprintf("slight-%d", i)))
		_, _ = a.Hire(fmt.Sprintf("slighter-%d", i))
		newsletter := a.SubscribeOutgoingMessageType(messages.MessageTypeAreYouReady)
		changing.Add(1)
		go func() {
			defer changing.Done()
			<-newsletter.Receive
			// Now we start changing our ready-state.
			for i := 0; i < changeTimeCount; i++ {
				a.HandleIncomingMessage(acting.ActorOutgoingMessage{
					MessageType: messages.MessageTypeReadyState,
					Content:     messages.MessageReadyState{IsReady: i%2 == 0},
				})
			}
			// And now the final ready-state.
			a.HandleIncomingMessage(acting.ActorOutgoingMessage{
				MessageType: messages.MessageTypeReadyState,
				Content:     messages.MessageReadyState{IsReady: true},
			})
		}()
		changingActors = append(changingActors, a)
	}
	// The actor that blocks until changing is done.
	finalActor := acting.NewMockActor("the-final-one")
	_, _ = finalActor.Hire("the-final-one")
	go func() {
		changing.Wait()
		finalActor.HandleIncomingMessage(acting.ActorOutgoingMessage{
			MessageType: messages.MessageTypeReadyState,
			Content:     messages.MessageReadyState{IsReady: true},
		})
	}()

	err := RequestAndAwaitReady(suite.ctx, suite.eatReadyStateUpdates(suite.ctx), changingActors...)
	suite.Require().Nilf(err, "should not fail but got: %s", errors.Prettify(err))
	suite.cancel()
}

func (suite *ReadyAwaiterTestSuite) TestAbort() {
	a := acting.NewMockActor("garden")
	_, _ = a.Hire("listen")

	suite.cancel()
	err := RequestAndAwaitReady(suite.ctx, suite.eatReadyStateUpdates(suite.ctx), a)
	suite.Require().NotNil(err, "should fail")
	suite.Assert().Nil(a.MessageCollector.AssureOutgoingMessageTypes(false, messages.MessageTypeAreYouReady,
		messages.MessageTypeReadyAccepted), "should have sent expected messages")
}

func TestRequestAndAwaitReady(t *testing.T) {
	suite.Run(t, new(ReadyAwaiterTestSuite))
}
