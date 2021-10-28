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
	"golang.org/x/sync/errgroup"
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
	// ID is used for identifying the player.
	id messages.UserID
	// IsAlive
	isAlive bool
	// lives is the amount of remaining lives.
	lives uint
}

// team holds the players that belong to it.
type team struct {
	// key is the identifier for the team.
	key string
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
	match.M.Lock()
	if match.Phase != messages.MatchPhaseInit {
		match.M.Unlock()
		return errors.NewMatchAlreadyStartedError()
	}
	match.Logger().Info("match start")
	match.Phase = messages.MatchPhaseSetup
	match.BaseMatch.Start = time.Now()
	match.M.Unlock()
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
	match.changePhase(messages.MatchPhaseSetup)
	// Allow player joins and await ready state.
	err = match.playerJoinsAndAwaitReady(ctx)
	if err != nil {
		games.AbortMatchBecauseOfErrorOrLog(match, errors.Wrap(err, "player joins and await ready"))
		return
	}
	// Setup.
	match.fillTeams()
	match.handOutLives()
}

// changePhase changes the match phase and broadcasts the match status via
// notifyMatchStatus.
func (match *Match) changePhase(phase messages.MatchPhase) {
	match.M.Lock()
	match.Phase = phase
	match.M.Unlock()
	match.notifyMatchStatus()
}

// notifyMatchStatus notifies of the current games.MatchStatus.
func (match *Match) notifyMatchStatus() {
	match.M.RLock()
	defer match.M.RUnlock()
	// Count players.
	playerCount := 0
	for _, team := range match.teams {
		playerCount += len(team.players)
	}
	match.StatusUpdates <- messages.MessageMatchStatus{
		GameMode:    match.GameMode,
		MatchPhase:  match.Phase,
		MatchConfig: match.config,
		IsActive:    match.IsActive,
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

func (match *Match) participatingActors() []acting.Actor {
	match.M.RLock()
	defer match.M.RUnlock()
	participating := make([]acting.Actor, 0)
	for _, t := range match.teams {
		participating = append(participating, t.bases...)
		participating = append(participating, t.monitors...)
	}
	participating = append(participating, match.monitors...)
	return participating
}

// performRoleAssignment assigns all roles that are needed in order to run the
// match.
func (match *Match) performRoleAssignment(ctx context.Context) error {
	match.M.Lock()
	defer match.M.Unlock()
	// Assure that we are in setup phase.
	if match.Phase != messages.MatchPhaseSetup {
		return errors.Error{
			Code:    errors.ErrInternal,
			Kind:    errors.KindMatchPhaseViolation,
			Message: fmt.Sprintf("perform role assignment only allowed in setup phase but was %v", match.Phase),
		}
	}
	// Prepare casting.
	matchCasting := casting.NewCasting(match.Agency)
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
			key:      fmt.Sprintf("team-%d", i),
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

// playerJoinsAndAwaitReady accepts player joins and blocks until everybody is
// ready.
func (match *Match) playerJoinsAndAwaitReady(ctx context.Context) error {
	wg, _ := errgroup.WithContext(ctx)
	handlers, cancelHandlers := context.WithCancel(ctx)
	// Open a player join office for each team.
	match.M.RLock()
	playerJoinOffices := make([]*games.PlayerJoinOffice, 0, match.config.TeamCount)
	for _, team := range match.teams {
		playerJoinOffice := &games.PlayerJoinOffice{
			Team:             games.PlayerManagementTeamKey(team.key),
			Logger:           match.BaseMatch.Logger,
			PlayerProvider:   match.PlayerProvider,
			PlayerManagement: match.PlayerManagement,
		}
		playerJoinOffice.Open(ctx, team.bases...)
	}
	match.M.RUnlock()
	// Handle player updates.
	wg.Go(func() error {
		for {
			select {
			case <-handlers.Done():
				return nil
			case update := <-match.PlayerManagementUpdates:
				err := games.BroadcastPlayerManagementUpdate(update, match.PlayerProvider, match.participatingActors()...)
				if err != nil {
					errors.Log(match.Logger(), errors.Wrap(err, "broadcast player management update"))
				}
			}
		}
	})
	// Request ready-state from team bases and game master.
	match.M.RLock()
	requestReadyFrom := make([]acting.Actor, 0, len(match.teams)+1)
	requestReadyFrom = append(requestReadyFrom, match.GameMaster)
	for _, team := range match.teams {
		requestReadyFrom = append(requestReadyFrom, team.bases...)
	}
	match.M.RUnlock()
	readyStateUpdates := make(chan games.ReadyStateUpdate)
	// Handle ready-state updates.
	wg.Go(func() error {
		for {
			select {
			case <-handlers.Done():
				return nil
			case update := <-readyStateUpdates:
				err := games.BroadcastReadyStateUpdate(update, match.participatingActors()...)
				if err != nil {
					errors.Log(match.Logger(), errors.Wrap(err, "broadcast ready-state update"))
				}
			}
		}
	})
	err := games.RequestAndAwaitReady(ctx, readyStateUpdates, requestReadyFrom...)
	var originalErr error
	if err != nil {
		originalErr = errors.Wrap(err, "request and await ready")
	}
	// Close all player join offices.
	for _, office := range playerJoinOffices {
		err = office.Close()
		if err != nil {
			err = errors.Wrap(err, "close player join office")
			if originalErr != nil {
				originalErr = err
			} else {
				// Only log the error.
				errors.Log(match.Logger(), err)
			}
		}
	}
	// Cancel handlers.
	cancelHandlers()
	if handlerErr := wg.Wait(); handlerErr != nil {
		handlerErr = errors.Wrap(err, "handlers")
		if originalErr != nil {
			originalErr = handlerErr
		} else {
			errors.Log(match.Logger(), handlerErr)
		}
	}
	// Done.
	return originalErr
	// TODO: Test
}

// fillTeams uses the Match.config and player management for filling Match.teams
// with initial players.
func (match *Match) fillTeams() {
	match.M.Lock()
	defer match.M.Unlock()
	for _, team := range match.teams {
		team.players = make([]*player, 0)
		for _, playerInTeam := range match.PlayerManagement.PlayersInTeam(games.PlayerManagementTeamKey(team.key)) {
			team.players = append(team.players, &player{
				id:      playerInTeam.User,
				isAlive: false,
			})
		}
	}
	// TODO: Test
}

func (match *Match) handOutLives() {
	match.M.Lock()
	defer match.M.Unlock()
	// TODO: Add lives to players and balance for uneven teams!
}
