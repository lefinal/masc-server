package games

import "github.com/google/uuid"

type MatchId uuid.UUID

type Match struct {
	Id   MatchId  `json:"id"`
	Mode GameMode `json:"mode"`
}

func (g *Match) Identify() uuid.UUID {
	return uuid.UUID(g.Id)
}
