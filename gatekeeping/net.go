package gatekeeping

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/logging"
	"github.com/LeFinal/masc-server/messages"
	"github.com/LeFinal/masc-server/stores"
	"github.com/LeFinal/masc-server/ws"
	"sync"
	"time"
)

// TODO

type GatekeeperStore interface {
	GetDevices() ([]stores.Device, error)
	// CreateNewDevice creates a new stores.Device with the empty values and only an
	// assigned id. The created device will be returned with the set id.
	CreateNewDevice(selfDescription string) (stores.Device, error)
	// RefreshLastSeenForDevice sets the last seen field for the Device to the
	// current time if online.
	RefreshLastSeenForDevice(deviceID messages.DeviceID) error
	// SetDeviceName sets the name for the Device with the given id.
	SetDeviceName(deviceID messages.DeviceID, name string) error
}

type NetGatekeeper struct {
	// store is used for persistent information regarding device information and
	// acceptance-state.
	store GatekeeperStore
	// protected is the entity to protect. This one will receive calls for device
	// events and management. If not nil, the NetGatekeeper is awake.
	protected Protected
	// onlineDevices holds all devices that are currently online.
	onlineDevices map[*ws.Client]*Device
	// m locks the NetGatekeeper.
	m sync.RWMutex
}

func NewNetGatekeeper(store GatekeeperStore) *NetGatekeeper {
	return &NetGatekeeper{
		store:         store,
		onlineDevices: make(map[*ws.Client]*Device),
	}
}

func (gk *NetGatekeeper) WakeUpAndProtect(protected Protected) error {
	gk.m.Lock()
	defer gk.m.Unlock()
	gk.protected = protected
	logging.GatekeepingLogger.Info("gatekeeper was woken in order to protect the weak")
	return nil
}

func (gk *NetGatekeeper) Retire() error {
	gk.m.Lock()
	defer gk.m.Unlock()
	gk.protected = nil
	logging.GatekeepingLogger.Info("gatekeeper retired")
	return nil
}

func (gk *NetGatekeeper) GetDevices() ([]messages.Device, error) {
	storeDevices, err := gk.store.GetDevices()
	if err != nil {
		return nil, errors.Wrap(err, "get devices from store")
	}
	// Index online devices for faster message building.
	onlineDevices := make(map[messages.DeviceID]*Device)
	for _, onlineDevice := range gk.onlineDevices {
		onlineDevices[onlineDevice.ID] = onlineDevice
	}
	messageDevices := make([]messages.Device, 0, len(storeDevices))
	for _, storeDevice := range storeDevices {
		// Check if online.
		onlineDevice, isOnline := onlineDevices[storeDevice.ID]
		messageDevice := messages.Device{
			ID:              storeDevice.ID,
			Name:            storeDevice.Name,
			SelfDescription: storeDevice.SelfDescription,
			IsConnected:     isOnline,
		}
		// Only set roles when online.
		if isOnline {
			messageDevice.Roles = onlineDevice.Roles
		}
		messageDevices = append(messageDevices, messageDevice)
	}
	return messageDevices, nil
}

func (gk *NetGatekeeper) SetDeviceName(deviceID messages.DeviceID, name string) error {
	return gk.store.SetDeviceName(deviceID, name)
}

func (gk *NetGatekeeper) AcceptClient(ctx context.Context, client *ws.Client) {
	// Wait for hello message.
	for {
		select {
		case <-ctx.Done():
			errors.Log(logging.GatekeepingLogger, errors.NewContextAbortedError("wait for hello message"))
		case helloMessageRaw, ok := <-client.Receive:
			if !ok {
				logging.GatekeepingLogger.Warnf("client %v disconnected while waiting for hello", client.ID)
				return
			}
			newDevice, err := gk.handleHelloFromNewClient(helloMessageRaw)
			if err != nil {
				logAndSendErrorMessage(err, client)
				continue
			}
			// Create send and receive channels.
			deviceSend := make(chan messages.MessageContainer)
			newDevice.Send = deviceSend
			deviceReceive := make(chan messages.MessageContainer)
			newDevice.Receive = deviceReceive
			// Build welcome message.
			welcomeMessageRaw, err := json.Marshal(messages.MessageWelcome{
				DeviceID: newDevice.ID,
				Name:     newDevice.Name,
			})
			if err != nil {
				logAndSendErrorMessage(errors.NewJSONError(err, "marshal welcome message", false), client)
				continue
			}
			// Start up the pumps.
			pumps, shutdownPumps := context.WithCancel(context.Background())
			go deviceIncomingPump(pumps, client, deviceReceive, newDevice.ID)
			go deviceOutgoingPump(pumps, client.Send, deviceSend, newDevice.ID)
			newDevice.ShutdownPumps = shutdownPumps
			// Send welcome message.
			newDevice.Send <- messages.MessageContainer{
				MessageType: messages.MessageTypeWelcome,
				DeviceID:    newDevice.ID,
				Content:     welcomeMessageRaw,
			}
			// Add new device to list.
			gk.m.Lock()
			gk.onlineDevices[client] = newDevice
			gk.m.Unlock()
			// Okay, so now everything is set up, and we can pass it forward to the protected.
			err = gk.protected.WelcomeDevice(newDevice)
			if err != nil {
				logAndSendErrorMessage(errors.Wrap(err, "welcome device"), client)
				continue
			}
			return
		}
	}
}

// deviceIncomingPump unmarshalls and logs incoming messages and forwards them
// to the given device channel.
func deviceIncomingPump(ctx context.Context, client *ws.Client, deviceReceive chan<- messages.MessageContainer, deviceID messages.DeviceID) {
	for {
		select {
		case <-ctx.Done():
			return
		case raw := <-client.Receive:
			// Parse message container.
			var messageContainer messages.MessageContainer
			err := json.Unmarshal(raw, &messageContainer)
			if err != nil {
				// Notify sender.
				errMessage, err := messageErrorFromError(errors.NewJSONError(err, "unmarshal incoming message", true))
				if err != nil {
					errors.Log(logging.GatekeepingLogger, errors.Wrap(err, "message error from error"))
					continue
				}
				marshalAndSendOrLog(messages.MessageTypeError, deviceID, errMessage, client)
				continue
			}
			// Log.
			logging.MessageLogger().WithFields(map[string]interface{}{
				"device":      messageContainer.DeviceID,
				"dir":         "incoming",
				"actor":       messageContainer.ActorID,
				"messageType": messageContainer.MessageType,
			}).Trace(string(messageContainer.Content))
			// Forward.
			select {
			case <-ctx.Done():
				logging.MessageLogger().Warnf("aborting incoming message forward for device %v: %v", deviceID, messageContainer)
				return
			case deviceReceive <- messageContainer:
			}
		}
	}
}

// deviceOutgoingPump marshals and logs outgoing messages and forwards them to
// the given client's channel.
func deviceOutgoingPump(ctx context.Context, send chan<- []byte, deviceSend <-chan messages.MessageContainer, deviceID messages.DeviceID) {
	for {
		select {
		case <-ctx.Done():
			return
		case message := <-deviceSend:
			// Marshal message container.
			raw, err := json.Marshal(message)
			if err != nil {
				errors.Log(logging.GatekeepingLogger, errors.Wrap(err, fmt.Sprintf("marshal outgoing message for device %v", deviceID)))
				continue
			}
			// Log.
			logging.MessageLogger().WithFields(map[string]interface{}{
				"device":      message.DeviceID,
				"dir":         "outgoing",
				"actor":       message.ActorID,
				"messageType": message.MessageType,
			}).Trace(string(message.Content))
			// Forward.
			select {
			case <-ctx.Done():
				logging.MessageLogger().Warnf("aborting outgoing message forward for device %v: %v", deviceID, message)
				return
			case send <- raw:
			}
		}
	}
}

// handleHelloFromNewClient handles a messages.MessageTypeHello from the given
// client. If the client sets the device id field, we search the store for an
// entry with a matching id. If none is found, we create a new device entry as
// we do so for hellos with no set device id. The device is then sent a
// messages.MessageTypeWelcome with the assigned device id. If errors occur,
// they are returned. The created device is returned as first return value.
//
// All fields are set except Device.Send, Device.Receive and
// Device.ShutdownPumps!
func (gk *NetGatekeeper) handleHelloFromNewClient(helloMessageRaw []byte) (*Device, error) {
	// Parse hello message.
	var helloMessageContainer messages.MessageContainer
	err := json.Unmarshal(helloMessageRaw, &helloMessageContainer)
	if err != nil {
		return nil, errors.NewJSONError(err, "unmarshal hello message container", true)
	}
	var helloMessage messages.MessageHello
	err = json.Unmarshal(helloMessageContainer.Content, &helloMessage)
	if err != nil {
		return nil, errors.NewJSONError(err, "unmarshal hello message content", true)
	}
	// Handle it.
	newDevice := &Device{
		SelfDescription: helloMessage.SelfDescription,
		IsConnected:     true,
		LastSeen:        time.Now(),
		Roles:           helloMessage.Roles,
	}
	// Check if device is new.
	foundInKnownDevices := false
	if helloMessageContainer.DeviceID != "" {
		// Device claims to be known. Let's verify that by comparing with known devices.
		knownDevices, err := gk.store.GetDevices()
		if err != nil {
			return nil, errors.Wrap(err, "get devices")
		}
		for _, knownDevice := range knownDevices {
			if knownDevice.ID == helloMessageContainer.DeviceID {
				foundInKnownDevices = true
				// Apply already known fields.
				newDevice.ID = knownDevice.ID
				newDevice.Name = knownDevice.Name
				newDevice.SelfDescription = knownDevice.SelfDescription
				break
			}
		}
	}
	if !foundInKnownDevices {
		// Create entry in store.
		createdDevice, err := gk.store.CreateNewDevice(helloMessage.SelfDescription)
		if err != nil {
			return nil, errors.Wrap(err, "create new device")
		}
		newDevice.ID = createdDevice.ID
		newDevice.Name = createdDevice.Name
		newDevice.SelfDescription = createdDevice.SelfDescription
	}
	err = gk.store.RefreshLastSeenForDevice(newDevice.ID)
	if err != nil {
		return nil, errors.Wrap(err, "set device online")
	}
	return newDevice, nil
}

func (gk *NetGatekeeper) SayGoodbyeToClient(client *ws.Client) {
	gk.m.Lock()
	defer gk.m.Unlock()
	device, ok := gk.onlineDevices[client]
	if !ok {
		// Device did not complete hello-process.
		logging.GatekeepingLogger.Warnf("client %v disconnected without completing handshake", client.ID)
		return
	}
	delete(gk.onlineDevices, client)
	// Update online status.
	err := gk.store.RefreshLastSeenForDevice(device.ID)
	if err != nil {
		errors.Log(logging.GatekeepingLogger, errors.Wrap(err, "set device offline"))
		return
	}
	err = gk.protected.SayGoodbyeToDevice(device.ID)
	if err != nil {
		errors.Log(logging.GatekeepingLogger, errors.Wrap(err, "make protected say goodbye to device"))
		return
	}
}

// logAndSendErrorMessage logs the given error and sends it to the given
// ws.Client WITHOUT device id!
func logAndSendErrorMessage(e error, client *ws.Client) {
	errors.Log(logging.GatekeepingLogger, e)
	errMessage, err := messageErrorFromError(e)
	if err != nil {
		errors.Log(logging.GatekeepingLogger, errors.Wrap(err, "message error from error"))
		return
	}
	marshalAndSendOrLog(messages.MessageTypeError, "", errMessage, client)
}

// messageErrorFromError creates a marshalled messages.MessageContainer with
// messages.MessageTypeError and the given error. However, it will NOT set the
// device or actor field.
func messageErrorFromError(err error) ([]byte, error) {
	c, err := json.Marshal(messages.MessageErrorFromError(err))
	if err != nil {
		return nil, errors.NewJSONError(err, "marshal error message", false)
	}
	return c, nil
}

func marshalAndSendOrLog(messageType messages.MessageType, deviceID messages.DeviceID, content json.RawMessage, client *ws.Client) bool {
	errMessage, err := json.Marshal(messages.MessageContainer{
		MessageType: messageType,
		DeviceID:    deviceID,
		Content:     content,
	})
	if err != nil {
		errors.Log(logging.GatekeepingLogger, errors.NewJSONError(err, "marshal error message container", false))
		return false
	}
	client.Send <- errMessage
	return true
}
