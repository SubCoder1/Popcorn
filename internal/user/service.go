// Service layer of the internal package user.

package user

import (
	"Popcorn/internal/entity"
	"Popcorn/internal/errors"
	"Popcorn/pkg/log"
	"context"

	"github.com/asaskevich/govalidator"
)

// Service layer of internal package user which encapsulates UserModel logic of Popcorn.
type Service interface {
	// Fetches User Data based on User-ID.
	getuser(ctx context.Context, username string) (entity.User, error)
	// Search for an user in Popcorn.
	searchuser(ctx context.Context, query entity.UserSearch) ([]entity.User, uint64, error)
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

func (s service) getuser(ctx context.Context, username string) (entity.User, error) {
	user, dberr := s.userRepo.GetUser(ctx, s.logger, username)
	if dberr != nil {
		// Error occured in Get()
		return entity.User{}, dberr
	}
	// Hide password
	user.Password = ""
	return user, nil
}

func (s service) searchuser(ctx context.Context, query entity.UserSearch) ([]entity.User, uint64, error) {
	// Validate the query data
	valerr := s.validateUserSearchData(ctx, query)
	if valerr != nil {
		// Error occured during validation
		return []entity.User{}, 0, valerr
	}
	return s.userRepo.SearchUser(ctx, s.logger, query)
}

// Helper to validate the user data against validation-tags mentioned in its entity.
func (s service) validateUserSearchData(ctx context.Context, ue entity.UserSearch) error {
	_, valerr := govalidator.ValidateStruct(ue)
	if valerr != nil {
		valerr := valerr.(govalidator.Errors).Errors()
		return errors.GenerateValidationErrorResponse(valerr)
	}
	return nil
}
