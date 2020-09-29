package games

type MatchConfigPreset struct {
	Name        string      `json:"name"`
	Description string      `json:"description"`
	MatchConfig interface{} `json:"match_config"`
}

type DeathmatchMatchConfig struct {
}
