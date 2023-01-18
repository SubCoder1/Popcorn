// User repository encapsulates the data access logic (interactions with the DB) related to Users in Popcorn.

package user

import (
	"Popcorn/internal/entity"
	"Popcorn/internal/errors"
	"Popcorn/pkg/db"
	"Popcorn/pkg/log"
	"context"
	"fmt"

	"github.com/go-redis/redis/v8"
)

type Repository interface {
	// Get returns the user with username if exists.
	GetUser(ctx context.Context, logger log.Logger, username string) (entity.User, error)
	// Set adds the user with credentials saved in ue into the DB.
	SetUser(ctx context.Context, logger log.Logger, user entity.User, userExistCheck bool, setProfPicOnly bool) (bool, error)
	// Has returns a boolean depending on user's availability.
	HasUser(ctx context.Context, logger log.Logger, username string) (bool, error)
	// Increments the current number of Users to 1 and returns the new total.
	// Util used during user registration.
	IncrTotalUsers(ctx context.Context, logger log.Logger) error
}

// repository struct of user Repository.
// Object of this will be passed around from main to internal.
// Helps to access the repository layer interface and call methods.
type repository struct {
	db *db.RedisDB
}

// Returns a new instance of repository for other packages to access its interface.
func NewRepository(dbwrp *db.RedisDB) Repository {
	return repository{db: dbwrp}
}

// Returns the user data object if user with the given username is found in the DB.
func (r repository) GetUser(ctx context.Context, logger log.Logger, username string) (entity.User, error) {
	user := entity.User{}
	available, dberr := r.db.Client().HExists(ctx, "user:"+username, "username").Result()
	if dberr != nil && dberr != redis.Nil {
		// Error during interacting with DB
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.HExists() in user.Get")
		return user, errors.InternalServerError("")
	} else if !available {
		// User not available
		return user, errors.NotFound("User not available")
	}
	if dberr := r.db.Client().HGetAll(ctx, "user:"+username).Scan(&user); dberr != nil {
		// Error during interacting with DB
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.HGetAll() in user.Get")
		return user, errors.InternalServerError("")
	}
	return user, nil
}

// Returns true if user successfully got added into the DB.
func (r repository) SetUser(ctx context.Context, logger log.Logger, ue entity.User, userExistCheck bool, setProfPicOnly bool) (bool, error) {
	if !userExistCheck {
		// Checking if an user with username ue.username exists in the DB
		available, dberr := r.HasUser(ctx, logger, ue.Username)
		if dberr != nil {
			// Issues in Exists()
			return false, dberr
		} else if available {
			return false, errors.BadRequest("User already exists")
		}
	}
	// Add the user into the DB
	key := "user:" + ue.Username
	if _, dberr := r.db.Client().Pipelined(ctx, func(client redis.Pipeliner) error {
		if !setProfPicOnly {
			client.HSet(ctx, key, "username", ue.Username)
			client.HSet(ctx, key, "full_name", ue.FullName)
			client.HSet(ctx, key, "password", ue.Password)
		}
		client.HSet(ctx, key, "user_profile_pic", ue.ProfilePic)
		return nil
	}); dberr != nil {
		// Error during interacting with DB
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.Pipelined() in user.Set")
		return false, errors.InternalServerError("")
	}
	return true, nil
}

// Returns true if user with the given username exists in Popcorn.
func (r repository) HasUser(ctx context.Context, logger log.Logger, username string) (bool, error) {
	available, dberr := r.db.Client().Exists(ctx, "user:"+username).Result()
	if dberr != nil && dberr != redis.Nil {
		// Error during interacting with DB
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.HExists() in user.Get")
		return false, errors.InternalServerError("")
	} else if available == 0 {
		// User not available
		return false, nil
	}
	return true, nil
}

// Increments the total number of users in Popcorn.
func (r repository) IncrTotalUsers(ctx context.Context, logger log.Logger) error {
	newTotal, dberr := r.db.Client().IncrBy(ctx, "users", 1).Result()
	if dberr != nil {
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.IncrBy in user.IncTotal")
		return errors.InternalServerError("")
	}
	logger.WithCtx(ctx).Info().Msg(fmt.Sprintf("Current total users in Popcorn - %d", newTotal))
	return nil
}
