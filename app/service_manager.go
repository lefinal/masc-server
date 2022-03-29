package app

import (
	"context"
	"fmt"
	"github.com/lefinal/masc-server/debugstats"
	"github.com/lefinal/masc-server/devicesvc"
	"github.com/lefinal/masc-server/errors"
	"github.com/lefinal/masc-server/portal"
	"github.com/lefinal/masc-server/service"
	"github.com/lefinal/masc-server/store"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"time"
)

type services map[string]service.Service

func createServices(appConfig Config, logger *zap.Logger, portalBase portal.Base, mall *store.Mall) (services, error) {
	services := make(services)
	// Debug stats service.
	s, err := debugstats.NewService(logger.Named("debug-stats"), debugstats.Config{
		IsEnabled: appConfig.Log.SystemDebugStatsInterval.Valid && appConfig.Log.SystemDebugStatsInterval.Int > 0,
		Interval:  time.Duration(appConfig.Log.SystemDebugStatsInterval.Int) * time.Minute,
	})
	if err != nil {
		return nil, errors.Wrap(err, "new debug stats service", nil)
	}
	services["debug-stats"] = s
	// Device service.
	services["device"] = devicesvc.NewDeviceService(logger, portalBase.NewPortal("device-service"), mall)
	return services, nil
}

func (s services) run(logger *zap.Logger, ctx context.Context) error {
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
