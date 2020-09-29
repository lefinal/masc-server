package games

import "github.com/google/uuid"

type MatchAttributeKey string

const (
	TeamCountMatchAttribute   MatchAttributeKey = "team-count"
	PlayerCountMatchAttribute MatchAttributeKey = "player-count"
	LifeCountMatchAttribute   MatchAttributeKey = "life-count"
)

type Match struct {
	Id   uuid.UUID `json:"id"`
	Mode GameMode  `json:"mode"`
}

func (g *Match) Identify() uuid.UUID {
	return g.Id
}
