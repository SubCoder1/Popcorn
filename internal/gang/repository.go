// Gang repository encapsulates the data access logic (interactions with the DB) related to Gang CRUD in Popcorn.

package gang

import (
	"Popcorn/internal/entity"
	"Popcorn/internal/errors"
	"Popcorn/pkg/db"
	"Popcorn/pkg/log"
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/go-redis/redis/v8"
	"github.com/xeonx/timeago"
)

type Repository interface {
	// HasGang returns a boolean depending on gang's availability.
	HasGang(ctx context.Context, logger log.Logger, admin string) (bool, error)
	// SetGang adds the gang data into the DB.
	SetGang(ctx context.Context, logger log.Logger, gang *entity.Gang) (bool, error)
	// SetGangMembers adds the gang member into a gang.
	SetGangMembers(ctx context.Context, logger log.Logger, gangMemberKey string, member string) (bool, error)
	// GetGang fetches created gang data from DB.
	GetGang(ctx context.Context, logger log.Logger, admin string, username string, existCheck bool) (map[string]any, error)
	// GetJoinedGang fetches joined gang data from DB.
	GetJoinedGang(ctx context.Context, logger log.Logger, username string) (map[string]any, error)
	// JoinGang adds user to a gang.
	JoinGang(ctx context.Context, logger log.Logger, gangKey entity.GangKey, username string) error
	// SearchGang returns paginated gang data depending on the query.
	SearchGang(ctx context.Context, logger log.Logger, query entity.GangSearch, username string) ([]map[string]any, uint64, error)
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
func (r repository) HasGang(ctx context.Context, logger log.Logger, gangKey string) (bool, error) {
	available, dberr := r.db.Client().Exists(ctx, gangKey).Result()
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
func (r repository) SetGang(ctx context.Context, logger log.Logger, gang *entity.Gang) (bool, error) {
	// Checking if an gang with admin gang.Admin exists in the DB
	available, dberr := r.HasGang(ctx, logger, "gang:"+gang.Admin)
	if dberr != nil {
		// Issues in Exists()
		return false, dberr
	} else if available {
		return false, errors.BadRequest("Gang already exists")
	}
	gangKey := "gang:" + gang.Admin
	if _, dberr := r.db.Client().Pipelined(ctx, func(client redis.Pipeliner) error {
		client.HSet(ctx, gangKey, "gang_admin", gang.Admin)
		client.HSet(ctx, gangKey, "gang_name", gang.Name)
		client.HSet(ctx, gangKey, "gang_pass_key", gang.PassKey)
		client.HSet(ctx, gangKey, "gang_member_limit", gang.Limit)
		client.HSet(ctx, gangKey, "gang_members_key", gang.MembersListKey)
		client.HSet(ctx, gangKey, "gang_created", gang.Created)
		return nil
	}); dberr != nil {
		// Error during interacting with DB
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.Pipelined() in gang.SetGang")
		return false, errors.InternalServerError("")
	}
	// Set gang:index -> gang:<gang.Admin>:<gang.Name> as index for quicker search
	gangIndex := fmt.Sprintf("gang:%s:%s", gang.Admin, strings.ReplaceAll(gang.Name, " ", "-"))
	_, dberr = r.db.Client().SAdd(ctx, "gang:index", gangIndex).Result()
	if dberr != nil {
		// Issues in SAdd()
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during setting gang index")
		return false, errors.InternalServerError("")
	}
	// Set gang-members:<member>
	_, err := r.SetGangMembers(ctx, logger, gang.MembersListKey, gang.Admin)
	if err != nil {
		// Issues in SetGangMembers
		return false, err
	}
	return true, nil
}

// Returns true if gang member got successfully added into the DB.
func (r repository) SetGangMembers(ctx context.Context, logger log.Logger, gangMemberKey string, member string) (bool, error) {
	_, dberr := r.db.Client().SAdd(ctx, gangMemberKey, member).Result()
	if dberr != nil {
		// Issues in SAdd()
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.SAdd() in gang.SetGangMembers")
		return false, errors.InternalServerError("")
	}
	return true, nil
}

// Returns gang data if user has created a gang.
func (r repository) GetGang(ctx context.Context, logger log.Logger, gangKey string, username string, existCheck bool) (map[string]any, error) {
	response := make(map[string]any)
	var gang entity.Gang

	if !existCheck {
		// Checking if gangKey exists in the DB
		available, dberr := r.HasGang(ctx, logger, gangKey)
		if dberr != nil {
			// Issues in Exists()
			return response, dberr
		} else if !available {
			return response, nil
		}
	}

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
		response["is_admin"] = username == gang.Admin
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

	return r.GetGang(ctx, logger, gangKey, username, true)
}

func (r repository) CanJoinGang(ctx context.Context, logger log.Logger, username string) (bool, error) {
	// Check if gang-joined:<username> is found in the DB
	// Which means user is still in a gang, cannot join without leaving the one he/she's still in
	canJoin, dberr := r.db.Client().Exists(ctx, "gang-joined:"+username).Result()
	if dberr != nil && dberr != redis.Nil {
		// Error during interacting with DB
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.Exists() in gang.CanJoinGang")
		return false, errors.InternalServerError("")
	}
	return canJoin == 0, nil
}

// Returns nil if user got successfully added to the gang.
func (r repository) JoinGang(ctx context.Context, logger log.Logger, gangKey entity.GangKey, username string) error {
	// Checking if an gang with gangKey exists in the DB
	available, dberr := r.HasGang(ctx, logger, gangKey.Key)
	if dberr != nil {
		// Issues in HasGang()
		return dberr
	} else if !available {
		// Delete index as this request was made through search or invite
		r.delGangIndex(ctx, logger, gangKey.Key+":"+gangKey.Admin)
		return errors.BadRequest("Gang doesn't exist")
	}
	// Check if user can join a gang
	canJoin, dberr := r.CanJoinGang(ctx, logger, username)
	if dberr != nil {
		// Issues in CanJoinGang()
		return dberr
	} else if !canJoin {
		// User already in a gang, cannot join
		return errors.BadRequest("Already in a gang")
	}
	// Add gang-joined:<username> to gangKey
	// set gang-joined:<member> to gang:<gang_admin>
	_, dberr = r.db.Client().Set(ctx, "gang-joined:"+username, gangKey, 0).Result()
	if dberr != nil {
		// Isses in Set()
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.Set() in gang.SetGangMembers")
		return errors.InternalServerError("")
	}
	// Fetch memberListKey from gang
	gangMemberKey, dberr := r.db.Client().HGet(ctx, gangKey.Key, "gang_members_key").Result()
	if dberr != nil {
		// Error during interacting with DB
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.HGet() in gang.JoinGang")
		return errors.InternalServerError("")
	} else if dberr == redis.Nil {
		// No key found which matches gangMemberKey
		logger.WithCtx(ctx).Error().Err(dberr).Msg(fmt.Sprintf("gang_members_key not found in %s", gangKey))
		r.delGangIndex(ctx, logger, gangKey.Key+":"+gangKey.Admin)
		return errors.BadRequest("Gang doesn't exist")
	}
	// Add user with username into the GangMembersList
	_, err := r.SetGangMembers(ctx, logger, gangMemberKey, username)
	if err != nil {
		// Issues in SetGangMembers()
		return err
	}

	return nil
}

// Returns paginated gang details of all the gangs matched by query in DB.
// query can either be by admin or gangName
func (r repository) SearchGang(ctx context.Context, logger log.Logger, gs entity.GangSearch, username string) ([]map[string]any, uint64, error) {
	searchResult := []map[string]any{}
	newCursor := uint64(0)
	// try searching gang index gang:*:query:index, assuming query as gang name
	query := strings.ReplaceAll(gs.Name, " ", "-")
	searchBy := fmt.Sprintf("gang:*:%s*", query)
	resultSet, newCursor, dberr := r.db.Client().SScan(ctx, "gang:index", uint64(gs.Cursor), searchBy, 10).Result()

	if dberr != nil {
		// Error during interacting with DB
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.SScan() in gang.SearchGang")
		return searchResult, newCursor, errors.InternalServerError("")
	}
	for _, index := range resultSet {
		gangKey, exterr := extractGangKeyFromIndex(ctx, logger, index)
		if exterr != nil {
			// Issues in extractGangKeyFromIndex()
			return searchResult, newCursor, exterr
		}
		gang, dberr := r.GetGang(ctx, logger, gangKey, username, false)
		if dberr != nil {
			// Issues in getGangByKey()
			return searchResult, newCursor, dberr
		} else if len(gang) == 0 {
			// Empty gang, must be expired
			// Remove from index and continue
			r.delGangIndex(ctx, logger, gangKey+":"+query)
		}
		searchResult = append(searchResult, gang)
	}

	return searchResult, newCursor, nil
}

// Helper to delete expired gang index from DB.
func (r repository) delGangIndex(ctx context.Context, logger log.Logger, index string) error {
	_, dberr := r.db.Client().SRem(ctx, "gang:index", index).Result()
	if dberr != nil {
		// Error during interacting with DB
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.SRem() in gang.delGangIndex")
		return errors.InternalServerError("")
	}
	return nil
}

// Helper to extract gangKey from gang index
func extractGangKeyFromIndex(ctx context.Context, logger log.Logger, index string) (string, error) {
	slice := strings.Split(index, ":")
	if len(slice) > 3 {
		// Issues in index
		logger.WithCtx(ctx).Error().Msg("Error occured during extraction of gangKey from index, improper : used in index?")
		return "", errors.BadRequest("")
	}
	return slice[0] + ":" + slice[1], nil
}
