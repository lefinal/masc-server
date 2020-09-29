package messages

import (
	"fmt"
	"masc-server/pkg/errors"
)

// CreateMessageContainerForType creates the correct message container for a given message type.
func CreateMessageContainerForType(msgType MessageType) (interface{}, *errors.MascError) {
	var res interface{}
	switch msgType {
	// General
	case MsgTypeOk:
		res = OkMessage{}
	// Errors
	case MsgTypeError:
		res = ErrorMessage{}
	// Gatekeeping
	case MsgTypeHello:
		res = HelloMessage{}
	case MsgTypeWelcome:
		res = WelcomeMessage{}
	// Scheduling
	case MsgTypeGetSchedule:
		res = GetScheduleMessage{}
	case MsgTypeSchedule:
		res = ScheduleMessage{}
	case MsgTypeScheduleEvent:
		res = ScheduleEventMessage{}
	case MsgTypeUpdateEvent:
		res = UpdateEventMessage{}
	case MsgTypeDeleteEvent:
		res = DeleteEventMessage{}
	// Games
	case MsgTypeNewMatch:
		res = NewMatchMessage{}
	case MsgTypeRequestGameModeMessage:
		res = RequestGameModeMessage{}
	case MsgTypeSetGameMode:
		res = SetGameModeMessage{}
	case MsgTypeMatchConfig:
		res = MatchConfigMessage{}
	case MsgTypeSetupMatch:
		res = SetupMatchMessage{}
	case MsgTypeRequestMatchConfigPresets:
		res = RequestMatchConfigPresetsMessage{}
	case MsgTypeMatchConfigPresets:
		res = MatchConfigPresetsMessage{}
	default:
		return nil, errors.NewMascError(fmt.Sprintf("find mapping for: %v", msgType),
			errors.UnknownMessageTypeError)
	}
	return res, nil

}
