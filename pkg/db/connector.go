// Initialization of Redis client to be used internally in Popcorn.

package db

import (
	"Popcorn/pkg/log"
	"context"
	"os"
	"strconv"
	"strings"

	"github.com/go-redis/redis/v8"
)

// RedisDB represents a redis client connection to be used internally in Popcorn.
type RedisDB struct {
	client *redis.Client
}

// Client returns the redis client wrapped by RedisDB.
func (db *RedisDB) Client() *redis.Client {
	return db.client
}

// Returns a new Redis DB connection wrapped up by RedisDB struct.
func NewDBConnection(ctx context.Context, logger log.Logger, client *redis.Client) *RedisDB {
	dbNumber, strerr := strconv.Atoi(strings.TrimSpace(os.Getenv("REDIS_DB_NUMBER")))
	if strerr != nil {
		logger.WithCtx(ctx).Fatal().Err(strerr).Msg("Couldn't parse ENV: REDIS_DB_NUMBER")
	}
	// Initializing a connection to Redis-server.
	client = redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDR") + ":" + os.Getenv("REDIS_PORT"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       dbNumber,
	})
	return &RedisDB{client: client}
}

// Helper to check connection status of redis client to redis-server.
// Equivalent to a PING request on redis-server, returns PONG on success.
func (db *RedisDB) CheckDBConnection(ctx context.Context, logger log.Logger) {
	logger.WithCtx(ctx).Info().Msg("Checking DB Connection . . .")
	// Pinging the Redis-server to check connection status
	cnterr := db.Client().Ping(ctx).Err()
	if cnterr != nil {
		// Most likely, DB connection failure
		logger.WithCtx(ctx).Fatal().Err(cnterr).Msg("Redis client couldn't PING the redis-server.")
	}
	// Connection successful
	logger.WithCtx(ctx).Info().Msg("Connection to DB Successful")
}

// Helper to check if key:value exists in the DB. Sets the key to 0 if not.
// Best to use this util before starting gin.
func (db *RedisDB) SetGlobKeyIfNotExists(ctx context.Context, logger log.Logger, keys map[string]interface{}) (map[string]bool, error) {
	res := make(map[string]bool)
	if _, dberr := db.Client().Pipelined(ctx, func(client redis.Pipeliner) error {
		for key, val := range keys {
			r, qryerr := client.SetNX(ctx, key, val, -1).Result()
			if qryerr != nil {
				logger.WithCtx(ctx).Error().Err(qryerr).Msg("Error during execution of queries inside redis.Pipelined() in conn.SetKeyIfNotExists")
				return qryerr
			} else {
				// true means set; false means already exist
				res[key] = r
			}
		}
		return nil
	}); dberr != nil {
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.Pipelined() in conn.SetKeyIfNotExists")
		return res, dberr
	}
	return res, nil
}

// Helper to close the RedisDB client, should be called before closing the server.
func (db *RedisDB) CloseDBConnection(ctx context.Context) error {
	return db.Client().Close()
}
