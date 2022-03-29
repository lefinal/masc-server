package logpublishsvc

import (
	"context"
	"github.com/lefinal/masc-server/event"
	"github.com/lefinal/masc-server/logging"
	"github.com/lefinal/masc-server/portal"
	"github.com/lefinal/masc-server/service"
	"go.uber.org/zap"
	"time"
)

// topicLogPublish the topic to publish log entries to.
const topicLogPublish portal.Topic = "lefinal/masc/log/next"

// publishDebounceDelay is the delay to wait for collecting log entries. This
// avoids blocking on every log call.
const publishDebounceDelay = 100 * time.Millisecond

// logPublishService publishes log entries from logEntriesIn to the portal.
type logPublishService struct {
	logger *zap.Logger
	portal portal.Portal
	// logEntriesIn is the channel to read log entries to publish from.
	logEntriesIn <-chan logging.LogEntry
}

// New creates a new log publish service that can be run. The given
// logging.LogEntry channel is the channel log entries will be read from.
func New(logger *zap.Logger, portal portal.Portal, logEntriesIn <-chan logging.LogEntry) service.Service {
	return &logPublishService{
		logger:       logger,
		portal:       portal,
		logEntriesIn: logEntriesIn,
	}
}

// Run the service until the given context.Context is done.
func (s *logPublishService) Run(ctx context.Context) error {
	for {
		select {
		case <-ctx.Done():
			return nil
		case entry, more := <-s.logEntriesIn:
			if !more {
				return nil
			}
			s.publishLogEntriesAfterTimeout(ctx, entry)
		}
	}
}

// publishLogEntriesAfterTimeout publishes the given logging.LogEntry to topicLogPublish.
func (s *logPublishService) publishLogEntriesAfterTimeout(ctx context.Context, firstEntry logging.LogEntry) {
	select {
	case <-ctx.Done():
		return
	case <-time.After(publishDebounceDelay):
		// Publish first and read all remaining entries.
		s.publishLogEntry(ctx, firstEntry)
		for {
			select {
			case entry := <-s.logEntriesIn:
				s.publishLogEntry(ctx, entry)
			default:
				return
			}
		}
	}
}

func (s *logPublishService) publishLogEntry(ctx context.Context, entry logging.LogEntry) {
	s.portal.Publish(ctx, topicLogPublish, event.NextLogEntryEvent{
		Time:       entry.Time,
		Message:    entry.Message,
		Level:      entry.Level.String(),
		LoggerName: entry.LoggerName,
		Fields:     entry.Fields,
	})
}
