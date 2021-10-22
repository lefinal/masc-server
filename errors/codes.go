package errors

type Code string

const (
	ErrAborted           Code = "aborted"
	ErrBadRequest        Code = "bad-request"
	ErrCommunication     Code = "communication"
	ErrProtocolViolation Code = "protocol-violation"
	ErrFatal             Code = "fatal"
	ErrNotFound          Code = "not-found"
	ErrInternal          Code = "internal"
	ErrSadLife           Code = "sad-life"
	ErrUnexpected        Code = "unexpected"
)

type Kind string

const (
	// KindActorAlreadyHired is used when someone wants to hire an acting.Actor that
	// has already been hired.
	KindActorAlreadyHired Kind = "actor-already-hired"
	// KindActorNotHired is used when someone instructs an acting.Actor to send a
	// message although not hired yet.
	KindActorNotHired Kind = "actor-not-hired"
	// KindAlreadyPerformingCasting is used when a role assignment is
	// requested although already performing the assignment.
	KindAlreadyPerformingCasting Kind = "already-performing-role-assignment"
	// KindCastingAborted is used when an expected
	// messages.MessageTypeRoleAssignments could not be received.
	KindCastingAborted Kind = "casting-aborted"
	// KindCastingAlreadyDone is used when a role assignment is requested
	// although the assigner is already done.
	KindCastingAlreadyDone Kind = "role-assignment-already-done"
	// KindCastingNotDone is used when winners are retrieved from a casting which is
	// not done yet.
	KindCastingNotDone Kind = "casting-not-done"
	// KindContextAborted is used when we were currently performing an operation but
	// the context got aborted.
	KindContextAborted Kind = "context-aborted"
	// KindCountDoesNotMatchExpected is used when a number of entities does not
	// match the expected count.
	KindCountDoesNotMatchExpected Kind = "count-does-not-match-expected"
	KindDecodeJSON                Kind = "parse-request-body-as-json"
	KindEncodeJSON                Kind = "encode-json"
	// KindForbiddenMessage is used when the protocol is being violated due to a
	// message with currently forbidden type.
	KindForbiddenMessage Kind = "protocol-violation"
	// KindInvalidCastingRequest is used when an invalid games.ActorRequest is added
	// to a casting.
	KindInvalidCastingRequest Kind = "invalid-casting-request"
	// KindInvalidConfigRequest is used when an invalid config is passed to match
	// creation.
	KindInvalidConfigRequest Kind = "invalid-config-request"
	// KindInvalidRoleAssignments is used when an invalid assignment with
	// messages.MessageRoleAssignments was received.
	KindInvalidRoleAssignments Kind = "invalid-role-assignments"
	// KindMatchAlreadyStarted is used when a games.Match is tried to be started,
	// although it is already running.
	KindMatchAlreadyStarted Kind = "match-already-started"
	// KindMalformedID is used when a passed ID is not in uuid.UUID format.
	KindMalformedID Kind = "malformed-id"
	// KindMatchPhaseViolation is used for operations that were performed although
	// not in the expected match phase.
	KindMatchPhaseViolation Kind = "match-phase-violation"
	KindMissingID           Kind = "missing-id"
	// KindNotRunning is used when actions are performed that require a running
	// entity.
	KindNotRunning Kind = "not-running"
	// KindPlayerAlreadyJoined is used when a player wants to join a match but has
	// already joined.
	KindPlayerAlreadyJoined Kind = "player-already-joined"
	// KindPlayerNotJoined is used when a player has not joined the match yet.
	KindPlayerNotJoined  Kind = "player-not-joined"
	KindResourceNotFound Kind = "resource-not-found"
	KindUUIDGenFail      Kind = "uuid-gen-fail"
	KindUnexpected       Kind = "unexpected"
	// KindUnknown is used for different unknown type values that are too special
	// for creating separate error kinds.
	KindUnknown Kind = "unknown"
	// KindUnknownActor is used when a message with an unknown actor is received.
	KindUnknownActor Kind = "unknown-actor"
	// KindUnknownCastingKey is used when winners are retrieved from a casting that
	// were not requested.
	KindUnknownCastingKey Kind = "unknown-casting-key"
	// KindUnknownDevice is used when an unknown device is being requested.
	KindUnknownDevice Kind = "unknown-device"
	// KindUnknownRole is used when a role is unknown.
	KindUnknownRole Kind = "unknown-role"
)
