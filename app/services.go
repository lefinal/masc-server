package app

import (
	"context"
	"fmt"
	"github.com/lefinal/masc-server/debugstatssvc"
	"github.com/lefinal/masc-server/devicesvc"
	"github.com/lefinal/masc-server/errors"
	"github.com/lefinal/masc-server/logging"
	"github.com/lefinal/masc-server/logpublishsvc"
	"github.com/lefinal/masc-server/portal"
	"github.com/lefinal/masc-server/service"
	"github.com/lefinal/masc-server/store"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"time"
)

type services map[string]service.Service

func createServices(appConfig Config, logger *zap.Logger, portalBase portal.Base, mall *store.Mall,
	logEntriesIn <-chan logging.LogEntry) (services, error) {
	services := make(services)
	// Debug stats service.
	s, err := debugstatssvc.NewService(logger.Named("debug-stats"), debugstatssvc.Config{
		IsEnabled: appConfig.Log.SystemDebugStatsInterval.Valid && appConfig.Log.SystemDebugStatsInterval.Int > 0,
		Interval:  time.Duration(appConfig.Log.SystemDebugStatsInterval.Int) * time.Minute,
	})
	if err != nil {
		return nil, errors.Wrap(err, "new debug stats service", nil)
	}
	services["debug-stats"] = s
	// Device service.
	services["device"] = devicesvc.NewDeviceService(logger.Named("device"), portalBase.NewPortal("device-service"), mall)
	// Log publishing service.
	services["log-publish"] = logpublishsvc.New(logger.Named("log-publish"), portalBase.NewPortal("log-publish"), logEntriesIn)
	return services, nil
}

func (s services) run(ctx context.Context, logger *zap.Logger) error {
	wg, lifetime := errgroup.WithContext(ctx)
	// Run each.
	for name, serviceToRun := range s {
		// Copy values.
		name, serviceToRun := name, serviceToRun
		wg.Go(func() error {
			logger.Debug(fmt.Sprintf("service %s up", name))
			defer logger.Debug(fmt.Sprintf("service %s down", name))
			if err := serviceToRun.Run(lifetime); err != nil {
				return errors.Wrap(err, "run service", errors.Details{"service_name": name})
			}
			return nil
		})
	}
	return wg.Wait()
}
