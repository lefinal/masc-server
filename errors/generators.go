package errors

//goland:noinspection SpellCheckingInspection

// NewResourceNotFoundError returns a new ErrNotFound error with kind
// KindResourceNotFound and the given message.
func NewResourceNotFoundError(message string, details Details) error {
	return Error{
		Code:    ErrNotFound,
		Kind:    KindResourceNotFound,
		Message: message,
		Details: details,
	}
}

// NewContextAbortedError returns a new ErrAborted error with kind
// KindContextAborted and the given operation in details.
func NewContextAbortedError(currentOperation string) error {
	return Error{
		Code:    ErrAborted,
		Kind:    KindContextAborted,
		Message: "context aborted",
		Details: Details{"currentOperation": currentOperation},
	}
}

// NewInvalidConfigRequestError returns a new ErrBadRequest error with kind
// KindInvalidConfigRequest and the given message.
func NewInvalidConfigRequestError(message string) error {
	return Error{
		Code:    ErrBadRequest,
		Kind:    KindInvalidConfigRequest,
		Message: message,
	}
}

// NewmatchAlreadyStartedError create
func NewMatchAlreadyStartedError() error {
	return Error{
		Code:    ErrInternal,
		Kind:    KindMatchAlreadyStarted,
		Message: "match already started",
	}
}
