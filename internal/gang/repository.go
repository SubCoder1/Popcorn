// Gang repository encapsulates the data access logic (interactions with the DB) related to Gang CRUD in Popcorn.

package gang

import (
	"Popcorn/internal/entity"
	"Popcorn/internal/errors"
	"Popcorn/pkg/db"
	"Popcorn/pkg/log"
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/go-redis/redis/v8"
)

type Repository interface {
	// HasGang returns a boolean depending on gang's availability.
	HasGang(ctx context.Context, logger log.Logger, gangKey string, gangName string) (bool, error)
	// SetOrUpdateGang adds or updates the gang data into the DB.
	SetOrUpdateGang(ctx context.Context, logger log.Logger, gang *entity.Gang, update bool) (bool, error)
	// DelGang deletes the gang data from the DB.
	DelGang(ctx context.Context, logger log.Logger, admin string) error
	// GetGang fetches created gang data from DB.
	GetGang(ctx context.Context, logger log.Logger, gangKey string, username string, existCheck bool) (entity.GangResponse, error)
	// GetGangPassKey fetches PassKey Hash for a gang, should be used before JoinGang.
	GetGangPassKey(ctx context.Context, logger log.Logger, gangKey entity.GangJoin) (string, error)
	// GetJoinedGang fetches joined gang data from DB.
	GetJoinedGang(ctx context.Context, logger log.Logger, username string) (entity.GangResponse, error)
	// GetGangMembers fetches joined gang members list from DB.
	GetGangMembers(ctx context.Context, logger log.Logger, username string) ([]string, error)
	// GetGangInvites returns a list of invites received by user in Popcorn.
	GetGangInvites(ctx context.Context, logger log.Logger, username string) ([]entity.GangInvite, error)
	// DelGangInvite deletes rejected or expired gang invites.
	DelGangInvite(ctx context.Context, logger log.Logger, invite entity.GangInvite) error
	// JoinGang adds user to a gang.
	JoinGang(ctx context.Context, logger log.Logger, gangKey entity.GangJoin, username string) error
	// LeaveGang removes an user from a gang.
	LeaveGang(ctx context.Context, logger log.Logger, boot entity.GangExit) error
	// SearchGang returns paginated gang data depending on the query.
	SearchGang(ctx context.Context, logger log.Logger, query entity.GangSearch, username string) ([]entity.GangResponse, uint64, error)
	// SendGangInvite adds the invite request metadata to respective receiver's gang-invites stack.
	SendGangInvite(ctx context.Context, logger log.Logger, invite entity.GangInvite) error
	// AcceptGangInvite accepts the invite request and joins the requested gang.
	AcceptGangInvite(ctx context.Context, logger log.Logger, invite entity.GangInvite) error
	// UpdateGangContentData updates content filename and ID from gang data.
	UpdateGangContentData(ctx context.Context, logger log.Logger, admin, cname, cID, cURL string, screen_share, streaming bool) error
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
func (r repository) HasGang(ctx context.Context, logger log.Logger, gangKey string, gangName string) (bool, error) {
	available, dberr := r.db.Client().Exists(ctx, gangKey).Result()
	if dberr != nil && dberr != redis.Nil {
		// Error during interacting with DB
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.Exists() in gang.HasGang")
		return false, errors.InternalServerError("")
	} else if available == 0 {
		// Gang not available
		return false, nil
	}
	if gangName != "" {
		// Useful in joinGang or accepting gangInvite
		name, dberr := r.db.Client().HGet(ctx, gangKey, "gang_name").Result()
		if dberr != nil && dberr != redis.Nil {
			// Error during interacting with DB
			logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.HGet() in gang.HasGang")
			return false, errors.InternalServerError("")
		} else if dberr == redis.Nil || name != gangName {
			return false, nil
		}
	}
	return true, nil
}

// Returns true if gang got successfully added into the DB.
func (r repository) SetOrUpdateGang(ctx context.Context, logger log.Logger, gang *entity.Gang, update bool) (bool, error) {
	// Checking if an gang with admin gang.Admin exists in the DB
	available, dberr := r.HasGang(ctx, logger, "gang:"+gang.Admin, "")
	if dberr != nil {
		// Issues in Exists()
		return false, dberr
	} else if available && !update {
		return false, errors.BadRequest("Gang already exists")
	}
	gangKey := "gang:" + gang.Admin
	txferr := func(key string) error {
		txf := func(tx *redis.Tx) error {
			// Operation is commited only if the watched keys remain unchanged
			_, dberr := r.db.Client().TxPipelined(ctx, func(client redis.Pipeliner) error {
				client.HSet(ctx, gangKey, "gang_name", gang.Name)
				if !update || (update && gang.PassKey != "PREVIOUSPASSKEY") {
					// Only used during createGang or when passKey is being updated
					client.HSet(ctx, gangKey, "gang_pass_key", gang.PassKey)
				}
				client.HSet(ctx, gangKey, "gang_member_limit", gang.Limit)
				client.HSet(ctx, gangKey, "gang_content_url", gang.ContentURL)
				client.HSet(ctx, gangKey, "gang_screen_share", gang.ContentScreenShare)
				client.HSet(ctx, gangKey, "gang_invite_hashcode", gang.InviteHashCode)
				if !update {
					// Only set during creating gang, some of these can be changed by server
					client.HSet(ctx, gangKey, "gang_admin", gang.Admin)
					client.HSet(ctx, gangKey, "gang_members_key", gang.MembersListKey)
					client.HSet(ctx, gangKey, "gang_created", gang.Created)
					client.HSet(ctx, gangKey, "gang_streaming", false)
					client.HSet(ctx, gangKey, "gang_content_name", "")
					client.HSet(ctx, gangKey, "gang_content_ID", "")
					client.HSet(ctx, gangKey, "gang_content_url", "")
					client.HSet(ctx, gangKey, "gang_screen_share", false)
				} else if len(gang.ContentID) != 0 && len(gang.ContentName) != 0 {
					// These values are only updated through server
					client.HSet(ctx, gangKey, "gang_content_name", gang.ContentName)
					client.HSet(ctx, gangKey, "gang_content_ID", gang.ContentID)
				}
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
	}(gangKey)
	if txferr != nil {
		logger.WithCtx(ctx).Error().Err(txferr).Msg("Error occured in SetUser transaction")
		return false, errors.InternalServerError("")
	}
	if !update {
		// Set gang:index -> gang:<gang.Admin>:<gang.Name> as index for quicker search
		gangIndex := fmt.Sprintf("gang:%s:%s", gang.Admin, strings.ToLower(gang.Name))
		_, dberr = r.db.Client().SAdd(ctx, "gang:index", gangIndex).Result()
		if dberr != nil {
			// Issues in SAdd()
			logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during setting gang index")
			return false, errors.InternalServerError("")
		}
		// Set gang-members:<member>
		err := r.SetGangMembers(ctx, logger, gang.MembersListKey, gang.Admin)
		if err != nil {
			// Issues in SetGangMembers
			return false, err
		}
	}
	return true, nil
}

// Returns nil if gang got successfully deleted from the DB.
func (r repository) DelGang(ctx context.Context, logger log.Logger, admin string) error {
	// Required gang metadata needed to delete related gang data from the DB
	gangData := struct {
		Key            string
		Name           string `json:"gang_name" redis:"gang_name,omitempty"`
		MembersListKey string `json:"gang_members_key,omitempty" redis:"gang_members_key"`
	}{Key: "gang:" + admin}
	dberr := r.db.Client().HGetAll(ctx, gangData.Key).Scan(&gangData)
	if dberr != nil && dberr != redis.Nil {
		// Issues in HGet()
		return dberr
	} else if dberr == redis.Nil {
		// Maybe expired or doesn't exist at all
		return errors.BadRequest("Gang doesn't exist")
	}
	// We need to delete every gang-joined:<member> of this gang's current members
	members, dberr := r.GetGangMembers(ctx, logger, admin)
	if dberr != nil {
		// Issues in GetGangMembers()
		return dberr
	}
	for _, mem := range members {
		dberr = r.db.Client().Del(ctx, "gang-joined:"+mem).Err()
		if dberr != nil {
			logger.WithCtx(ctx).Error().Err(dberr).Msg("Error during deleting joined member's keys in DelGang()")
			return errors.InternalServerError("")
		}
	}
	// Delete gang members from DB
	dberr = r.db.Client().Del(ctx, gangData.MembersListKey).Err()
	if dberr != nil && dberr != redis.Nil {
		// Issues in Del()
		return dberr
	}
	// Delete gang data from DB
	dberr = r.db.Client().Del(ctx, gangData.Key).Err()
	if dberr != nil && dberr != redis.Nil {
		// Issues in Del()
		return dberr
	}
	// Delete gang index from DB
	r.delGangIndex(ctx, logger, fmt.Sprintf("gang:%s:%s", admin, strings.ToLower(gangData.Name)))
	return nil
}

// Returns nil if gang member got successfully added into the DB.
func (r repository) SetGangMembers(ctx context.Context, logger log.Logger, gangMemberKey string, member string) error {
	_, dberr := r.db.Client().SAdd(ctx, gangMemberKey, member).Result()
	if dberr != nil {
		// Issues in SAdd()
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.SAdd() in gang.SetGangMembers")
		return errors.InternalServerError("")
	}
	return nil
}

// Returns nil if gang member got successfully removed from the gang.
func (r repository) DelGangMember(ctx context.Context, logger log.Logger, gangMemberKey string, member string) error {
	_, dberr := r.db.Client().SRem(ctx, gangMemberKey, member).Result()
	if dberr != nil {
		// Issues in SAdd()
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.SAdd() in gang.SetGangMembers")
		return errors.InternalServerError("")
	}
	return nil
}

// Returns gang data if user has created a gang.
func (r repository) GetGang(ctx context.Context, logger log.Logger, gangKey string, username string, existCheck bool) (entity.GangResponse, error) {
	if !existCheck {
		// Checking if gangKey exists in the DB
		available, dberr := r.HasGang(ctx, logger, gangKey, "")
		if dberr != nil {
			// Issues in HasGang()
			return entity.GangResponse{}, dberr
		} else if !available {
			return entity.GangResponse{}, nil
		}
	}

	var gangResp entity.GangResponse
	if dberr := r.db.Client().HGetAll(ctx, gangKey).Scan(&gangResp); dberr != nil {
		// Error during interacting with DB
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.HGetAll() in gang.GetGang")
		return entity.GangResponse{}, errors.InternalServerError("")
	}
	joined_count, dberr := r.db.Client().SCard(ctx, "gang-members:"+gangResp.Admin).Result()
	if dberr != nil {
		// Error during interacting with DB
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.SCard() in gang.GetGang")
		return entity.GangResponse{}, errors.InternalServerError("")
	}

	if len(gangResp.Name) != 0 {
		// use timeago on gang_created
		gangResp.Count = int(joined_count)
		gangResp.IsAdmin = username == gangResp.Admin
	}

	return gangResp, nil
}

// Returns gang passkey, used to validate incoming passkey before JoinGang is called
func (r repository) GetGangPassKey(ctx context.Context, logger log.Logger, gangKey entity.GangJoin) (string, error) {
	// Checking if an gang with gangKey with the same gang name exists in the DB
	available, dberr := r.HasGang(ctx, logger, gangKey.Key, gangKey.Name)
	if dberr != nil {
		// Issues in HasGang()
		return "", dberr
	} else if !available {
		// Delete index as this request was made through search or invite
		idx := gangKey.Key + ":" + strings.ToLower(gangKey.Name)
		r.delGangIndex(ctx, logger, idx)
		return "", errors.BadRequest("Gang doesn't exist")
	}
	// Fetch Gang PassKey hash
	passKey, dberr := r.db.Client().HGet(ctx, gangKey.Key, "gang_pass_key").Result()
	if dberr != nil {
		// Error during interacting with DB
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.HGet() in gang.GetGangPassKey")
		return "", errors.InternalServerError("")
	}
	return passKey, nil
}

// Returns a list of GangInvite objects consisting invite metadata.
func (r repository) GetGangInvites(ctx context.Context, logger log.Logger, username string) ([]entity.GangInvite, error) {
	inviteKeys, dberr := r.db.Client().ZRevRange(ctx, "gang-invites:"+username, 0, -1).Result()
	if dberr != nil && dberr != redis.Nil {
		// Error during interacting with DB
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.SMembers() in gang.GetGangInvites")
		return []entity.GangInvite{}, errors.InternalServerError("")
	}
	invites := []entity.GangInvite{}
	for _, inviteKey := range inviteKeys {
		// invite is of format <GangInvite.Admin>:<GangInvite.GangName>:<Created_UNIX_Timestamp>
		gangInvite, err := extDataFromInviteIndex(ctx, logger, inviteKey)
		if err != nil {
			// Issues in extractGangInviteData()
			return []entity.GangInvite{}, err
		}
		invites = append(invites, gangInvite)
	}
	return invites, nil
}

// Returns gang data if user has joined a gang.
func (r repository) GetJoinedGang(ctx context.Context, logger log.Logger, username string) (entity.GangResponse, error) {
	gangKey, dberr := r.db.Client().Get(ctx, "gang-joined:"+username).Result()
	if dberr != nil && dberr != redis.Nil {
		// Error during interacting with DB
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.Get() in gang.GetJoinedGang")
		return entity.GangResponse{}, errors.InternalServerError("")
	} else if len(gangKey) == 0 {
		// User has not joined any gang
		return entity.GangResponse{}, nil
	}

	return r.GetGang(ctx, logger, gangKey, username, true)
}

// Returns a list of joined gang members.
func (r repository) GetGangMembers(ctx context.Context, logger log.Logger, admin string) ([]string, error) {
	membersList, dberr := r.db.Client().SMembers(ctx, "gang-members:"+admin).Result()
	if dberr != nil && dberr != redis.Nil {
		// Error during interacting with DB
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.SMembers() in gang.GetGangMembers")
		return []string{}, errors.InternalServerError("")
	} else if dberr == redis.Nil {
		// GangMembersList missing
		return []string{}, errors.BadRequest("")
	}
	return membersList, nil
}

// Leaves the current joined gang.
func (r repository) LeaveGang(ctx context.Context, logger log.Logger, boot entity.GangExit) error {
	// Checking if a gang with gangKey and same gangName exists in the DB
	available, dberr := r.HasGang(ctx, logger, boot.Key, boot.Name)
	if dberr != nil {
		// Issues in HasGang()
		return dberr
	} else if !available {
		// Delete index as this request was made through search or invite
		idx := boot.Key + ":" + strings.ToLower(boot.Name)
		r.delGangIndex(ctx, logger, idx)
		return errors.BadRequest("Gang doesn't exist")
	}

	// If boot.Type == leave, get the gang key in which member (username) is currently at
	// Then, remove member from gangKey:gangMemberKey list and delete gang-joined:member key:value from DB
	joinedKey := "gang-joined:" + boot.Member
	memInGang, dberr := r.db.Client().Get(ctx, joinedKey).Result()
	if dberr != nil && dberr != redis.Nil {
		// Error during interacting with DB
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.Get() in gang.LeaveGang")
		return errors.InternalServerError("")
	} else if dberr == redis.Nil {
		// memInGang is empty
		if boot.Type == "boot" {
			return errors.BadRequest("Invalid Boot Request")
		}
		return nil
	}
	// If boot.Type == boot, we have to make sure the member user is trying to kick out of his/her own gang actually is in his/her gang
	if boot.Type != "leave" {
		if memInGang != boot.Key {
			// This member is not in the gang by the key boot.Key
			return errors.BadRequest("Member cannot be kicked")
		}
	} else {
		boot.Key = memInGang
	}

	// Delete gang-joined:<boot.Member> as member is leaving or is forced out of the gang
	_, dberr = r.db.Client().Del(ctx, joinedKey).Result()
	if dberr != nil && dberr != redis.Nil {
		// Error during interacting with DB
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.Del() in gang.LeaveGang")
		return errors.InternalServerError("")
	}
	// Fetch gangMemberKey
	gangMemberKey, dberr := r.db.Client().HGet(ctx, boot.Key, "gang_members_key").Result()
	if dberr != nil && dberr != redis.Nil {
		// Error during interacting with DB
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.HGet() in gang.LeaveGang")
		return errors.InternalServerError("")
	} else if dberr == redis.Nil || gangMemberKey == "" {
		// GangMembers doesn't exist
		idx := boot.Key + ":" + strings.ToLower(boot.Name)
		go r.delGangIndex(ctx, logger, idx)
		return nil
	}
	dberr = r.DelGangMember(ctx, logger, gangMemberKey, boot.Member)
	if dberr != nil {
		// Issue in DelGangMember()
		return dberr
	}
	return nil
}

// Returns nil if user got successfully added to the gang.
func (r repository) JoinGang(ctx context.Context, logger log.Logger, join entity.GangJoin, username string) error {
	// Check if gang can take a member in by checking if current gang members count + 1 < members_limit
	gangLimitStr, dberr := r.db.Client().HGet(ctx, join.Key, "gang_member_limit").Result()
	idx := join.Key + ":" + strings.ToLower(join.Name)
	if dberr != nil {
		if dberr == redis.Nil || len(gangLimitStr) == 0 {
			go r.delGangIndex(ctx, logger, idx)
			return errors.BadRequest("Gang doesn't exist")
		}
		// Error during interacting with DB
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.HGet() in gang.JoinGang")
		return errors.InternalServerError("")
	}
	gangLimit, prserr := strconv.Atoi(gangLimitStr)
	if prserr != nil {
		// Parse error
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Parse error during parsing gangLimit in gang.JoinGang")
		return errors.InternalServerError("")
	}
	// Fetch memberListKey from gang
	gangMemberKey, dberr := r.db.Client().HGet(ctx, join.Key, "gang_members_key").Result()
	if dberr != nil {
		if dberr == redis.Nil || len(gangMemberKey) == 0 {
			go r.delGangIndex(ctx, logger, idx)
			return errors.BadRequest("Gang doesn't exist")
		}
		// Error during interacting with DB
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.HGet() in gang.JoinGang")
		return errors.InternalServerError("")
	}
	currMembersCount, dberr := r.db.Client().SCard(ctx, gangMemberKey).Result()
	if dberr != nil {
		if dberr == redis.Nil || currMembersCount == 0 {
			go r.delGangIndex(ctx, logger, idx)
			return errors.BadRequest("Gang doesn't exist")
		}
		// Error interacting with DB
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.SCard() in gang.JoinGang")
		return errors.InternalServerError("")
	} else if int(currMembersCount)+1 > gangLimit {
		// Limit will exceed on adding this new member into the gang
		return errors.BadRequest("Gang Limit Exceeded")
	}

	// Remove user from existing joined gang (if any)
	boot := entity.GangExit{
		Member: username,
		Name:   join.Name,
		Key:    "gang:" + join.Admin,
		Type:   "leave",
	}
	dberr = r.LeaveGang(ctx, logger, boot)
	if dberr != nil {
		// Issues in LeaveGang
		return dberr
	}

	// Add gang-joined:<username> to join.Key
	// Set gang-joined:<member> to gang:<gang_admin>
	_, dberr = r.db.Client().Set(ctx, "gang-joined:"+username, join.Key, 0).Result()
	if dberr != nil {
		// Isses in Set()
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.Set() in gang.JoinGang")
		return errors.InternalServerError("")
	}

	// Add user with username into the joined gang's GangMembersList
	err := r.SetGangMembers(ctx, logger, gangMemberKey, username)
	if err != nil {
		// Issues in SetGangMembers()
		return err
	}

	return nil
}

// Returns paginated gang details of all the gangs matched by query (gang_name) in DB.
func (r repository) SearchGang(ctx context.Context, logger log.Logger, gs entity.GangSearch, username string) ([]entity.GangResponse, uint64, error) {
	// try searching gang index gang:*:query:index, assuming query as gang name
	searchBy := fmt.Sprintf("gang:*:%s*", strings.ToLower(gs.Name))
	initialResult, newCursor, dberr := r.db.Client().SScan(ctx, "gang:index", uint64(gs.Cursor), searchBy, 10).Result()
	if dberr != nil && dberr != redis.Nil {
		// Error during interacting with DB
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.SScan() in gang.SearchGang")
		return []entity.GangResponse{}, uint64(0), errors.InternalServerError("")
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
		freshList, freshCursor, dberr := r.db.Client().SScan(ctx, "gang:index", newCursor, searchBy, 10).Result()
		if dberr != nil && dberr != redis.Nil {
			// Error during interacting with DB
			logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.SScan() in gang.SearchGang")
			return []entity.GangResponse{}, uint64(0), errors.InternalServerError("")
		}
		newCursor = freshCursor
		addIntoResultSet(freshList)
	}

	searchResult := []entity.GangResponse{}
	for gangIndex := range resultSet {
		gangKey, gangName, exterr := extDataFromGangIndex(ctx, logger, gangIndex)
		if exterr != nil {
			// Issues in extractGangKeyFromIndex()
			return searchResult, uint64(0), exterr
		}
		gang, dberr := r.GetGang(ctx, logger, gangKey, username, false)
		if dberr != nil {
			// Issues in GetGang()
			return searchResult, uint64(0), dberr
		} else if gang.Admin == "" {
			// Empty gang, must be expired
			// Remove from index and continue
			idx := gangKey + ":" + strings.ToLower(gangName)
			r.delGangIndex(ctx, logger, idx)
		}
		searchResult = append(searchResult, gang)
	}

	return searchResult, newCursor, nil
}

// Deletes gang invites, usually triggered by gang invite decline.
func (r repository) DelGangInvite(ctx context.Context, logger log.Logger, invite entity.GangInvite) error {
	query := invite.Admin + ":" + invite.Name + ":*"
	inviteKey := "gang-invites:" + invite.For
	existingInvites, _, dberr := r.db.Client().ZScan(ctx, inviteKey, 0, query, 100).Result()
	if dberr != nil && dberr != redis.Nil {
		// Error during interacting with DB
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.Sscan() in gang.SendGangInvite")
		return errors.InternalServerError("")
	}
	if len(existingInvites) == 0 {
		// 0 means Invite doesn't exist, maybe expired or invalid
		return errors.BadRequest("Expired or Invalid Gang Invite")
	}
	for _, extinvite := range existingInvites {
		_, dberr = r.db.Client().ZRem(ctx, "gang-invites:"+invite.For, extinvite).Result()
		if dberr != nil && dberr != redis.Nil {
			// Error during interacting with DB
			logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.SRem() in gang.DelGangInvite")
			return errors.InternalServerError("")
		}
	}
	return nil
}

// Adds incoming invite request to receiver's gang-invites: set in DB.
func (r repository) SendGangInvite(ctx context.Context, logger log.Logger, invite entity.GangInvite) error {
	// check if gang exists
	available, dberr := r.HasGang(ctx, logger, "gang:"+invite.Admin, invite.Name)
	if dberr != nil {
		// Error occured in HasGang()
		return dberr
	} else if !available {
		// Gang doesn't exist
		return errors.BadRequest("Invalid Gang Invite")
	}
	// Delete any duplicate invite from the same <invite.Admin>:<invite.Name>:*
	invitesKey := "gang-invites:" + invite.For
	dberr = r.DelGangInvite(ctx, logger, invite)
	if dberr != nil {
		// We can ignore existingInvites == 0 check during sendInvites
		err, ok := dberr.(errors.ErrorResponse)
		if !ok {
			return dberr
		} else if err.Status != 400 {
			// Issues in DelGangInvite()
			return dberr
		}
	}
	// gang-invites:<invite.For> -> <invite.Admin>:<invite.Name>:<Created_UNIX_Timestamp>
	score, dberr := r.db.Client().ZCard(ctx, invitesKey).Result()
	if dberr != nil {
		// Error during interacting with DB
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.SCard() in gang.SendGangInvite")
		return errors.InternalServerError("")
	}
	inviteIndex := fmt.Sprintf("%s:%s:%d", invite.Admin, invite.Name, invite.CreatedTimeAgo)
	_, dberr = r.db.Client().ZAdd(ctx, invitesKey, &redis.Z{Score: float64(score + 1), Member: inviteIndex}).Result()
	if dberr != nil {
		// Error during interacting with DB
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.SAdd() in gang.SendGangInvite")
		return errors.InternalServerError("")
	}
	return nil
}

// Accepts a gang invite request and joins the gang.
func (r repository) AcceptGangInvite(ctx context.Context, logger log.Logger, invite entity.GangInvite) error {
	if invite.InviteHashCode == "NOTREQUIRED" {
		// Delete the invite from user's gang-invites: set
		dberr := r.DelGangInvite(ctx, logger, invite)
		if dberr != nil {
			// Issues in DelGangIndex()
			return dberr
		}
	}
	// Validate if gang by the key gang:<invite.Admin> with <invite.Name> exists
	gangKey := "gang:" + invite.Admin
	gangExists, dberr := r.HasGang(ctx, logger, gangKey, invite.Name)
	if dberr != nil {
		// Issues in HasGang()
		return dberr
	} else if !gangExists {
		// Gang doesn't exist, invalid invite
		if invite.InviteHashCode == "NOTREQUIRED" {
			gangIndex := fmt.Sprintf("gang:%s:%s", invite.Admin, strings.ToLower(invite.Name))
			go r.delGangIndex(ctx, logger, gangIndex)
		}
		return errors.BadRequest("Expired or Invalid Gang Invite")
	}
	gangJoin := &entity.GangJoin{
		Admin:   invite.Admin,
		Name:    invite.Name,
		Key:     gangKey,
		PassKey: "joiningThroughInvite",
	}
	return r.JoinGang(ctx, logger, *gangJoin, invite.For)
}

// Updates gang content ID and filename from gang data.
func (r repository) UpdateGangContentData(ctx context.Context, logger log.Logger, admin, cname, cID, cURL string, screen_share, streaming bool) error {
	// Checking if an gang with admin exists in the DB
	available, dberr := r.HasGang(ctx, logger, "gang:"+admin, "")
	if dberr != nil {
		// Issues in Exists()
		return dberr
	} else if !available {
		return errors.BadRequest("Gang doesn't exist")
	}
	gangKey := "gang:" + admin
	txferr := func(key string) error {
		txf := func(tx *redis.Tx) error {
			// Operation is commited only if the watched keys remain unchanged
			_, dberr := r.db.Client().TxPipelined(ctx, func(client redis.Pipeliner) error {
				client.HSet(ctx, gangKey, "gang_content_name", cname)
				client.HSet(ctx, gangKey, "gang_content_ID", cID)
				client.HSet(ctx, gangKey, "gang_content_url", cURL)
				client.HSet(ctx, gangKey, "gang_screen_share", screen_share)
				client.HSet(ctx, gangKey, "gang_streaming", streaming)
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
	}(gangKey)
	if txferr != nil {
		logger.WithCtx(ctx).Error().Err(txferr).Msg("Error occured in EraseGangContentData transaction")
		return errors.InternalServerError("")
	}
	return nil
}

// Helper to delete expired gang index from DB.
func (r repository) delGangIndex(ctx context.Context, logger log.Logger, index string) error {
	_, dberr := r.db.Client().SRem(ctx, "gang:index", index).Result()
	if dberr != nil && dberr != redis.Nil {
		// Error during interacting with DB
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.SRem() in gang.delGangIndex")
		return errors.InternalServerError("")
	}
	return nil
}

// Helper to extract gangKey from gang index.
func extDataFromGangIndex(ctx context.Context, logger log.Logger, gangIndex string) (string, string, error) {
	slice := strings.Split(gangIndex, ":")
	if len(slice) != 3 {
		// Issues in index
		logger.WithCtx(ctx).Error().Msg("Error occured during extraction of gangKey from index, improper : used in index?")
		return "", "", errors.BadRequest("")
	}
	gangKey := slice[0] + ":" + slice[1]
	gangName := slice[2]
	return gangKey, gangName, nil
}

// Helper to extract GangInvite metadata from invite key.
func extDataFromInviteIndex(ctx context.Context, logger log.Logger, inviteKey string) (entity.GangInvite, error) {
	slice := strings.Split(inviteKey, ":")
	if len(slice) != 3 {
		// Issues in index
		logger.WithCtx(ctx).Error().Msg("Error occured during extraction of GangInvite data from inviteKey, improper : used in index?")
		return entity.GangInvite{}, errors.BadRequest("")
	}
	var invite entity.GangInvite
	invite.Admin = slice[0]
	invite.Name = slice[1]
	var prserr error
	invite.CreatedTimeAgo, prserr = strconv.ParseInt(slice[2], 10, 64)
	if prserr != nil {
		// Parsing error in strconv.Atoi()
		logger.WithCtx(ctx).Error().Msg("Error during conversion of created_unix from inviteKey in extractGangInviteData()")
		return entity.GangInvite{}, errors.InternalServerError("")
	}

	return invite, nil
}
