package games

import (
	"context"
	"github.com/LeFinal/masc-server/acting"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

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
	// configs.
	MatchPhaseInit MatchPhase = "init"
	// MatchPhaseSetup is used for players to join and assigning teams.
	MatchPhaseSetup MatchPhase = "setup"
	// MatchPhasePrepare is used for making last changes to the match before
	// starting.
	MatchPhasePrepare MatchPhase = "prepare"
	// MatchPhaseAwaitReady waits until everybody is ready for start.
	MatchPhaseAwaitReady MatchPhase = "await-ready"
	// MatchPhaseRunning is used while the match is running.
	MatchPhaseRunning MatchPhase = "running"
	// MatchPhasePost is used for any post stuff. Currently, not used but maybe in
	// the future.
	MatchPhasePost MatchPhase = "post"
	// MatchPhaseEnd is used when a match has ended.
	MatchPhaseEnd MatchPhase = "end"
)

// MatchStatus is a container for status information regarding a Match. This can
// be used for example by acting.RoleGlobalMonitor.
type MatchStatus struct {
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

// MatchDoneReason is a reason for why a match is done.
type MatchDoneReason string

const (
	// MatchDoneReasonFinish is used when a match was finished cleanly.
	MatchDoneReasonFinish MatchDoneReason = "finish"
	// MatchDoneReasonError is used when an error led to match finish.
	MatchDoneReasonError MatchDoneReason = "error"
)

// MatchDone holds information regarding the finish of a Match.
type MatchDone struct {
	// Reason for why the Match is done.
	Reason MatchDoneReason
	// Err is an optional error that led to Match finish.
	Err error
}

// Match allows handling and receiving status updates from a match.
type Match interface {
	// Status receives status information regarding the match.
	Status() <-chan MatchStatus
	// Abort aborts the match.
	Abort() error
	// Done receives when the match is done.
	Done() <-chan MatchDone
	// Logger is the match logger which contains all match information.
	Logger() *logrus.Logger
}

// BaseMatch holds some basic fields that every Match needs and provides the
// Match.Status and Match.Done methods.
type BaseMatch struct {
	// M is a lock for the whole match state. This can be used when BaseMatch is
	// composed with the actual match.
	M sync.RWMutex
	// Ctx is the context of the match. This can be used in order to cancel ongoing
	// operations when the match is being aborted.
	Ctx context.Context
	// abort cancels Ctx.
	abort context.CancelFunc
	// GameMode is the GameMode the match uses.
	GameMode GameMode
	// Phase is the MatchPhase the match is currently in.
	Phase MatchPhase
	// IsActive determines whether the match is currently active or done/not
	// started.
	IsActive bool
	// Agency is where actors are hired from.
	Agency acting.Agency
	// GameMaster is the acting.Actor with acting.RoleGameMaster. This is the one
	// who started the match and controls it.
	GameMaster acting.Actor
	// StatusUpdates sends StatusUpdates updates.
	StatusUpdates chan MatchStatus
	// DoneUpdates sends when the match has finished.
	DoneUpdates chan MatchDone
	// Logger holds all information regarding the match.
	Logger *logrus.Logger
}

func (match *BaseMatch) Status() <-chan MatchStatus {
	return match.StatusUpdates
}

func (match *BaseMatch) Done() <-chan MatchDone {
	return match.DoneUpdates
}

func StartMatch(gameMode GameMode, agency acting.Agency, gameMaster acting.Actor) (Match, error) {
	_ = &BaseMatch{
		GameMode:   gameMode,
		Phase:      MatchPhaseInit,
		IsActive:   true,
		Agency:     agency,
		GameMaster: gameMaster,
		Logger:     logrus.New(),
	}
	// TODO
	panic("implement me")
}
