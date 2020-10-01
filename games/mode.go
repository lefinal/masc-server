package games

type GameMode string

// We are currently only using one mode because all the other ones (1v1, elimination, ...) can be set
// in through match config.
const (
	GameModeDeathmatch GameMode = "team-deathmatch"
)
