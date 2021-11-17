package lighting

import (
	"context"
	"fmt"
	"github.com/LeFinal/masc-server/acting"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/logging"
	"github.com/LeFinal/masc-server/messages"
	"sync"
)

// FixtureOperatorHandlers implements acting.ActorNewsletterRecipient for
// handling new actors with acting.RoleTypeFixtureOperator.
type FixtureOperatorHandlers struct {
	// agency from which new actors are subscribed.
	agency acting.Agency
	// manager is the Manager that is used for performing operations.
	manager Manager
	// activeManagers holds all active provider handlers.
	activeManagers map[*fixtureOperatorHandler]struct{}
	// managerCounter is a counter that is incremented for each new manager in
	// activeManagers in order to set the displayed name.
	managerCounter int
	// m locks activeManagers and managerCounter.
	m   sync.Mutex
	ctx context.Context
}

// NewFixtureOperatorHandlers creates a new ManagementHandlers that can
// be run via ManagementHandlers.Run.
//
// **Warning**: This will read from Manager.GetFixtureStatesBroadcast so be sure
// that no one else reads from it!
func NewFixtureOperatorHandlers(agency acting.Agency, manager Manager) *FixtureOperatorHandlers {
	return &FixtureOperatorHandlers{
		agency:         agency,
		manager:        manager,
		activeManagers: make(map[*fixtureOperatorHandler]struct{}),
	}
}

// Run the handler. It subscribes to the agency and unsubscribes when the given
// context.Context is done.
func (dm *FixtureOperatorHandlers) Run(ctx context.Context) {
	dm.ctx = ctx
	dm.agency.SubscribeNewActors(dm)
	go dm.handleFixtureStateBroadcastUpdates(ctx)
	<-ctx.Done()
	dm.agency.UnsubscribeNewActors(dm)
}

// handleFixtureStateBroadcastUpdates reads from
// Manager.GetFixtureStatesBroadcast and broadcasts updates to all registered
// handlers.
func (dm *FixtureOperatorHandlers) handleFixtureStateBroadcastUpdates(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
		case m := <-dm.manager.GetFixtureStatesBroadcast():
			dm.m.Lock()
			for handler := range dm.activeManagers {
				acting.SendOrLogError(logging.ActingLogger, handler.Actor, acting.ActorOutgoingMessage{
					MessageType: messages.MessageTypeFixtureStates,
					Content:     m,
				})
			}
			dm.m.Unlock()
		}
	}
}

func (dm *FixtureOperatorHandlers) HandleNewActor(actor acting.Actor, role acting.RoleType) {
	if role != acting.RoleTypeFixtureOperator {
		return
	}
	actorDM := &fixtureOperatorHandler{
		Actor:   actor,
		manager: dm.manager,
		ctx:     dm.ctx,
	}
	// Add to active ones.
	dm.m.Lock()
	dm.activeManagers[actorDM] = struct{}{}
	dm.managerCounter++
	dm.m.Unlock()
	// Hire.
	err := actorDM.Hire(fmt.Sprintf("fixture-operator-%d", dm.managerCounter))
	if err != nil {
		errors.Log(logging.AppLogger, errors.Wrap(err, "hire"))
		return
	}
	<-actorDM.Quit()
	// Remove from active ones.
	dm.m.Lock()
	delete(dm.activeManagers, actorDM)
	dm.m.Unlock()
}

// fixtureOperatorHandler is an Actor that implements handling for
// acting.RoleTypeFixtureOperator.
type fixtureOperatorHandler struct {
	acting.Actor
	// manager is used for retrieving and managing fixtures.
	manager Manager
	ctx     context.Context
}

func (a *fixtureOperatorHandler) Hire(displayedName string) error {
	// Hire normally.
	err := a.Actor.Hire(displayedName)
	if err != nil {
		return errors.Wrap(err, "hire actor")
	}
	// Message handlers.
	go func() {
		newsletter := acting.SubscribeMessageTypeGetFixtureStates(a)
		for range newsletter.Receive {
			a.handleGetFixtureStates()
		}
	}()
	go func() {
		newsletter := acting.SubscribeMessageTypeSetFixturesEnabled(a)
		for m := range newsletter.Receive {
			a.handleSetFixturesEnabled(m)
		}
	}()
	go func() {
		newsletter := acting.SubscribeMessageTypeSetFixturesLocating(a)
		for m := range newsletter.Receive {
			a.handleSetFixturesLocating(m)
		}
	}()
	return nil
}

func (a *fixtureOperatorHandler) handleGetFixtureStates() {
	acting.SendOrLogError(logging.ActingLogger, a.Actor, acting.ActorOutgoingMessage{
		MessageType: messages.MessageTypeFixtureStates,
		Content:     a.manager.FixtureStates(),
	})
}

// handleSetFixturesEnabled handles a received
// messages.MessageTypeSetFixturesEnabled message.
func (a *fixtureOperatorHandler) handleSetFixturesEnabled(message messages.MessageSetFixturesEnabled) {
	// Create map of fixtures for easy access.
	fixtureMap := make(map[messages.FixtureID]Fixture)
	for _, fixture := range a.manager.Fixtures() {
		fixtureMap[fixture.ID()] = fixture
	}
	// Apply updates.
	for _, newFixtureState := range message.Fixtures {
		fixture, ok := fixtureMap[newFixtureState.FixtureID]
		if !ok {
			acting.SendOrLogError(logging.ActingLogger, a.Actor, acting.ActorErrorMessageFromError(errors.Error{
				Code:    errors.ErrBadRequest,
				Kind:    errors.KindUnknownFixture,
				Message: fmt.Sprintf("fixture %v not found", newFixtureState.FixtureID),
				Details: errors.Details{"fixture": newFixtureState.FixtureID},
			}))
			return
		}
		fixture.SetEnabled(newFixtureState.IsEnabled)
		err := fixture.Apply()
		if err != nil {
			err = errors.Wrap(err, "apply update after setting enabled state")
			errors.Log(logging.LightingLogger, err)
			acting.SendOrLogError(logging.ActingLogger, a.Actor, acting.ActorErrorMessageFromError(err))
			return
		}
	}
	acting.SendOKOrLogError(logging.ActingLogger, a.Actor)
}

// handleSetFixturesLocating handles a received
// messages.MessageTypeSetFixturesLocating message.
func (a *fixtureOperatorHandler) handleSetFixturesLocating(message messages.MessageSetFixturesLocating) {
	// Create map of fixtures for easy access.
	fixtureMap := make(map[messages.FixtureID]Fixture)
	for _, fixture := range a.manager.Fixtures() {
		fixtureMap[fixture.ID()] = fixture
	}
	// Apply updates.
	for _, newFixtureState := range message.Fixtures {
		fixture, ok := fixtureMap[newFixtureState.FixtureID]
		if !ok {
			acting.SendOrLogError(logging.ActingLogger, a.Actor, acting.ActorErrorMessageFromError(errors.Error{
				Code:    errors.ErrBadRequest,
				Kind:    errors.KindUnknownFixture,
				Message: fmt.Sprintf("fixture %v not found", newFixtureState.FixtureID),
				Details: errors.Details{"fixture": newFixtureState.FixtureID},
			}))
			return
		}
		fixture.SetLocating(newFixtureState.IsLocating)
		err := fixture.Apply()
		if err != nil {
			err = errors.Wrap(err, "apply update after setting locating-mode")
			errors.Log(logging.LightingLogger, err)
			acting.SendOrLogError(logging.ActingLogger, a.Actor, acting.ActorErrorMessageFromError(err))
			return
		}
	}
	acting.SendOKOrLogError(logging.ActingLogger, a.Actor)
}
