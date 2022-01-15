package stores

import (
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/messages"
	"github.com/doug-martin/goqu/v9"
)

func (m *Mall) GetDeviceIDByMQTTID(mqttID string) (messages.DeviceID, error) {
	q, _, err := goqu.From(goqu.T("mqtt_devices")).
		Select(goqu.C("device_id")).
		Where(goqu.C("mqtt_id").Eq(mqttID)).ToSQL()
	if err != nil {
		return "", errors.NewQueryToSQLError(err, nil)
	}
	var deviceID messages.DeviceID
	err = m.db.QueryRow(q).Scan(&deviceID)
	if err != nil {
		return "", errors.NewScanSingleDBRowError("device not found", err, nil)
	}
	return deviceID, nil
}

func (m *Mall) RememberDevice(mqttID string, deviceID messages.DeviceID) error {
	q, _, err := goqu.Insert(goqu.T("mqtt_devices")).Rows(goqu.Record{
		"mqtt_id":   mqttID,
		"device_id": deviceID,
	}).ToSQL()
	if err != nil {
		return errors.NewQueryToSQLError(err, nil)
	}
	result, err := m.db.Exec(q)
	if err != nil {
		return errors.NewExecQueryError(err, q, nil)
	}
	err = assureNRowsAffected(result, 1)
	if err != nil {
		return errors.Wrap(err, "assure one row affected", nil)
	}
	return nil
}
