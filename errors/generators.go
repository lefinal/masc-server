package errors

import (
	"database/sql"
	nativeerrors "errors"
	"github.com/jackc/pgconn"
	"strings"
)

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

func NewJSONError(err error, operation string, blameUser bool) error {
	e := Error{
		Kind:    KindJSON,
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
		Kind:    KindMatchAlreadyStarted,
		Message: "match already started",
	}
}

// NewQueryToSQLError creates a new ErrInternal error with kind KindQueryToSQL.
func NewQueryToSQLError(err error, details Details) error {
	return Error{
		Code:    ErrInternal,
		Kind:    KindQueryToSQL,
		Err:     err,
		Message: "query to sql",
		Details: details,
	}
}

// NewScanSingleDBRowError returns the correct error for scanning a single row from QueryRow. If the error is because
// of no rows, an ErrNotFound error is returned with the given message. Otherwise, an ErrInternal error with kind
// KindScanDBRow is returned.
func NewScanSingleDBRowError(notFoundMessage string, err error, details Details) error {
	// Check if error is because of no rows --> not found.
	if err == sql.ErrNoRows {
		return Error{
			Code:    ErrNotFound,
			Kind:    KindResourceNotFound,
			Message: notFoundMessage,
			Details: details,
		}
	}
	return NewScanDBRowError(err, details)
}

// NewScanDBRowError returns an ErrInternal error with kind KindScanDBRow is returned.
// If you are using this to scan a row from QueryRow, then use NewScanSingleDBRowError for generating an ErrNotFound
// error if necessary.
func NewScanDBRowError(err error, details Details) error {
	return Error{
		Code:    ErrInternal,
		Kind:    KindScanDBRow,
		Err:     err,
		Message: "scan db row",
		Details: details,
	}
}

// NewExecQueryError creates a new ErrBadRequest error if the error is a constraint violation or data exception.
// Otherwise, an ErrInternal error is created. The query will be added to the details as well as the additional ones.
// Error codes taken from https://en.wikipedia.org/wiki/SQLSTATE.
func NewExecQueryError(err error, query string, details Details) error {
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
				Kind:    KindDBConstraintViolation,
				Message: "exed db query: constraint violation",
				Err:     err,
				Details: details,
			}
		}
		if strings.HasPrefix(pgErr.Code, "22") {
			return Error{
				Code:    ErrBadRequest,
				Kind:    KindDBDataException,
				Message: "exec db query: data exception",
				Err:     err,
				Details: details,
			}
		}
		// Check if syntax error.
		if strings.HasPrefix(pgErr.Code, "42") {
			return Error{
				Code:    ErrInternal,
				Kind:    KindDBSyntaxError,
				Message: "exec db query: syntax error",
				Err:     err,
				Details: details,
			}
		}
		// Otherwise, probably internal error.
		return Error{
			Code:    ErrInternal,
			Kind:    KindDBQuery,
			Message: "exec db query",
			Err:     err,
			Details: details,
		}
	}
	if nativeerrors.Is(err, sql.ErrTxDone) {
		return Error{
			Code:    ErrInternal,
			Kind:    KindDBTxDone,
			Message: "exec db query",
			Err:     err,
			Details: details,
		}
	}
	if nativeerrors.Is(err, sql.ErrConnDone) {
		return Error{
			Code:    ErrFatal,
			Kind:    KindDB,
			Message: "connection done",
			Err:     err,
			Details: details,
		}
	}
	// Any other internal error.
	return Error{
		Code:    ErrInternal,
		Kind:    KindDBQuery,
		Message: "exec db query",
		Err:     err,
		Details: details,
	}
}

// NewDBTxBeginError returns an errors.Error with code errors.ErrInternal and kind errors.KindDBTxBegin.
func NewDBTxBeginError(err error) error {
	return Error{
		Code:    ErrInternal,
		Kind:    KindDBTxBegin,
		Message: "begin tx",
		Err:     err,
	}
}

// NewDBTxCommitError creates a new error with KindDBTxCommit.
func NewDBTxCommitError(err error) error {
	return Error{
		Code:    ErrInternal,
		Kind:    KindDBTxCommit,
		Message: "tx commit",
		Err:     err,
	}
}
