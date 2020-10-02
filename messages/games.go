package messages

import (
	"github.com/google/uuid"
)

const (
	MsgTypeNewMatch                  MessageType = "new-match"
	MsgTypeRequestGameMode           MessageType = "request-game-mode"
	MsgTypeSetGameMode               MessageType = "set-game-mode"
	MsgTypeMatchConfig               MessageType = "match-config"
	MsgTypeSetupMatch                MessageType = "setup-match"
	MsgTypeRequestMatchConfigPresets MessageType = "request-match-config-presets"
	MsgTypeMatchConfigPresets        MessageType = "match-config-presets"
	MsgTypeConfirmMatchConfig        MessageType = "confirm-match-config"
	MsgTypeRequestRoleAssignments    MessageType = "request-role-assignments"
	MsgTypeAssignRoles               MessageType = "assign-roles"
	MsgTypeYouAreIn                  MessageType = "you-are-in"
	MsgTypePlayerLoginStatus         MessageType = "player-login-status"
	MsgTypeLoginPlayer               MessageType = "login-player"
	MsgTypeReadyForMatchStart        MessageType = "ready-for-match-start"
	MsgTypeMatchStartReadyStates     MessageType = "match-start-ready-states"
	MsgTypeStartMatch                MessageType = "start-match"
	MsgTypePrepareForCountdown       MessageType = "prepare-for-countdown"
	MsgTypeCountdown                 MessageType = "countdown"
	MsgTypeMatchStart                MessageType = "match-start"
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
}

type Performer struct {
	Id     uuid.UUID `json:"id"`
	Role   string    `json:"role"`
	Device Device    `json:"device"`
}

type RoleDetails struct {
	Role        string `json:"role"`
	Name        string `json:"name"`
	Description string `json:"description"`
}

type RoleAssignmentOffer struct {
	PerformerId      uuid.UUID   `json:"performer_id"`
	RoleDetails      RoleDetails `json:"role_details"`
	AvailableDevices []Device    `json:"available_devices"`
}

// RequestRoleAssignmentsMessage is sent by the server to the game master after the match config is being confirmed.
// The game master is then expected to send several AssignRolesMessage s.
type RequestRoleAssignmentsMessage struct {
	MatchId uuid.UUID             `json:"match_id"`
	Roles   []RoleAssignmentOffer `json:"roles"`
}

type RoleAssignment struct {
	PerformerId uuid.UUID `json:"performer_id"`
	DeviceId    uuid.UUID `json:"device_id"`
}

// AssignRolesMessage is sent by the game master for assigning devices to roles for a certain match.
type AssignRolesMessage struct {
	MatchId         uuid.UUID        `json:"match_id"`
	RoleAssignments []RoleAssignment `json:"role_assignments"`
}

type Contract struct {
	PerformerId uuid.UUID   `json:"performer_id"`
	RoleDetails RoleDetails `json:"role_details"`
}

// YouAreInMessage is sent by the server to all devices which got assigned to a role for a certain match.
type YouAreInMessage struct {
	MatchId     uuid.UUID   `json:"match_id"`
	TeamConfig  TeamConfig  `json:"team_config"`
	MatchConfig interface{} `json:"match_config"`
	Contracts   []Contract  `json:"contracts"`
}

type Player struct {
	UserId           uuid.UUID     `json:"user_id"`
	PlayerDetails    PlayerDetails `json:"player_details"`
	Alive            bool          `json:"alive"`
	LifeCount        int           `json:"life_count"`
	RespawnCountdown int           `json:"respawn_countdown"`
}

type PlayerDetails struct {
	Name     string `json:"name"`
	CallSign string `json:"call_sign"`
	Tag      string `json:"tag"`
	Level    string `json:"level"`
}

type Team struct {
	TeamConfig TeamConfig `json:"team_config"`
	Players    []Player   `json:"players"`
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
	MatchId     uuid.UUID `json:"match_id"`
	PerformerId uuid.UUID `json:"performer_id"`
	UserId      uuid.UUID `json:"user_id"`
	TeamId      uuid.UUID `json:"team_id"`
}

// ReadyForMatchStartMessage is sent by all performers after they are ready for match start. Currently this cannot be
// undone.
type ReadyForMatchStartMessage struct {
	MatchId     uuid.UUID `json:"match_id"`
	PerformerId uuid.UUID `json:"performer_id"`
}

type ReadyState struct {
	PerformerId uuid.UUID   `json:"performer_id"`
	RoleDetails RoleDetails `json:"role_details"`
	Ready       bool        `json:"ready"`
}

// MatchStartReadyStatesMessage is sent to all performers after a ReadyForMatchStartMessage is received.
type MatchStartReadyStatesMessage struct {
	MatchId     uuid.UUID    `json:"match_id"`
	ReadyStates []ReadyState `json:"ready_states"`
}

// StartMatchMessage is sent by the game master when he wants to start the match after each device has signaled that it
// is ready for match start. This sends one final PlayerLoginStatusMessage with adjusted player login open indicator.
// The server then sends a PrepareForCountdownMessage.
type StartMatchMessage struct {
	MatchId     uuid.UUID `json:"match_id"`
	PerformerId uuid.UUID `json:"performer_id"`
}

// PrepareForCountdownMessage is sent by the server after the match has been started by the game master via
// StartMatchMessage. Preparation time allows for example dimming a team display or an intro for the countdown display
// and is retrieved from server config.
type PrepareForCountdownMessage struct {
	MatchId         uuid.UUID `json:"match_id"`
	PreparationTime int       `json:"preparation_time"`
}

// CountdownMessage is sent by the server for the match start. The duration is retrieved from server config.
type CountdownMessage struct {
	MatchId  uuid.UUID `json:"match_id"`
	Duration int       `json:"duration"`
}

// MatchStartMessage is sent by the server after the countdown has finished. Now the match has begun.
type MatchStartMessage struct {
	MatchId uuid.UUID `json:"match_id"`
}

type MatchStatusMessage struct {
	MatchId uuid.UUID `json:"match_id"`
	Teams   []Team    `json:"teams"`
}

// PlayerHitMessage is sent by a player control for telling the server that a player is hit and arrived at the spawn.
type PlayerHitMessage struct {
	MatchId     uuid.UUID `json:"match_id"`
	PerformerId uuid.UUID `json:"performer_id"`
	UserId      uuid.UUID `json:"user_id"`
}

// RespawnPlayersMessage is sent by the server for telling player controls that players can now respawn. As often there
// are multiple players respawning at the same time, multiple user ids are packed in this message.
type RespawnPlayersMessage struct {
	MatchId uuid.UUID   `json:"match_id"`
	UserIds []uuid.UUID `json:"user_ids"`
}

// MatchEventMessage
type MatchEventMessage struct {
	MatchId     uuid.UUID `json:"match_id"`
	PerformerId uuid.UUID `json:"performer_id,omitempty"`
	EventName   string    `json:"event_name"`
	EventData   string    `json:"event_data"`
}

type MatchEndTeamStats struct {
	Team     Team `json:"team"`
	IsWinner bool `json:"is_winner"`
}

// MatchEndMessage is sent by the server when a match ends.
type MatchEndMessage struct {
	MatchId   uuid.UUID           `json:"match_id"`
	TeamStats []MatchEndTeamStats `json:"team_stats"`
}
