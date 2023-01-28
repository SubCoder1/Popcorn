// Service layer of the internal package gang.

package gang

import (
	"Popcorn/internal/entity"
	"Popcorn/internal/errors"
	"Popcorn/internal/user"
	"Popcorn/pkg/log"
	"context"
	"time"

	"github.com/asaskevich/govalidator"
	"golang.org/x/crypto/bcrypt"
)

// Service layer of internal package gang which encapsulates gang CRUD logic of Popcorn.
type Service interface {
	// Creates a gang in Popcorn.
	creategang(ctx context.Context, gang *entity.Gang) error
	// Get user created or joined gang data in Popcorn.
	getgang(ctx context.Context, username string) ([]entity.GangResponse, bool, bool, error)
	// Get gang invites received by user in Popcorn.
	getganginvites(ctx context.Context, username string) ([]entity.GangInvite, error)
	// Get list of gang members of user created gang in Popcorn.
	getgangmembers(ctx context.Context, username string) ([]entity.User, error)
	// Join user into a gang
	joingang(ctx context.Context, username string, gangKey entity.GangKey) error
	// Search for a gang
	searchgang(ctx context.Context, query entity.GangSearch, username string) ([]entity.GangResponse, uint64, error)
}

// Object of this will be passed around from main to routers to API.
// Helps to access the service layer interface and call methods.
// Also helps to pass objects to be used from outer layer.
type service struct {
	gangRepo Repository
	userRepo user.Repository
	logger   log.Logger
}

// Helps to access the service layer interface and call methods. Service object is passed from main.
func NewService(gangRepo Repository, userRepo user.Repository, logger log.Logger) Service {
	return service{gangRepo, userRepo, logger}
}

func (s service) creategang(ctx context.Context, gang *entity.Gang) error {
	valerr := s.validateGangData(ctx, gang)
	if valerr != nil {
		// Error occured during validation
		return valerr
	}
	// Check if user already has an unexpired gang created in Popcorn
	available, dberr := s.gangRepo.HasGang(ctx, s.logger, "gang:"+gang.Admin)
	if dberr != nil {
		// Error occured in HasGang()
		return dberr
	} else if available {
		// User cannot create more than 1 gang at a time
		valerr := errors.New("gang:User cannot create more than 1 gang at a time")
		return errors.GenerateValidationErrorResponse([]error{valerr})
	}

	// Set gang creation timestamp
	gang.Created = time.Now().Unix()
	// Set gang members list foreign key
	gang.MembersListKey = "gang-members:" + gang.Admin
	// Encrypt gang passkey
	hashedgangpk, hasherr := s.generatePassKeyHash(ctx, gang.PassKey)
	if hasherr != nil {
		return hasherr
	}
	gang.PassKey = hashedgangpk

	// Save gang data in DB
	_, dberr = s.gangRepo.SetGang(ctx, s.logger, gang)
	if dberr != nil {
		return dberr
	}

	return nil
}

func (s service) getgang(ctx context.Context, username string) ([]entity.GangResponse, bool, bool, error) {
	// Get gang data from DB
	data := []entity.GangResponse{}
	canCreate := false
	canJoin := false
	// get gang details created by user (if any)(if any)
	gangKey := "gang:" + username
	gangData, dberr := s.gangRepo.GetGang(ctx, s.logger, gangKey, username, false)
	if dberr != nil {
		// Error occured in GetGang()
		return data, canCreate, canJoin, dberr
	}
	// Don't send empty gang data
	if (gangData != entity.GangResponse{}) {
		data = append(data, gangData)
	} else {
		// User can create a gang
		canCreate = true
	}
	// get gang details joined by user (if any)
	gangJoinedData, dberr := s.gangRepo.GetJoinedGang(ctx, s.logger, username)
	if dberr != nil {
		// Error occured in GetJoinedGang()
		return data, canCreate, canJoin, dberr
	}
	// Don't send empty gang data
	if (gangJoinedData != entity.GangResponse{}) {
		data = append(data, gangJoinedData)
	} else {
		// User can join a gang
		canJoin = true
	}
	return data, canCreate, canJoin, nil
}

func (s service) getganginvites(ctx context.Context, username string) ([]entity.GangInvite, error) {
	return s.gangRepo.GetGangInvites(ctx, s.logger, username)
}

func (s service) getgangmembers(ctx context.Context, username string) ([]entity.User, error) {
	membersList, dberr := s.gangRepo.GetGangMembers(ctx, s.logger, username)
	if dberr != nil {
		// Error occured in GetGangMembers()
		return []entity.User{}, dberr
	}
	members := []entity.User{}
	for _, member := range membersList {
		user, dberr := s.userRepo.GetUser(ctx, s.logger, member)
		if dberr != nil {
			// Error occured in GetUser()
			return members, dberr
		}
		members = append(members, user)
	}
	return members, nil
}

func (s service) joingang(ctx context.Context, username string, gangKey entity.GangKey) error {
	valerr := s.validateGangData(ctx, gangKey)
	if valerr != nil {
		// Error occured during validation
		return valerr
	}
	// Fetch passkey hash for the gang and match with incoming one
	gangPassKeyHash, dberr := s.gangRepo.GetGangPassKey(ctx, s.logger, gangKey)
	if dberr != nil {
		// Error occured in GetGangPassKey()
		return dberr
	} else if !s.verifyPassKeyHash(ctx, gangKey.PassKey, gangPassKeyHash) {
		// Passkey didn't match
		return errors.Unauthorized("PassKey didn't match")
	}
	dberr = s.gangRepo.JoinGang(ctx, s.logger, gangKey, username)
	if dberr != nil {
		// Error occured in JoinGang()
		return dberr
	}
	return nil
}

func (s service) searchgang(ctx context.Context, query entity.GangSearch, username string) ([]entity.GangResponse, uint64, error) {
	valerr := s.validateGangData(ctx, query)
	if valerr != nil {
		// Error occured during validation
		return []entity.GangResponse{}, 0, valerr
	}
	return s.gangRepo.SearchGang(ctx, s.logger, query, username)
}

// Helper to validate the user data against validation-tags mentioned in its entity.
func (s service) validateGangData(ctx context.Context, gang interface{}) error {
	_, valerr := govalidator.ValidateStruct(gang)
	if valerr != nil {
		valerr := valerr.(govalidator.Errors).Errors()
		return errors.GenerateValidationErrorResponse(valerr)
	}
	return nil
}

// Helper to generate password hash and return in string type.
// Uses external package "bcrypt" and its function GenerateFromPassword.
func (s service) generatePassKeyHash(ctx context.Context, passkey string) (string, error) {
	pwdbyte, err := bcrypt.GenerateFromPassword([]byte(passkey), bcrypt.DefaultCost)
	if err != nil {
		s.logger.WithCtx(ctx).Error().Err(err).Msg("Error occured during Password encryption.")
		return "", errors.InternalServerError("")
	}
	return string(pwdbyte), nil
}

// Helper to verify incoming passkey with the actual hash of gang's set passkey.
// Helpful during gang join verification in Popcorn.
func (s service) verifyPassKeyHash(ctx context.Context, passkey, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(passkey))
	return err == nil
}
