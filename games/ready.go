package games

import (
	"context"
	"github.com/LeFinal/masc-server/acting"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/messages"
	"sync"
)

// ReadyStateUpdate holds information regarding the ready-state of actors.
type ReadyStateUpdate struct {
	// IsEverybodyReady describes whether all actors are ready.
	IsEverybodyReady bool
	// ActorStates holds the individual ready-states for the actors.
	ActorStates []ReadyStateUpdateActorState
}

func (update *ReadyStateUpdate) Message() messages.MessageReadyStateUpdate {
	actorStates := make([]messages.ReadyStateUpdateActorState, 0, len(update.ActorStates))
	for _, state := range update.ActorStates {
		actorStates = append(actorStates, messages.ReadyStateUpdateActorState{
			Actor:   state.Actor.Message(),
			IsReady: state.IsReady,
		})
	}
	return messages.MessageReadyStateUpdate{
		IsEverybodyReady: update.IsEverybodyReady,
		ActorStates:      actorStates,
	}
}

// ReadyStateUpdateActorState is the ready-state for a specific Actor.
type ReadyStateUpdateActorState struct {
	// Actor is the actor the ready state is for.
	Actor acting.ActorRepresentation
	// IsReady describes whether the actor is ready.
	IsReady bool
}

// readyAwaiter holds shared fields for requesting ready-state from actors.
type readyAwaiter struct {
	// isAcceptingReadyStateUpdates is false when all ready-states are accepted and no changes are
	// allowed. This is needed in order to drop "late" ready-state updates.
	isAcceptingReadyStateUpdates bool
	// modeMutex locks isAcceptingReadyStateUpdates.
	modeMutex sync.Mutex
	// readyStateUpdates is the channel to send ready state updates to.
	readyStateUpdates chan<- ReadyStateUpdate
	// readyStates holds the actors with their ready-states.
	readyStates map[acting.Actor]bool
	// readyStatesMutex locks readyStates.
	readyStatesMutex sync.Mutex
	// originalErr is the first error that occurred during ready-state requests for
	// actors.
	originalErr error
	// requests are the ongoing ready-state requests.
	requests sync.WaitGroup
	// done receives when everybody is ready.
	done chan struct{}
}

// RequestAndAwaitReady sends message with messages.MessageTypeAreYouReady to
// all given actors. It awaits their ready-states until everybody is ready.
// Actor quits are NOT handled. Actors can send messages with
// messages.MessageTypeReadyState freely and how often they want until everybody
// is ready. Then each actor is sent a messages.MessageTypeReadyAccepted.
func RequestAndAwaitReady(ctx context.Context, readyStateUpdates chan<- ReadyStateUpdate,
	actors ...acting.Actor) error {
	// Check if there are even actors as then we can skip.
	if len(actors) == 0 {
		return nil
	}
	ra := &readyAwaiter{
		isAcceptingReadyStateUpdates: true,
		readyStates:                  make(map[acting.Actor]bool),
		readyStateUpdates:            readyStateUpdates,
		done:                         make(chan struct{}),
	}
	// Start for each actor.
	readyStateRequests, cancelRequests := context.WithCancel(context.Background())
	ra.readyStatesMutex.Lock()
	defer ra.requests.Wait()
	for _, actor := range actors {
		ra.requests.Add(1)
		// Set initial ready-state to not ready.
		ra.readyStates[actor] = false
		go ra.areYouReadyForActor(readyStateRequests, actor)
	}
	ra.readyStatesMutex.Unlock()
	// Wait until completion.
	select {
	case <-ctx.Done():
		// Cancelled.
		cancelRequests()
		return errors.NewContextAbortedError("wait until everybody ready")
	case <-ra.done:
		// Done because of everybody ready or error.
		cancelRequests()
		if ra.originalErr != nil {
			return ra.originalErr
		}
		return nil
	}
}

// handleReadyStateUpdate sends an update to readyAwaiter.readyStatesUpdated if
// the ready-state was not accepted yet. Otherwise, it does nothing in order to
// forget "late" ready-state updates.
func (ra *readyAwaiter) handleReadyStateUpdate(ctx context.Context) {
	ra.modeMutex.Lock()
	defer ra.modeMutex.Unlock()
	if !ra.isAcceptingReadyStateUpdates {
		// We drop the update.
		return
	}
	// Build status update and while building we also check if everybody is ready.
	isEverybodyReady := true
	ra.readyStatesMutex.Lock()
	actorStates := make([]ReadyStateUpdateActorState, 0, len(ra.readyStates))
	for actor, isReady := range ra.readyStates {
		if !isReady {
			isEverybodyReady = false
		}
		actorStates = append(actorStates, ReadyStateUpdateActorState{
			Actor: acting.ActorRepresentation{
				ID:   actor.ID(),
				Name: actor.Name(),
			},
			IsReady: isReady,
		})
	}
	ra.readyStatesMutex.Unlock()
	// Send status update.
	ra.readyStateUpdates <- ReadyStateUpdate{
		IsEverybodyReady: isEverybodyReady,
		ActorStates:      actorStates,
	}
	if !isEverybodyReady {
		return
	}
	// Accept ready.
	ra.isAcceptingReadyStateUpdates = false
	select {
	case <-ctx.Done():
	case ra.done <- struct{}{}:
	}
}

// handleError handles an occurred error during ready-state retrieval.
func (ra *readyAwaiter) handleError(ctx context.Context, err error) {
	ra.modeMutex.Lock()
	defer ra.modeMutex.Unlock()
	if !ra.isAcceptingReadyStateUpdates {
		return
	}
	if ra.originalErr != nil {
		// An error already occurred.
		return
	}
	ra.originalErr = err
	ra.isAcceptingReadyStateUpdates = false
	select {
	case <-ctx.Done():
	case ra.done <- struct{}{}:
	}
}

// areYouReadyForActor performs a ready-state request for the given
// acting.Actor. It sends the messages.MessageTypeAreYouReady to the given actor
// and awaits ready states. Updates are saved to readyAwaiter.readyStates and
// readyAwaiter.handleReadyStateUpdate is called. When the context is cancelled,
// the actor is sent a messages.MessageTypeReadyAccepted.
func (ra *readyAwaiter) areYouReadyForActor(readyStateRequest context.Context, actor acting.Actor) {
	defer ra.requests.Done()
	// Subscribe to ready-state updates.
	newsletter := acting.SubscribeMessageTypeReadyState(actor)
	defer acting.UnsubscribeOrLogError(newsletter.Newsletter)
	// Send ready-state request.
	err := actor.Send(acting.ActorOutgoingMessage{MessageType: messages.MessageTypeAreYouReady})
	if err != nil {
		ra.handleError(readyStateRequest, errors.Wrap(err, "send ready-state request to actor"))
		return
	}
	// Handle ready-state updates.
	for {
		select {
		case <-readyStateRequest.Done():
			err = actor.Send(acting.ActorOutgoingMessage{MessageType: messages.MessageTypeReadyAccepted})
			ra.handleError(readyStateRequest, errors.Wrap(err, "send ready accepted to actor"))
			return
		case message, more := <-newsletter.Receive:
			if !more {
				// We do NOT handle actor quits!
				return
			}
			ra.readyStatesMutex.Lock()
			ra.readyStates[actor] = message.IsReady
			ra.readyStatesMutex.Unlock()
			// Notify of ready-state update.
			ra.handleReadyStateUpdate(readyStateRequest)
		}
	}
}
