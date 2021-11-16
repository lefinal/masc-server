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

// ManagementHandlers implements acting.ActorNewsletterRecipient for handling
// new actors with acting.RoleTypeFixtureManager.
type ManagementHandlers struct {
	// agency from which new actors are subscribed.
	agency acting.Agency
	// manager is the Manager that is used for management operations.
	manager Manager
	// activeManagers holds all active management handlers.
	activeManagers map[*managementHandler]struct{}
	// managerCounter is a counter that is incremented for each new manager in
	// activeManagers in order to set the displayed name.
	managerCounter int
	// m locks activeManagers and managerCounter.
	m sync.Mutex
}

// NewManagementHandlers creates a new ManagementHandlers that can
// be run via ManagementHandlers.Run.
func NewManagementHandlers(agency acting.Agency, manager Manager) *ManagementHandlers {
	return &ManagementHandlers{
		agency:         agency,
		manager:        manager,
		activeManagers: make(map[*managementHandler]struct{}),
	}
}

// Run the handler. It subscribes to the agency and unsubscribes when the given
// context.Context is done.
func (dm *ManagementHandlers) Run(ctx context.Context) {
	dm.agency.SubscribeNewActors(dm)
	<-ctx.Done()
	dm.agency.UnsubscribeNewActors(dm)
}

func (dm *ManagementHandlers) HandleNewActor(actor acting.Actor, role acting.RoleType) {
	if role != acting.RoleTypeFixtureManager {
		return
	}
	actorDM := &managementHandler{
		Actor:   actor,
		manager: dm.manager,
	}
	// Add to active ones.
	dm.m.Lock()
	dm.activeManagers[actorDM] = struct{}{}
	dm.managerCounter++
	dm.m.Unlock()
	// Hire.
	err := actorDM.Hire(fmt.Sprintf("fixture-manager-%d", dm.managerCounter))
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

// managementHandler is an Actor that implements handling for
// acting.RoleTypeFixtureManager.
type managementHandler struct {
	acting.Actor
	// manager is used for retrieving and managing fixtures.
	manager Manager
}

func (a *managementHandler) Hire(displayedName string) error {
	// Hire normally.
	err := a.Actor.Hire(displayedName)
	if err != nil {
		return errors.Wrap(err, "hire actor")
	}
	// Setup message handlers. We do not need to unsubscribe because this will be
	// done when the actor is fired. Handle device retrieval.
	go func() {
		newsletter := acting.SubscribeMessageTypeSetFixtureName(a)
		for message := range newsletter.Receive {
			a.handleSetFixtureName(message)
		}
	}()
	go func() {
		newsletter := acting.SubscribeMessageTypeGetFixtures(a)
		for range newsletter.Receive {
			a.handleGetFixtures()
		}
	}()
	go func() {
		newsletter := acting.SubscribeMessageTypeDeleteFixture(a)
		for message := range newsletter.Receive {
			a.handleDeleteFixture(message)
		}
	}()
	return nil
}

// handleSetFixtureName handles an incoming message with type
// messages.MessageTypeSetFixtureName.
func (a *managementHandler) handleSetFixtureName(message messages.MessageSetFixtureName) {
	err := a.manager.SetFixtureName(message.FixtureID, message.Name)
	if err != nil {
		acting.SendOrLogError(logging.ActingLogger, a, acting.ActorErrorMessageFromError(errors.Wrap(err, "set fixture name")))
		return
	}
	acting.SendOKOrLogError(logging.ActingLogger, a)
}

// handleGetFixtures handles an incoming message with type
// messages.MessageTypeGetFixtures.
func (a *managementHandler) handleGetFixtures() {
	// Respond with all devices.
	fixtures := a.manager.Fixtures()
	res := messages.MessageFixtureList{
		Fixtures: make([]messages.Fixture, len(fixtures)),
	}
	for i, fixture := range fixtures {
		res.Fixtures[i] = messages.Fixture{
			ID:         fixture.ID(),
			DeviceID:   fixture.DeviceID(),
			ProviderID: fixture.ProviderID(),
			IsEnabled:  fixture.IsEnabled(),
			Type:       fixture.Type(),
			Name:       fixture.Name(),
			Features:   fixture.Features(),
			IsLocating: fixture.IsLocating(),
			IsOnline:   fixture.IsOnline(),
			LastSeen:   fixture.LastSeen(),
		}
	}
	acting.SendOrLogError(logging.AppLogger, a, acting.ActorOutgoingMessage{
		MessageType: messages.MessageTypeFixtureList,
		Content:     res,
	})
}

// handleDeleteFixture handles an incoming message with type
// messages.MessageTypeDeleteFixture.
func (a *managementHandler) handleDeleteFixture(message messages.MessageDeleteFixture) {
	err := a.manager.DeleteFixture(message.FixtureID)
	if err != nil {
		acting.SendOrLogError(logging.ActingLogger, a, acting.ActorErrorMessageFromError(errors.Wrap(err, "delete fixture")))
		return
	}
	acting.SendOKOrLogError(logging.ActingLogger, a)
}
