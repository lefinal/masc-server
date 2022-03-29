package store

import (
	"context"
	"github.com/doug-martin/goqu/v9"
	"github.com/gobuffalo/nulls"
	"github.com/lefinal/masc-server/errors"
	"time"
)

// Device is a container for information regarding a known device.
type Device struct {
	// ID is the self-assigned ID of the device.
	ID string
	// Type is the self-assigned device type.
	Type string
	// LastSeen is the last time the Device was seen.
	LastSeen time.Time
	// Name is an optional human-readable name.
	Name nulls.String
	// Config is an optional configuration for the device.
	Config nulls.ByteSlice
}

// DeviceByID retrieves a Device by its id.
func (m *Mall) DeviceByID(ctx context.Context, deviceID string) (Device, error) {
	// Build query.
	q, _, err := m.dialect.From("devices").
		Select(goqu.C("id"),
			goqu.C("type"),
			goqu.C("last_seen"),
			goqu.C("name"),
			goqu.C("config")).
		Where(goqu.C("id").Eq(deviceID)).ToSQL()
	if err != nil {
		return Device{}, errors.NewInternalErrorFromErr(err, "query to sql", nil)
	}
	// Query.
	rows, err := m.db.Query(ctx, q)
	if err != nil {
		return Device{}, errors.NewExecQueryError(err, "query db", q)
	}
	defer rows.Close()
	// Scan.
	if !rows.Next() {
		return Device{}, errors.NewResourceNotFoundError("device not found", nil)
	}
	var device Device
	err = rows.Scan(&device.ID,
		&device.Type,
		&device.LastSeen,
		&device.Name,
		&device.Config)
	if err != nil {
		return Device{}, errors.NewScanDBRowError(err, "scan row", q)
	}
	return device, nil
}

// RegisterDevice retrieves the Device with the given id. If none was found, a
// new one is created with default values and returned. The second return value
// describes whether the device was created. In any case, the last seen
// timestamp is set to the current time. If the device was found and the type
// changed, the config will get cleared.
func (m *Mall) RegisterDevice(ctx context.Context, deviceID string, deviceType string) (Device, bool, error) {
	// Begin tx.
	txCtx, cancelTx := context.WithCancel(ctx)
	defer cancelTx()
	tx, err := m.db.Begin(txCtx)
	if err != nil {
		return Device{}, false, errors.NewDBTxBeginError(err)
	}
	// Build lookup and type check query.
	q, _, err := m.dialect.From("devices").
		Select(goqu.C("type")).
		Where(goqu.C("id").Eq(deviceID)).ToSQL()
	if err != nil {
		return Device{}, false, errors.NewInternalErrorFromErr(err, "lookup query to sql", nil)
	}
	// Query.
	rows, err := tx.Query(ctx, q)
	if err != nil {
		return Device{}, false, errors.NewExecQueryError(err, "exec lookup query", q)
	}
	found := rows.Next()
	rows.Close()
	if found {
		// Scan type.
		var oldType string
		err = rows.Scan(&oldType)
		if err != nil {
			return Device{}, false, errors.NewScanDBRowError(err, "scan device type", q)
		}
		rows.Close()
		// Update last seen timestamp.
		newRecord := goqu.Record{
			"last_seen": time.Now(),
		}
		// Check if type changed and if so, clear config.
		if deviceType != oldType {
			newRecord["type"] = deviceType
			newRecord["config"] = nulls.ByteSlice{}
		}
		q, _, err = m.dialect.Update("devices").Set(newRecord).Where(goqu.C("id").Eq(deviceID)).ToSQL()
		if err != nil {
			return Device{}, false, errors.NewInternalErrorFromErr(err, "update last seen query to sql", nil)
		}
		_, err = tx.Exec(ctx, q)
		if err != nil {
			return Device{}, false, errors.NewExecQueryError(err, "exec update last seen query", q)
		}
	} else {
		// Not found -> create.
		q, _, err = m.dialect.Insert("devices").Rows(goqu.Record{
			"id":        deviceID,
			"type":      deviceType,
			"last_seen": time.Now(),
		}).ToSQL()
		if err != nil {
			return Device{}, false, errors.NewInternalErrorFromErr(err, "create query to sql", nil)
		}
		// Query.
		result, err := tx.Exec(ctx, q)
		if err != nil {
			return Device{}, false, errors.NewExecQueryError(err, "exec create query", q)
		}
		if result.RowsAffected() != 1 {
			return Device{}, false, errors.NewInternalError("new device not created", errors.Details{"query": q})
		}
	}
	// Build final retrieve query.
	q, _, err = m.dialect.From("devices").
		Select(goqu.C("id"),
			goqu.C("type"),
			goqu.C("last_seen"),
			goqu.C("name"),
			goqu.C("config")).
		Where(goqu.C("id").Eq(deviceID)).ToSQL()
	if err != nil {
		return Device{}, false, errors.NewInternalErrorFromErr(err, "retrieve query to sql", nil)
	}
	// Query.
	rows, err = tx.Query(ctx, q)
	if err != nil {
		return Device{}, false, errors.NewExecQueryError(err, "query final device", q)
	}
	defer rows.Close()
	// Scan.
	if !rows.Next() {
		return Device{}, false, errors.NewInternalError("missing device although should be created", nil)
	}
	var device Device
	err = rows.Scan(&device.ID,
		&device.Type,
		&device.LastSeen,
		&device.Name,
		&device.Config)
	if err != nil {
		return Device{}, false, errors.NewScanDBRowError(err, "scan final row", q)
	}
	rows.Close()
	// Commit.
	err = tx.Commit(ctx)
	if err != nil {
		return Device{}, false, errors.NewDBTxCommitError(err)
	}
	return device, !found, nil
}

// UpdateDeviceLastSeen updates the last seen timestamp for the device with the
// given id.
func (m *Mall) UpdateDeviceLastSeen(ctx context.Context, deviceID string) error {
	// Build query.
	q, _, err := m.dialect.Update("devices").Set(goqu.Record{
		"last_seen": time.Now(),
	}).Where(goqu.C("id").Eq(deviceID)).ToSQL()
	if err != nil {
		return errors.NewInternalErrorFromErr(err, "query to sql", nil)
	}
	// Exec.
	result, err := m.db.Exec(ctx, q)
	if err != nil {
		return errors.NewExecQueryError(err, "exec query", q)
	}
	// Assure found.
	if result.RowsAffected() != 1 {
		return errors.NewResourceNotFoundError("device not found", nil)
	}
	return nil
}
