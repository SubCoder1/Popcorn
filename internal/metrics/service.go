// Service layer of the internal package metrics.

package metrics

import (
	"Popcorn/internal/entity"
	"Popcorn/pkg/log"
	"context"
	"sync"
	"time"
)

// Service layer of internal package metrics which encapsulates metrics CRUD logic of Popcorn.
type Service interface {
	// get Popcorn metrics
	GetMetrics(ctx context.Context) (entity.Metrics, error)
	// set or update Popcorn metrics
	SetOrUpdateMetrics(ctx context.Context, metrics *entity.Metrics) error
	// reset metrics in the beginning of every month
	ResetMetrics(ctx context.Context)
}

// Object of this will be passed around from main to routers to API.
// Helps to access the service layer interface and call methods.
// Also helps to pass objects to be used from outer layer.
type service struct {
	livekit_config entity.LivekitConfig
	metricsRepo    Repository
	logger         log.Logger
}

// sync.Once singleton is used to make sure metrics ticker instantiation is done only once.
var once sync.Once

// ticker used in ResetMetrics to trigger a reset action at the beginning of every month
var ticker *time.Ticker

// stopResetMetrics channel used to stop long running ResetMetrics() method
var stopResetMetrics chan bool

// day variable to check current day
var day int

func NewService(livekit_config entity.LivekitConfig, metricsRepo Repository, logger log.Logger) Service {
	return service{livekit_config: livekit_config, metricsRepo: metricsRepo, logger: logger}
}

func (s service) GetMetrics(ctx context.Context) (entity.Metrics, error) {
	return s.metricsRepo.GetMetrics(ctx, s.logger)
}

func (s service) SetOrUpdateMetrics(ctx context.Context, metrics *entity.Metrics) error {
	return s.metricsRepo.SetOrUpdateMetrics(ctx, s.logger, metrics)
}

func (s service) ResetMetrics(ctx context.Context) {
	once.Do(func() {
		ticker = time.NewTicker(5 * time.Hour)
		stopResetMetrics = make(chan bool)
		// this is for initial check
		go s.revertingressquota(ctx)
	})
	s.logger.WithCtx(ctx).Info().Msg("Launching ResetMetrics()")
	for {
		select {
		case <-ticker.C:
			s.revertingressquota(ctx)
		case <-stopResetMetrics:
			ticker.Stop()
			s.logger.WithCtx(ctx).Info().Msg("Successfully stopped ResetMetrics()")
			return
		}
	}
}

func (s service) revertingressquota(ctx context.Context) {
	day = time.Now().UTC().Day()
	if day == 1 {
		// Reset the metrics on the first day of every month
		s.logger.WithCtx(ctx).Info().Msg("Setting metrics.IngressQuotaExceeded to False")
		metrics, _ := s.GetMetrics(ctx)
		metrics.IngressQuotaExceeded = false
		s.SetOrUpdateMetrics(ctx, &metrics)
		s.logger.WithCtx(ctx).Info().Msg("Metrics Reset Successful")
	}
}

func Cleanup(ctx context.Context) {
	time.Sleep(1 * time.Second)
	stopResetMetrics <- true
	close(stopResetMetrics)
}
