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
	m *SubscriptionManager
}

func (suite *SubscriptionManagerTestSuite) SetupTest() {
	suite.m = NewSubscriptionManager()
}

func (suite *SubscriptionManagerTestSuite) TestNew() {
	suite.Assert().NotNil(suite.m.subscriptionsByToken, "subscriptions by token should be created")
	suite.Assert().NotNil(suite.m.subscriptionsByMessageType, "subscriptions by message type should be created")
}

func (suite *SubscriptionManagerTestSuite) TestSubscribeOK() {
	messageType := messages.MessageTypeHello
	messagesToSend := 5
	sub := suite.m.SubscribeMessageType(messageType)
	// Check if receiving works.
	ctx, cancel := context.WithTimeout(context.Background(), waitTimeout)
	var wg sync.WaitGroup
	wg.Add(messagesToSend)
	go func() {
		received := 0
		for {
			select {
			case <-sub.Ctx.Done():
				if received != messagesToSend {
					suite.Failf("incorrect messages received", "should have received %d messages but got: %d", messagesToSend, received)
				}
				cancel()
			case <-sub.out:
				received++
				wg.Done()
			}
		}
	}()
	// Send messages.
	for i := 0; i < int(messagesToSend); i++ {
		forwarded := suite.m.HandleMessage(Message{MessageType: messageType})
		suite.Assert().Equal(1, forwarded, "should have forwarded one message")
	}
	wg.Wait()
	suite.m.CancelAllSubscriptions()
	<-ctx.Done()
	if ctx.Err() == context.DeadlineExceeded {
		suite.Fail("timeout", "timeout while waiting for all messages being received")
	}
}

func (suite *SubscriptionManagerTestSuite) TestUnsubscribeNotFound() {
	sub := suite.m.SubscribeMessageType(messages.MessageTypeHello)
	sub.token = 200
	err := suite.m.Unsubscribe(sub)
	suite.Assert().NotNil(err, "unsubscribe should fail")
}

func (suite *SubscriptionManagerTestSuite) TestUnsubscribeOK() {
	messageType := messages.MessageTypeHello
	// Subscribe.
	sub := suite.m.SubscribeMessageType(messageType)
	// Unsubscribe.
	err := suite.m.Unsubscribe(sub)
	suite.Require().Nilf(err, "unsubscribe should not fail but got: %s", errors.Prettify(err))
	// Try to send message, and we expect it not to block.
	ctx, cancel := context.WithTimeout(context.Background(), waitTimeout)
	go func() {
		forwarded := suite.m.HandleMessage(Message{MessageType: messageType})
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
	ctx, cancel := context.WithTimeout(context.Background(), waitTimeout)
	for i := 0; i < subscriberCount; i++ {
		sub := suite.m.SubscribeMessageType(messageType)
		go func() {
			defer wg.Done()
			select {
			case <-ctx.Done():
				suite.Fail("timeout", "timeout while waiting for subscription to be inactive")
			case <-sub.Ctx.Done():
				// Yay.
			}
		}()
	}
	// Unsubscribe all.
	suite.m.CancelAllSubscriptions()
	wg.Wait()
	cancel()
	suite.Assert().Equal(context.Canceled, ctx.Err(), "timeout should have been canceled and not exceeded")
}

func Test_subscriptionManager(t *testing.T) {
	suite.Run(t, new(SubscriptionManagerTestSuite))
}
