package lighting

import (
	nativeerrors "errors"
	"github.com/LeFinal/masc-server/acting"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/messages"
	"github.com/LeFinal/masc-server/stores"
	"github.com/gobuffalo/nulls"
	"github.com/stretchr/testify/suite"
	"testing"
	"time"
)

type MockFixtureTestSuite struct {
	suite.Suite
	fixture *MockFixture
}

func (suite *MockFixtureTestSuite) SetupTest() {
	suite.fixture = NewMockFixture(MockFixtureProps{})
}

func (suite *MockFixtureTestSuite) TestNewMockFixture() {
	props := MockFixtureProps{
		id:          766,
		deviceID:    messages.DeviceID("food"),
		providerID:  messages.FixtureProviderFixtureID("boast"),
		isEnabled:   true,
		fixtureType: "beak",
		name:        nulls.NewString("gate"),
		isLocating:  true,
		actor:       acting.NewMockActor("knife"),
		features:    []messages.FixtureFeature{messages.FixtureFeatureDimmer},
	}
	suite.Equal(MockFixture{
		id:          props.id,
		deviceID:    props.deviceID,
		providerID:  props.providerID,
		isEnabled:   props.isEnabled,
		fixtureType: props.fixtureType,
		name:        props.name,
		isLocating:  props.isLocating,
		actor:       props.actor,
	}, *NewMockFixture(props), "should be as set from props")
}

func (suite *MockFixtureTestSuite) TestID() {
	suite.Equal(suite.fixture.id, suite.fixture.ID(), "id should be as set")
}

func (suite *MockFixtureTestSuite) TestDeviceID() {
	suite.Equal(suite.fixture.deviceID, suite.fixture.DeviceID(), "device id should be as set")
}

func (suite *MockFixtureTestSuite) TestProviderID() {
	suite.Equal(suite.fixture.providerID, suite.fixture.ProviderID(), "provider id should be as set")
}

func (suite *MockFixtureTestSuite) TestIsEnabled() {
	suite.fixture.isEnabled = true
	suite.Equal(suite.fixture.isEnabled, suite.fixture.IsEnabled(), "enabled-state should be as set")
}

func (suite *MockFixtureTestSuite) TestSetEnabled() {
	suite.fixture.SetEnabled(true)
	suite.True(suite.fixture.isEnabled, "should set enabled-state correctly")
}

func (suite *MockFixtureTestSuite) TestReset() {
	start := time.Now()
	suite.fixture.Reset()
	suite.True(suite.fixture.lastReset.After(start), "last reset should be after start")
}

func (suite *MockFixtureTestSuite) TestType() {
	suite.fixture.fixtureType = messages.FixtureTypeDimmer
	suite.Equal(suite.fixture.fixtureType, suite.fixture.Type(), "fixture type should be as set")
}

func (suite *MockFixtureTestSuite) TestName() {
	suite.fixture.name = nulls.NewString("fork")
	suite.Equal(suite.fixture.name, suite.fixture.Name(), "name should be as set")
}

func (suite *MockFixtureTestSuite) TestSetName() {
	newName := nulls.NewString("western")
	suite.fixture.setName(newName)
	suite.Equal(newName, suite.fixture.name, "name should be as set")
}

func (suite *MockFixtureTestSuite) TestFeatures() {
	suite.fixture.features = []messages.FixtureFeature{messages.FixtureFeatureDimmer}
	suite.Equal(suite.fixture.features, suite.fixture.Features(), "features should be as set")
}

func (suite *MockFixtureTestSuite) TestIsLocating() {
	suite.fixture.isLocating = true
	suite.Equal(suite.fixture.isLocating, suite.fixture.IsLocating(), "locating-state should be as set")
}

func (suite *MockFixtureTestSuite) TestSetLocating() {
	suite.fixture.SetLocating(true)
	suite.True(suite.fixture.isLocating, "locating-state should be as set")
}

func (suite *MockFixtureTestSuite) TestApplyFail() {
	start := time.Now()
	suite.fixture.ApplyErr = nativeerrors.New("compose")
	err := suite.fixture.Apply()
	suite.NotNil(err, "should fail")
	suite.True(start.After(suite.fixture.lastApply), "should not change last apply")
}

func (suite *MockFixtureTestSuite) TestApplyOK() {
	start := time.Now()
	err := suite.fixture.Apply()
	suite.Nilf(err, "should not fail but got: %s", errors.Prettify(err))
	suite.True(suite.fixture.lastApply.After(start), "last apply should be after start")
}

func (suite *MockFixtureTestSuite) TestIsOnline() {
	suite.fixture.actor = acting.NewMockActor("voice")
	suite.True(suite.fixture.IsOnline(), "online-state should be as set")
}

func (suite *MockFixtureTestSuite) TestSetActorOnline() {
	suite.fixture.actor = nil
	suite.fixture.setActor(acting.NewMockActor("shop"))
	suite.NotNil(suite.fixture.actor, "actor should be set")
}

func (suite *MockFixtureTestSuite) TestSetActorOffline() {
	suite.fixture.actor = acting.NewMockActor("enclose")
	suite.fixture.setActor(nil)
	suite.Nil(suite.fixture.actor, "actor should be unset")
}

func (suite *MockFixtureTestSuite) TestActorID() {
	actorID := messages.ActorID("frighten")
	suite.fixture.actor = acting.NewMockActor(actorID)
	suite.Equal(actorID, suite.fixture.actorID(), "actor id should be as set")
}

func TestMockFixture(t *testing.T) {
	suite.Run(t, new(MockFixtureTestSuite))
}

type MockManagerStoreTestSuite struct {
	suite.Suite
	manager      *MockManagerStore
	knownFixture stores.Fixture
}

func (suite *MockManagerStoreTestSuite) SetupTest() {
	suite.manager = &MockManagerStore{}
}

// TODO: MockManager tests

func TestMockManagerStore(t *testing.T) {
	suite.Run(t, new(MockManagerStoreTestSuite))
}
