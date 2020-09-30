package messages

import (
	"github.com/LeFinal/masc-server/pkg/devices"
	"github.com/LeFinal/masc-server/pkg/games"
	"github.com/google/uuid"
)

const (
	MsgTypeNewMatch                  MessageType = "new-match"
	MsgTypeRequestGameModeMessage    MessageType = "request-game-mode"
	MsgTypeSetGameMode               MessageType = "set-game-mode"
	MsgTypeMatchConfig               MessageType = "match-config"
	MsgTypeSetupMatch                MessageType = "setup-match"
	MsgTypeRequestMatchConfigPresets MessageType = "request-match-config-presets"
	MsgTypeMatchConfigPresets        MessageType = "match-config-presets"
)

// NewMatchMessage is sent by the game master if he wants to create a new match.
// The client then expects a RequestGameModeMessage.
type NewMatchMessage struct {
}

// RequestGameModeMessage is sent by the server after the game master has started a new match.
// The server then expects a SetGameModeMessage.
type RequestGameModeMessage struct {
	MatchId          uuid.UUID        `json:"match_id"`
	OfferedGameModes []games.GameMode `json:"offered_game_modes"`
}

// SetGameModeMessage is sent by the game master as a response to the RequestGameModeMessage.
// The game master then expects a MatchConfigMessage.
type SetGameModeMessage struct {
	MatchId  uuid.UUID      `json:"match_id"`
	GameMode games.GameMode `json:"game_mode"`
}

// MatchConfigMessage is sent by the server after the game mode has been set by the game master via SetGameModeMessage.
type MatchConfigMessage struct {
	MatchId     uuid.UUID      `json:"match_id"`
	GameMode    games.GameMode `json:"game_mode"`
	MatchConfig interface{}    `json:"match_config"`
}

// SetupMatchMessage is sent by the game master if he wants to setup a match in order to start a game.
type SetupMatchMessage struct {
	MatchId     uuid.UUID   `json:"match_id"`
	MatchConfig interface{} `json:"match_config"`
}

// RequestMatchConfigPresetsMessage is sent by a client if he wants to request match config presets for
// a target game mode. The client expects an MatchConfigPresetsMessage.
type RequestMatchConfigPresetsMessage struct {
	GameMode games.GameMode `json:"game_mode"`
}

// MatchConfigPresetsMessage is sent by the server as a response to RequestMatchConfigPresetsMessage.
type MatchConfigPresetsMessage struct {
	GameMode games.GameMode            `json:"game_mode"`
	Presets  []games.MatchConfigPreset `json:"presets"`
}

// ConfirmMatchConfigMessage is sent by the game master if he wants to confirm the match config.
// A RequestRoleAssignmentsMessage is then expected to be sent by the server.
type ConfirmMatchConfigMessage struct {
	MatchId uuid.UUID `json:"match_id"`
}

// RequestRoleAssignmentsMessage is sent by the server to the game master after the match config is being confirmed.
// The game master is then expected to send several AssignRoleMessage s.
type RequestRoleAssignmentsMessage struct {
	MatchId uuid.UUID                     `json:"match_id"`
	Roles   []devices.RoleAssignmentOffer `json:"roles"`
}

// AssignRoleMessage is sent by the game master for assigning devices to roles for a certain match.
type AssignRoleMessage struct {
	MatchId  uuid.UUID    `json:"match_id"`
	Role     devices.Role `json:"role"`
	DeviceId uuid.UUID    `json:"device_id"`
}

// PlayerLoginStatusMessage is sent by the server to game master and team bases in order to allow the login of players.
// After each player login the PlayerLoginStatusMessage is sent again but with adjusted open slots count.
type PlayerLoginStatusMessage struct {
	MatchId uuid.UUID    `json:"match_id"`
	Teams   []games.Team `json:"teams"`
}

// LoginPlayerMessage is sent by player controls to the server after they received a PlayerLoginStatusMessage with
// available slots.
type LoginPlayerMessage struct {
	MatchId uuid.UUID `json:"match_id"`
	UserId  uuid.UUID `json:"user_id"`
	TeamId  uuid.UUID `json:"team_id"`
}
