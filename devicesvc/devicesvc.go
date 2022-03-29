package devicesvc

import (
	"context"
	"github.com/lefinal/masc-server/errors"
	"github.com/lefinal/masc-server/event"
	"github.com/lefinal/masc-server/portal"
	"github.com/lefinal/masc-server/service"
	"github.com/lefinal/masc-server/store"
	"go.uber.org/zap"
	"sync"
)

// deviceID is the unique id to use for the server device. This is fixed,
// because we do not expect any other servers.
const deviceID string = "masc-server"

// Topics.
const (
	// topicReport is used for requesting an online report to topicDeviceOnline.
	topicReport portal.Topic = "lefinal/masc/devices/report"
	// topicDeviceOnline is where devices announce their presence.
	topicDeviceOnline portal.Topic = "lefinal/masc/devices/online"
	// topicDeviceOnlineDetailed is where devices are announced in detailed version.
	topicDeviceOnlineDetailed portal.Topic = "lefinal/masc/devices/online/detailed"
	// topicDeviceOffline is where devices announce that they are offline.
	topicDeviceOffline portal.Topic = "lefinal/masc/devices/offline"
)

// Store are the dependencies needed for NewDeviceService.
type Store interface {
	// RegisterDevice retrieves the Device with the given id. If none was found, a
	// new one is created with default values and returned. The second return value
	// describes whether the device was created. In any case, the last seen
	// timestamp is set to the current time. If the device was found and the type
	// changed, the config will get cleared.
	RegisterDevice(ctx context.Context, deviceID string, deviceType string) (store.Device, bool, error)
	// UpdateDeviceLastSeen updates the last seen timestamp for the device with the
	// given id.
	UpdateDeviceLastSeen(ctx context.Context, deviceID string) error
}

// deviceService is used for managing devices as well as notifying of the
// server's presence.
type deviceService struct {
	logger *zap.Logger
	// portal to use for communication.
	portal portal.Portal
	// store with all persistence dependencies.
	store Store
}

// NewDeviceService creates a new service.Service ready to run.
func NewDeviceService(logger *zap.Logger, portal portal.Portal, store Store) service.Service {
	return &deviceService{
		logger: logger,
		portal: portal,
		store:  store,
	}
}

// Run the service, request online-report and serve.
func (s *deviceService) Run(ctx context.Context) error {
	var wg sync.WaitGroup
	// Subscribe to online-report request.
	onlineReportNewsletter := portal.Subscribe[event.EmptyEvent](ctx, s.portal, topicReport)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for range onlineReportNewsletter.Receive {
			s.handleReportOnlineEvent(ctx)
		}
	}()
	// Handle new online devices.
	onlineDevicesNewsletter := portal.Subscribe[event.DeviceOnlineEvent](ctx, s.portal, topicDeviceOnline)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for e := range onlineDevicesNewsletter.Receive {
			s.handleDeviceOnlineEvent(ctx, e.Payload)
		}
	}()
	// Handle offline devices.
	offlineDevicesNewsletter := portal.Subscribe[event.DeviceOfflineEvent](ctx, s.portal, topicDeviceOffline)
	wg.Add(1)
	go func() {
		defer wg.Done()
		for e := range offlineDevicesNewsletter.Receive {
			s.handleDeviceOfflineEvent(ctx, e.Payload)
		}
	}()
	// Request online report after all handlers up.
	s.requestDeviceOnlineReport(ctx)
	// Wait until all services done.
	wg.Wait()
	return nil
}

// handleReportOnlineEvent handles topicReport that announces its presence to
// topicDeviceOnline.
func (s *deviceService) handleReportOnlineEvent(ctx context.Context) {
	s.portal.Publish(ctx, topicDeviceOnline, event.DeviceOnlineEvent{
		DeviceID:   deviceID,
		DeviceType: "masc-server",
	})
}

// handleDeviceOnlineEvent handles topicDeviceOnline.
func (s *deviceService) handleDeviceOnlineEvent(ctx context.Context, e event.DeviceOnlineEvent) {
	device, _, err := s.store.RegisterDevice(ctx, e.DeviceID, e.DeviceType)
	if err != nil {
		errors.Log(s.logger, errors.Wrap(err, "register device", errors.Details{
			"device_id":   e.DeviceID,
			"device_type": e.DeviceType,
		}))
		return
	}
	// Publish detailed.
	s.portal.Publish(ctx, topicDeviceOnlineDetailed, event.DeviceOnlineDetailedEvent{
		DeviceID:   device.ID,
		DeviceType: device.Type,
		LastSeen:   device.LastSeen,
		Name:       device.Name,
	})
}

// handleDeviceOfflineEvent handles topicDeviceOffline.
func (s *deviceService) handleDeviceOfflineEvent(ctx context.Context, e event.DeviceOfflineEvent) {
	err := s.store.UpdateDeviceLastSeen(ctx, e.DeviceID)
	if err != nil {
		errors.Log(s.logger, errors.Wrap(err, "update device last seen", errors.Details{"device_id": e.DeviceID}))
		return
	}
}

// requestDeviceOnlineReport publishes to topicReport in order to request online
// reports from all devices.
func (s *deviceService) requestDeviceOnlineReport(ctx context.Context) {
	s.portal.Publish(ctx, topicReport, event.EmptyEvent{})
}
