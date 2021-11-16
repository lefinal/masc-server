package device_management

import (
	"context"
	"fmt"
	"github.com/LeFinal/masc-server/acting"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/gatekeeping"
	"github.com/LeFinal/masc-server/logging"
	"github.com/LeFinal/masc-server/messages"
	"sync"
)

// DeviceManagementHandlers implements acting.ActorNewsletterRecipient for
// handling new actors with acting.RoleTypeDeviceManager.
type DeviceManagementHandlers struct {
	// agency from which new actors are subscribed.
	agency acting.Agency
	// gatekeeper is the gatekeeping.Gatekeeper that is passed to every created
	// actorDeviceManager.
	gatekeeper gatekeeping.Gatekeeper
	// activeManagers holds all active device managers.
	activeManagers map[*actorDeviceManager]struct{}
	// managerCounter is a counter that is incremented for each new manager in
	// activeManagers in order to set the displayed name.
	managerCounter int
	// m locks activeManagers and managerCounter.
	m sync.Mutex
}

// NewDeviceManagementHandlers creates a new deviceManagementHandlers that can
// be run via DeviceManagementHandlers.Run.
func NewDeviceManagementHandlers(agency acting.Agency, gatekeeper gatekeeping.Gatekeeper) *DeviceManagementHandlers {
	return &DeviceManagementHandlers{
		agency:         agency,
		gatekeeper:     gatekeeper,
		activeManagers: make(map[*actorDeviceManager]struct{}),
	}
}

// Run the handler. It subscribes to the agency and unsubscribes when the given
// context.Context is done.
func (dm *DeviceManagementHandlers) Run(ctx context.Context) {
	dm.agency.SubscribeNewActors(dm)
	<-ctx.Done()
	dm.agency.UnsubscribeNewActors(dm)
}

func (dm *DeviceManagementHandlers) HandleNewActor(actor acting.Actor, role acting.RoleType) {
	if role != acting.RoleTypeDeviceManager {
		return
	}
	actorDM := &actorDeviceManager{
		Actor:      actor,
		gatekeeper: dm.gatekeeper,
	}
	// Add to active ones.
	dm.m.Lock()
	dm.activeManagers[actorDM] = struct{}{}
	dm.managerCounter++
	dm.m.Unlock()
	// Hire.
	err := actorDM.Hire(fmt.Sprintf("device-manager-%d", dm.managerCounter))
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

// actorDeviceManager is an Actor that implements handling for
// acting.RoleTypeDeviceManager.
type actorDeviceManager struct {
	acting.Actor
	// gatekeeper is used for retrieving and managing devices.
	gatekeeper gatekeeping.Gatekeeper
}

func (a *actorDeviceManager) Hire(displayedName string) error {
	// Hire normally.
	err := a.Actor.Hire(displayedName)
	if err != nil {
		return errors.Wrap(err, "hire actor")
	}
	// Setup message handlers. We do not need to unsubscribe because this will be
	// done when the actor is fired. Handle device retrieval.
	go func() {
		newsletter := acting.SubscribeMessageTypeGetDevices(a)
		for range newsletter.Receive {
			a.handleGetDevices()
		}
	}()
	go func() {
		newsletter := acting.SubscribeMessageTypeSetDeviceName(a)
		for message := range newsletter.Receive {
			a.handleSetDeviceName(message)
		}
	}()
	go func() {
		newsletter := acting.SubscribeMessageTypeDeleteDevice(a)
		for message := range newsletter.Receive {
			a.handleDeleteDevice(message)
		}
	}()
	return nil
}

// handleGetDevices handles an incoming message with type
// messages.MessageTypeGetDevices.
func (a *actorDeviceManager) handleGetDevices() {
	// Respond with all devices.
	devices, err := a.gatekeeper.GetDevices()
	if err != nil {
		acting.SendOrLogError(logging.ActingLogger, a, acting.ActorErrorMessageFromError(errors.Wrap(err, "get devices")))
		return
	}
	res := messages.MessageDeviceList{
		Devices: make([]messages.Device, len(devices)),
	}
	for i, device := range devices {
		res.Devices[i] = messages.Device{
			ID:          device.ID,
			Name:        device.Name,
			IsConnected: device.IsConnected,
			LastSeen:    device.LastSeen,
			Roles:       device.Roles,
		}
	}
	acting.SendOrLogError(logging.AppLogger, a, acting.ActorOutgoingMessage{
		MessageType: messages.MessageTypeDeviceList,
		Content:     res,
	})
}

// handleSetDeviceName handles an incoming message with type
// messages.MessageTypeSetDeviceName.
func (a *actorDeviceManager) handleSetDeviceName(message messages.MessageSetDeviceName) {
	err := a.gatekeeper.SetDeviceName(message.DeviceID, message.Name)
	if err != nil {
		acting.SendOrLogError(logging.ActingLogger, a, acting.ActorErrorMessageFromError(errors.Wrap(err, "set device name")))
		return
	}
	acting.SendOKOrLogError(logging.ActingLogger, a)
}

// handleDeleteDevice handles an incoming message with type
// messages.MessageTypeDeleteDevice.
func (a *actorDeviceManager) handleDeleteDevice(message messages.MessageDeleteDevice) {
	err := a.gatekeeper.DeleteDevice(message.DeviceID)
	if err != nil {
		acting.SendOrLogError(logging.ActingLogger, a, acting.ActorErrorMessageFromError(errors.Wrap(err, "delete device")))
		return
	}
	acting.SendOKOrLogError(logging.ActingLogger, a)
}
