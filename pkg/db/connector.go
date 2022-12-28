// Initialization of Redis client to be used internally in Popcorn.

package db

import (
	"Popcorn/pkg/logger"
	"os"
	"strconv"
	"strings"

	"github.com/go-redis/redis/v8"
)

// A DAO (Data-Access-Object) to be used internally all over Popcorn.
var redisDAO *redis.Client

func init() {
	dbNumber, strerr := strconv.Atoi(strings.TrimSpace(os.Getenv("REDIS_DB_NUMBER")))
	if strerr != nil {
		logger.Logger.Fatal().Err(strerr).Msg("Couldn't parse ENV: REDIS_DB_NUMBER")
	}
	// Initializing a connection to Redis-server.
	redisDAO = redis.NewClient(&redis.Options{
		Addr:     os.Getenv("REDIS_ADDR") + ":" + os.Getenv("REDIS_PORT"),
		Password: os.Getenv("REDIS_PASSWORD"),
		DB:       dbNumber,
	})
}

func CloseDBConnection() error {
	return redisDAO.Close()
}
