package messages

import "time"

// MatchID is used in order to identify a games.Match.
type MatchID string

// MatchBase is used for all messages that are related a certain match.
type MatchBase struct {
	// MatchID is used for identifying a match.
	MatchID MatchID `json:"match_id"`
}

// MessageReadyState is used for MessageTypeReadyState.
type MessageReadyState struct {
	IsReady bool `json:"is_ready"`
}

// GameMode is the type of game mode.
type GameMode string

// Game modes.
const (
	// GameModeTeamDeathmatch is the classic deathmatch where each player has a
	// limited amount of lives and the last team surviving wins.
	GameModeTeamDeathmatch GameMode = "team-deathmatch"
)

// MatchPhase is a fixed phase type for being used in matches.
type MatchPhase string

const (
	// MatchPhaseInit is used while the match is initializing like setting up
	// configs and assigning roles.
	MatchPhaseInit MatchPhase = "init"
	// MatchPhaseSetup is used for players to join and assigning teams.
	MatchPhaseSetup MatchPhase = "setup"
	// MatchPhaseCountdown is when the countdown is running.
	MatchPhaseCountdown MatchPhase = "countdown"
	// MatchPhaseRunning is used while the match is running.
	MatchPhaseRunning MatchPhase = "running"
	// MatchPhasePost is used for any post stuff. Currently, not used but maybe in
	// the future.
	MatchPhasePost MatchPhase = "post"
	// MatchPhaseEnd is used when a match has ended.
	MatchPhaseEnd MatchPhase = "end"
)

// MessageMatchStatus is used with MessageTypeMatchStatus.
type MessageMatchStatus struct {
	// GameMode is the mode the match uses.
	GameMode GameMode `json:"game_mode"`
	// MatchPhase is the phase the match is currently in.
	MatchPhase MatchPhase `json:"match_phase"`
	// MatchConfig is an optional configuration that may be of interest.
	MatchConfig interface{} `json:"match_config"`
	// IsActive determines whether the match is currently active.
	IsActive bool `json:"is_active"`
	// Start is the start timestamp of the match.
	Start time.Time `json:"start"`
	// End is the end timestamp of the match.
	End time.Time `json:"end"`
	// PlayerCount is the count of players who have joined the match.
	PlayerCount int `json:"player_count"`
}

// MessagePlayerJoinOpen is used with MessageTypePlayerJoinOpen.
type MessagePlayerJoinOpen struct {
	// Team is the assigned team. This is used in order to distinguish player joins
	// and leaves from other teams and maybe handle this information differently.
	Team string `json:"team"`
}

// MessagePlayerJoin is used with MessageTypePlayerJoin.
type MessagePlayerJoin struct {
	// User is the id of the player who wants to join. If RequestGuest is set to
	// true, this can be omitted.
	User UserID `json:"user"`
	// RequestGuest states that a guest user is wanted.
	RequestGuest bool `json:"request_guest"`
}

// MessagePlayerJoined is used with MessageTypePlayerJoined.
type MessagePlayerJoined struct {
	// PlayerDetails holds user information regarding the player.
	PlayerDetails User `json:"player_details"`
}

// MessagePlayerLeave is used with MessageTypePlayerLeave.
type MessagePlayerLeave struct {
	// Player is the id of the player that wants to leave.
	Player UserID `json:"player"`
}

// MessagePlayerLeft is used with MessageTypePlayerLeft.
type MessagePlayerLeft struct {
	// Player is the id of the player that left.
	Player UserID `json:"player"`
}
