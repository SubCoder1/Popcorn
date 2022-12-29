// Initialization of Redis client to be used internally in Popcorn.

package db

import (
	"Popcorn/internal/errors"
	logger "Popcorn/pkg/log"
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
func NewDBConnection(client *redis.Client) *RedisDB {
	dbNumber, strerr := strconv.Atoi(strings.TrimSpace(os.Getenv("REDIS_DB_NUMBER")))
	if strerr != nil {
		logger.Logger.Fatal().Err(strerr).Msg("Couldn't parse ENV: REDIS_DB_NUMBER")
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
func (db *RedisDB) CheckDBConnection(ctx context.Context) error {
	logger.Logger.Info().Msg("Checking DB Connection . . .")
	// Pinging the Redis-server to check connection status
	cnterr := db.Client().Ping(ctx).Err()
	if cnterr != nil {
		// Most likely, DB connection failure
		return errors.InternalServerError(cnterr.Error())
	}
	// Connection successful
	logger.Logger.Info().Msg("Connection to DB Successful")
	return nil
}

// Helper to close the RedisDB client, should be called before closing the server.
func (db *RedisDB) CloseDBConnection() error {
	return db.Client().Close()
}
