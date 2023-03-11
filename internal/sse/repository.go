// sse repository encapsulates the data access logic (interactions with the DB) related to sse clients in Popcorn.

package sse

import (
	"Popcorn/internal/errors"
	"Popcorn/pkg/db"
	"Popcorn/pkg/log"
	"context"
)

type Repository interface {
	// AddClient adds incoming (SSE) client username to DB.
	// Helpful to initialize or scale if the server has to reload or a new instance has to be created.
	AddClient(ctx context.Context, logger log.Logger, username string) error
	// RemoveClient removes disconnected SSE client username from DB.
	RemoveClient(ctx context.Context, logger log.Logger, username string) error
}

// repository struct of sse Repository.
// Object of this will be passed around from main to internal.
// Helps to access the repository layer interface and call methods.
type repository struct {
	db *db.RedisDB
}

// Returns a new instance of sse repository for other packages to access its interface.
func NewRepository(dbwrp *db.RedisDB) Repository {
	return repository{db: dbwrp}
}

// Returns nil if client with username got successfully added into the DB.
func (r repository) AddClient(ctx context.Context, logger log.Logger, username string) error {
	dberr := r.db.Client().SAdd(ctx, "sse_clients", username).Err()
	if dberr != nil {
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of SAdd in sse.AddClient")
		return errors.InternalServerError("")
	}
	return dberr
}

// Returns nil if client with username got successfully removed into the DB.
func (r repository) RemoveClient(ctx context.Context, logger log.Logger, username string) error {
	dberr := r.db.Client().SRem(ctx, "sse_clients", username).Err()
	if dberr != nil {
		logger.WithCtx(ctx).Error().Err(dberr).Msg("Error occured during execution of SRem in sse.RemoveClient")
		return errors.InternalServerError("")
	}
	return dberr
}
