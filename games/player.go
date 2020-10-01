package games

import (
	"github.com/LeFinal/masc-server/users"
)

type Player struct {
	UserId    users.UserId
	Alive     bool
	LifeCount int
}
