// Handles all of the Redis Database operations going around in Popcorn.

package db

import (
	"Popcorn/pkg/logger"
	"context"

	"github.com/go-redis/redis/v8"
)

// Helper to check connection status of redis client to redis-server.
// Equivalent to a PING request on redis-server, returns PONG on success.
func PingToRedisServer(client *redis.Client) error {
	// Pinging the Redis-server to check connection status.
	logger.Logger.Info().Msg("Checking DB Connection . . .")
	ctx := context.Background()
	cnterr := client.Ping(ctx).Err()
	if cnterr != nil {
		// Most likely, DB connection failure
		return cnterr
	}
	// Successful Operation
	logger.Logger.Info().Msg("Connection to DB Successful")
	return nil
}
