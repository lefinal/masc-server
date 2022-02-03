package lightswitch

import (
	"context"
	"fmt"
	"github.com/LeFinal/masc-server/acting"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/logging"
	"github.com/LeFinal/masc-server/messages"
	"github.com/LeFinal/masc-server/stores"
	"github.com/google/uuid"
	"sync"
)

type actorReception struct {
	// agency from which new actors are subscribed.
	agency acting.Agency
	// manager is the Manager that is used for management operations.
	manager Manager
	ctx     context.Context
}

// RunActorReception runs an acting.ActorNewsletterRecipient for the given
// acting.Agency that handles messages using the given Manager.
func RunActorReception(ctx context.Context, agency acting.Agency, manager Manager) {
	r := &actorReception{
		agency:  agency,
		manager: manager,
		ctx:     ctx,
	}
	r.agency.SubscribeNewActors(r)
	<-ctx.Done()
	r.agency.UnsubscribeNewActors(r)
}

func (r *actorReception) HandleNewActor(actor acting.Actor, role acting.RoleType) {
	switch role {
	case acting.RoleTypeLightSwitchManager:
		go runHandler(&actorManagerHandler{
			Actor:   actor,
			manager: r.manager,
		}, "light-switch-manager")
	case acting.RoleTypeLightSwitchProvider:
		go runHandler(&actorProviderHandler{
			Actor:   actor,
			manager: r.manager,
			ctx:     r.ctx,
		}, "light-switch-provider")
	}
}

func runHandler(actor acting.Actor, displayedNamePrefix string) {
	err := actor.Hire(fmt.Sprintf("%s%s", displayedNamePrefix, uuid.New().String()))
	if err != nil {
		errors.Log(logging.LightSwitchLogger, errors.Wrap(err, "hire", nil))
		return
	}
}

// actorManagerHandler handles all stuff for actors with
// acting.RoleTypeLightSwitchManager.
type actorManagerHandler struct {
	acting.Actor
	// manager is used for retrieving and managing light switches.
	manager Manager
}

func (h *actorManagerHandler) Hire(displayedName string) error {
	err := h.Actor.Hire(displayedName)
	if err != nil {
		return errors.Wrap(err, "hire actor", nil)
	}
	// Message handlers.
	go func() {
		newsletter := acting.SubscribeNotifyForMessageType(messages.MessageTypeGetLightSwitches, h)
		for range newsletter.Receive {
			h.handleGetLightSwitches()
		}
	}()
	go func() {
		newsletter := acting.SubscribeMessageTypeDeleteLightSwitch(h)
		for m := range newsletter.Receive {
			h.handleDeleteLightSwitch(m)
		}
	}()
	go func() {
		newsletter := acting.SubscribeMessageTypeUpdateLightSwitch(h)
		for m := range newsletter.Receive {
			h.handleUpdateLightSwitch(m)
		}
	}()
	go func() {
		newsletter := acting.SubscribeMessageTypeGetFixtures(h)
		for range newsletter.Receive {
			h.handleGetFixtures()
		}
	}()
	go func() {
		// Read from quit channel as this would block otherwise somewhere else.
		<-h.Quit()
	}()
	return nil
}

// handleGetLightSwitches handles an incoming message with type
// messages.MessageTypeGetLightSwitches.
func (h *actorManagerHandler) handleGetLightSwitches() {
	// Respond with all light switches.
	lightSwitches, err := h.manager.LightSwitches()
	if err != nil {
		acting.LogErrorAndSendOrLog(logging.AppLogger, h, errors.Wrap(err, "light switches", nil))
		return
	}
	res := messages.MessageLightSwitchList{
		LightSwitches: lightSwitches,
	}
	acting.SendOrLogError(h, acting.ActorOutgoingMessage{
		MessageType: messages.MessageTypeLightSwitchList,
		Content:     res,
	})
}

// handleUpdateLightSwitches handles messages with
// messages.MessageTypeUpdateLightSwitch.
func (h *actorManagerHandler) handleUpdateLightSwitch(message messages.MessageUpdateLightSwitch) {
	err := h.manager.UpdateLightSwitchByID(message.LightSwitchID, stores.EditLightSwitch{
		Name:        message.Name,
		Assignments: message.Assignments,
	})
	if err != nil {
		acting.LogErrorAndSendOrLog(logging.LightSwitchLogger, h, errors.Wrap(err, "update light switch by id",
			errors.Details{
				"light_switch_id": message.LightSwitchID,
				"message":         message,
			},
		))
		return
	}
	acting.SendOKOrLogError(h)
}

// handleDeleteLightSwitch handles messages with
// messages.MessageTypeDeleteLightSwitch.
func (h *actorManagerHandler) handleDeleteLightSwitch(message messages.MessageDeleteLightSwitch) {
	err := h.manager.DeleteLightSwitchByID(message.LightSwitchID)
	if err != nil {
		acting.LogErrorAndSendOrLog(logging.LightSwitchLogger, h, errors.Wrap(err, "delete light switch by id",
			errors.Details{"light_switch_id": message.LightSwitchID}))
		return
	}
	acting.SendOKOrLogError(h)
}

// handleGetFixtures handles messages with messages.MessageTypeGetFixtures.
func (h *actorManagerHandler) handleGetFixtures() {
	fixtures, err := h.manager.GetFixtures()
	if err != nil {
		acting.LogErrorAndSendOrLog(logging.LightSwitchLogger, h, errors.Wrap(err, "get fixtures", nil))
		return
	}
	res := messages.MessageFixtureList{
		Fixtures: make([]messages.Fixture, 0, len(fixtures)),
	}
	for _, fixture := range fixtures {
		res.Fixtures = append(res.Fixtures, messages.Fixture{
			ID:         fixture.ID,
			DeviceID:   fixture.Device,
			ProviderID: fixture.ProviderID,
			Type:       fixture.Type,
			Name:       fixture.Name,
			LastSeen:   fixture.LastSeen,
		})
	}
	acting.SendOrLogError(h, acting.ActorOutgoingMessage{
		MessageType: messages.MessageTypeFixtureList,
		Content:     res,
	})
}

// actorProviderHandler handles all stuff for actors with
// acting.RoleTypeLightSwitchProvider.
type actorProviderHandler struct {
	acting.Actor
	// manager is used for registering light switches.
	manager Manager
	ctx     context.Context
}

func (h *actorProviderHandler) Hire(displayedName string) error {
	err := h.Actor.Hire(displayedName)
	if err != nil {
		return errors.Wrap(err, "hire actor", nil)
	}
	lightSwitchProviderAlive, killLightSwitchProvider := context.WithCancel(h.ctx)
	var wg sync.WaitGroup
	// Register.
	wg.Add(1)
	go func() {
		defer wg.Done()
		err := h.manager.AcceptLightSwitchProvider(lightSwitchProviderAlive, h)
		if err != nil {
			acting.LogErrorAndSendOrLog(logging.LightSwitchLogger, h,
				errors.Wrap(err, "accept light switch provider", nil))
			// Fire.
			err = h.Fire()
			if err != nil {
				errors.Log(logging.LightSwitchLogger, errors.Wrap(err, "fire", nil))
			}
			return
		}
	}()
	select {
	case <-h.ctx.Done():
	case <-h.Quit():
	}
	killLightSwitchProvider()

	// No message handlers needed. We also do NOT read from the quit channel because
	// the manager will do so.
	return nil
}
