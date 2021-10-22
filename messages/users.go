package messages

import "time"

// User holds information regarding a registered user.
type User struct {
	// ID identifies the player.
	ID UserID `json:"id"`
	// Rank is the rank of the player.
	Rank PlayerRank `json:"rank"`
	// BattleTag is something like B-15.
	BattleTag string `json:"battle_tag"`
	// CallSign is how the player is called.
	CallSign string `json:"call_sign"`
	// Mantra is a customizable text that may be shown when the player joins a game.
	Mantra string `json:"mantra"`
	// JoinDate is when the player was created.
	JoinDate time.Time `json:"join_date"`
	// ProfilePicture is the URL where the profile picture is stored.
	ProfilePicture string `json:"profile_picture"`
}
