package devices

// Role is a set of abilities that a device can provide and so provide
// a certain functionality.
type Role string

const (
	// RoleScheduler schedules events
	RoleScheduler Role = "scheduler"
	// RoleGameMaster sets up and controls matches
	RoleGameMaster Role = "game-master"
	// RoleReferee can view the current match status and specific information
	RoleReferee Role = "ref"
	// RoleTeamBase provides team specific information and interaction
	RoleTeamBase Role = "team-base"
	// RolePlayerControl provides interaction for players
	RolePlayerControl Role = "player-control"
	// RoleMatchStatsCollector is allowed to request
	RoleMatchStatsCollector Role = "match-stats-collector"
	// RoleGlobalInformationCollector is allowed to request global information about the system, schedules and matches
	RoleGlobalInformationCollector Role = "global-information-collector"
	// RoleObjectiveTest provides a functionality for certain game modes and will be extended later
	RoleObjectiveTest Role = "objective-test"
)

type RoleAssignmentOffer struct {
	Role             Role
	Description      string
	CountRequired    int
	AssignedDevices  []Device
	AvailableDevices []Device
}

type RoleAssignment struct {
	Role Role `json:"role"`
}
