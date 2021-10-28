package stores

import (
	"github.com/LeFinal/masc-server/messages"
	"time"
)

// User holds information regarding a registered user.
type User struct {
	// ID identifies the player.
	ID messages.UserID
	// Rank is the rank of the player.
	Rank messages.PlayerRank
	// BattleTag is something like B-15.
	BattleTag string
	// CallSign is how the player is called.
	CallSign string
	// Mantra is a customizable text that may be shown when the player joins a game.
	Mantra string
	// JoinDate is when the player was created.
	JoinDate time.Time
	// ProfilePicture is the URL where the profile picture is stored.
	ProfilePicture string
}

func (user *User) Message() messages.User {
	return messages.User{
		ID:             user.ID,
		Rank:           user.Rank,
		BattleTag:      user.BattleTag,
		CallSign:       user.CallSign,
		Mantra:         user.Mantra,
		JoinDate:       user.JoinDate,
		ProfilePicture: user.ProfilePicture,
	}
}
