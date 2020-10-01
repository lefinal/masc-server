package games

type MatchConfigPreset struct {
	Name        string
	Description string
	TeamConfigs []TeamConfig
	MatchConfig interface{}
}

type DeathmatchMatchConfig struct {
}
