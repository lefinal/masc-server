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
	// KindAlreadyPerformingRoleAssignment is used when a role assignment is
	// requested although already performing the assignment.
	KindAlreadyPerformingRoleAssignment Kind = "already-performing-role-assignment"
	// KindContextAborted is used when we were currently performing an operation but
	// the context got aborted.
	KindContextAborted     Kind = "context-aborted"
	KindCommunicationWrite Kind = "communication-write"
	KindDecodeJSON         Kind = "parse-request-body-as-json"
	KindEncodeJSON         Kind = "encode-json"
	// KindForbiddenMessage is used when the protocol is being violated due to a
	// message with currently forbidden type.
	KindForbiddenMessage Kind = "protocol-violation"
	// KindMalformedID is used when a passed ID is not in uuid.UUID format.
	KindMalformedID Kind = "malformed-id"
	KindMissingID   Kind = "missing-id"
	// KindNotRunning is used when actions are performed that require a running
	// entity.
	KindNotRunning       Kind = "not-running"
	KindResourceNotFound Kind = "resource-not-found"
	// KindRoleAssignmentAlreadyDone is used when a role assignment is requested
	// although the assigner is already done.
	KindRoleAssignmentAlreadyDone Kind = "role-assignment-already-done"
	KindUUIDGenFail               Kind = "uuid-gen-fail"
	KindUnexpected                Kind = "unexpected"
	// KindUnknown is used for different unknown type values that are too special
	// for creating separate error kinds.
	KindUnknown Kind = "unknown"
	// KindUnknownActor is used when a message with an unknown actor is received.
	KindUnknownActor Kind = "unknown-actor"
	// KindUnknownDevice is used when an unknown device is being requested.
	KindUnknownDevice Kind = "unknown-device"
	// KindUnknownRole is used when a role is unknown.
	KindUnknownRole Kind = "unknown-role"
)
