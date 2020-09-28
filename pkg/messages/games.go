package messages

import "github.com/google/uuid"

const (
	MsgTypeSetupMatch MessageType = "setup-match"
	MsgTypeNewMatch   MessageType = "new-match"
)

// NewMatchMessage is sent by the client if he wants to create a new match.
type NewMatchMessage struct {
}

type RequestGameModeMessage struct {
}

// SetupMatchMessage is sent by the client if he wants to setup
// a match in order to start a game.
type SetupMatchMessage struct {
	MatchId uuid.UUID `json:"match_id"`
}
