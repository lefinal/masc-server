package messages

// MessageYouAreIn is used with MessageTypeYouAreIn.
type MessageYouAreIn struct {
	// ActorID is the assigned ID that the device needs to use from now on when acting.
	ActorID ActorID `json:"actor_id"`
	// Role is the role the device got assigned to.
	Role string `json:"role"`
}
