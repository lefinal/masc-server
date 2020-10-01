package messages

import (
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
	MatchId          uuid.UUID `json:"match_id"`
	OfferedGameModes []string  `json:"offered_game_modes"`
}

// SetGameModeMessage is sent by the game master as a response to the RequestGameModeMessage.
// The game master then expects a MatchConfigMessage.
type SetGameModeMessage struct {
	MatchId  uuid.UUID `json:"match_id"`
	GameMode string    `json:"game_mode"`
}

// MatchConfigMessage is sent by the server after the game mode has been set by the game master via SetGameModeMessage.
type MatchConfigMessage struct {
	MatchId     uuid.UUID   `json:"match_id"`
	GameMode    string      `json:"game_mode"`
	MatchConfig interface{} `json:"match_config"`
}

// SetupMatchMessage is sent by the game master if he wants to setup a match in order to start a game.
type SetupMatchMessage struct {
	MatchId     uuid.UUID   `json:"match_id"`
	MatchConfig interface{} `json:"match_config"`
}

type TeamConfig struct {
	TeamId              uuid.UUID `json:"team_id"`
	Name                string    `json:"name"`
	MaxPlayers          int       `json:"max_players"`
	LifeCount           int       `json:"life_count"`
	MinRespawnGroupSize int       `json:"min_respawn_group_size"`
}

type MatchConfigPreset struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	TeamConfigs []TeamConfig `json:"team_configs"`
	MatchConfig interface{}  `json:"match_config"`
}

// RequestMatchConfigPresetsMessage is sent by a client if he wants to request match config presets for
// a target game mode. The client expects an MatchConfigPresetsMessage.
type RequestMatchConfigPresetsMessage struct {
	GameMode string `json:"game_mode"`
}

// MatchConfigPresetsMessage is sent by the server as a response to RequestMatchConfigPresetsMessage.
type MatchConfigPresetsMessage struct {
	GameMode string              `json:"game_mode"`
	Presets  []MatchConfigPreset `json:"presets"`
}

// ConfirmMatchConfigMessage is sent by the game master if he wants to confirm the match config.
// A RequestRoleAssignmentsMessage is then expected to be sent by the server.
type ConfirmMatchConfigMessage struct {
	MatchId uuid.UUID `json:"match_id"`
}

type Device struct {
	Id          uuid.UUID `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Roles       []string  `json:"roles"`
}

type RoleAssignmentOffer struct {
	Role             string   `json:"role"`
	Description      string   `json:"description"`
	CountRequired    int      `json:"count_required"`
	AssignedDevices  []Device `json:"assigned_devices"`
	AvailableDevices []Device `json:"available_devices"`
}

// RequestRoleAssignmentsMessage is sent by the server to the game master after the match config is being confirmed.
// The game master is then expected to send several AssignRoleMessage s.
type RequestRoleAssignmentsMessage struct {
	MatchId uuid.UUID             `json:"match_id"`
	Roles   []RoleAssignmentOffer `json:"roles"`
}

// AssignRoleMessage is sent by the game master for assigning devices to roles for a certain match.
type AssignRoleMessage struct {
	MatchId  uuid.UUID `json:"match_id"`
	Role     string    `json:"role"`
	DeviceId uuid.UUID `json:"device_id"`
}

type Player struct {
	UserId    uuid.UUID
	Alive     bool
	LifeCount int
}

type Team struct {
	TeamConfig TeamConfig
	Players    []Player
}

// PlayerLoginStatusMessage is sent by the server to game master and team bases in order to allow the login of players.
// After each player login the PlayerLoginStatusMessage is sent again but with adjusted open slots count.
type PlayerLoginStatusMessage struct {
	MatchId         uuid.UUID `json:"match_id"`
	PlayerLoginOpen bool      `json:"player_login_open"`
	Teams           []Team    `json:"teams"`
}

// LoginPlayerMessage is sent by player controls to the server after they received a PlayerLoginStatusMessage with
// available slots.
type LoginPlayerMessage struct {
	MatchId uuid.UUID `json:"match_id"`
	UserId  uuid.UUID `json:"user_id"`
	TeamId  uuid.UUID `json:"team_id"`
}
