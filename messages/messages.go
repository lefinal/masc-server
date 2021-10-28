// Provide basic message functionality.

package messages

import (
	"encoding/json"
	"github.com/LeFinal/masc-server/errors"
)

// MessageType is the type of message and serves for using the correct parsing
// method.
type MessageType string

// DeviceID is a UUID that is used to identify a Device.
type DeviceID string

// ActorID is a UUID that is used to identify an acting.Actor.
type ActorID string

// UserID is a UUID that is used to identify a stores.Player.
type UserID string

// PlayerRank is the rank of a player.
type PlayerRank int

// Role is used for provided functionality.
type Role string

// MessageContainer is a container for all messages that are sent and received.
// It holds some meta information as well as the actual payload.
type MessageContainer struct {
	// MessageType is the type of the message.
	MessageType MessageType `json:"message_type"`
	// DeviceID is the id that is used for identifying the devices.Device the
	// message belongs to.
	DeviceID DeviceID `json:"device_id"`
	// ActorID is the optional ID of the actor. This is used for concurrent
	// communication with actors that use the same device.
	ActorID ActorID `json:"actor_id,omitempty"`
	// Content is the actual message content.
	Content json.RawMessage `json:"content"`
}

// All message types.
const (
	// MessageTypeAbortMatch is used
	MessageTypeAbortMatch MessageType = "abort-match"
	// MessageTypeAcceptDevice is used with MessageAcceptDevice for accepting new
	// devices and allowing them to communicate with MASC.
	MessageTypeAcceptDevice MessageType = "welcome-device"
	// MessageTypeAreYouReady is used for requesting ready-state from actors. Actors
	// can send messages with MessageTypeReadyState for notifying of their current
	// ready-state. Ready request is finished with MessageTypeReadyAccepted.
	MessageTypeAreYouReady MessageType = "are-you-ready"
	// MessageTypeDeviceList is used with MessageDeviceList as an answer to
	// MessageTypeGetDevices.
	MessageTypeDeviceList MessageType = "device-list"
	// MessageTypeError is used for error messages. The content is being set to the
	// detailed error.
	MessageTypeError MessageType = "error"
	// MessageTypeFired is used when an actor is fired.
	MessageTypeFired MessageType = "fired"
	// MessageTypeGetDevices is received when devices are requested.
	MessageTypeGetDevices MessageType = "get-devices"
	// MessageTypeGoAway is sent to the client when he wants to say hello with an
	// unknown device ID.
	MessageTypeGoAway MessageType = "go-away"
	// MessageTypeHello is received with MessageHello for saying hello to the
	// server.
	MessageTypeHello MessageType = "hello"
	// MessageTypeMatchStatus is a container for status information regarding a
	// Match.
	MessageTypeMatchStatus MessageType = "match-status"
	// MessageTypeOK is used only for confirmation of actions that do not require a
	// detailed response.
	MessageTypeOK MessageType = "ok"
	// MessageTypePlayerJoin is used for joining a player for a match.
	MessageTypePlayerJoin MessageType = "player-join"
	// MessageTypePlayerJoinClosed is used for notifying that no more player can
	// join a match.
	MessageTypePlayerJoinClosed MessageType = "player-join-closed"
	// MessageTypePlayerJoinOpen notifies that players can now join.
	MessageTypePlayerJoinOpen MessageType = "player-join-open"
	// MessageTypePlayerJoined is sent to everyone participating in a match when a
	// player joined.
	MessageTypePlayerJoined MessageType = "player-joined"
	// MessageTypePlayerLeave is received when a player wants so leave a match.
	MessageTypePlayerLeave MessageType = "player-leave"
	// MessageTypePlayerLeft is sent to everyone participating in a match when a
	// player left.
	MessageTypePlayerLeft MessageType = "player-left"
	// MessageTypeReadyAccepted is used for ending ready-state requests that were
	// initially started with MessageTypeAreYouReady.
	MessageTypeReadyAccepted MessageType = "ready-accepted"
	// MessageTypeReadyState is used with MessageReadyState for notifying that an
	// actor is (not) ready.
	MessageTypeReadyState MessageType = "ready-state"
	// MessageTypeReadyStateUpdate is used with MessageReadyStateUpdate for
	// broadcasting ready-states to all actors participating in a match.
	MessageTypeReadyStateUpdate MessageType = "ready-state-update"
	// MessageTypeRequestRoleAssignments is used with MessageRequestRoleAssignments
	// for requesting role assignments. Usually, this is sent to a game master.
	MessageTypeRequestRoleAssignments MessageType = "request-role-assignments"
	// MessageTypeRoleAssignments is used with MessageRoleAssignments for when an
	// assignment request was fulfilled.
	MessageTypeRoleAssignments MessageType = "role-assignments"
	// MessageTypeWelcome is sent to the client when he is welcomed at the server.
	// Used with MessageWelcome.
	MessageTypeWelcome MessageType = "welcome"
	// MessageTypeYouAreIn is used with MessageYouAreIn for notifying an actor that he now has a job.
	MessageTypeYouAreIn MessageType = "you-are-in"
)

// MessageError is used with MessageTypeError for errors that need to be sent to devices.
type MessageError struct {
	// Code is the error code from errors.Error.
	Code string `json:"code"`
	// Kind is the kind code from errors.Error.
	Kind string `json:"kind"`
	// Err is the error from errors.Error.
	Err string `json:"err"`
	// Message is the message from errors.Error.
	Message string `json:"message"`
	// Details are error details from errors.Error.
	Details map[string]interface{} `json:"details"`
}

// MessageErrorFromError creates a MessageError from the given error.
func MessageErrorFromError(err error) MessageError {
	e, _ := errors.Cast(err)
	if !errors.BlameUser(err) {
		return MessageError{
			Code:    string(e.Code),
			Message: "internal server error",
		}
	}
	return MessageError{
		Code:    string(e.Code),
		Kind:    string(e.Kind),
		Err:     e.Error(),
		Message: e.Message,
		Details: e.Details,
	}
}
