package games

type MatchConfigPreset struct {
	Name        string       `json:"name"`
	Description string       `json:"description"`
	TeamConfigs []TeamConfig `json:"team_configs"`
	MatchConfig interface{}  `json:"match_config"`
}

type DeathmatchMatchConfig struct {
}
