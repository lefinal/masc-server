package app

import (
	"context"
	"github.com/lefinal/masc-server/debugstats"
	"github.com/lefinal/masc-server/errors"
	"github.com/lefinal/masc-server/service"
	"github.com/lefinal/masc-server/store"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"time"
)

type services map[string]service.Service

func createServices(appConfig Config, logger *zap.Logger, mall *store.Mall) (services, error) {
	services := make(services)
	// Debug stats service.
	s, err := debugstats.NewService(logger.Named("debug-stats"), debugstats.Config{
		IsEnabled: appConfig.Log.SystemDebugStatsInterval.Valid,
		Interval:  time.Duration(appConfig.Log.SystemDebugStatsInterval.Int) * time.Minute,
	})
	if err != nil {
		return nil, errors.Wrap(err, "new debug stats service", nil)
	}
	services["debug-stats"] = s
	return services, nil
}

func (s services) run(ctx context.Context) error {
	wg, lifetime := errgroup.WithContext(ctx)
	// Run each.
	for name, serviceToRun := range s {
		wg.Go(func() error {
			if err := serviceToRun.Run(lifetime); err != nil {
				return errors.Wrap(err, "run service", errors.Details{"service_name": name})
			}
			return nil
		})
	}
	return wg.Wait()
}
