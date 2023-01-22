// Gang repository encapsulates the data access logic (interactions with the DB) related to Gang CRUD in Popcorn.

package gang

import (
	"Popcorn/internal/entity"
	"Popcorn/internal/errors"
	"Popcorn/pkg/db"
	"Popcorn/pkg/log"
	"context"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/xeonx/timeago"
)

type Repository interface {
	// HasGang returns a boolean depending on gang's availability.
	HasGang(ctx context.Context, logger log.Logger, admin string) (bool, error)
	// SetGang adds the gang data into the DB.
	SetGang(ctx context.Context, logger log.Logger, gang *entity.Gang, userExistCheck bool) (bool, error)
	// SetGangMembers adds the gang member into a gang.
	SetGangMembers(ctx context.Context, logger log.Logger, gangMemberKey string, member string) (bool, error)
	// GetGang fetches created gang data from DB.
	GetGang(ctx context.Context, logger log.Logger, admin string, username string) (map[string]any, error)
	// GetJoinedGang fetches joined gang data from DB.
	GetJoinedGang(ctx context.Context, logger log.Logger, username string) (map[string]any, error)
}

// repository struct of gang Repository.
// Object of this will be passed around from main to internal.
// Helps to access the repository layer interface and call methods.
type repository struct {
	db *db.RedisDB
}

// Returns a new instance of gang repository for other packages to access its interface.
func NewRepository(dbwrp *db.RedisDB) Repository {
	return repository{db: dbwrp}
}

// Returns true if gang:<gang_admin> exists in Popcorn.
func (r repository) HasGang(ctx context.Context, logger log.Logger, admin string) (bool, error) {
	available, dberr := r.db.Client().Exists(ctx, "gang:"+admin).Result()
	if dberr != nil && dberr != redis.Nil {
		// Error during interacting with DB
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.Exists() in gang.Exists")
		return false, errors.InternalServerError("")
	} else if available == 0 {
		// Gang not available
		return false, nil
	}
	return true, nil
}

// Returns true if gang got successfully added into the DB.
func (r repository) SetGang(ctx context.Context, logger log.Logger, gang *entity.Gang, gangExistCheck bool) (bool, error) {
	if !gangExistCheck {
		// Checking if an gang with admin gang.Admin exists in the DB
		available, dberr := r.HasGang(ctx, logger, gang.Admin)
		if dberr != nil {
			// Issues in Exists()
			return false, dberr
		} else if available {
			return false, errors.BadRequest("Gang already exists")
		}
	}
	key := "gang:" + gang.Admin
	if _, dberr := r.db.Client().Pipelined(ctx, func(client redis.Pipeliner) error {
		client.HSet(ctx, key, "gang_admin", gang.Admin)
		client.HSet(ctx, key, "gang_name", gang.Name)
		client.HSet(ctx, key, "gang_pass_key", gang.PassKey)
		client.HSet(ctx, key, "gang_member_limit", gang.Limit)
		client.HSet(ctx, key, "gang_members_key", gang.MembersListKey)
		client.HSet(ctx, key, "gang_created", gang.Created)
		return nil
	}); dberr != nil {
		// Error during interacting with DB
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.Pipelined() in gang.SetGang")
		return false, errors.InternalServerError("")
	}
	return true, nil
}

// Returns true if gang member got successfully added into the DB.
func (r repository) SetGangMembers(ctx context.Context, logger log.Logger, gangMemberKey string, member string) (bool, error) {
	_, dberr := r.db.Client().SAdd(ctx, gangMemberKey, member).Result()
	if dberr != nil {
		// Issue in SAdd()
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.SAdd() in gang.SetGangMembers")
		return false, errors.InternalServerError("")
	}
	return true, nil
}

// Returns gang data if user has created a gang.
func (r repository) GetGang(ctx context.Context, logger log.Logger, gangKey string, username string) (map[string]any, error) {
	response := make(map[string]any)
	var gang entity.Gang

	if dberr := r.db.Client().HGetAll(ctx, gangKey).Scan(&gang); dberr != nil {
		// Error during interacting with DB
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.HGetAll() in gang.GetGang")
		return response, errors.InternalServerError("")
	}
	joined_count, dberr := r.db.Client().SCard(ctx, gang.MembersListKey).Result()
	if dberr != nil {
		// Error during interacting with DB
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.SCard() in gang.GetGang")
		return response, errors.InternalServerError("")
	}

	if len(gang.Name) != 0 {
		// Delete gang passkey hash
		gang.PassKey = ""
		// use timeago on gang_created
		response["gang_created_timeago"] = timeago.English.Format(time.Unix(gang.Created, 0))
		response["gang"] = gang
		response["members"] = joined_count
		response["isAdmin"] = username == gang.Admin
	}

	return response, nil
}

// Returns gang data if user has joined a gang.
func (r repository) GetJoinedGang(ctx context.Context, logger log.Logger, username string) (map[string]any, error) {
	response := make(map[string]any)

	gangKey, dberr := r.db.Client().Get(ctx, "gang-joined:"+username).Result()
	if dberr != nil && dberr != redis.Nil {
		// Error during interacting with DB
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.Get() in gang.GetJoinedGang")
		return response, errors.InternalServerError("")
	} else if len(gangKey) == 0 {
		// User has not joined any gang
		return response, nil
	}

	return r.GetGang(ctx, logger, gangKey, username)
}
