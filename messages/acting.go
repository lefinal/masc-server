package messages

import "github.com/gobuffalo/nulls"

// MessageYouAreIn is used with MessageTypeYouAreIn.
type MessageYouAreIn struct {
	// ActorID is the assigned ID that the device needs to use from now on when
	// acting.
	ActorID ActorID `json:"actor_id"`
	// Role is the role the device got assigned to.
	Role Role `json:"role"`
}

// RoleAssignmentKey that identifies a role in an RoleAssignmentOrder and RoleAssignment.
type RoleAssignmentKey string

// RoleAssignmentOrder is used in MessageRequestRoleAssignments.
type RoleAssignmentOrder struct {
	// DisplayedName is a human-readable description for what this role does.
	DisplayedName string `json:"displayed_name"`
	// Key that identifies this role. This is needed for example when 2 team bases
	// are needed. This allows the assigner to distinguish between them because both
	// share the same role type.
	Key string `json:"key"`
	// Role is the role type that is requested.
	Role Role `json:"role"`
	// Min is optional the minimum amount of requested actor assignments.
	Min nulls.UInt32 `json:"min"`
	// Max is optional the maximum amount of requested actor assignments.
	Max nulls.UInt32 `json:"max"`
}

// MessageRequestRoleAssignments is used with MessageTypeRequestRoleAssignments.
type MessageRequestRoleAssignments struct {
	// Orders for assignments.
	Orders []RoleAssignmentOrder `json:"orders"`
}

type RoleAssignment struct {
	// Key is the key which was used from MessageRequestRoleAssignments.
	Key string `json:"key"`
	// ActorID is the id of the assigned acting.Actor.
	ActorID ActorID `json:"actor_id"`
}

// MessageRoleAssignments is used with MessageTypeRoleAssignments.
type MessageRoleAssignments struct {
	// Assignments that were made.
	Assignments []RoleAssignment `json:"assignments"`
}
