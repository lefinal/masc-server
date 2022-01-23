package messages

import (
	"time"
)

const (
	// MessageTypeAbortMatch is used
	MessageTypeAbortMatch MessageType = "abort-match"
	// MessageTypeAreYouReady is used for requesting ready-state from actors. Actors
	// can send messages with MessageTypeReadyState for notifying of their current
	// ready-state. Ready request is finished with MessageTypeReadyAccepted.
	MessageTypeAreYouReady MessageType = "are-you-ready"
	// MessageTypeMatchStatus is a container for status information regarding a
	// Match.
	MessageTypeMatchStatus MessageType = "match-status"
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
	// for requesting role assignments. Usually, this is sent to a game master. Used
	// with MessageRequestRoleAssignments.
	MessageTypeRequestRoleAssignments MessageType = "request-role-assignments"
	// MessageTypeRoleAssignments is used with MessageRoleAssignments for when an
	// assignment request was fulfilled.
	MessageTypeRoleAssignments MessageType = "role-assignments"
)

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

// MessageReadyStateUpdate is used with MessageTypeReadyStateUpdate.
type MessageReadyStateUpdate struct {
	// IsEverybodyReady describes whether all actors are ready.
	IsEverybodyReady bool `json:"is_everybody_ready"`
	// ActorStates holds the individual ready-states for the actors.
	ActorStates []ReadyStateUpdateActorState `json:"actor_states"`
}

// ReadyStateUpdateActorState is the ready-state for a specific Actor.
type ReadyStateUpdateActorState struct {
	// Actor is the actor the ready state is for.
	Actor ActorRepresentation `json:"actor"`
	// IsReady describes whether the actor is ready.
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
