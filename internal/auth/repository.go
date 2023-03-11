// Auth repository encapsulates the data access logic (interactions with the DB) related to Authentication in Popcorn.

package auth

import (
	"Popcorn/internal/errors"
	"Popcorn/pkg/db"
	"Popcorn/pkg/log"
	"context"
	"time"

	"github.com/go-redis/redis/v8"
)

type Repository interface {
	// SetToken adds the user's AccessTokenUUID:userID and RefreshTokenUUID:userID in the DB.
	SetToken(ctx context.Context, logger log.Logger, jwtData *JWTdata) error
	// HasToken checks whether token:username exists in the DB.
	HasToken(ctx context.Context, logger log.Logger, tokenUUID string, username string) (bool, error)
	// DelToken deletes TokenUUID from DB (if exists).
	DelToken(ctx context.Context, logger log.Logger, tokenUUID string) error
}

// repository struct of auth Repository.
// Object of this will be passed around from main to internal.
// Helps to access the repository layer interface and call methods.
type repository struct {
	db *db.RedisDB
}

// Returns a new instance of auth repository for other packages to access its interface.
func NewRepository(dbwrp *db.RedisDB) Repository {
	return repository{db: dbwrp}
}

// Returns nil if Token got successfully added into the DB else error.
func (r repository) SetToken(ctx context.Context, logger log.Logger, jwtData *JWTdata) error {
	now := time.Now()
	// Set AccessTokenUUID:userID in the DB
	accTokenExp := time.Unix(jwtData.AccTokenExp, 0)
	_, dberr := r.db.Client().Set(ctx, jwtData.AccessTokenUUID, jwtData.Username, accTokenExp.Sub(now)).Result()
	if dberr != nil {
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.Set in auth.SetToken")
		return errors.InternalServerError("")
	}
	// Set RefreshTokenUUID:userID in the DB
	refTokenExp := time.Unix(jwtData.RefTokenExp, 0)
	_, dberr = r.db.Client().Set(ctx, jwtData.RefTokenUUID, jwtData.Username, refTokenExp.Sub(now)).Result()
	if dberr != nil {
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.Set in auth.SetToken")
		return errors.InternalServerError("")
	}
	return nil
}

// Returns boolean if tokenUUID:Username exists in the DB or not.
// tokenUUID can be both AccessToken or RefreshToken.
func (r repository) HasToken(ctx context.Context, logger log.Logger, tokenUUID string, username string) (bool, error) {
	val, dberr := r.db.Client().Get(ctx, tokenUUID).Result()
	if dberr != nil && dberr != redis.Nil {
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.Get in auth.TokenExists")
		return false, errors.InternalServerError("")
	} else if dberr == redis.Nil {
		// Key doesn't exist, maybe got expired
		return false, nil
	}
	return username == val, nil
}

// Returns nil if tokenUUID was deleted from DB (if found) else error.
func (r repository) DelToken(ctx context.Context, logger log.Logger, tokenUUID string) error {
	deleted, dberr := r.db.Client().Del(ctx, tokenUUID).Result()
	if dberr != nil {
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.Del in auth.DelToken")
		return errors.InternalServerError("")
	} else if deleted == 0 {
		// tokenUUID didn't even exist in the DB
		return errors.NotFound("")
	}
	return nil
}
