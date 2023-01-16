// Service layer of the internal package user.

package user

import (
	"Popcorn/internal/entity"
	"Popcorn/internal/errors"
	"Popcorn/pkg/log"
	"context"
)

// Service layer of internal package user which encapsulates UserModel logic of Popcorn.
type Service interface {
	// Fetches User Data based on User-ID
	getuser(context.Context) (entity.User, error)
}

// Object of this will be passed around from main to routers to API.
// Helps to access the service layer interface and call methods.
// Also helps to pass objects to be used from outer layer.
type service struct {
	userRepo Repository
	logger   log.Logger
}

func NewService(userRepo Repository, logger log.Logger) Service {
	return service{userRepo, logger}
}

func (s service) getuser(ctx context.Context) (entity.User, error) {
	// get userID from context
	username := ctx.Value("Username")
	if username == nil {
		// userID missing from context
		return entity.User{}, errors.InternalServerError("")
	}
	user, dberr := s.userRepo.Get(ctx, s.logger, username.(string))
	if dberr != nil {
		// Error occured in Get()
		return entity.User{}, dberr
	}
	return user, nil
}
