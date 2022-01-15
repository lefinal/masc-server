package lighting

import (
	"context"
	"fmt"
	"github.com/LeFinal/masc-server/acting"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/logging"
	"sync"
)

// FixtureProviderHandlers implements acting.ActorNewsletterRecipient for
// handling new actors with acting.RoleTypeFixtureProvider.
type FixtureProviderHandlers struct {
	// agency from which new actors are subscribed.
	agency acting.Agency
	// manager is the Manager that is used for accepting new fixture providers.
	manager Manager
	// activeManagers holds all active provider handlers.
	activeManagers map[*fixtureProviderHandler]struct{}
	// managerCounter is a counter that is incremented for each new manager in
	// activeManagers in order to set the displayed name.
	managerCounter int
	// m locks activeManagers and managerCounter.
	m   sync.Mutex
	ctx context.Context
}

// NewFixtureProviderHandlers creates a new ManagementHandlers that can
// be run via ManagementHandlers.Run.
func NewFixtureProviderHandlers(agency acting.Agency, manager Manager) *FixtureProviderHandlers {
	return &FixtureProviderHandlers{
		agency:         agency,
		manager:        manager,
		activeManagers: make(map[*fixtureProviderHandler]struct{}),
	}
}

// Run the handler. It subscribes to the agency and unsubscribes when the given
// context.Context is done.
func (dm *FixtureProviderHandlers) Run(ctx context.Context) {
	dm.ctx = ctx
	dm.agency.SubscribeNewActors(dm)
	<-ctx.Done()
	dm.agency.UnsubscribeNewActors(dm)
}

// HandleNewActor is executed in its own goroutine, however, we cannot hire
// (will wait for fixture offers) as quits would then not be handled.
func (dm *FixtureProviderHandlers) HandleNewActor(actor acting.Actor, role acting.RoleType) {
	if role != acting.RoleTypeFixtureProvider {
		return
	}
	actorDM := &fixtureProviderHandler{
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
	err := actorDM.Hire(fmt.Sprintf("fixture-provider-%d", dm.managerCounter))
	if err != nil {
		errors.Log(logging.AppLogger, errors.Wrap(err, "hire", nil))
		return
	}
	<-actorDM.Quit()
	err = actorDM.cleanUp()
	if err != nil {
		errors.Log(logging.AppLogger, errors.Wrap(err, "clean up", nil))
	}
	// Remove from active ones.
	dm.m.Lock()
	delete(dm.activeManagers, actorDM)
	dm.m.Unlock()
}

// fixtureProviderHandler is an Actor that implements handling for
// acting.RoleTypeFixtureProvider.
type fixtureProviderHandler struct {
	acting.Actor
	// manager is used for retrieving and managing fixtures.
	manager Manager
	ctx     context.Context
}

func (a *fixtureProviderHandler) Hire(displayedName string) error {
	// Hire normally.
	err := a.Actor.Hire(displayedName)
	if err != nil {
		return errors.Wrap(err, "hire actor", nil)
	}
	// We do not need any message handlers but pass it directly to the manager.
	go func() {
		err = a.manager.AcceptFixtureProvider(a.ctx, a.Actor)
		if err != nil {
			err = errors.Wrap(err, "accept fixture provider", nil)
			errors.Log(logging.ActingLogger, err)
			acting.SendOrLogError(logging.ActingLogger, a.Actor, acting.ActorErrorMessageFromError(err))
			return
		}
	}()
	return nil
}

// cleanUp cleans up by unregistering from manager.
func (a *fixtureProviderHandler) cleanUp() error {
	err := a.manager.SayGoodbyeToFixtureProvider(a.Actor)
	if err != nil {
		return errors.Wrap(err, "unregister fixture provider", nil)
	}
	return nil
}
