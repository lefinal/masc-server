package errors

import "fmt"

type ErrorCode string

type MascError struct {
	Where     string
	ErrorCode ErrorCode
}

// NewMascError creates a new error with given location, and the error code.
// For propagating use PropagateMascError.
func NewMascError(where string, errorCode ErrorCode) *MascError {
	return &MascError{
		Where:     where,
		ErrorCode: errorCode,
	}
}

// PropagateMascError extends the where field of the error by the given string.
func PropagateMascError(where string, error *MascError) *MascError {
	error.Where = fmt.Sprintf("%s: %s", where, error.Where)
	return error
}

// NewMascErrorFromError creates a new error with the given location, error code and an error.
func NewMascErrorFromError(where string, errorCode ErrorCode, err error) *MascError {
	return NewMascError(fmt.Sprintf("%s: %s", where, err), errorCode)
}

func (m *MascError) Error() string {
	return fmt.Sprintf("%s: %v", m.Where, m.ErrorCode)
}
