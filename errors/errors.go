package errors

import (
	"encoding/json"
	"fmt"
	"github.com/sirupsen/logrus"
)

// Details holds additional error details that can be viewed and logged.
type Details map[string]interface{}

// Error is the general error type for appearing errors in MASC.
type Error struct {
	// Code is the error code.
	Code Code
	// Kind is the kind of code.
	Kind Kind
	// Err is the original error that occurred.
	Err error
	// Message is the manually created message that can be used in order to trace the error.
	Message string
	// Details holds any error details.
	Details Details
}

func (e Error) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Err)
	}
	return e.Message
}

// Cast casts the given error to Error. If the given one is not of type Error, an unknown one with error code
// ErrUnexpected is created and false returned
func Cast(err error) (Error, bool) {
	if e, ok := err.(Error); ok {
		return e, ok
	}
	e := Error{
		Code:    ErrUnexpected,
		Kind:    KindUnexpected,
		Err:     err,
		Message: "unknown operation",
		Details: make(map[string]interface{}),
	}
	return e, false
}

// Wrap wraps the given error with the given message.
func Wrap(err error, message string, details Details) error {
	e, ok := Cast(err)
	// Check whether to append to message or replace.
	var errMsg string
	if ok {
		errMsg = fmt.Sprintf("%s: %s", message, e.Message)
	} else {
		errMsg = message
	}
	// Add details.
	if details != nil && e.Details == nil {
		e.Details = make(Details)
	}
	for k, v := range details {
		// Check if detail with same key already set.
		if originalV, ok := e.Details[k]; ok {
			// Add prefix to original key. Original value will be overwritten after this
			// block.
			e.Details[fmt.Sprintf("_%s", k)] = originalV
		}
		e.Details[k] = v
	}
	wrappedErr := Error{
		Code:    e.Code,
		Kind:    e.Kind,
		Err:     e.Err,
		Message: errMsg,
		Details: e.Details,
	}
	return wrappedErr
}

// FromErr creates an Error with the given details.
func FromErr(message string, code Code, kind Kind, err error, details Details) error {
	createdError := Error{
		Code:    code,
		Kind:    kind,
		Err:     err,
		Message: message,
		Details: details,
	}
	createdError.Err = err
	return createdError
}

// detailsAsJSON encodes the Details of the given Error as JSON string.
func detailsAsJSON(logger *logrus.Entry, err error) []byte {
	e, _ := Cast(err)
	if e.Details == nil {
		return nil
	}
	b, err := json.Marshal(e.Details)
	if err != nil {
		Log(logger, Error{
			Code:    ErrInternal,
			Kind:    KindEncodeJSON,
			Message: "marshal error details",
			Err:     err,
			Details: Details{
				"toMarshal": fmt.Sprintf("%+v", e.Details),
			},
		})
		return nil
	}
	return b
}

// Log logs the given error with its details. If the error is ErrFatal, the error will be logged is fatal.
func Log(logger *logrus.Entry, err error) {
	e, _ := Cast(err)
	fields := logrus.Fields{
		"err_code":    e.Code,
		"err_kind":    e.Kind,
		"err_details": string(detailsAsJSON(logger, err)),
	}

	// Add each details entry as separate field for better readability.
	for k, v := range e.Details {
		fields[fmt.Sprintf("err_details_v_%s", k)] = fmt.Sprintf("%+v", v)
	}

	if e.Err != nil {
		fields["err_orig"] = err.Error()
	}
	entry := logger.WithFields(fields)
	switch e.Code {
	case ErrBadRequest, ErrProtocolViolation, ErrNotFound:
		entry.Debug(e.Error())
	case ErrFatal:
		entry.Fatal(e.Error())
	default:
		entry.Error(e.Error())
	}
}

// Prettify returns a detailed error string with error details.
func Prettify(err error) string {
	e, _ := Cast(err)
	return fmt.Sprintf("Code: %s\nKind: %s\nOriginal Error: %+v\nMessage: %s\nDetails: %s\n",
		e.Code, e.Kind, e.Err, e.Message, detailsAsJSON(nil, e))
}

// BlameUser checks if the given error is ErrBadRequest, ErrProtocolViolation or
// ErrNotFound.
func BlameUser(err error) bool {
	e, ok := Cast(err)
	if !ok {
		// Unexpected.
		return false
	}
	switch e.Code {
	case ErrBadRequest,
		ErrProtocolViolation,
		ErrNotFound:
		return true
	}
	// Otherwise.
	return false
}
