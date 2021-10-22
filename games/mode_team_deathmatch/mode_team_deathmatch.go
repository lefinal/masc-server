package mode_team_deathmatch

import (
	"context"
	"fmt"
	"github.com/LeFinal/masc-server/acting"
	"github.com/LeFinal/masc-server/casting"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/games"
	"github.com/LeFinal/masc-server/messages"
	"github.com/gobuffalo/nulls"
	"github.com/sirupsen/logrus"
	"time"
)

// ConfigRequest for a Match. Will be used for creating a Config.
type ConfigRequest struct {
	// TeamCount determines how many teams are playing.
	TeamCount nulls.Int `json:"team_count"`
	// LivesPerPlayer determines how many lives each player owns. Matchmaking might
	// increase lives for certain players in order to create a fair match.
	LivesPerPlayer nulls.Int `json:"lives_per_player"`
	// RespawnTeamSize determines how many players of a team are needed in order to respawn.
	RespawnTeamSize nulls.Float64 `json:"respawn_team_size"`
}

// Default values for config.
const (
	// defaultConfigTeamCount is the default value for Config.TeamCount.
	defaultConfigTeamCount uint = 2
	// defaultLivesPerPlayer is the default value for Config.LivesPerPlayer.
	defaultLivesPerPlayer uint = 3
	// defaultRespawnTeamSize is the default value for Config.RespawnTeamSize.
	defaultRespawnTeamSize float64 = 0.4
)

// Config is the actual Match configuration.
type Config struct {
	// TeamCount determines how many teams are playing.
	TeamCount uint `json:"team_count"`
	// LivesPerPlayer determines how many lives each player owns. Matchmaking might
	// increase lives for certain players in order to create a fair match.
	LivesPerPlayer uint `json:"lives_per_player"`
	// RespawnTeamSize determines how many players of a team are needed in order to
	// respawn.
	RespawnTeamSize float64 `json:"respawn_team_size"`
}

// configFromRequest assures that we have a valid Config based on the given
// ConfigRequest. If stuff is missing, we set it from default values. If the
// config has invalid values, it returns an error. If everything is okay, the
// created Config is returned. The passed logger is used for logging warnings
// for unset fields.
func configFromRequest(logger *logrus.Logger, r ConfigRequest) (Config, error) {
	config := Config{}
	// Team count.
	if r.TeamCount.Valid {
		if r.TeamCount.Int <= 0 {
			return Config{}, errors.NewInvalidConfigRequestError(fmt.Sprintf("team count must be greater 0 but was %d",
				r.TeamCount.Int))
		}
		config.TeamCount = uint(r.TeamCount.Int)
	} else {
		logger.Warnf("config request contains no team count. using default value: %d",
			defaultConfigTeamCount)
		config.TeamCount = defaultConfigTeamCount
	}
	// Lives per player.
	if r.LivesPerPlayer.Valid {
		if r.LivesPerPlayer.Int <= 0 {
			return Config{}, errors.NewInvalidConfigRequestError(fmt.Sprintf("lives per player must be greater 0 but was %d",
				r.LivesPerPlayer.Int))
		}
		config.LivesPerPlayer = uint(r.LivesPerPlayer.Int)
	} else {
		logger.Warnf("config request contains no lives per player. using default value: %d",
			defaultLivesPerPlayer)
		config.LivesPerPlayer = defaultLivesPerPlayer
	}
	// Respawn team size.
	if r.RespawnTeamSize.Valid {
		if r.RespawnTeamSize.Float64 < 0 {
			return Config{}, errors.NewInvalidConfigRequestError(fmt.Sprintf("respawn team size must be greater or equal 0 but was %g",
				r.RespawnTeamSize.Float64))
		}
		if r.RespawnTeamSize.Float64 > 1 {
			return Config{}, errors.NewInvalidConfigRequestError(fmt.Sprintf("respawn team size must be less or equal 1 but was %g",
				r.RespawnTeamSize.Float64))
		}
		config.RespawnTeamSize = r.RespawnTeamSize.Float64
	} else {
		logger.Warnf("config request contains no respawn team size. using default value: %g",
			defaultRespawnTeamSize)
		config.RespawnTeamSize = defaultRespawnTeamSize
	}
	return config, nil
}

// player who participates in a Match.
type player struct {
	acting.Actor
	// ID is used for identifying the player.
	id messages.UserID
	// IsAlive
	isAlive bool
	// lives is the amount of remaining lives.
	lives int
}

// team holds the players that belong to it.
type team struct {
	// bases is where players login and notify hits. As there should be support for
	// multiple bases in a team, we use a list here.
	bases []acting.Actor
	// monitors only provide readonly information.
	monitors []acting.Actor
	// players that belong to the team.
	players []*player
}

// Match is an actual team deathmatch match. Create one with NewMatch and start
// it with Start.
type Match struct {
	*games.BaseMatch
	// config describes how many teams play, etc.
	config Config
	// teams that participate in the match.
	teams []*team
	// monitors receive status information for the Match.
	monitors []acting.Actor
}

// NewMatch creates a new Match with the given ConfigRequest. Start it with
// Match.Start.
func NewMatch(baseMatch *games.BaseMatch, configRequest ConfigRequest) (*Match, error) {
	match := &Match{
		BaseMatch: baseMatch,
		teams:     make([]*team, 0),
	}
	// Set config.
	config, err := configFromRequest(baseMatch.Logger, configRequest)
	if err != nil {
		return nil, errors.Wrap(err, "config from request")
	}
	match.config = config
	return match, nil
}

func (match *Match) Abort(string) error {
	match.BaseMatch.Abort()
	panic("implement me")
}

func (match *Match) Start(ctx context.Context) error {
	match.BaseMatch.M.Lock()
	if match.BaseMatch.Phase != messages.MatchPhaseInit {
		match.BaseMatch.M.Unlock()
		return errors.NewMatchAlreadyStartedError()
	}
	match.Logger().Info("match start")
	match.BaseMatch.Phase = messages.MatchPhaseSetup
	match.BaseMatch.Start = time.Now()
	match.BaseMatch.M.Unlock()
	// Let's go.
	match.notifyMatchStatus()
	go match.run(ctx)
	return nil
	// TODO: Remember that we have a mutex for match state!!!
}

// run the Match.
func (match *Match) run(ctx context.Context) {
	go match.subscribeMatchAbort(ctx)
	// Castings.
	err := match.performRoleAssignment(ctx)
	if err != nil {
		games.AbortMatchBecauseOfErrorOrLog(match, errors.Wrap(err, "perform role assignments"))
		return
	}
	// We are ready.

}

// notifyMatchStatus notifies of the current games.MatchStatus.
func (match *Match) notifyMatchStatus() {
	match.BaseMatch.M.RLock()
	defer match.BaseMatch.M.RUnlock()
	// Count players.
	playerCount := 0
	for _, team := range match.teams {
		playerCount += len(team.players)
	}
	match.StatusUpdates <- messages.MessageMatchStatus{
		GameMode:    match.BaseMatch.GameMode,
		MatchPhase:  match.BaseMatch.Phase,
		MatchConfig: match.config,
		IsActive:    match.BaseMatch.IsActive,
		Start:       match.BaseMatch.Start,
		End:         match.BaseMatch.End,
		PlayerCount: playerCount,
	}
}

// subscribeMatchAbort subscribes to the messages.MessageTypeAbortMatch from the
// game master in order to abort the match.
func (match *Match) subscribeMatchAbort(ctx context.Context) {
	sub := acting.SubscribeMessageTypeAbortMatch(match.GameMaster)
	select {
	case <-ctx.Done():
		return
	case _, more := <-sub.Receive:
		if !more {
			return
		}
		acting.UnsubscribeOrLogError(sub.Newsletter)
		games.AbortMatchOrLog(match, "abort requested from game master")
	}
}

func (match *Match) Logger() *logrus.Logger {
	return match.BaseMatch.Logger
}

func (match *Match) performRoleAssignment(ctx context.Context) error {
	match.BaseMatch.M.Lock()
	defer match.BaseMatch.M.Unlock()
	// Assure that we are in setup phase.
	if match.BaseMatch.Phase != messages.MatchPhaseSetup {
		return errors.Error{
			Code:    errors.ErrInternal,
			Kind:    errors.KindMatchPhaseViolation,
			Message: fmt.Sprintf("perform role assignment only allowed in setup phase but was %v", match.BaseMatch.Phase),
		}
	}
	// Prepare casting.
	matchCasting := casting.NewCasting(match.BaseMatch.Agency)
	teamBaseRequests := make([]casting.ActorRequest, match.config.TeamCount)
	teamBaseMonitorRequests := make([]casting.ActorRequest, match.config.TeamCount)
	for i := uint(0); i < match.config.TeamCount; i++ {
		teamBaseRequests[i] = casting.ActorRequest{
			DisplayedName: fmt.Sprintf("Team Base %d", i),
			Key:           casting.RequestKey(fmt.Sprintf("team-base-%d", i)),
			Role:          acting.RoleTypeTeamBase,
			Min:           nulls.NewUInt32(1),
		}
		err := matchCasting.AddRequest(teamBaseRequests[i])
		if err != nil {
			return errors.Wrap(err, "add actor request for team base")
		}
		teamBaseMonitorRequests[i] = casting.ActorRequest{
			DisplayedName: "Team Base Monitor",
			Key:           "team-base-monitor",
			Role:          acting.RoleTypeTeamBaseMonitor,
		}
		err = matchCasting.AddRequest(teamBaseMonitorRequests[i])
		if err != nil {
			return errors.Wrap(err, "add actor request for team base monitor")
		}
	}
	matchMonitorRequest := casting.ActorRequest{
		DisplayedName: "Match Monitor",
		Key:           casting.RequestKey("match-monitor"),
		Role:          acting.RoleTypeMatchMonitor,
	}
	err := matchCasting.AddRequest(matchMonitorRequest)
	if err != nil {
		return errors.Wrap(err, "add actor request for match monitor")
	}
	// Perform casting.
	err = matchCasting.PerformAndHire(ctx, match.GameMaster)
	if err != nil {
		return errors.Wrap(err, "perform role assignment")
	}
	// Retrieve winners.
	// Team bases and monitors.
	match.teams = make([]*team, 0, len(teamBaseRequests))
	for i := uint(0); i < match.config.TeamCount; i++ {
		teamBases, err := matchCasting.GetWinners(teamBaseRequests[i].Key)
		if err != nil {
			return errors.Wrap(err, "retrieve team base winners")
		}
		teamBaseMonitors, err := matchCasting.GetWinners(teamBaseMonitorRequests[i].Key)
		if err != nil {
			return errors.Wrap(err, "retrieve team base monitor winners")
		}
		t := team{
			bases:    make([]acting.Actor, 0, len(teamBases)),
			monitors: make([]acting.Actor, 0, len(teamBaseMonitors)),
			players:  make([]*player, 0),
		}
		for _, base := range teamBases {
			t.bases = append(t.bases, base)
		}
		for _, monitor := range teamBaseMonitors {
			t.monitors = append(t.monitors, monitor)
		}
		match.teams = append(match.teams, &t)
	}
	// Match monitors.
	matchMonitors, err := matchCasting.GetWinners(matchMonitorRequest.Key)
	if err != nil {
		return errors.Wrap(err, "retrieve match monitor winners")
	}
	match.monitors = make([]acting.Actor, len(matchMonitors))
	for i, monitor := range matchMonitors {
		matchMonitors[i] = monitor
	}
	return nil
}
