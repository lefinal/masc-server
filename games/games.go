package games

import (
	"context"
	"github.com/LeFinal/masc-server/acting"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/messages"
	"github.com/google/uuid"
	"github.com/sirupsen/logrus"
	"sync"
	"time"
)

// Match allows handling and receiving status updates from a match.
type Match interface {
	// Start starts the Match.
	Start(ctx context.Context) error
	// Status receives status information regarding the match.
	Status() <-chan messages.MessageMatchStatus
	// Abort aborts the match.
	Abort(reason string) error
	// Done receives when the match is done.
	Done() <-chan MatchDone
	// Logger is the match logger which contains all match information.
	Logger() *logrus.Entry
}

// BaseMatch holds some basic fields that every Match needs and provides the
// Match.Status and Match.Done methods.
type BaseMatch struct {
	// ID identifies a match.
	ID string
	// M is a lock for the whole match state. This can be used when BaseMatch is
	// composed with the actual match.
	M sync.RWMutex
	// Abort cancels the at Match.Start provided match context. This should only be
	// used from within an actual match and not from outside. In this case use
	// Match.Abort. The only reason why this exists is that the match can abort
	// itself.
	Abort context.CancelFunc
	// GameMode is the GameMode the match uses.
	GameMode messages.GameMode
	// Phase is the MatchPhase the match is currently in.
	Phase messages.MatchPhase
	// IsActive determines whether the match is currently active. If it is inactive
	// in can be dumped away.
	IsActive bool
	// Agency is where actors are hired from.
	Agency acting.Agency
	// GameMaster is the acting.Actor with acting.RoleTypeGameMaster. This is the one
	// who started the match and controls it.
	GameMaster acting.Actor
	// PlayerManagement allows easy player management which can also be used for
	// player joins.
	PlayerManagement *PlayerManagement
	// PlayerManagementUpdates receives when PlayerManagement welcomes new players,
	// or they leave.
	PlayerManagementUpdates chan PlayerManagementUpdate
	// PlayerProvider allows retrieving users and requesting guest accounts.
	PlayerProvider PlayerProvider
	// StatusUpdates sends StatusUpdates updates.
	StatusUpdates chan messages.MessageMatchStatus
	// DoneUpdates sends when the match has finished.
	DoneUpdates chan MatchDone
	// Logger holds all information regarding the match.
	Logger *logrus.Entry
	// Start is the time when the match was started.
	Start time.Time
	// End is the time when the match had finished.
	End time.Time
}

func (match *BaseMatch) Status() <-chan messages.MessageMatchStatus {
	return match.StatusUpdates
}

func (match *BaseMatch) Done() <-chan MatchDone {
	return match.DoneUpdates
}

func StartMatch(gameMode messages.GameMode, agency acting.Agency, gameMaster acting.Actor) (Match, error) {
	playerManagementUpdates := make(chan PlayerManagementUpdate)
	matchID := uuid.New().String()
	_ = &BaseMatch{
		ID:                      matchID,
		GameMode:                gameMode,
		Phase:                   messages.MatchPhaseInit,
		IsActive:                true,
		Agency:                  agency,
		GameMaster:              gameMaster,
		PlayerManagement:        NewPlayerManagement(playerManagementUpdates),
		PlayerManagementUpdates: playerManagementUpdates,
		PlayerProvider:          nil, // TODO: ADD PLEASE
		Logger:                  logrus.New().WithField("match", matchID),
	}
	// TODO
	// TODO: Remember to listen for game master quit in order to abort the match.
	panic("implement me")
}

// AbortMatchOrLog aborts the match or logs the occurred error.
func AbortMatchOrLog(match Match, reason string) {
	err := match.Abort(reason)
	errors.Log(match.Logger().WithFields(logrus.Fields{}), errors.Wrap(err, "abort match", nil))
}

// AbortMatchBecauseOfErrorOrLog logs the given error to the given logger
// and aborts the Match. If aborting fails, the error is also logged to the
// logger.
func AbortMatchBecauseOfErrorOrLog(match Match, e error) {
	match.Logger().Errorf("abort match because of error: %v", e)
	AbortMatchOrLog(match, "internal error")
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
