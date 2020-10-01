package games

import (
	"github.com/LeFinal/masc-server/users"
)

type Player struct {
	UserId    users.UserId `json:"user_id"`
	Alive     bool         `json:"alive"`
	LifeCount int          `json:"life_count"`
}
