// User repository encapsulates the data access logic (interactions with the DB) related to Users in Popcorn.

package user

import (
	"Popcorn/internal/entity"
	"Popcorn/internal/errors"
	"Popcorn/pkg/db"
	logger "Popcorn/pkg/log"
	"context"

	"github.com/go-redis/redis/v8"
)

type Repository interface {
	// User repository function Get returns the user with username if exists
	Get(ctx context.Context, username string) (entity.User, error)
	// User repository function Exists returns a boolean depending on user's availibility.
	Exists(ctx context.Context, username string) (bool, error)
}

// repository struct of user Repository.
// Object of this will be passed around from main to routers.
// Helps to access the repository layer interface and call methods.
type repository struct {
	db *db.RedisDB
}

// Returns a new instance of repository for other packages to access its interface.
func NewRepository(dbwrp *db.RedisDB) Repository {
	return repository{db: dbwrp}
}

// Returns the user data object if user with the given username is found in the DB.
func (r repository) Get(ctx context.Context, username string) (entity.User, error) {
	user := entity.User{}
	available, dberr := r.db.Client().HExists(ctx, "user:"+username, username).Result()
	if dberr != nil && dberr != redis.Nil {
		// Error during interacting with DB
		logger.Logger.Error().Err(dberr).Msg("Error occured during execution of redis.HExists() in user.Get")
		return user, dberr
	} else if !available {
		// User not available
		return user, nil
	}
	if dberr := r.db.Client().HGetAll(ctx, "user:"+username).Scan(&user); dberr != nil {
		// Error during interacting with DB
		logger.Logger.Error().Err(dberr).Msg("Error occured during execution of redis.HGetAll() in user.Get")
		return user, dberr
	}
	return user, nil
}

// Returns true if user with the given username exists in Popcorn.
func (r repository) Exists(ctx context.Context, username string) (bool, error) {
	available, dberr := r.db.Client().HExists(context.Background(), "user:"+username, username).Result()
	if dberr != nil && dberr != redis.Nil {
		// Error during interacting with DB
		logger.Logger.Error().Err(dberr).Msg("Error occured during execution of redis.HExists() in user.Get")
		return false, errors.InternalServerError("")
	} else if !available {
		// User not available
		return false, nil
	}
	return true, nil
}
