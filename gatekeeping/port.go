package gatekeeping

import (
	"fmt"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/logging"
	"github.com/LeFinal/masc-server/messages"
	"github.com/LeFinal/masc-server/networking"
	"github.com/google/uuid"
)

// GatePort is the port which holds the client as well as receive and send channels for the device itself.
// The port is run in a separate go routine and ensures message parsing as well as gatekeeping.
type GatePort struct {
	logger               logging.Logger
	AllowedMessages      *messages.AllowedMessageCollection
	DeviceId             uuid.UUID
	client               networking.Client
	receivedFirstMessage bool
	Receive              chan messages.MessageContainer
	Send                 chan messages.MessageContainer
}

func newGatePort(c networking.Client) *GatePort {
	newUUID := uuid.New()
	return &GatePort{
		logger:          logging.NewLogger(fmt.Sprintf("GATE_PORT-%s", newUUID)),
		AllowedMessages: messages.NewAllowedMessageCollection(messages.AllowedMessagesLoggedOut),
		DeviceId:        newUUID,
		client:          c,
		Receive:         make(chan messages.MessageContainer, 256),
		Send:            make(chan messages.MessageContainer, 256),
	}
}

// run should be run in a separate go routine and manages incoming and outgoing
// communication of a gate port. This includes checking for meta data and
// building messages.
func (port *GatePort) run() {
	port.logger.Info("Up and running!")
	for {
		select {
		case msg := <-port.client.Inbox():
			port.handleReceivedMessage(msg)
		case container := <-port.Send:
			port.handleMessageToBeSent(container)
		}
	}
}

func (port *GatePort) handleReceivedMessage(message networking.InboundMessage) {
	// Parse message.
	meta, payload, parseErr := messages.ParseMessage([]byte(message.Message))
	if parseErr != nil {
		mascErr := errors.PropagateMascError("parse message", parseErr)
		port.logger.MascError(mascErr)
		port.handleMessageToBeSent(messages.NewMessageContainerForError(messages.NewErrorMessageFromMascError(mascErr)))
		return
	}
	// Check if allowed message.
	if !port.isAllowedMessage(meta.MessageType()) {
		allowedMessages := port.AllowedMessages.String()
		mascErr := errors.NewMascErrorWithMessage(fmt.Sprintf("message type not in allowed ones (%s)", allowedMessages),
			errors.MessageTypeNotAllowedError,
			fmt.Sprintf("Allowed messages are: %s", allowedMessages))
		port.logger.MascError(mascErr)
		port.handleMessageToBeSent(messages.NewMessageContainerForError(messages.NewErrorMessageFromMascError(mascErr)))
		return
	}
	// Check if logged in.
	if port.receivedFirstMessage {
		// Check device id.
		if port.DeviceId != meta.DeviceId {
			mascErr := errors.NewMascError("check device id", errors.InvalidDeviceIdError)
			port.logger.MascError(mascErr)
			port.handleMessageToBeSent(messages.NewMessageContainerForError(messages.NewErrorMessageFromMascError(mascErr)))
			return
		}
	} else {
		port.receivedFirstMessage = true
		port.logger.Info("First contact with client.")
	}
	// Forward to device.
	port.Receive <- messages.MessageContainer{
		MessageType: meta.MessageType(),
		Payload:     payload,
	}
}

func (port *GatePort) isAllowedMessage(messageType messages.MessageType) bool {
	return port.AllowedMessages.IsAllowed(messageType)
}

func (port *GatePort) handleMessageToBeSent(container messages.MessageContainer) {
	// Marshal payload.
	payload, mascErr := messages.MarshalPayload(container)
	if mascErr != nil {
		logger.MascError(errors.PropagateMascError("marshal payload", mascErr))
		return
	}
	// Marshal general message.
	message, err := messages.MarshalMessage(messages.GeneralMessage{
		MessageMeta: messages.MessageMeta{
			Type:     container.MessageType.String(),
			DeviceId: port.DeviceId,
		},
		Payload: payload,
	})
	if err != nil {
		logger.MascError(errors.NewMascErrorFromError("marshal general message",
			errors.MarshalMessageErrorError, err))
		return
	}
	// sendMessage message to client.
	port.client.SendMessage(networking.OutboundMessage{
		Message: message,
	})
}
