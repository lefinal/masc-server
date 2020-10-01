package games

import "github.com/google/uuid"

type TeamId uuid.UUID

type TeamConfig struct {
	TeamId              TeamId
	Name                string
	MaxPlayers          int
	LifeCount           int
	MinRespawnGroupSize int
}

type Team struct {
	TeamConfig TeamConfig
	Players    []Player
}
