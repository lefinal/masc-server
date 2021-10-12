package mode_team_deathmatch

import (
	"github.com/LeFinal/masc-server/acting"
	"github.com/LeFinal/masc-server/games"
	"github.com/LeFinal/masc-server/messages"
)

// Config for a Match.
type Config struct {
	// TeamCount determines how many teams are playing.
	TeamCount int `json:"team_count"`
	// LivesPerPlayer determines how many lives each player owns. Matchmaking might
	// increase lives for certain players in order to create a fair match.
	LivesPerPlayer int `json:"lives_per_player"`
	// RespawnTeamSize determines how many players of a team are needed in order to respawn.
	RespawnTeamSize float64 `json:"respawn_team_size"`
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
	// players that belong to the team.
	players []*player
}

type Match struct {
	games.BaseMatch
	// teams that participate in the match.
	teams []*team
}

func (match *Match) Abort() error {
	panic("implement me")
}

func (match *Match) Start() {
	// Subscribe to match abort.
	// subMatchAbort, _ := match.GameMaster.SubscribeMessageType(messages.MessageTypeAbortMatch)
	// go func() {
	// 	for _ = range subMatchAbort {
	// 		err := match.Abort()
	// 		errors.Log(match.Logger, errors.Wrap(err, "abort match"))
	// 	}
	// }()
	// ra := games.NewCasting(match.BaseMatch.Agency, match.BaseMatch.GameMaster)
	match.Logger.Info("match start.")
	panic("implement me")
	// TODO: Remember that we have a mutex for match state!!!
}
