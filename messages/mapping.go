package messages

import (
	"fmt"
	"github.com/LeFinal/masc-server/errors"
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
	case MsgTypeRequestGameMode:
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
	case MsgTypeConfirmMatchConfig:
		res = ConfirmMatchConfigMessage{}
	case MsgTypeRequestRoleAssignments:
		res = RequestRoleAssignmentsMessage{}
	case MsgTypeAssignRoles:
		res = AssignRolesMessage{}
	case MsgTypeYouAreIn:
		res = YouAreInMessage{}
	case MsgTypePlayerLoginStatus:
		res = PlayerLoginStatusMessage{}
	case MsgTypeLoginPlayer:
		res = LoginPlayerMessage{}
	case MsgTypeReadyForMatchStart:
		res = ReadyForMatchStartMessage{}
	case MsgTypeMatchStartReadyStates:
		res = MatchStartReadyStatesMessage{}
	case MsgTypeStartMatch:
		res = StartMatchMessage{}
	case MsgTypePrepareForCountdown:
		res = PrepareForCountdownMessage{}
	case MsgTypeCountdown:
		res = CountdownMessage{}
	case MsgTypeMatchStart:
		res = MatchStartMessage{}
	default:
		return nil, errors.NewMascError(fmt.Sprintf("find mapping for: %v", msgType),
			errors.UnknownMessageTypeError)
	}
	return res, nil

}
