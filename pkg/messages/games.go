package messages

import (
	"github.com/google/uuid"
	"masc-server/pkg/games"
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

// NewMatchMessage is sent by the client if he wants to create a new match.
// The client then expects a RequestGameModeMessage.
type NewMatchMessage struct {
}

// RequestGameModeMessage is sent by the server after the client has started a new match.
// The server then expects a SetGameModeMessage.
type RequestGameModeMessage struct {
	MatchId          uuid.UUID        `json:"match_id"`
	OfferedGameModes []games.GameMode `json:"offered_game_modes"`
}

// SetGameModeMessage is sent by the client as a response to the RequestGameModeMessage.
// The client then expects a MatchConfigMessage.
type SetGameModeMessage struct {
	MatchId  uuid.UUID      `json:"match_id"`
	GameMode games.GameMode `json:"game_mode"`
}

// MatchConfigMessage is sent by the server after the game mode has been set by the client via SetGameModeMessage.
type MatchConfigMessage struct {
	MatchId     uuid.UUID      `json:"match_id"`
	GameMode    games.GameMode `json:"game_mode"`
	MatchConfig interface{}    `json:"match_config"`
}

// SetupMatchMessage is sent by the client if he wants to setup
// a match in order to start a game.
type SetupMatchMessage struct {
	MatchId     uuid.UUID   `json:"match_id"`
	MatchConfig interface{} `json:"match_config"`
}

// RequestMatchConfigPresetsMessage is sent by the client if he wants to request match config presets for
// a target game mode. The client expects an MatchConfigPresetsMessage.
type RequestMatchConfigPresetsMessage struct {
	GameMode games.GameMode `json:"game_mode"`
}

// MatchConfigPresetsMessage is sent by the server as a response to RequestMatchConfigPresetsMessage.
type MatchConfigPresetsMessage struct {
	GameMode games.GameMode            `json:"game_mode"`
	Presets  []games.MatchConfigPreset `json:"presets"`
}
