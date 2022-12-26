// Initialization of Redis client to be used internally in Popcorn.

package db

import (
	"Popcorn/pkg/logger"
	"context"
	"os"
	"strconv"
	"strings"

	"github.com/go-redis/redis/v8"
)

// A DAO (Data-Access-Object) to be used internally all over Popcorn.
var RedisDAO *redis.Client

func init() {
	redisDB, strerr := strconv.Atoi(strings.TrimSpace(os.Getenv("REDIS_DB")))
	if strerr != nil {
		panic(strerr)
	}
	// Initializing a connection to Redis-server.
	RedisDAO = redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDR") + ":" + os.Getenv("REDIS_PORT"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       redisDB,
	})
	// Pinging the Redis-server to check connection status.
	ctx := context.Background()
	cnterr := RedisDAO.Ping(ctx).Err()
	if cnterr != nil {
		panic(cnterr)
	}
	logger.Logger.Info().Msg("Connection to Redis-server is successful.")
}
