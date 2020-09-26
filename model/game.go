package model

import (
	"github.com/google/uuid"
	"time"
)

type GameMode string

const (
	GameModeTeamDeathmatch GameMode = "team-deathmatch"
	GameMode1V1                     = "1v1"
	GameModeElimination             = "elimination"
)

type Game struct {
	Id          uuid.UUID `json:"id"`
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Mode        GameMode  `json:"mode"`
	StartTime   time.Time `json:"start_time"`
	EndTime     time.Time `json:"end_time"`
}

func (g *Game) EventId() uuid.UUID {
	return g.Id
}

func (g *Game) EventType() EventType {
	return EventTypeGame
}

func (g *Game) EventTitle() string {
	return g.Title
}

func (g *Game) EventDescription() string {
	return g.Description
}

func (g *Game) EventStartTime() time.Time {
	return g.StartTime
}

func (g *Game) EventEndTime() time.Time {
	return g.EndTime
}
