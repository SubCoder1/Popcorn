// Auth repository encapsulates the data access logic (interactions with the DB) related to Authentication in Popcorn.

package auth

import (
	"Popcorn/internal/errors"
	"Popcorn/pkg/db"
	"Popcorn/pkg/log"
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/go-redis/redis/v8"
)

type Repository interface {
	// SetToken adds the user's AccessTokenUUID:userID and RefreshTokenUUID:userID in the DB.
	SetToken(context.Context, log.Logger, *JWTdata) error
	// TokenExists checks whether token:userID exists in the DB.
	TokenExists(context.Context, log.Logger, string, uint64) (bool, error)
}

// repository struct of user Repository.
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
	_, dberr := r.db.Client().Set(ctx, jwtData.AccessTokenUUID, jwtData.UserID, accTokenExp.Sub(now)).Result()
	if dberr != nil {
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.Set in auth.SetToken")
		return errors.InternalServerError("")
	}
	// Set RefreshTokenUUID:userID in the DB
	refTokenExp := time.Unix(jwtData.RefTokenExp, 0)
	_, dberr = r.db.Client().Set(ctx, jwtData.RefTokenUUID, jwtData.UserID, refTokenExp.Sub(now)).Result()
	if dberr != nil {
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.Set in auth.SetToken")
		return errors.InternalServerError("")
	}
	return nil
}

// Returns boolean if tokenUUID:UserID exists in the DB or not.
// tokenUUID can be both AccessToken or RefreshToken.
func (r repository) TokenExists(ctx context.Context, logger log.Logger, tokenUUID string, UserID uint64) (bool, error) {
	val, dberr := r.db.Client().Get(ctx, tokenUUID).Result()
	if dberr != nil && dberr != redis.Nil {
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of redis.Get in auth.TokenExists")
		return false, errors.InternalServerError("")
	} else if dberr == redis.Nil {
		fmt.Println(tokenUUID)
		// Key doesn't exist, maybe got expired
		return false, nil
	}
	// Parse the UserID stored as a string in val to uint64 before comparing
	id, prserr := strconv.Atoi(val)
	if prserr != nil {
		// Parsing error
		logger.WithCtx(ctx).Error().Err(prserr).Msg("Parsing error in auth.TokenExists")
		return false, errors.InternalServerError("")
	}
	return uint64(id) == UserID, nil
}
