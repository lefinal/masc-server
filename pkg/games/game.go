package games

import (
	"github.com/google/uuid"
	"masc-server/pkg/scheduling"
)

type GameMode string

const (
	GameModeTeamDeathmatch GameMode = "team-deathmatch"
	GameMode1V1            GameMode = "1v1"
	GameModeElimination    GameMode = "elimination"
)

type Game struct {
	Event scheduling.Event
	Mode  GameMode `json:"mode"`
}

func (g *Game) Identify() uuid.UUID {
	return g.Event.Id
}

func (g *Game) ProvideEvent() scheduling.Event {
	return g.Event
}
