// User repository encapsulates the data access logic (interactions with the DB) related to Users in Popcorn.

package user

import (
	"Popcorn/internal/entity"
	"Popcorn/internal/errors"
	"Popcorn/pkg/db"
	"Popcorn/pkg/log"
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

type Repository interface {
	// GetUser returns the user with username if exists.
	GetUser(ctx context.Context, logger log.Logger, username string) (entity.User, error)
	// SetUser adds the user with credentials saved in ue into the DB.
	SetOrUpdateUser(ctx context.Context, logger log.Logger, user entity.User, userExistCheck bool) (bool, error)
	// HasUser returns a boolean depending on user's availability.
	HasUser(ctx context.Context, logger log.Logger, username string) (bool, error)
	// SearchGang returns paginated gang data depending on the query.
	SearchUser(ctx context.Context, logger log.Logger, query entity.UserSearch) ([]entity.User, uint64, error)
	// AddStreamingToken adds streaming token credentials to DB.
	AddStreamingToken(ctx context.Context, logger log.Logger, username, token string)
	// GetStreamingToken fetches the user streaming token from DB if available.
	GetStreamingToken(ctx context.Context, logger log.Logger, username string) string
	// DelStreamingToken deletes the user streaming token from DB.
	DelStreamingToken(ctx context.Context, logger log.Logger, username string)
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

// Returns true if user got successfully added or updated into the DB.
func (r repository) SetOrUpdateUser(ctx context.Context, logger log.Logger, ue entity.User, userExistCheck bool) (bool, error) {
	if !userExistCheck {
		// Checking if an user with username ue.username exists in the DB
		available, dberr := r.HasUser(ctx, logger, ue.Username)
		if dberr != nil {
			// Issues in HasUser()
			return false, dberr
		} else if available {
			return false, errors.BadRequest("User already exists")
		}
	}
	// Transaction to set user data
	key := "user:" + ue.Username
	txferr := func(key string) error {
		txf := func(tx *redis.Tx) error {
			// Operation is commited only if the watched keys remain unchanged
			_, dberr := r.db.Client().TxPipelined(ctx, func(client redis.Pipeliner) error {
				client.HSet(ctx, key, "username", ue.Username)
				client.HSet(ctx, key, "full_name", ue.FullName)
				client.HSet(ctx, key, "password", ue.Password)
				client.HSet(ctx, key, "user_profile_pic", ue.ProfilePic)
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
	}(key)
	if txferr != nil {
		logger.WithCtx(ctx).Error().Err(txferr).Msg("Error occured in SetUser transaction")
		return false, errors.InternalServerError("")
	}

	// Add user to user:index for faster searches
	_, dberr := r.db.Client().SAdd(ctx, "user:index", ue.Username).Result()
	if dberr != nil {
		// Issues in SAdd()
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during setting user index")
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

// Returns user data matching incoming query in DB.
func (r repository) SearchUser(ctx context.Context, logger log.Logger, query entity.UserSearch) ([]entity.User, uint64, error) {
	searchBy := query.Username + "*"
	initialResult, newCursor, dberr := r.db.Client().SScan(ctx, "user:index", uint64(query.Cursor), searchBy, 10).Result()
	if dberr != nil && dberr != redis.Nil {
		// Error during interacting with DB
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.SScan() in user.SearchUser")
		return []entity.User{}, uint64(0), errors.InternalServerError("")
	}
	resultSet := make(map[string]struct{}) // Empty set
	// Helper to add values from SScan() into resultSet
	addIntoResultSet := func(resultList []string) {
		for _, u := range resultList {
			resultSet[u] = struct{}{}
		}
	}
	addIntoResultSet(initialResult)
	// Have to repeat SScan() until we get 10 results or cursor returned by the server is 0 again
	// Else unpredictable searchResult will be returned to the client
	for len(resultSet) <= 10 && newCursor != 0 {
		freshList, freshCursor, dberr := r.db.Client().SScan(ctx, "user:index", newCursor, searchBy, 10).Result()
		if dberr != nil && dberr != redis.Nil {
			// Error during interacting with DB
			logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.SScan() in user.SearchUser")
			return []entity.User{}, uint64(0), errors.InternalServerError("")
		}
		newCursor = freshCursor
		addIntoResultSet(freshList)
	}

	searchResult := []entity.User{}
	for username := range resultSet {
		userData, err := r.GetUser(ctx, logger, username)
		if err != nil {
			// Issues in GetUser()
			return searchResult, uint64(0), err
		}
		// Hide password
		userData.Password = ""
		searchResult = append(searchResult, userData)
	}
	return searchResult, newCursor, nil
}

// Adds a newly created user gang content streaming token to DB.
func (r repository) AddStreamingToken(ctx context.Context, logger log.Logger, username, token string) {
	dberr := r.db.Client().Set(ctx, "stream_token:"+username, token, time.Hour*3).Err()
	if dberr != nil {
		// Error during interacting with DB
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.Set() in user.AddStreamingToken")
	}
}

// Get user streaming token if available.
func (r repository) GetStreamingToken(ctx context.Context, logger log.Logger, username string) string {
	token, dberr := r.db.Client().Get(ctx, "stream_token:"+username).Result()
	if dberr != nil {
		if dberr != redis.Nil {
			// Error during interacting with DB
			logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.Get() in user.GetStreamingToken")
		}
		return token
	}
	return token
}

// Delete user streaming token.
func (r repository) DelStreamingToken(ctx context.Context, logger log.Logger, username string) {
	_, dberr := r.db.Client().Del(ctx, "stream_token:"+username).Result()
	if dberr != nil {
		if dberr != redis.Nil {
			// Error during interacting with DB
			logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.Del() in user.DelStreamingToken")
		}
	}
}
