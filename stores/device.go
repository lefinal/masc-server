package stores

import (
	"fmt"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/messages"
	"github.com/doug-martin/goqu/v9"
	"github.com/gobuffalo/nulls"
	"github.com/google/uuid"
	"time"
)

// Device holds all information regarding a gatekeeping.Device.
type Device struct {
	// ID is the assigned device id.
	ID messages.DeviceID
	// Name is a human-readable name of the device.
	Name nulls.String
	// SelfDescription is how the device describe itself on first connect.
	SelfDescription string
	// LastSeen is the last time the device updated its online-state.
	LastSeen time.Time
}

func (m *Mall) GetDevices() ([]Device, error) {
	q, _, err := m.dialect.From(goqu.T("devices")).
		Select(goqu.C("id"), goqu.C("name"), goqu.C("self_description"), goqu.C("last_seen")).ToSQL()
	if err != nil {
		return nil, errors.NewQueryToSQLError(err, nil)
	}
	rows, err := m.db.Query(q)
	if err != nil {
		return nil, errors.NewExecQueryError(err, q, nil)
	}
	defer closeRows(rows)
	devices := make([]Device, 0)
	for rows.Next() {
		var device Device
		err = rows.Scan(&device.ID, &device.Name, &device.SelfDescription, &device.LastSeen)
		if err != nil {
			return nil, errors.NewScanDBRowError(err, nil)
		}
		devices = append(devices, device)
	}
	return devices, nil
}

func (m *Mall) CreateNewDevice(selfDescription string) (Device, error) {
	createdDevice := Device{
		Name:            nulls.String{},
		SelfDescription: selfDescription,
		LastSeen:        time.Now(),
	}
	q, _, err := m.dialect.Insert(goqu.T("devices")).Rows(goqu.Record{
		"id":               uuid.New().String(),
		"self_description": createdDevice.SelfDescription,
		"last_seen":        createdDevice.LastSeen,
	}).Returning(goqu.C("id")).ToSQL()
	if err != nil {
		return Device{}, errors.NewQueryToSQLError(err, errors.Details{"selfDescription": selfDescription})
	}
	row := m.db.QueryRow(q)
	err = row.Scan(&createdDevice.ID)
	if err != nil {
		return Device{}, errors.NewScanSingleDBRowError(fmt.Sprintf("what"), err,
			errors.Details{"selfDescription": selfDescription})
	}
	return createdDevice, nil
}

func (m *Mall) RefreshLastSeenForDevice(deviceID messages.DeviceID) error {
	q, _, err := m.dialect.Update(goqu.T("devices")).
		Set(goqu.Record{
			"last_seen": time.Now(),
		}).
		Where(goqu.C("id").Eq(deviceID)).ToSQL()
	if err != nil {
		return errors.NewQueryToSQLError(err, errors.Details{"device": deviceID})
	}
	result, err := m.db.Exec(q)
	if err != nil {
		return errors.NewExecQueryError(err, q, errors.Details{"device": deviceID})
	}
	err = assureOneRowAffectedForNotFound(result, fmt.Sprintf("device %v not found", deviceID), "devices", deviceID, q)
	if err != nil {
		return errors.Wrap(err, "assure found")
	}
	return nil
}

func (m *Mall) SetDeviceName(deviceID messages.DeviceID, name string) error {
	errDetails := errors.Details{
		"device": deviceID,
		"name":   name,
	}
	q, _, err := m.dialect.Update(goqu.T("devices")).
		Set(goqu.Record{
			"name": name,
		}).
		Where(goqu.C("id").Eq(deviceID)).ToSQL()
	if err != nil {
		return errors.NewQueryToSQLError(err, errDetails)
	}
	result, err := m.db.Exec(q)
	if err != nil {
		return errors.NewExecQueryError(err, q, errDetails)
	}
	err = assureOneRowAffectedForNotFound(result, fmt.Sprintf("device %v not found", deviceID), "devices", deviceID, q)
	if err != nil {
		return errors.Wrap(err, "assure found")
	}
	return nil
}
