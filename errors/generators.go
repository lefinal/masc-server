package errors

import (
	"database/sql"
	nativeerrors "errors"
	"github.com/jackc/pgconn"
	"strings"
)

//goland:noinspection SpellCheckingInspection

// NewInternalError creates a new ErrInternal with the given message and
// details.
func NewInternalError(message string, details Details) error {
	return NewInternalErrorFromErr(nil, message, details)
}

// NewInternalErrorFromErr creates a new ErrInternal with the given message,
// original error and details.
func NewInternalErrorFromErr(err error, message string, details Details) error {
	return Error{
		Code:    ErrInternal,
		Err:     err,
		Message: message,
		Details: details,
	}
}

// NewResourceNotFoundError returns a new ErrNotFound error with the given
// message.
func NewResourceNotFoundError(message string, details Details) error {
	return Error{
		Code:    ErrNotFound,
		Message: message,
		Details: details,
	}
}

// NewContextAbortedError returns a new ErrAborted error with the given
// operation in details.
func NewContextAbortedError(currentOperation string) error {
	return Error{
		Code:    ErrAborted,
		Message: "context aborted",
		Details: Details{"currentOperation": currentOperation},
	}
}

// NewInvalidConfigRequestError returns a new ErrBadRequest error and the given
// message.
func NewInvalidConfigRequestError(message string) error {
	return Error{
		Code:    ErrBadRequest,
		Message: message,
	}
}

func NewJSONError(err error, operation string, blameUser bool) error {
	e := Error{
		Err:     err,
		Message: operation,
	}
	if blameUser {
		e.Code = ErrBadRequest
	} else {
		e.Code = ErrInternal
	}
	return e
}

// NewmatchAlreadyStartedError create
func NewMatchAlreadyStartedError() error {
	return Error{
		Code:    ErrInternal,
		Message: "match already started",
	}
}

// NewQueryToSQLError creates a new ErrInternal error.
func NewQueryToSQLError(err error, details Details) error {
	return Error{
		Code:    ErrInternal,
		Err:     err,
		Message: "query to sql",
		Details: details,
	}
}

// NewScanSingleDBRowError returns the correct error for scanning a single row
// from QueryRow. If the error is because of no rows, an ErrNotFound error is
// returned with the given message. Otherwise, an ErrInternal error is returned.
func NewScanSingleDBRowError(err error, notFoundMessage string, query string) error {
	// Check if error is because of no rows --> not found.
	if err == sql.ErrNoRows {
		return Error{
			Code:    ErrNotFound,
			Message: notFoundMessage,
		}
	}
	return NewScanDBRowError(err, query)
}

// NewScanDBRowError returns an ErrInternal error. If you are using this to scan
// a row from QueryRow, then use NewScanSingleDBRowError for generating an
// ErrNotFound error if necessary.
func NewScanDBRowError(err error, query string) error {
	return Error{
		Code:    ErrInternal,
		Err:     err,
		Message: "scan db row",
		Details: Details{"query": query},
	}
}

// NewExecQueryError creates a new ErrBadRequest error if the error is a constraint violation or data exception.
// Otherwise, an ErrInternal error is created. The query will be added to the details as well as the additional ones.
// Error codes taken from https://en.wikipedia.org/wiki/SQLSTATE.
func NewExecQueryError(err error, query string, details Details) error {
	if details == nil {
		details = Details{}
	}
	details["query"] = query
	// Check if error is postgres error.
	var pgErr *pgconn.PgError
	if nativeerrors.As(err, &pgErr) {
		// It is postgres error.
		details["pgErr"] = *pgErr
		details["sqlstate"] = pgErr.Code
		// Check if constraint violation.
		if strings.HasPrefix(pgErr.Code, "23") {
			return Error{
				Code:    ErrBadRequest,
				Message: "exec db query: constraint violation",
				Err:     err,
				Details: details,
			}
		}
		if strings.HasPrefix(pgErr.Code, "22") {
			return Error{
				Code:    ErrBadRequest,
				Message: "exec db query: data exception",
				Err:     err,
				Details: details,
			}
		}
		// Check if syntax error.
		if strings.HasPrefix(pgErr.Code, "42") {
			return Error{
				Code:    ErrInternal,
				Message: "exec db query: syntax error",
				Err:     err,
				Details: details,
			}
		}
		// Otherwise, probably internal error.
		return Error{
			Code:    ErrInternal,
			Message: "exec db query",
			Err:     err,
			Details: details,
		}
	}
	if nativeerrors.Is(err, sql.ErrTxDone) {
		return Error{
			Code:    ErrInternal,
			Message: "exec db query",
			Err:     err,
			Details: details,
		}
	}
	if nativeerrors.Is(err, sql.ErrConnDone) {
		return Error{
			Code:    ErrFatal,
			Message: "connection done",
			Err:     err,
			Details: details,
		}
	}
	// Any other internal error.
	return Error{
		Code:    ErrInternal,
		Message: "exec db query",
		Err:     err,
		Details: details,
	}
}

// NewDBTxBeginError returns an errors.Error with code errors.ErrInternal.
func NewDBTxBeginError(err error) error {
	return Error{
		Code:    ErrInternal,
		Message: "begin tx",
		Err:     err,
	}
}

// NewDBTxCommitError creates a new error for when a tx cannot be committed.
func NewDBTxCommitError(err error) error {
	return Error{
		Code:    ErrInternal,
		Message: "tx commit",
		Err:     err,
	}
}

// NewBadRequestErr creates a new error with ErrBadRequest.
func NewBadRequestErr(message string, details Details) error {
	return Error{
		Code:    ErrBadRequest,
		Message: message,
		Details: details,
	}
}
