package acting

import (
	"context"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/messages"
	"github.com/stretchr/testify/suite"
	"sync"
	"testing"
)

type SubscriptionManagerTestSuite struct {
	suite.Suite
	m *subscriptionManager
}

func (suite *SubscriptionManagerTestSuite) SetupTest() {
	suite.m = newSubscriptionManager()
}

func (suite *SubscriptionManagerTestSuite) TestNew() {
	suite.Assert().NotNil(suite.m.subscriptionsByToken, "subscriptions by token should be created")
	suite.Assert().NotNil(suite.m.subscriptionsByMessageType, "subscriptions by message type should be created")
}

func (suite *SubscriptionManagerTestSuite) TestSubscribeOK() {
	messageType := messages.MessageTypeHello
	messagesToSend := int32(5)
	cMessages, _ := suite.m.subscribeMessageType(messageType)
	// Check if receiving works.
	ctx, cancel := context.WithTimeout(context.Background(), waitTimeout)
	go func() {
		received := int32(0)
		for range cMessages {
			received++
		}
		if received != messagesToSend {
			suite.Failf("incorrect messages received", "should have received %d messages but got: %d", messagesToSend, received)
		}
		cancel()
	}()
	// Send messages.
	for i := 0; i < int(messagesToSend); i++ {
		forwarded := suite.m.handleMessage(ActorIncomingMessage{MessageType: messageType})
		suite.Assert().Equal(1, forwarded, "should have forwarded one message")
	}
	suite.m.cancelAllSubscriptions()
	<-ctx.Done()
	if ctx.Err() == context.DeadlineExceeded {
		suite.Fail("timeout", "timeout while waiting for all messages being received")
	}
}

func (suite *SubscriptionManagerTestSuite) TestUnsubscribeNotFound() {
	err := suite.m.unsubscribe(100)
	suite.Assert().NotNil(err, "unsubscribe should fail")
}

func (suite *SubscriptionManagerTestSuite) TestUnsubscribeOK() {
	messageType := messages.MessageTypeHello
	// Subscribe.
	_, sub := suite.m.subscribeMessageType(messageType)
	// Unsubscribe.
	err := suite.m.unsubscribe(sub)
	suite.Require().Nilf(err, "unsubscribe should not fail but got: %s", errors.Prettify(err))
	// Try to send message, and we expect it not to block.
	ctx, cancel := context.WithTimeout(context.Background(), waitTimeout)
	go func() {
		forwarded := suite.m.handleMessage(ActorIncomingMessage{MessageType: messageType})
		suite.Assert().Equal(0, forwarded, "should have forwarded to no one")
		cancel()
	}()
	<-ctx.Done()
	suite.Assert().Equal(context.Canceled, ctx.Err(), "timeout should have been canceled and not exceeded")
}

func (suite *SubscriptionManagerTestSuite) TestCancelAllSubscriptions() {
	messageType := messages.MessageTypeHello
	subscriberCount := 3
	var wg sync.WaitGroup
	// Make all subscribe.
	wg.Add(subscriberCount)
	for i := 0; i < subscriberCount; i++ {
		sub, _ := suite.m.subscribeMessageType(messageType)
		go func() {
			defer wg.Done()
			for range sub {
			}
		}()
	}
	ctx, cancel := context.WithTimeout(context.Background(), waitTimeout)
	go func() {
		wg.Wait()
		cancel()
	}()
	// Unsubscribe all.
	suite.m.cancelAllSubscriptions()
	<-ctx.Done()
	suite.Assert().Equal(context.Canceled, ctx.Err(), "timeout should have been canceled and not exceeded")
}

func Test_subscriptionManager(t *testing.T) {
	suite.Run(t, new(SubscriptionManagerTestSuite))
}
