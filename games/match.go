package games

import "github.com/google/uuid"

type Match struct {
	Id   uuid.UUID `json:"id"`
	Mode GameMode  `json:"mode"`
}

func (g *Match) Identify() uuid.UUID {
	return g.Id
}
