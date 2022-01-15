package stores

import (
	"fmt"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/messages"
	"github.com/doug-martin/goqu/v9"
	"github.com/gobuffalo/nulls"
	"time"
)

// Fixture is the store representation of lighting.Fixture.
type Fixture struct {
	// ID identifies the fixture.
	ID messages.FixtureID
	// Device is the device id of the provider which is needed in order to reuse
	// fixtures.
	Device messages.DeviceID
	// ProviderID is the id for the fixture the provider assigned to it.
	ProviderID messages.FixtureProviderFixtureID
	// Name is a human-readable description of the fixture.
	Name nulls.String
	// Type is the fixture type.
	Type messages.FixtureType
	// LastSeen is the last time the fixture online state changed.
	LastSeen time.Time
}

func (m *Mall) GetFixtures() ([]Fixture, error) {
	q, _, err := m.dialect.From(goqu.T("fixtures")).
		Select(goqu.C("id"), goqu.C("device"), goqu.C("provider_id"), goqu.C("name"),
			goqu.C("type"), goqu.C("last_seen")).
		Order(goqu.C("last_seen").Desc()).ToSQL()
	if err != nil {
		return nil, errors.NewQueryToSQLError(err, nil)
	}
	rows, err := m.db.Query(q)
	if err != nil {
		return nil, errors.NewExecQueryError(err, q, nil)
	}
	defer closeRows(rows)
	fixtures := make([]Fixture, 0)
	for rows.Next() {
		var fixture Fixture
		err = rows.Scan(&fixture.ID, &fixture.Device, &fixture.ProviderID, &fixture.Name, &fixture.Type, &fixture.LastSeen)
		if err != nil {
			return nil, errors.NewScanDBRowError(err, nil)
		}
		fixtures = append(fixtures, fixture)
	}
	return fixtures, nil
}

func (m *Mall) CreateFixture(fixture Fixture) (Fixture, error) {
	q, _, err := m.dialect.Insert(goqu.T("fixtures")).
		Rows(goqu.Record{
			"device":      fixture.Device,
			"provider_id": fixture.ProviderID,
			"name":        fixture.Name,
			"type":        fixture.Type,
			"last_seen":   fixture.LastSeen,
		}).
		Returning(goqu.C("id")).ToSQL()
	if err != nil {
		return Fixture{}, errors.NewQueryToSQLError(err, errors.Details{"fixture": fixture})
	}
	row := m.db.QueryRow(q)
	err = row.Scan(&fixture.ID)
	if err != nil {
		return Fixture{}, errors.NewScanSingleDBRowError("sad life", err, errors.Details{"fixture": fixture})
	}
	return fixture, nil
}

func (m *Mall) DeleteFixture(fixtureID messages.FixtureID) error {
	q, _, err := m.dialect.Delete(goqu.T("fixtures")).
		Where(goqu.C("id").Eq(fixtureID)).ToSQL()
	if err != nil {
		return errors.NewQueryToSQLError(err, errors.Details{"fixture": fixtureID})
	}
	result, err := m.db.Exec(q)
	if err != nil {
		return errors.NewExecQueryError(err, q, errors.Details{"fixture": fixtureID})
	}
	err = assureOneRowAffectedForNotFound(result, fmt.Sprintf("fixture %v not found", fixtureID), "fixtures", fixtureID, q)
	if err != nil {
		return errors.Wrap(err, "assure one affected", nil)
	}
	return nil
}

func (m *Mall) SetFixtureName(fixtureID messages.FixtureID, name nulls.String) error {
	q, _, err := m.dialect.Update(goqu.T("fixtures")).Set(goqu.Record{
		"name": name,
	}).Where(goqu.C("id").Eq(fixtureID)).ToSQL()
	if err != nil {
		return errors.NewQueryToSQLError(err, errors.Details{"fixture": fixtureID, "name": name})
	}
	result, err := m.db.Exec(q)
	if err != nil {
		return errors.NewExecQueryError(err, q, errors.Details{"fixture": fixtureID, "name": name})
	}
	err = assureOneRowAffectedForNotFound(result, fmt.Sprintf("fixture %v not found", fixtureID), "fixtures", fixtureID, q)
	if err != nil {
		return errors.Wrap(err, "assure one affected", nil)
	}
	return nil
}

func (m *Mall) RefreshLastSeenForFixture(fixtureID messages.FixtureID) error {
	q, _, err := m.dialect.Update(goqu.T("fixtures")).Set(goqu.Record{
		"last_seen": time.Now(),
	}).Where(goqu.C("id").Eq(fixtureID)).ToSQL()
	if err != nil {
		return errors.NewQueryToSQLError(err, errors.Details{"fixture": fixtureID})
	}
	result, err := m.db.Exec(q)
	if err != nil {
		return errors.NewExecQueryError(err, q, errors.Details{"fixture": fixtureID})
	}
	err = assureOneRowAffectedForNotFound(result, fmt.Sprintf("fixture %v not found", fixtureID), "fixtures", fixtureID, q)
	if err != nil {
		return errors.Wrap(err, "assure one affected", nil)
	}
	return nil
}
