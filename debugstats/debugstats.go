package debugstats

import (
	"context"
	"fmt"
	"github.com/LeFinal/masc-server/service"
	"go.uber.org/zap"
	"runtime"
	"time"
)

type Config struct {
	// IsEnabled describes whether periodic debug stats logging is desired.
	IsEnabled bool
	// Interval in which to log debug stats.
	Interval time.Duration
}

type debugStatsService struct {
	logger *zap.Logger
	config Config
}

func NewService(logger *zap.Logger, config Config) (service.Service, error) {
	return &debugStatsService{
		logger: logger,
		config: config,
	}, nil
}

func (s *debugStatsService) Run(ctx context.Context) error {
	if !s.config.IsEnabled {
		return nil
	}
	s.logger.Debug(fmt.Sprintf("logging system state every %gs", s.config.Interval.Seconds()))
	logSystemDebugStats(ctx, s.config.Interval, s.logger)
	return nil
}

// logSystemDebugStats logs the current system state like memory stats, current
// stack, etc. to the given zap.Logger in the given interval.
func logSystemDebugStats(ctx context.Context, interval time.Duration, logger *zap.Logger) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-time.After(interval):
			// Num CPU.
			numCPU := runtime.NumCPU()
			// Num goroutines.
			numGoroutine := runtime.NumGoroutine()
			// Memory usage.
			var memStats runtime.MemStats
			runtime.ReadMemStats(&memStats)
			memoryUsageMB := memStats.Sys / 1000 / 1000
			// Get current stack.
			buf := make([]byte, 1<<16)
			stackSize := runtime.Stack(buf, true)
			// Log it.
			logger.Debug(fmt.Sprintf(`
----------BEGIN OF DEBUG SYSTEM STATS-----------
       Num CPU: %d
Num goroutines: %d
 Memory in use: %dMB

----------BEGIN OF STACK----------
%s
----------END OF STACK------------
----------END OF DEBUG SYSTEM STATS-------------
`, numCPU, numGoroutine, memoryUsageMB, string(buf[0:stackSize])))
		}
	}
}
