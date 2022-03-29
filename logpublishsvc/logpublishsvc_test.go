package logpublishsvc

import (
	"github.com/lefinal/masc-server/logging"
	"github.com/lefinal/masc-server/portal"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"testing"
)

func TestNew(t *testing.T) {
	logger := zap.New(zapcore.NewNopCore())
	portalStub := &portal.Stub{}
	logEntriesIn := make(<-chan logging.LogEntry)
	s := New(logger, portalStub, logEntriesIn).(*logPublishService)
	require.NotNil(t, s, "should create")
	assert.Equal(t, logger, s.logger, "should set correct logger")
	assert.Equal(t, portalStub, s.portal, "should set correct portal")
	assert.Equal(t, logEntriesIn, s.logEntriesIn, "should set correct log entries in channel")
}
