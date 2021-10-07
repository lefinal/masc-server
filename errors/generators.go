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
