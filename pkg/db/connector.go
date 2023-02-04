// Initialization of Redis client to be used internally in Popcorn.

package db

import (
	"Popcorn/pkg/log"
	"context"
	"os"
	"strconv"
	"strings"
	"sync"

	"github.com/go-redis/redis/v8"
)

// RedisDB represents a redis client connection to be used internally in Popcorn.
type RedisDB struct {
	client *redis.Client
}

// Global DB instance to be used all over Popcorn.
var globalDbClient *RedisDB

// sync.Once singleton is used to make sure configs and DB instantiation is done only once.
var once sync.Once

// Client returns the redis client wrapped by RedisDB.
func (db *RedisDB) Client() *redis.Client {
	return db.client
}

// Returns a new Redis DB connection wrapped up by RedisDB struct.
func NewDbConnection(ctx context.Context, logger log.Logger) *RedisDB {
	once.Do(func() {
		dbNumber, strerr := strconv.Atoi(strings.TrimSpace(os.Getenv("REDIS_DB_NUMBER")))
		if strerr != nil {
			logger.WithCtx(ctx).Fatal().Err(strerr).Msg("Couldn't parse ENV: REDIS_DB_NUMBER")
		}
		// Initializing a connection to Redis-server
		client := redis.NewClient(&redis.Options{
			Addr:     os.Getenv("REDIS_ADDR") + ":" + os.Getenv("REDIS_PORT"),
			Password: os.Getenv("REDIS_PASSWORD"),
			DB:       dbNumber,
		})
		// Instantiate globalDbClient once
		globalDbClient = &RedisDB{client: client}
	})
	return globalDbClient
}

// Helper to check connection status of redis client to redis-server.
// Equivalent to a PING request on redis-server, returns PONG on success.
func (db *RedisDB) CheckDbConnection(ctx context.Context, logger log.Logger) {
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

// Helper to clean up test db after finishing Popcorn tests.
func (db *RedisDB) CleanTestDbData(ctx context.Context, logger log.Logger) {
	if db.Client().Options().DB == 1 {
		dberr := db.Client().FlushDB(ctx).Err()
		if dberr != nil {
			// Error during flushing test db
			logger.Error().Err(dberr).Msg("Error occured during the execution of FlushDB() in db.CleanTestDbData")
		}
	}
}

// Helper to close the RedisDB client, should be called before closing the server.
func (db *RedisDB) CloseDbConnection(ctx context.Context) error {
	return db.Client().Close()
}
