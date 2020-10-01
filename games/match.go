package games

import "github.com/google/uuid"

type MatchId uuid.UUID

type Match struct {
	Id   MatchId
	Mode GameMode
}

func (g *Match) Identify() uuid.UUID {
	return uuid.UUID(g.Id)
}
