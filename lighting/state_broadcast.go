package lighting

import (
	"context"
	"github.com/LeFinal/masc-server/logging"
	"github.com/LeFinal/masc-server/messages"
	"sync"
	"time"
)

// FixtureStateUpdateNotifier is used for notifying of fixture-apply-updates.
type FixtureStateUpdateNotifier interface {
	HandleFixtureStateUpdated()
}

// fixtureStateBroadcast is used for broadcasting fixture updates with a certain
// buffer time in order to avoid information overload.
type fixtureStateBroadcaster struct {
	ctx context.Context
	// statesBroadcast is the channel where all fixture state updates are posted to.
	statesBroadcast chan messages.MessageFixtureStates
	// manager is the Manager where fixture states are retrieved from.
	manager Manager
	// bufferDuration is how long to wait until we publish fixture updates.
	bufferDuration time.Duration
	// performPublish is used for requesting the broadcaster to publish after the
	// buffer.
	performPublish chan struct{}
	// isPublishing is used when the buffer timer is running, or we are currently
	// publishing.
	isPublishing bool
	// isPublishingMutex locks isPublishing and ctx.
	isPublishingMutex sync.Mutex
}

// newFixtureStateBroadcaster creates a new fixtureStateBroadcaster that can be
// run via fixtureStateBroadcaster.run.
func newFixtureStateBroadcaster(manager Manager, bufferDuration time.Duration) *fixtureStateBroadcaster {
	return &fixtureStateBroadcaster{
		statesBroadcast: make(chan messages.MessageFixtureStates),
		manager:         manager,
		bufferDuration:  bufferDuration,
		performPublish:  make(chan struct{}),
		isPublishing:    false,
	}
}

// run actually runs the broadcaster. Updates are sent to statesBroadcast.
func (bc *fixtureStateBroadcaster) run(ctx context.Context) {
	bc.isPublishingMutex.Lock()
	bc.ctx = ctx
	bc.isPublishingMutex.Unlock()
	for {
		select {
		case <-ctx.Done():
		case <-bc.performPublish:
			bc.publishAfterBuffer()
		}
	}
}

// publishAfterBuffer calls publish after the bufferDuration has passed.
func (bc *fixtureStateBroadcaster) publishAfterBuffer() {
	timer := time.NewTimer(bc.bufferDuration)
	select {
	case <-bc.ctx.Done():
		if !timer.Stop() {
			<-timer.C
		}
	case <-timer.C:
		bc.publish()
	}
}

// publish actually publishes the state.
func (bc *fixtureStateBroadcaster) publish() {
	// Reset publishing state here as otherwise update calls might get lost during
	// channel send.
	bc.isPublishingMutex.Lock()
	bc.isPublishing = false
	bc.isPublishingMutex.Unlock()
	// Publish.
	select {
	case <-bc.ctx.Done():
		return
	case bc.statesBroadcast <- bc.manager.FixtureStates():
	}
}

func (bc *fixtureStateBroadcaster) HandleFixtureStateUpdated() {
	bc.isPublishingMutex.Lock()
	if bc.ctx == nil {
		bc.isPublishingMutex.Unlock()
		logging.LightingLogger.Warn("dropping fixture state update because of broadcaster not running")
		return
	}
	if bc.isPublishing {
		bc.isPublishingMutex.Unlock()
		return
	}
	bc.isPublishing = true
	bc.isPublishingMutex.Unlock()
	select {
	case <-bc.ctx.Done():
	case bc.performPublish <- struct{}{}:
	}
}
