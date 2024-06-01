// Metrics repository encapsulates the data access logic (interactions with the DB) related to Metrics CRUD in Popcorn.

package metrics

import (
	"Popcorn/internal/entity"
	"Popcorn/internal/errors"
	"Popcorn/pkg/db"
	"Popcorn/pkg/log"
	"context"

	"github.com/go-redis/redis/v8"
)

var metricsDbKey string = "popcorn:metrics"

type Repository interface {
	// Get Popcorn Metrics data
	GetMetrics(ctx context.Context, logger log.Logger) (entity.Metrics, error)
	// Set or Update Popcorn Metrics data
	SetOrUpdateMetrics(ctx context.Context, logger log.Logger, metrics *entity.Metrics) error
}

// repository struct of gang Repository.
// Object of this will be passed around from main to internal.
// Helps to access the repository layer interface and call methods.
type repository struct {
	db *db.RedisDB
}

// Returns a new instance of metrics repository for other packages to access its interface.
func NewRepository(dbwrp *db.RedisDB) Repository {
	return repository{db: dbwrp}
}

func (r repository) GetMetrics(ctx context.Context, logger log.Logger) (entity.Metrics, error) {
	// check if metrics data is available in the db
	available, dberr := r.db.Client().Exists(ctx, metricsDbKey).Result()
	if dberr != nil && dberr != redis.Nil {
		// Error during interacting with DB
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.Exists() in metrics.GetMetrics")
		return entity.Metrics{}, errors.InternalServerError("")
	} else if available == 0 {
		// no metrics data is available
		return entity.Metrics{}, nil
	}
	var metrics entity.Metrics
	if dberr = r.db.Client().HGetAll(ctx, metricsDbKey).Scan(&metrics); dberr != nil {
		// Error during interacting with DB
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.HGetAll() in metrics.GetMetrics")
		return entity.Metrics{}, errors.InternalServerError("")
	}
	return metrics, nil
}

func (r repository) SetOrUpdateMetrics(ctx context.Context, logger log.Logger, metrics *entity.Metrics) error {
	txferr := func(key string) error {
		txf := func(tx *redis.Tx) error {
			// Operation is commited only if the watched keys remain unchanged
			_, dberr := r.db.Client().TxPipelined(ctx, func(client redis.Pipeliner) error {
				client.HSet(ctx, metricsDbKey, "active_ingress", metrics.ActiveIngress)
				client.HSet(ctx, metricsDbKey, "ingress_quota_exceeded", metrics.IngressQuotaExceeded)
				return nil
			})
			return dberr
		}
		for i := 0; i < r.db.GetMaxRetries(); i++ {
			dberr := r.db.Client().Watch(ctx, txf, key)
			if dberr == nil {
				return nil
			} else if dberr == redis.TxFailedErr {
				// Optimistic lock lost. Retry.
				continue
			}
			// Return any other error.
			return dberr
		}
		return errors.New("increment reached maximum number of retries")
	}(metricsDbKey)
	if txferr != nil {
		logger.WithCtx(ctx).Error().Err(txferr).Msg("Error occured in SetOrUpdateMetrics transaction")
		return errors.InternalServerError("")
	}
	return nil
}
