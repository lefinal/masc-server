package model

import "github.com/google/uuid"

type GameMode string

const (
	GameModeTeamDeathmatch GameMode = "team-deathmatch"
	GameMode1V1            GameMode = "1v1"
	GameModeElimination    GameMode = "elimination"
)

type Game struct {
	Event Event
	Mode  GameMode `json:"mode"`
}

func (g *Game) Identify() uuid.UUID {
	return g.Event.Id
}

func (g *Game) ProvideEvent() Event {
	return g.Event
}
