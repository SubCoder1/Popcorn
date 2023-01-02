// Service layer of the internal package authentication.

package auth

import (
	"Popcorn/internal/entity"
	"Popcorn/internal/errors"
	"Popcorn/pkg/db"
	"Popcorn/pkg/log"
	"context"

	"github.com/asaskevich/govalidator"

	"Popcorn/internal/user"

	"golang.org/x/crypto/bcrypt"
)

// Service layer of internal package auth which encapsulates authentication logic of Popcorn.
type Service interface {
	// Registers an user in Popcorn with valid user credentials
	Register(context.Context, *db.RedisDB, entity.User) (string, error)
}

// Object of this will be passed around from main to routers to API.
// Helps to access the service layer interface and call methods.
// Also helps to pass objects to be used from outer layer.
type service struct {
	logger log.Logger
}

// Helps to access the service layer interface and call methods. Service object is passed from main.
func NewService(logger log.Logger) Service {
	return service{logger}
}

func (s service) Register(ctx context.Context, dbwrp *db.RedisDB, ue entity.User) (string, error) {
	// Validate the received user data which is serialized to entity.User struct
	valerr := s.validateUserData(ctx, ue)
	if valerr != nil {
		// Error occured during validation
		return "", valerr
	}

	// Check for user availability against user.Username
	// Need to access internal package user's repository function Exists() and Set()
	userrepo := user.NewRepository(dbwrp)

	available, dberr := userrepo.Exists(ctx, s.logger, ue.Username)
	if dberr != nil {
		// Error occured in Exists()
		return "", dberr
	} else if available {
		// User by the received username is already available in the platform
		valerr := errors.New("username:username is already taken")
		return "", errors.GenerateValidationErrorResponse([]error{valerr})
	}

	// Increment users by 1 and return the total
	// currTotal will be used as userID
	currTotal, dberr := userrepo.IncrTotal(ctx, s.logger)
	if dberr != nil {
		// Error occured in IncrTotal()
		return "", dberr
	}
	ue.ID = currTotal

	// Hash user password and save the credentials in the user object
	hasheduserpwd, hasherr := s.generatePwDHash(ctx, ue.Password)
	if hasherr != nil {
		return "", hasherr
	}
	ue.Password = hasheduserpwd

	// Save the user in the DB
	_, dberr = userrepo.Set(ctx, s.logger, ue)
	if dberr != nil {
		// Error occured in Set()
		return "", dberr
	}

	// Generate JWT token for the newly created user

	return "token", nil
}

// Helper to validate the user data against validation-tags mentioned in its entity.
func (s service) validateUserData(ctx context.Context, ue entity.User) error {
	_, valerr := govalidator.ValidateStruct(ue)
	if valerr != nil {
		valerr := valerr.(govalidator.Errors).Errors()
		return errors.GenerateValidationErrorResponse(valerr)
	}
	return nil
}

// Helper to generate password hash and return in string type.
// Uses external package "bcrypt" and its function GenerateFromPassword.
func (s service) generatePwDHash(ctx context.Context, password string) (string, error) {
	pwdbyte, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.WithCtx(ctx).Error().Err(err).Msg("Error occured during Password encryption.")
		return "", errors.InternalServerError("")
	}
	return string(pwdbyte), nil
}

// Helper to verify incoming password with the actual hash of user's set password.
// Helpful during login verification of an user in Popcorn.
func (s service) verifyPwDHash(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}
