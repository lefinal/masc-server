package stores

import (
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/messages"
	"github.com/doug-martin/goqu/v9"
	"github.com/gobuffalo/nulls"
	"time"
)

// LightSwitch is the store representation of lightswitch.LightSwitch.
type LightSwitch struct {
	// ID identifies the light switch.
	ID messages.LightSwitchID
	// Device is the device id of the provider which is needed in order to reuse
	// light switches.
	Device messages.DeviceID
	// DeviceName is the optional human-readable name of the associated device.
	DeviceName nulls.String
	// ProviderID is the id for the light switch the provider assigned to it.
	ProviderID messages.ProviderID
	// Name is a human-readable description of the light switch.
	Name nulls.String
	// Type is the light switch type.
	Type messages.LightSwitchType
	// LastSeen is the last time the light switch online-state changed.
	LastSeen time.Time
	// Assignments are the assigned fixtures.
	Assignments []messages.FixtureID
}

// EditLightSwitch is used in order to edit a LightSwitch.
type EditLightSwitch struct {
	// Name is the human-readable description of the light switch.
	Name nulls.String
	// Assignments are the assigned fixtures.
	Assignments []messages.FixtureID
}

// LightSwitches retrieves all available light switches.
func (m *Mall) LightSwitches() ([]LightSwitch, error) {
	// Build query for retrieving light switches.
	lightSwitchQ, _, err := m.dialect.From(goqu.T("light_switches")).
		InnerJoin(goqu.T("devices"), goqu.On(goqu.I("light_switches.device").Eq(goqu.I("devices.id")))).
		Select(goqu.I("light_switches.id"),
			goqu.I("light_switches.device"),
			goqu.I("devices.name"),
			goqu.I("light_switches.provider_id"),
			goqu.I("light_switches.name"),
			goqu.I("light_switches.type"),
			goqu.I("light_switches.last_seen")).
		Order(goqu.I("light_switches.last_seen").Desc()).ToSQL()
	if err != nil {
		return nil, errors.NewQueryToSQLError(err, nil)
	}
	// Build query for retrieving assignments.
	assignmentsQ, _, err := m.dialect.From(goqu.T("light_switch_assignments")).
		Select(goqu.C("light_switch"),
			goqu.C("fixture")).ToSQL()
	if err != nil {
		return nil, errors.NewQueryToSQLError(err, nil)
	}
	// Exec.
	tx, err := m.db.Begin()
	if err != nil {
		return nil, errors.NewDBTxBeginError(err)
	}
	// Query light switches first.
	rows, err := tx.Query(lightSwitchQ)
	if err != nil {
		rollbackTx(tx, "light switch query failed")
		return nil, errors.NewExecQueryError(err, lightSwitchQ, nil)
	}
	defer closeRows(rows)
	// We save it in an array because we want to keep the order.
	lightSwitches := make([]LightSwitch, 0)
	for rows.Next() {
		lightSwitch := LightSwitch{
			Assignments: make([]messages.FixtureID, 0),
		}
		err = rows.Scan(&lightSwitch.ID,
			&lightSwitch.Device,
			&lightSwitch.DeviceName,
			&lightSwitch.ProviderID,
			&lightSwitch.Name,
			&lightSwitch.Type,
			&lightSwitch.LastSeen)
		if err != nil {
			rollbackTx(tx, "light switch scan failed")
			return nil, errors.NewScanDBRowError(err, lightSwitchQ)
		}
		lightSwitches = append(lightSwitches, lightSwitch)
	}
	// Then query the assignments.
	rows, err = tx.Query(assignmentsQ)
	if err != nil {
		rollbackTx(tx, "light switch assignment query failed")
		return nil, errors.NewExecQueryError(err, assignmentsQ, nil)
	}
	defer closeRows(rows)
scanAssignmentRows:
	for rows.Next() {
		var assignedLightSwitch messages.LightSwitchID
		var assignedFixture messages.FixtureID
		err = rows.Scan(&assignedLightSwitch, &assignedFixture)
		if err != nil {
			rollbackTx(tx, "light switch assignment scan failed")
			return nil, errors.NewScanDBRowError(err, assignmentsQ)
		}
		// Search and update the retrieved light switch.
		for i, lightSwitch := range lightSwitches {
			if lightSwitch.ID != assignedLightSwitch {
				continue
			}
			// Update assignments.
			lightSwitch.Assignments = append(lightSwitch.Assignments, assignedFixture)
			lightSwitches[i] = lightSwitch
			continue scanAssignmentRows
		}
		// Not found.
		rollbackTx(tx, "assure light switch exists failed")
		availableOnes := make([]messages.LightSwitchID, 0, len(lightSwitches))
		for _, lightSwitch := range lightSwitches {
			availableOnes = append(availableOnes, lightSwitch.ID)
		}
		return nil, errors.NewInternalError("retrieved light switch in assignment not found in available ones",
			errors.Details{
				"assigned_light_switch": assignedLightSwitch,
				"available_ones":        availableOnes,
			})
	}
	// Commit tx.
	err = tx.Commit()
	if err != nil {
		rollbackTx(tx, "commit failed")
		return nil, errors.NewDBTxCommitError(err)
	}
	return lightSwitches, nil
}

// LightSwitchByDeviceAndProviderID retrieves stores.LightSwitch by its device
// and provider id.
func (m *Mall) LightSwitchByDeviceAndProviderID(deviceID messages.DeviceID, providerID messages.ProviderID) (LightSwitch, error) {
	// Build light switch query.
	lightSwitchQ, _, err := m.dialect.From(goqu.T("light_switches")).
		InnerJoin(goqu.T("devices"), goqu.On(goqu.I("light_switches.device").Eq(goqu.I("devices.id")))).
		Select(goqu.I("light_switches.id"),
			goqu.I("light_switches.device"),
			goqu.I("devices.name"),
			goqu.I("light_switches.provider_id"),
			goqu.I("light_switches.name"),
			goqu.I("light_switches.type"),
			goqu.I("light_switches.last_seen")).
		Where(goqu.And(goqu.I("light_switches.device").Eq(deviceID),
			goqu.I("light_switches.provider_id").Eq(providerID))).ToSQL()
	if err != nil {
		return LightSwitch{}, errors.NewQueryToSQLError(err, errors.Details{"step": "light switch query to sql"})
	}
	// Begin tx.
	tx, err := m.db.Begin()
	if err != nil {
		return LightSwitch{}, errors.NewDBTxBeginError(err)
	}
	// Query light switch.
	lightSwitch := LightSwitch{
		Assignments: make([]messages.FixtureID, 0),
	}
	err = tx.QueryRow(lightSwitchQ).Scan(&lightSwitch.ID,
		&lightSwitch.Device,
		&lightSwitch.DeviceName,
		&lightSwitch.ProviderID,
		&lightSwitch.Name,
		&lightSwitch.Type,
		&lightSwitch.LastSeen)
	if err != nil {
		rollbackTx(tx, "query and scan light switch failed")
		return LightSwitch{}, errors.NewScanSingleDBRowError(err, "light switch not found", lightSwitchQ)
	}
	// Build query for assignments.
	assignmentsQ, _, err := m.dialect.From(goqu.C("light_switch_assignments")).
		Select(goqu.C("fixture")).
		Where(goqu.C("light_switch").Eq(lightSwitch.ID)).ToSQL()
	if err != nil {
		rollbackTx(tx, "assignments query to sql")
		return LightSwitch{}, errors.NewQueryToSQLError(err, errors.Details{"step": "assignments query to sql"})
	}
	// Query assignments.
	rows, err := tx.Query(assignmentsQ)
	if err != nil {
		rollbackTx(tx, "assignments query failed")
		return LightSwitch{}, errors.NewExecQueryError(err, assignmentsQ, nil)
	}
	defer closeRows(rows)
	// Scan.
	for rows.Next() {
		var assignment messages.FixtureID
		err = rows.Scan(&assignment)
		if err != nil {
			rollbackTx(tx, "scan assignment failed")
			return LightSwitch{}, errors.NewScanDBRowError(err, assignmentsQ)
		}
		lightSwitch.Assignments = append(lightSwitch.Assignments, assignment)
	}
	// Commit.
	err = tx.Commit()
	if err != nil {
		return LightSwitch{}, errors.NewDBTxCommitError(err)
	}
	return lightSwitch, nil
}

// UpdateLightSwitchByID updates the LightSwitch with the given id.
func (m *Mall) UpdateLightSwitchByID(id messages.LightSwitchID, payload EditLightSwitch) error {
	// Build queries.
	updateNameQ, _, err := m.dialect.Update(goqu.T("light_switches")).Set(goqu.Record{
		"name": payload.Name,
	}).Where(goqu.C("id").Eq(id)).ToSQL()
	if err != nil {
		return errors.NewQueryToSQLError(err, errors.Details{"step": "update name"})
	}
	clearAssignmentsQ, _, err := m.dialect.Delete(goqu.T("light_switch_assignments")).
		Where(goqu.C("light_switch").Eq(id)).ToSQL()
	if err != nil {
		return errors.NewQueryToSQLError(err, errors.Details{"step": "clear assignments"})
	}
	assignments := make([]interface{}, 0, len(payload.Assignments))
	for _, assignment := range payload.Assignments {
		assignments = append(assignments, goqu.Record{
			"light_switch": id,
			"fixture":      assignment,
		})
	}
	assignQ, _, err := m.dialect.Insert(goqu.T("light_switch_assignments")).
		Rows(assignments...).ToSQL()
	if err != nil {
		return errors.NewQueryToSQLError(err, errors.Details{"step": "assign"})
	}
	// Begin tx.
	tx, err := m.db.Begin()
	if err != nil {
		return errors.NewDBTxBeginError(err)
	}
	// Update name.
	result, err := tx.Exec(updateNameQ)
	if err != nil {
		rollbackTx(tx, "update name failed")
		return errors.NewExecQueryError(err, updateNameQ, nil)
	}
	// Assure found.
	err = assureOneRowAffectedForNotFound(result, "light switch not found", id, updateNameQ)
	if err != nil {
		rollbackTx(tx, "assure light switch found failed")
		return errors.Wrap(err, "assure found", nil)
	}
	// Clear assignments.
	_, err = tx.Exec(clearAssignmentsQ)
	if err != nil {
		rollbackTx(tx, "clear assignments failed")
		return errors.NewExecQueryError(err, clearAssignmentsQ, nil)
	}
	if len(assignments) > 0 {
		// Perform new assignments.
		_, err = tx.Exec(assignQ)
		if err != nil {
			rollbackTx(tx, "assign failed")
			return errors.NewExecQueryError(err, assignQ, nil)
		}
	}
	// Commit.
	err = tx.Commit()
	if err != nil {
		rollbackTx(tx, "commit failed")
		return errors.NewDBTxCommitError(err)
	}
	return nil
}

// DeleteLightSwitchByID deletes the LightSwitch with the given id.
func (m *Mall) DeleteLightSwitchByID(id messages.LightSwitchID) error {
	// Build query.
	q, _, err := m.dialect.Delete(goqu.T("light_switches")).
		Where(goqu.C("id").Eq(id)).ToSQL()
	if err != nil {
		return errors.NewQueryToSQLError(err, nil)
	}
	// Exec.
	result, err := m.db.Exec(q)
	if err != nil {
		return errors.NewExecQueryError(err, q, nil)
	}
	err = assureOneRowAffectedForNotFound(result, "light switch not found", id, q)
	if err != nil {
		return errors.Wrap(err, "assure found", nil)
	}
	return nil
}

// CreateLightSwitch creates the given LightSwitch.
func (m *Mall) CreateLightSwitch(lightSwitch LightSwitch) (LightSwitch, error) {
	lightSwitchQ, _, err := m.dialect.Insert(goqu.T("light_switches")).
		Rows(goqu.Record{
			"device":      lightSwitch.Device,
			"provider_id": lightSwitch.ProviderID,
			"name":        lightSwitch.Name,
			"type":        lightSwitch.Type,
			"last_seen":   lightSwitch.LastSeen,
		}).
		Returning(goqu.C("id")).ToSQL()
	if err != nil {
		return LightSwitch{}, errors.NewQueryToSQLError(err, errors.Details{"step": "light switch query"})
	}
	// Begin tx.
	tx, err := m.db.Begin()
	if err != nil {
		return LightSwitch{}, errors.NewDBTxBeginError(err)
	}
	// Create light switch.
	err = tx.QueryRow(lightSwitchQ).Scan(&lightSwitch.ID)
	if err != nil {
		rollbackTx(tx, "create light switch failed")
		return LightSwitch{}, errors.NewScanSingleDBRowError(err, "what", lightSwitchQ)
	}
	// Perform assignments.
	assignments := make([]interface{}, 0, len(lightSwitch.Assignments))
	for _, assignment := range lightSwitch.Assignments {
		assignments = append(assignments, goqu.Record{
			"light_switch": lightSwitch.ID,
			"fixture":      assignment,
		})
	}
	if len(assignments) > 0 {
		assignQ, _, err := m.dialect.Insert(goqu.T("light_switch_assignments")).
			Rows(assignments...).ToSQL()
		if err != nil {
			rollbackTx(tx, "assignments query to sql failed")
			return LightSwitch{}, errors.NewQueryToSQLError(err, errors.Details{"step": "assignments query to sql"})
		}
		_, err = tx.Exec(assignQ)
		if err != nil {
			rollbackTx(tx, "assignments failed")
			return LightSwitch{}, errors.NewExecQueryError(err, assignQ, nil)
		}
	}
	// Commit tx.
	err = tx.Commit()
	if err != nil {
		rollbackTx(tx, "commit failed")
		return LightSwitch{}, errors.NewDBTxCommitError(err)
	}
	return lightSwitch, nil
}

// RefreshLastSeenForLightSwitchByID refreshes the last seen timestamp for the
// LightSwitch with the given id.
func (m *Mall) RefreshLastSeenForLightSwitchByID(id messages.LightSwitchID) error {
	q, _, err := m.dialect.Update(goqu.T("light_switches")).Set(goqu.Record{
		"last_seen": time.Now(),
	}).Where(goqu.C("id").Eq(id)).ToSQL()
	if err != nil {
		return errors.NewQueryToSQLError(err, nil)
	}
	result, err := m.db.Exec(q)
	if err != nil {
		return errors.NewExecQueryError(err, q, nil)
	}
	err = assureOneRowAffectedForNotFound(result, "light switch not found", id, q)
	if err != nil {
		return errors.Wrap(err, "assure found", nil)
	}
	return nil
}
