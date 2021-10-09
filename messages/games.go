package messages

// MatchID is used in order to identify a games.Match.
type MatchID string

// MatchBase is used for all messages that are related a certain match.
type MatchBase struct {
	// MatchID is used for identifying a match.
	MatchID MatchID `json:"match_id"`
}
