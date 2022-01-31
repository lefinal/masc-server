package logpublish

import (
	"context"
	"fmt"
	"github.com/LeFinal/masc-server/acting"
	"github.com/LeFinal/masc-server/errors"
	"github.com/LeFinal/masc-server/logging"
	"github.com/LeFinal/masc-server/messages"
	"github.com/google/uuid"
	"sync"
	"time"
)

type actorReception struct {
	// activeLogMonitors are all active actors with acting.RoleTypeLogMonitor. All
	// incoming log entries will be published to these actors.
	activeLogMonitors      map[acting.Actor]struct{}
	activeLogMonitorsMutex sync.RWMutex
	ctx                    context.Context
}

// RunActorReception runs a reception that subscribes
func RunActorReception(ctx context.Context, agency acting.ActorNewsletter, logEntriesToPublish <-chan logging.LogEntry) {
	r := &actorReception{
		ctx:               ctx,
		activeLogMonitors: make(map[acting.Actor]struct{}),
	}
	agency.SubscribeNewActors(r)
run:
	for {
		select {
		case <-ctx.Done():
			break run
		case e := <-logEntriesToPublish:
			// Read for 200ms in order to avoid to many single messages.
			entries := make([]logging.LogEntry, 0, 1)
			entries = append(entries, e)
			readTimeout, cancel := context.WithTimeout(ctx, 200*time.Millisecond)
		readBufferedEntries:
			for {
				select {
				case <-readTimeout.Done():
					break readBufferedEntries
				case e = <-logEntriesToPublish:
					entries = append(entries, e)
				}
			}
			cancel()
			r.handleNewLogEntries(entries)
		}
	}
	agency.UnsubscribeNewActors(r)
}

// handleNewLogEntries takes a list of new log entries and publishes them to all
// actorReception.activeLogMonitors.
func (r *actorReception) handleNewLogEntries(entries []logging.LogEntry) {
	// Create the message.
	messageEntries := make([]messages.MessageNextLogEntriesEntry, 0, len(entries))
	for _, entry := range entries {
		messageEntries = append(messageEntries, messageLogEntryFromLogEntry(entry))
	}
	message := messages.MessageNextLogEntries{Entries: messageEntries}
	// Publish to all.
	r.activeLogMonitorsMutex.RLock()
	defer r.activeLogMonitorsMutex.RUnlock()
	var wg sync.WaitGroup
	wg.Add(len(r.activeLogMonitors))
	for actor := range r.activeLogMonitors {
		go func(actor acting.Actor) {
			defer wg.Done()
			err := actor.Send(acting.ActorOutgoingMessage{
				MessageType: messages.MessageTypeNextLogEntries,
				Content:     message,
			})
			if err != nil {
				errors.Log(logging.NoPublishLogger, errors.Wrap(err, "send",
					errors.Details{"message_content": message}))
				return
			}
		}(actor)
	}
	wg.Wait()
}

// messageLogEntryFromLogEntry creates a messages.MessageNextLogEntriesEntry
// from the given logrus.Entry.
func messageLogEntryFromLogEntry(entry logging.LogEntry) messages.MessageNextLogEntriesEntry {
	return messages.MessageNextLogEntriesEntry{
		Time:    entry.Time,
		Message: entry.Message,
		Level:   entry.Level.String(),
		Fields:  entry.Fields,
	}
}

// HandleNewActor hires actors with acting.RoleTypeLogMonitor.
func (r *actorReception) HandleNewActor(actor acting.Actor, role acting.RoleType) {
	if role != acting.RoleTypeLogMonitor {
		return
	}
	go func(ctx context.Context, actor acting.Actor) {
		// Hire.
		err := actor.Hire(fmt.Sprintf("log-monitor-%s", uuid.New().String()))
		if err != nil {
			errors.Log(logging.LogPublishLogger, errors.Wrap(err, "hire actor", nil))
			return
		}
		// Add to active ones.
		r.activeLogMonitorsMutex.Lock()
		r.activeLogMonitors[actor] = struct{}{}
		r.activeLogMonitorsMutex.Unlock()
		// Listen for quit.
		select {
		case <-ctx.Done():
		case <-actor.Quit():
		}
		// Unregister.
		r.activeLogMonitorsMutex.Lock()
		delete(r.activeLogMonitors, actor)
		r.activeLogMonitorsMutex.Unlock()
	}(r.ctx, actor)
}
