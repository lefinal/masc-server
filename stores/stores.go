package stores

import (
	"database/sql"
	"fmt"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/logging"
	"github.com/doug-martin/goqu/v9"
)

// Mall implements all database operations.
type Mall struct {
	// db is the actual database to perform operations in.
	db *sql.DB
	// dialect is the SQL dialect for building queries.
	dialect goqu.DialectWrapper
}

// NewMall creates a new Mall using the given database. It uses the PostgreSQL
// dialect for queries.
func NewMall(db *sql.DB) *Mall {
	return &Mall{
		db:      db,
		dialect: goqu.Dialect("postgres"),
	}
}

func closeRows(rows *sql.Rows) {
	err := rows.Close()
	if err != nil {
		errors.Log(logging.DBLogger, errors.Error{
			Code:    errors.ErrInternal,
			Kind:    errors.KindDB,
			Err:     err,
			Message: "close rows",
		})
	}
}

// extractAffectedRows extracts the affected rows from the given sql.Result.
// Encapsulation is done, because this operation might not be supported by the
// current database driver and therefore an errors.ErrFatal error with kind
// errors.KindRowsAffectedNotSupported is returned.
func extractAffectedRows(result sql.Result) (int, error) {
	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return -1, errors.Error{
			Code:    errors.ErrFatal,
			Kind:    errors.KindShouldNotHappen,
			Message: "rows affected not supported by current database driver",
			Err:     err,
		}
	}
	return int(rowsAffected), nil
}

// assureNRowsAffected assures that the given amount of rows are affected in the
// sql.Result. If this operation is not supported, an errors.ErrFatal error with
// kind errors.KindRowsAffectedNotSupported is returned. Otherwise, if the
// affected row count does not match the expected one, an errors.ErrInternal
// with kind errors.KindWrongRowsAffected is returned.
func assureNRowsAffected(result sql.Result, n int) error {
	rowsAffected, err := extractAffectedRows(result)
	if err != nil {
		return errors.Wrap(err, "extract affected rows", nil)
	}
	if rowsAffected != n {
		return errors.Error{
			Code:    errors.ErrInternal,
			Kind:    errors.KindDB,
			Message: fmt.Sprintf("expected %d affected rows but only got %d", n, rowsAffected),
			Details: errors.Details{
				"expectedAffectedRows": n,
				"actualAffectedRows":   rowsAffected,
			},
		}
	}
	return nil
}

// assureOneRowAffectedForNotFound makes sure that exact one row for the given
// sql.Result is affected. If getting the affected rows is not possible, an
// errors.ErrFatal error is returned. If the affected rows do not equal 1, an
// errors.ErrNotFound error is returned with the given details.
func assureOneRowAffectedForNotFound(result sql.Result, notFoundMessage, table string, id interface{}, q string) error {
	rowsAffected, err := extractAffectedRows(result)
	if err != nil {
		return errors.Wrap(err, "extract affected rows", nil)
	}
	if rowsAffected != 1 {
		return errors.NewResourceNotFoundError(notFoundMessage, errors.Details{
			"table":        table,
			"id":           id,
			"query":        q,
			"rowsAffected": rowsAffected,
		})
	}
	return nil
}

// extractIDFromResult extracts the id from the given sql.Rows. There should be
// exactly one row with one column which value will be returned. Warning: The
// error being returned is NO application error! Use errors.NewExecQueryError.
func extractIDFromResult(result *sql.Row) (int, error) {
	id := -1
	err := result.Scan(&id)
	// TODO: What happens for insert failure like violated constraints? Where do we catch that error for ErrBadRequest?
	if err != nil {
		return -1, err
	}
	return id, nil
}

// rollbackTx rolls back the given sql.Tx. The encapsulation is needed because
// rolling back might return an error which does not need to be returned but
// definitely logged with the original reason the rollback was performed.
func rollbackTx(tx *sql.Tx, reason string) {
	err := tx.Rollback()
	if err != nil {
		errors.Log(logging.DBLogger, errors.Error{
			Code:    errors.ErrInternal,
			Kind:    errors.KindDBRollback,
			Message: "rollback tx",
			Err:     err,
			Details: errors.Details{"rollbackReason": reason},
		})
	}
}
