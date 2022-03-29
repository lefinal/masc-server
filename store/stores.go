package store

import (
	"database/sql"
	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgx/v4/pgxpool"
	"github.com/lefinal/masc-server/errors"
	"go.uber.org/zap"
)

// Mall implements all database operations.
type Mall struct {
	logger *zap.Logger
	// db is the actual database to perform operations in.
	db *pgxpool.Pool
	// dialect is the SQL dialect for building queries.
	dialect goqu.DialectWrapper
}

// NewMall creates a new Mall using the given database. It uses the PostgreSQL
// dialect for queries.
func NewMall(logger *zap.Logger, db *pgxpool.Pool) *Mall {
	return &Mall{
		logger:  logger,
		db:      db,
		dialect: goqu.Dialect("postgres"),
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
			Message: "rows affected not supported by current database driver",
			Err:     err,
		}
	}
	return int(rowsAffected), nil
}

// rollbackTx rolls back the given sql.Tx. The encapsulation is needed because
// rolling back might return an error which does not need to be returned but
// definitely logged with the original reason the rollback was performed.
func (m *Mall) rollbackTx(tx *sql.Tx, reason string) {
	err := tx.Rollback()
	if err != nil {
		errors.Log(m.logger, errors.Error{
			Code:    errors.ErrInternal,
			Message: "rollback tx",
			Err:     err,
			Details: errors.Details{"rollbackReason": reason},
		})
	}
}
