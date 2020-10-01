package games

import "github.com/google/uuid"

type TeamId uuid.UUID

type TeamConfig struct {
	TeamId              TeamId `json:"team_id"`
	Name                string `json:"name"`
	MaxPlayers          int    `json:"max_players"`
	LifeCount           int    `json:"life_count"`
	MinRespawnGroupSize int    `json:"min_respawn_group_size"`
}

type Team struct {
	TeamConfig TeamConfig `json:"team_config"`
	Players    []Player   `json:"players"`
}
