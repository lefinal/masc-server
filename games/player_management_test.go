package games

import (
	"context"
	"encoding/json"
	nativeerrors "errors"
	"fmt"
	"github.com/LeFinal/masc-server/acting"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/messages"
	"github.com/google/uuid"
	"github.com/stretchr/testify/suite"
	"sync"
	"testing"
)

type PlayerManagementTestSuite struct {
	suite.Suite
	pm *PlayerManagement
}

func (suite *PlayerManagementTestSuite) SetupTest() {
	suite.pm = NewPlayerManagement(nil)
}

func (suite *PlayerManagementTestSuite) TestNew() {
	suite.Assert().NotNil(suite.pm.active)
}

func (suite *PlayerManagementTestSuite) TestIsActiveActive() {
	suite.pm.active["hello"] = "world"
	suite.Assert().True(suite.pm.IsActive("hello"), "should be active")
}

func (suite *PlayerManagementTestSuite) TestIsActiveInactive() {
	suite.Assert().False(suite.pm.IsActive("hello"), "should be inactive")
}

func (suite *PlayerManagementTestSuite) TestAddPlayerDuplicate() {
	// Use another key for adding.
	suite.pm.active["hello"] = "world"
	suite.Assert().False(suite.pm.AddPlayer("hello", "!"), "should return false")
	suite.Assert().EqualValues("world", suite.pm.active["hello"], "should not have overwritten team")
}

func (suite *PlayerManagementTestSuite) TestAddPlayerOK() {
	suite.Assert().True(suite.pm.AddPlayer("hello", "world"), "should return true")
	suite.Assert().EqualValues("world", suite.pm.active["hello"], "should have added player")
}

func (suite *PlayerManagementTestSuite) TestAddPlayerOKUpdateBroadcast() {
	updates := make(chan PlayerManagementUpdate)
	suite.pm.updates = updates
	go func() {
		suite.Assert().True(suite.pm.AddPlayer("hello", "world"), "should return false")
	}()
	ctx, cancel := context.WithTimeout(context.Background(), waitTimeout)
	defer cancel()
	select {
	case <-ctx.Done():
		suite.Fail("timeout", "timeout while waiting for update")
	case <-updates:
	}
}

func (suite *PlayerManagementTestSuite) TestRemovePlayerNotFound() {
	suite.Assert().False(suite.pm.RemovePlayer("unknown"), "should return false")
}

func (suite *PlayerManagementTestSuite) TestRemovePlayerOK() {
	suite.pm.active["hello"] = "world"
	suite.Assert().True(suite.pm.RemovePlayer("hello"), "should return true")
	_, ok := suite.pm.active["hello"]
	suite.Assert().False(ok, "should have removed player")
}

func (suite *PlayerManagementTestSuite) TestRemovePlayerOKUpdateBroadcast() {
	updates := make(chan PlayerManagementUpdate)
	suite.pm.updates = updates
	suite.pm.active["hello"] = "wold"
	go func() {
		suite.Assert().True(suite.pm.RemovePlayer("hello"), "should return true")
	}()
	ctx, cancel := context.WithTimeout(context.Background(), waitTimeout)
	defer cancel()
	select {
	case <-ctx.Done():
		suite.Fail("timeout", "timeout while waiting for update")
	case <-updates:
	}
}

func (suite *PlayerManagementTestSuite) TestPlayersInTeamNone() {
	suite.pm.active["should"] = "marriage"
	suite.Assert().Len(suite.pm.PlayersInTeam("empty"), 0, "should return correct count")
}

func (suite *PlayerManagementTestSuite) TestPlayersInTeamMultiplePlayers() {
	suite.pm.active["should"] = "marriage"
	suite.pm.active["worm"] = "marriage"
	suite.Assert().Len(suite.pm.PlayersInTeam("marriage"), 2, "should return correct count")
}

func (suite *PlayerManagementTestSuite) TestPlayersInTeamMultipleTeams() {
	suite.pm.active["should"] = "marriage"
	suite.pm.active["prompt"] = "nobody"
	suite.pm.active["suit"] = "woman"
	suite.pm.active["poverty"] = "marriage"
	suite.pm.active["shield"] = "woman"
	suite.pm.active["baggage"] = "woman"
	suite.Assert().Len(suite.pm.PlayersInTeam("woman"), 3, "should return correct count")
}

func (suite *PlayerManagementTestSuite) TestActivePlayersEmpty() {
	suite.Assert().Empty(suite.pm.ActivePlayers(), "should be empty")
}

func (suite *PlayerManagementTestSuite) TestActivePlayersOK() {
	suite.pm.active["tent"] = "enough"
	suite.pm.active["explore"] = "pressure"
	suite.pm.active["coin"] = "effect"
	suite.Assert().Len(suite.pm.ActivePlayers(), len(suite.pm.active), "should return correct player count")
}

func TestPlayerManagement(t *testing.T) {
	suite.Run(t, new(PlayerManagementTestSuite))
}

type PlayerJoinOfficeTestSuite struct {
	suite.Suite
	office           *PlayerJoinOffice
	ctx              context.Context
	cancel           context.CancelFunc
	playerManagement *PlayerManagement
	playerProvider   *MockPlayerProvider
	playerUpdates    chan PlayerManagementUpdate
}

func (suite *PlayerJoinOfficeTestSuite) SetupTest() {
	suite.ctx, suite.cancel = context.WithCancel(context.Background())
	suite.playerProvider = NewMockPlayerProvider()
	suite.playerUpdates = make(chan PlayerManagementUpdate)
	suite.playerManagement = NewPlayerManagement(suite.playerUpdates)
	l := logrus.New()
	l.SetLevel(logrus.PanicLevel)
	suite.office = &PlayerJoinOffice{
		Team:             "test-team",
		Logger:           l.WithField("match", uuid.New().String()),
		PlayerProvider:   suite.playerProvider,
		PlayerManagement: suite.playerManagement,
	}
}

func (suite *PlayerJoinOfficeTestSuite) TeardownTest() {
	close(suite.playerUpdates)
}

func (suite *PlayerJoinOfficeTestSuite) closeOffice(expectUpdates int) error {
	ctx, cancel := context.WithTimeout(context.Background(), waitTimeout)
	done := make(chan error)
	// We remember the original channel because in teardown we close the channel
	// from the suite but do not fait until the following handler has finished. Then
	// a race might occur.
	updateChan := suite.playerUpdates
	go func() {
		updates := make([]PlayerManagementUpdate, 0, expectUpdates)
		for {
			select {
			case <-ctx.Done():
				done <- fmt.Errorf("timeout with %d/%d updates: %v", len(updates), expectUpdates, updates)
				return
			case update, ok := <-updateChan:
				if !ok {
					return
				}
				updates = append(updates, update)
				if len(updates) == expectUpdates {
					cancel()
					done <- nil
					return
				} else if len(updates) > expectUpdates {
					suite.Failf("more than expected", "received %d updates although %d expected",
						updates, expectUpdates)
				}
			}
		}
	}()
	err := <-done
	suite.Require().Nilf(err, "player join should not time out but got: %s", err)
	return suite.office.Close()
}

func (suite *PlayerJoinOfficeTestSuite) TestOKWithoutUserBlameErrors() {
	suite.playerProvider.SetPlayerKnown("user-0")
	suite.playerProvider.SetPlayerKnown("user-1")
	actor0 := acting.NewMockActor("actor-0")
	_, _ = actor0.Hire("actor-0")
	actor1 := acting.NewMockActor("actor-1")
	_, _ = actor1.Hire("actor-1")

	actor0Newsletter := actor0.SubscribeOutgoingMessageType(messages.MessageTypePlayerJoinOpen)
	var wg sync.WaitGroup
	wg.Add(1)

	go func() {
		defer wg.Done()
		<-actor0Newsletter.Receive
		_ = actor0.UnsubscribeOutgoing(actor0Newsletter.Subscription)
		actor0.HandleIncomingMessage(acting.ActorOutgoingMessage{
			MessageType: messages.MessageTypePlayerJoin,
			Content:     messages.MessagePlayerJoin{User: "user-0"},
		})
	}()
	actor1Newsletter := actor1.SubscribeOutgoingMessageType(messages.MessageTypePlayerJoinOpen)
	wg.Add(1)
	go func() {
		defer wg.Done()
		<-actor1Newsletter.Receive
		_ = actor1.UnsubscribeOutgoing(actor1Newsletter.Subscription)
		actor1.HandleIncomingMessage(acting.ActorOutgoingMessage{
			MessageType: messages.MessageTypePlayerJoin,
			Content:     messages.MessagePlayerJoin{User: "user-1"},
		})
	}()

	suite.office.Open(suite.ctx, actor0, actor1)
	err := suite.closeOffice(2)
	actorsDoneCtx, actorsDone := context.WithTimeout(context.Background(), waitTimeout)
	actorsDoneChan := make(chan struct{})
	go func() {
		wg.Wait()
		actorsDoneChan <- struct{}{}
	}()
	select {
	case <-actorsDoneCtx.Done():
		suite.Fail("timeout", "waiting for actors")
		actorsDone()
	case <-actorsDoneChan:
		actorsDone()
	}
	suite.Require().Nilf(err, "should not fail but got: %s", errors.Prettify(err))
	suite.Require().NotNil(actor0.MessageCollector.AssureOutgoingMessageTypes(false, messages.MessageTypeError),
		"should not have sent error message")
	suite.Require().NotNil(actor1.MessageCollector.AssureOutgoingMessageTypes(false, messages.MessageTypeError),
		"should not have sent error message")
}

func (suite *PlayerJoinOfficeTestSuite) TestOKWithUserBlameErrors() {
	suite.playerProvider.SetPlayerKnown("user-0")
	actor := acting.NewMockActor("actor")
	_, _ = actor.Hire("actor")

	actorNewsletter := actor.SubscribeOutgoingMessageType(messages.MessageTypePlayerJoinOpen)
	go func() {
		<-actorNewsletter.Receive
		_ = actor.UnsubscribeOutgoing(actorNewsletter.Subscription)
		// Try to join unknown player.
		actor.HandleIncomingMessage(acting.ActorOutgoingMessage{
			MessageType: messages.MessageTypePlayerJoin,
			Content:     messages.MessagePlayerJoin{User: "unknown-user"},
		})
		actor.HandleIncomingMessage(acting.ActorOutgoingMessage{
			MessageType: messages.MessageTypePlayerJoin,
			Content:     messages.MessagePlayerJoin{User: "user-0"},
		})
	}()

	suite.office.Open(suite.ctx, actor)
	err := suite.closeOffice(1)
	suite.Require().Nilf(err, "should not fail but got: %s", errors.Prettify(err))
	suite.Require().Nil(actor.MessageCollector.AssureOutgoingMessageTypes(false, messages.MessageTypeError),
		"should have sent error message")
}

func (suite *PlayerJoinOfficeTestSuite) TestControlMessages() {
	suite.playerProvider.SetPlayerKnown("user-0")
	actor := acting.NewMockActor("actor")
	_, _ = actor.Hire("actor")

	actorNewsletter := actor.SubscribeOutgoingMessageType(messages.MessageTypePlayerJoinOpen)
	go func() {
		m := <-actorNewsletter.Receive
		_ = actor.UnsubscribeOutgoing(actorNewsletter.Subscription)
		var playerJoinOpenMessage messages.MessagePlayerJoinOpen
		suite.Assert().Nil(json.Unmarshal(m, &playerJoinOpenMessage), "unmarshalling player join open message should not fail")
		suite.Assert().EqualValues(suite.office.Team, playerJoinOpenMessage.Team, "join open message should have correct team")
		actor.HandleIncomingMessage(acting.ActorOutgoingMessage{
			MessageType: messages.MessageTypePlayerJoin,
			Content:     messages.MessagePlayerJoin{User: "user-0"},
		})
	}()

	suite.office.Open(suite.ctx, actor)
	err := suite.closeOffice(1)
	suite.Require().Nilf(err, "should not fail but got: %s", errors.Prettify(err))
	suite.Require().Nil(actor.MessageCollector.AssureOutgoingMessageTypes(false, messages.MessageTypePlayerJoinOpen,
		messages.MessageTypePlayerJoinClosed), "should have sent open and close messages")
}

func (suite *PlayerJoinOfficeTestSuite) TestPlayerLeave() {
	suite.playerProvider.SetPlayerKnown("improve")
	actor := acting.NewMockActor("actor")
	_, _ = actor.Hire("actor")

	actorNewsletter := actor.SubscribeOutgoingMessageType(messages.MessageTypePlayerJoinOpen)
	// This is a bit hacky, but I don't know how to handle this. We need to wait
	// until the player has joined before we handle the leave message. This is
	// needed, because newsletter are handled concurrently.
	var updateChanReady sync.WaitGroup
	updateChanReady.Add(1)
	// Wait group for go routines that might interfere with other tests.
	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		<-actorNewsletter.Receive
		_ = actor.UnsubscribeOutgoing(actorNewsletter.Subscription)
		actor.HandleIncomingMessage(acting.ActorOutgoingMessage{
			MessageType: messages.MessageTypePlayerJoin,
			Content:     messages.MessagePlayerJoin{User: "improve"},
		})
		firstJoinUpdate := <-suite.playerUpdates
		actor.HandleIncomingMessage(acting.ActorOutgoingMessage{
			MessageType: messages.MessageTypePlayerLeave,
			Content:     messages.MessagePlayerLeave{Player: "improve"},
		})
		leaveUpdate := <-suite.playerUpdates
		// Now we pass the caught updates in the update channel again, and we are done.
		updateChanReady.Done()
		go func() {
			defer wg.Done()
			suite.playerUpdates <- firstJoinUpdate
			suite.playerUpdates <- leaveUpdate
		}()
		actor.HandleIncomingMessage(acting.ActorOutgoingMessage{
			MessageType: messages.MessageTypePlayerJoin,
			Content:     messages.MessagePlayerJoin{User: "improve"},
		})
	}()

	suite.office.Open(suite.ctx, actor)
	updateChanReady.Wait()
	err := suite.closeOffice(3)
	wg.Wait()
	suite.Require().Nilf(err, "should not fail but got: %s", errors.Prettify(err))
	suite.Require().NotNil(actor.MessageCollector.AssureOutgoingMessageTypes(false, messages.MessageTypeError),
		"actor should have no outgoing error messages")
}

func TestPlayerJoinOffice(t *testing.T) {
	suite.Run(t, new(PlayerJoinOfficeTestSuite))
}

type BroadcastPlayerManagementUpdateTestSuite struct {
	suite.Suite
	playerProvider *MockPlayerProvider
}

func (suite *BroadcastPlayerManagementUpdateTestSuite) SetupTest() {
	suite.playerProvider = NewMockPlayerProvider()
}

func (suite *BroadcastPlayerManagementUpdateTestSuite) TestUserRetrievalFail() {
	actor := acting.NewMockActor("")
	_, _ = actor.Hire("")

	err := BroadcastPlayerManagementUpdate(PlayerManagementUpdate{
		User:      "unknown-user",
		Team:      "team",
		HasJoined: true,
	}, suite.playerProvider, actor)
	suite.Require().NotNil(err, "should fail")
}

func (suite *BroadcastPlayerManagementUpdateTestSuite) TestSendFail() {
	actor := acting.NewMockActor("")
	_, _ = actor.Hire("")
	actor.SendErr = nativeerrors.New("sad life")

	err := BroadcastPlayerManagementUpdate(PlayerManagementUpdate{
		HasJoined: false,
	}, suite.playerProvider, actor)
	suite.Require().NotNil(err, "should fail")
}

func (suite *BroadcastPlayerManagementUpdateTestSuite) TestPlayerJoinedSingleOK() {
	actor := acting.NewMockActor("")
	_, _ = actor.Hire("")
	suite.playerProvider.SetPlayerKnown("hello-world")

	err := BroadcastPlayerManagementUpdate(PlayerManagementUpdate{
		User:      "hello-world",
		Team:      "team",
		HasJoined: true,
	}, suite.playerProvider, actor)
	suite.Require().Nilf(err, "should not fail but got: %s", errors.Prettify(err))
	suite.Require().Nilf(actor.MessageCollector.AssureOutgoingMessageTypes(false, messages.MessageTypePlayerJoined),
		"should have sent player join but did: %s", actor.MessageCollector)
}

func (suite *BroadcastPlayerManagementUpdateTestSuite) TestPlayerLeftSingleOK() {
	actor := acting.NewMockActor("")
	_, _ = actor.Hire("")
	suite.playerProvider.SetPlayerKnown("hello-world")

	err := BroadcastPlayerManagementUpdate(PlayerManagementUpdate{
		User:      "hello-world",
		Team:      "team",
		HasJoined: false,
	}, suite.playerProvider, actor)
	suite.Require().Nilf(err, "should not fail but got: %s", errors.Prettify(err))
	suite.Require().Nilf(actor.MessageCollector.AssureOutgoingMessageTypes(false, messages.MessageTypePlayerLeft),
		"should have sent player left but did: %s", actor.MessageCollector)
}

func (suite *BroadcastPlayerManagementUpdateTestSuite) TestMultiple() {
	timeout, cancelTimeout := context.WithTimeout(context.Background(), waitTimeout)

	// Prepare actors and make them waiting for one message.
	var actorMessageWaiters sync.WaitGroup
	actors := make([]acting.Actor, 0, 8)
	for i := 0; i < 8; i++ {
		actor := acting.NewMockActor("actor")
		_, _ = actor.Hire("actor")
		newsletter := actor.SubscribeOutgoingMessageType(messages.MessageTypePlayerLeft)
		actorMessageWaiters.Add(1)
		go func() {
			defer actorMessageWaiters.Done()
			select {
			case <-timeout.Done():
				suite.Fail("timeout", "timeout while waiting for message to receive")
			case <-newsletter.Receive:
				_ = actor.UnsubscribeOutgoing(newsletter.Subscription)
			}
		}()
		actors = append(actors, actor)
	}

	// Perform broadcast.
	err := BroadcastPlayerManagementUpdate(PlayerManagementUpdate{
		User:      "hello-world",
		Team:      "team-bla",
		HasJoined: false,
	}, nil, actors...)
	actorMessageWaiters.Wait()
	cancelTimeout()
	suite.Require().Nilf(err, "should not fail but got: %s", errors.Prettify(err))
}

func TestBroadcastPlayerManagementUpdate(t *testing.T) {
	suite.Run(t, new(BroadcastPlayerManagementUpdateTestSuite))
}

type BroadcastReadyStateUpdateTestSuite struct {
	suite.Suite
}

func (suite *BroadcastReadyStateUpdateTestSuite) TestSendFail() {
	actor := acting.NewMockActor("")
	_, _ = actor.Hire("")
	actor.SendErr = nativeerrors.New("sad life")

	err := BroadcastReadyStateUpdate(ReadyStateUpdate{
		IsEverybodyReady: true,
		ActorStates:      []ReadyStateUpdateActorState{},
	}, actor)
	suite.Require().NotNil(err, "should fail")
}

func (suite *BroadcastReadyStateUpdateTestSuite) TestSingleOK() {
	actor := acting.NewMockActor("")
	_, _ = actor.Hire("")

	err := BroadcastReadyStateUpdate(ReadyStateUpdate{
		IsEverybodyReady: true,
		ActorStates: []ReadyStateUpdateActorState{
			{
				Actor:   acting.ActorRepresentation{ID: "hello-world", Name: "yeah"},
				IsReady: true,
			},
			{
				Actor:   acting.ActorRepresentation{ID: "hot", Name: "shield"},
				IsReady: true,
			},
		},
	}, actor)
	suite.Require().Nilf(err, "should not fail but got: %s", errors.Prettify(err))
	suite.Require().Nilf(actor.MessageCollector.AssureOutgoingMessageTypes(false, messages.MessageTypeReadyStateUpdate),
		"should have sent ready-state update but did: %s", actor.MessageCollector)
}

func (suite *BroadcastReadyStateUpdateTestSuite) TestMultiple() {
	timeout, cancelTimeout := context.WithTimeout(context.Background(), waitTimeout)

	// Prepare actors and make them waiting for one message.
	var actorMessageWaiters sync.WaitGroup
	actors := make([]acting.Actor, 0, 8)
	for i := 0; i < 8; i++ {
		actor := acting.NewMockActor("actor")
		_, _ = actor.Hire("actor")
		newsletter := actor.SubscribeOutgoingMessageType(messages.MessageTypeReadyStateUpdate)
		actorMessageWaiters.Add(1)
		go func() {
			defer actorMessageWaiters.Done()
			select {
			case <-timeout.Done():
				suite.Fail("timeout", "timeout while waiting for message to receive")
			case <-newsletter.Receive:
				_ = actor.UnsubscribeOutgoing(newsletter.Subscription)
			}
		}()
		actors = append(actors, actor)
	}

	// Perform broadcast.
	err := BroadcastReadyStateUpdate(ReadyStateUpdate{
		IsEverybodyReady: false,
		ActorStates: []ReadyStateUpdateActorState{
			{
				Actor:   acting.ActorRepresentation{ID: "hello-world", Name: "yeah"},
				IsReady: false,
			},
			{
				Actor:   acting.ActorRepresentation{ID: "hot", Name: "shield"},
				IsReady: true,
			},
		},
	}, actors...)
	actorMessageWaiters.Wait()
	cancelTimeout()
	suite.Require().Nilf(err, "should not fail but got: %s", errors.Prettify(err))
}

func TestBroadcastReadyStateUpdate(t *testing.T) {
	suite.Run(t, new(BroadcastReadyStateUpdateTestSuite))
}
