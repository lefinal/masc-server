package games

import "github.com/google/uuid"

type Player struct {
	UserId    uuid.UUID `json:"user_id"`
	TeamId    uuid.UUID `json:"team_id"`
	Alive     bool      `json:"alive"`
	LifeCount int       `json:"life_count"`
}
