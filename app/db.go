package app

import (
	"database/sql"
	"embed"
	nativeerrors "errors"
	"fmt"
	"github.com/LeFinal/masc-server/errors"
	"github.com/doug-martin/goqu/v9"
	"github.com/jackc/pgconn"
	_ "github.com/jackc/pgx/v4/stdlib"
	"go.uber.org/zap"
	"io"
	"sort"
	"strconv"
	"strings"
)

// TODO: Move to migrations
var dbMigrationLogger *zap.Logger

// defaultMaxDBConnections is the maximum number of database connections that is used when no other one is provided
// in the Config.
const defaultMaxDBConnections = 16

// dbVersion is used for determining the current database version. This is saved in a special table when properly set
// up. If the version does not exist, one can know that the database needs to be initialized. If it is and the latest
// version is greater, migrations can be performed.
type dbVersion int32

// dbVersionZero is used when no database version could be found, and therefore we conclude that it has not been
// initialized yet.
const dbVersionZero dbVersion = 0

// dbMigration is used for performing and checking database migrations. They lie in dbMigrations which is an ordered
// list of versions with their migrations.
type dbMigration struct {
	version dbVersion
	up      string
}

//go:embed db_migrations
var migrations embed.FS

func getOrderedMigrations() ([]dbMigration, error) {
	const numberPrefix = 5
	// Read embedded migrations.
	files, err := migrations.ReadDir("db_migrations")
	if err != nil {
		return nil, errors.NewInternalErrorFromErr(err, "read embedded migrations failed", nil)
	}
	// Read file names.
	fileNames := make([]string, 0, len(files))
	for _, file := range files {
		fileNames = append(fileNames, file.Name())
	}
	// Sort.
	sort.Strings(fileNames)
	dbMigrations := make([]dbMigration, 0, len(files))
	for _, name := range fileNames {
		// Read file.
		up, err := migrations.Open(fmt.Sprintf("db_migrations/%s", name))
		if err != nil {
			return nil, errors.NewInternalErrorFromErr(err, "open db migration failed", nil)
		}
		upContent, err := io.ReadAll(up)
		_ = up.Close()
		if err != nil {
			return nil, errors.NewInternalErrorFromErr(err, "read db migration failed", errors.Details{
				"migration": name,
			})
		}
		// Extract migration version number.
		if len(strings.Split(name, ".sql")[0]) < numberPrefix {
			return nil, errors.NewInternalError("db migration file name too short", errors.Details{
				"expect": numberPrefix,
				"name":   name,
			})
		}
		version, err := strconv.Atoi(name[:numberPrefix])
		if err != nil {
			return nil, errors.NewInternalErrorFromErr(err, "db migration file name number prefix not numeric",
				errors.Details{"name": name})
		}
		dbMigrations = append(dbMigrations, dbMigration{
			version: dbVersion(version),
			up:      string(upContent),
		})
	}
	return dbMigrations, nil
}

// connectDB connects to the database with the given connection string and returns the connection pool.
func connectDB(connectionStr string, maxDBConnections int) (*sql.DB, error) {
	dbPool, err := sql.Open("pgx", connectionStr)
	if err != nil {
		return nil, errors.Error{
			Code:    errors.ErrFatal,
			Err:     err,
			Message: "connection to database failed",
			Details: errors.Details{"connectionStr": connectionStr},
		}
	}
	dbPool.SetMaxOpenConns(maxDBConnections)
	// Perform test query.
	err = testDBConnection(dbPool)
	if err != nil {
		return nil, errors.Wrap(err, "test db connection", nil)
	}
	// Perform db migrations.
	err = performDBMigrations(dbPool)
	if err != nil {
		return nil, errors.Wrap(err, "perform db migrations", nil)
	}
	return dbPool, nil
}

// testDBConnection tests the database connection by simply querying 1.
func testDBConnection(db *sql.DB) error {
	// Build test query.
	q, _, err := goqu.Select(goqu.V(1)).ToSQL()
	if err != nil {
		return errors.NewQueryToSQLError(err, errors.Details{})
	}
	// Query database.
	result := db.QueryRow(q)
	var got int
	err = result.Scan(&got)
	if err != nil {
		return errors.NewScanSingleDBRowError(err, "test query failed", q)
	}
	// Assure that we got 1.
	if got != 1 {
		return errors.Error{
			Code:    errors.ErrFatal,
			Message: fmt.Sprintf("test db connection: expected 1 as result but got %d", got),
			Details: errors.Details{
				"got": got,
			},
		}
	}
	return nil
}

// performDBMigrations performs all needed database migrations according to the (un)set database version.
func performDBMigrations(db *sql.DB) error {
	currentVersion, err := retrieveCurrentDBVersion(db)
	if err != nil {
		return errors.Wrap(err, "retrieve current db version", nil)
	}
	dbMigrationLogger.Info("extracted current database version",
		zap.Any("db_version", currentVersion))
	migrationsToDo, err := getDBMigrationsToDo(currentVersion)
	if err != nil {
		return errors.Wrap(err, "get db migrations to do", nil)
	}
	// Check if migrations need to be performed.
	if len(migrationsToDo) == 0 {
		return nil
	}
	// Begin tx for avoiding database destruction if something fails.
	tx, err := db.Begin()
	if err != nil {
		return errors.NewDBTxBeginError(err)
	}
	// Perform migrations.
	var newVersion dbVersion
	for i, migration := range migrationsToDo {
		dbMigrationLogger.Info(fmt.Sprintf("performing database migration %d/%d...", i+1, len(migrationsToDo)))
		// Perform migration according to the version.
		_, err = tx.Exec(migration.up)
		if err != nil {
			rollbackTx(tx, "database migration failed")
			return errors.NewExecQueryError(err, migration.up, errors.Details{"target_version": migration.version})
		}
		newVersion = migration.version
		dbMigrationLogger.Debug("finished database migration step", zap.Any("new_version", newVersion))
	}
	err = tx.Commit()
	if err != nil {
		return errors.NewDBTxCommitError(err)
	}
	// New tx for database version update.
	tx, err = db.Begin()
	if err != nil {
		return errors.NewDBTxBeginError(err)
	}
	// Update database version.
	dbMigrationLogger.Debug("updating database version...")
	var updateDBVersionQuery string
	if currentVersion == dbVersionZero {
		updateDBVersionQuery, _, err = goqu.Dialect("postgres").Insert(goqu.T("masc")).Rows(goqu.Record{
			"key":   "db-version",
			"value": newVersion,
		}).ToSQL()
	} else {
		updateDBVersionQuery, _, err = goqu.Dialect("postgres").Update(goqu.T("masc")).
			Set(goqu.Record{"value": newVersion}).
			Where(goqu.C("key").Eq("db-version")).ToSQL()
	}
	if err != nil {
		rollbackTx(tx, "update database version query to sql failed")
		return nil
	}
	_, err = tx.Exec(updateDBVersionQuery)
	if err != nil {
		rollbackTx(tx, "update database version failed")
		return nil
	}
	// Commit tx.
	err = tx.Commit()
	if err != nil {
		return errors.NewDBTxCommitError(err)
	}
	// All done.
	dbMigrationLogger.Debug("finished database version update")
	return nil
}

// getDBMigrationsToDo retrieves all database migrations that need to be performed. If the version is dbVersionZero, it
// will return all migrations. If the version is unknown, an error will be returned.
func getDBMigrationsToDo(currentVersion dbVersion) ([]dbMigration, error) {
	dbMigrations, err := getOrderedMigrations()
	if err != nil {
		return nil, errors.Wrap(err, "get ordered migrations", nil)
	}
	// Check if empty version.
	if currentVersion == dbVersionZero {
		return dbMigrations, nil
	}
	found := false
	migrationsToDo := make([]dbMigration, 0)
	for _, migration := range dbMigrations {
		if migration.version == currentVersion {
			// Match found.
			if found {
				// This should not happen and is an internal error as the versions are not properly set up. What did you
				// do?
				return nil, errors.Error{
					Code:    errors.ErrInternal,
					Message: fmt.Sprintf("duplicate database version %v in available migrations", currentVersion),
					Details: errors.Details{"version": currentVersion},
				}
			}
			// Set found flag.
			found = true
			// Continue with next one as we already performed everything for this database version.
			continue
		}
		if found {
			// Append migration to todos.
			migrationsToDo = append(migrationsToDo, migration)
		}
	}
	// Check if found.
	if !found {
		return nil, errors.NewResourceNotFoundError(fmt.Sprintf("no database version found matching %v", currentVersion),
			errors.Details{"version": currentVersion})
	}
	// Done.
	return migrationsToDo, nil
}

// retrieveCurrentDBVersion retrieves the current dbVersion from the given database. If no version could be found,
// dbVersionZero will be returned.
func retrieveCurrentDBVersion(db *sql.DB) (dbVersion, error) {
	versionStr, err := retrieveKeyValFromDB(db, "db-version")
	if err != nil {
		if e, ok := errors.Cast(err); ok {
			if e.Code == errors.ErrNotFound {
				return dbVersionZero, nil
			}
			// Check if error is postgres error. Then we can check if the error occurred because of the table not
			// existing.
			var pgErr *pgconn.PgError
			if nativeerrors.As(e.Err, &pgErr) && pgErr.Code == "42P01" {
				return dbVersionZero, nil
			}
		}
		return 0, errors.Wrap(err, "retrieve key val from db", nil)
	}
	version, err := strconv.Atoi(versionStr)
	if err != nil {
		return 0, errors.NewInternalErrorFromErr(err, "retrieved current db version not numeric", errors.Details{
			"was": versionStr,
		})
	}
	return dbVersion(version), nil
}

// retrieveKeyValFromDB retrieves the value for the given key from the given database.
func retrieveKeyValFromDB(db *sql.DB, key string) (string, error) {
	// Build query.
	q, _, err := goqu.Dialect("postgres").From(goqu.T("masc")).
		Select(goqu.C("value")).
		Where(goqu.C("key").Eq(key)).ToSQL()
	if err != nil {
		return "", errors.NewQueryToSQLError(err, errors.Details{"key": key})
	}
	// Exec query and scan value.
	var value string
	err = db.QueryRow(q).Scan(&value)
	if err != nil {
		// Check if error is because relation does not exist as then it's a not-found error.
		var pgErr *pgconn.PgError
		if nativeerrors.As(err, &pgErr) && pgErr.Code == "42P01" {
			return "", errors.NewResourceNotFoundError("key-value relation not found", errors.Details{})
		}
		return "", errors.NewScanSingleDBRowError(err, fmt.Sprintf("no entry with key %s found", key), q)
	}
	// Done.
	return value, nil
}

// rollbackTx rolls back the given sql.Tx. The encapsulation is needed because rolling back might return an error which
// does not need to be returned but definitely logged with the original reason the rollback was performed.
func rollbackTx(tx *sql.Tx, reason string) {
	err := tx.Rollback()
	if err != nil {
		errors.Log(dbMigrationLogger, errors.Error{
			Code:    errors.ErrInternal,
			Message: "rollback tx",
			Err:     err,
			Details: errors.Details{"rollbackReason": reason},
		})
	}
}
