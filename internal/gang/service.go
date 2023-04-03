// Service layer of the internal package gang.

package gang

import (
	"Popcorn/internal/entity"
	"Popcorn/internal/errors"
	"Popcorn/internal/sse"
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
	// Updates a gang in Popcorn.
	updategang(ctx context.Context, gang *entity.Gang) error
	// Get user created or joined gang data in Popcorn.
	getgang(ctx context.Context, username string) (interface{}, bool, bool, error)
	// Get gang invites received by user in Popcorn.
	getganginvites(ctx context.Context, username string) ([]entity.GangInvite, error)
	// Get list of gang members of user created gang in Popcorn.
	getgangmembers(ctx context.Context, username string) ([]entity.User, error)
	// Join user into a gang
	joingang(ctx context.Context, user entity.User, joinGangData entity.GangJoin) error
	// Search for a gang
	searchgang(ctx context.Context, query entity.GangSearch, username string) ([]entity.GangResponse, uint64, error)
	// Send gang invite to an user
	sendganginvite(ctx context.Context, invite entity.GangInvite) error
	// Accept gang invite for an user
	acceptganginvite(ctx context.Context, user entity.User, invite entity.GangInvite) error
	// Reject gang invite for an user
	rejectganginvite(ctx context.Context, invite entity.GangInvite) error
	// kicks a member out of a gang
	bootmember(ctx context.Context, admin string, boot entity.GangExit) error
	// leave a gang
	leavegang(ctx context.Context, boot entity.GangExit) error
	// delete a gang before expiry
	delgang(ctx context.Context, admin string) error
}

// Object of this will be passed around from main to routers to API.
// Helps to access the service layer interface and call methods.
// Also helps to pass objects to be used from outer layer.
type service struct {
	gangRepo   Repository
	userRepo   user.Repository
	sseService sse.Service
	logger     log.Logger
}

// Helps to access the service layer interface and call methods. Service object is passed from main.
func NewService(gangRepo Repository, userRepo user.Repository, sseService sse.Service, logger log.Logger) Service {
	return service{gangRepo, userRepo, sseService, logger}
}

func (s service) creategang(ctx context.Context, gang *entity.Gang) error {
	valerr := s.validateGangData(ctx, gang)
	if valerr != nil {
		// Error occured during validation
		return valerr
	}
	// Check if user already has an unexpired gang created in Popcorn
	available, dberr := s.gangRepo.HasGang(ctx, s.logger, "gang:"+gang.Admin, "")
	if dberr != nil {
		// Error occured in HasGang()
		return dberr
	} else if available {
		// User cannot create more than 1 gang at a time
		valerr := errors.New("gang:User cannot create more than 1 gang at a time")
		return errors.GenerateValidationErrorResponse([]error{valerr})
	}
	// Check if user has already joined a gang in Popcorn
	joined, dberr := s.gangRepo.HasGang(ctx, s.logger, "gang-joined:"+gang.Admin, "")
	if dberr != nil {
		// Error occured in HasGang()
		return dberr
	} else if joined {
		// User can only create or join a gang at a time.
		valerr := errors.New("gang:User can only join or create a gang at a time.")
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
	_, dberr = s.gangRepo.SetOrUpdateGang(ctx, s.logger, gang, false)
	if dberr != nil {
		err, ok := dberr.(errors.ErrorResponse)
		if ok && err.StatusCode() == 400 {
			// User cannot create more than 1 gang at a time
			valerr := errors.New("gang:User cannot create more than 1 gang at a time")
			return errors.GenerateValidationErrorResponse([]error{valerr})
		}
		return dberr
	}

	return nil
}

func (s service) updategang(ctx context.Context, gang *entity.Gang) error {
	// Check if user already has an unexpired gang created in Popcorn
	available, dberr := s.gangRepo.HasGang(ctx, s.logger, "gang:"+gang.Admin, "")
	if dberr != nil {
		// Error occured in HasGang()
		return dberr
	} else if !available {
		// User doesn't have their own gang to update
		valerr := errors.New("gang:User haven't created a gang")
		return errors.GenerateValidationErrorResponse([]error{valerr})
	}

	if gang.PassKey == "" {
		// Just to pass validation
		gang.PassKey = "PREVIOUSPASSKEY"
	} else {
		// Encrypt gang passkey
		hashedgangpk, hasherr := s.generatePassKeyHash(ctx, gang.PassKey)
		if hasherr != nil {
			return hasherr
		}
		gang.PassKey = hashedgangpk
	}
	valerr := s.validateGangData(ctx, gang)
	if valerr != nil {
		// Error occured during validation
		return valerr
	}
	_, dberr = s.gangRepo.SetOrUpdateGang(ctx, s.logger, gang, true)
	if dberr == nil {
		// Send notifications to gang Members about the updates
		members, dberr := s.gangRepo.GetGangMembers(ctx, s.logger, gang.Admin)
		if dberr == nil {
			for _, member := range members {
				go func(member string) {
					data := entity.SSEData{
						Data: nil,
						Type: "gangUpdate",
						To:   member,
					}
					s.sseService.GetOrSetEvent(ctx).Message <- data
				}(member)
			}
		}
	}
	return dberr
}

func (s service) getgang(ctx context.Context, username string) (interface{}, bool, bool, error) {
	// Get gang data from DB
	canCreate := false
	canJoin := false
	// get gang details created by user (if any)(if any)
	gangKey := "gang:" + username
	gangData, dberr := s.gangRepo.GetGang(ctx, s.logger, gangKey, username, false)
	if dberr != nil {
		// Error occured in GetGang()
		return entity.GangResponse{}, canCreate, canJoin, dberr
	}
	// Don't send empty gang data
	if (gangData != entity.GangResponse{}) {
		return gangData, canCreate, canJoin, dberr
	}
	// get gang details joined by user (if any)
	gangJoinedData, dberr := s.gangRepo.GetJoinedGang(ctx, s.logger, username)
	if dberr != nil {
		// Error occured in GetJoinedGang()
		return entity.GangResponse{}, canCreate, canJoin, dberr
	}
	// Don't send empty gang data
	if (gangJoinedData != entity.GangResponse{}) {
		return gangJoinedData, canCreate, canJoin, dberr
	}
	canCreate, canJoin = true, true
	return struct{}{}, canCreate, canJoin, nil
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

func (s service) joingang(ctx context.Context, user entity.User, joinGangData entity.GangJoin) error {
	// Check if user already has an unexpired gang created in Popcorn
	available, dberr := s.gangRepo.HasGang(ctx, s.logger, "gang:"+user.Username, "")
	if dberr != nil {
		// Error occured in HasGang()
		return dberr
	} else if available {
		// User cannot create more than 1 gang at a time
		valerr := errors.New("gang:User can only join or create a gang at a time.")
		return errors.GenerateValidationErrorResponse([]error{valerr})
	}
	// Check if user has already joined a gang in Popcorn
	joined, dberr := s.gangRepo.HasGang(ctx, s.logger, "gang-joined:"+user.Username, "")
	if dberr != nil {
		// Error occured in HasGang()
		return dberr
	} else if joined {
		// User can only create or join a gang at a time.
		valerr := errors.New("gang:User can only join or create a gang at a time.")
		return errors.GenerateValidationErrorResponse([]error{valerr})
	}

	valerr := s.validateGangData(ctx, joinGangData)
	if valerr != nil {
		// Error occured during validation
		return valerr
	}
	// Fetch passkey hash for the gang and match with incoming one
	gangPassKeyHash, dberr := s.gangRepo.GetGangPassKey(ctx, s.logger, joinGangData)
	if dberr != nil {
		// Error occured in GetGangPassKey()
		return dberr
	} else if !s.verifyPassKeyHash(ctx, joinGangData.PassKey, gangPassKeyHash) {
		// Passkey didn't match
		return errors.Unauthorized("PassKey didn't match")
	}
	dberr = s.gangRepo.JoinGang(ctx, s.logger, joinGangData, user.Username)
	if dberr != nil {
		// Error occured in JoinGang()
		return dberr
	}
	// Send notification to the gang page
	go func() {
		user.Password = ""
		data := entity.SSEData{
			Data: user,
			Type: "gangJoin",
			To:   joinGangData.Admin,
		}
		s.sseService.GetOrSetEvent(ctx).Message <- data
	}()
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

func (s service) sendganginvite(ctx context.Context, invite entity.GangInvite) error {
	valerr := s.validateGangData(ctx, invite)
	if valerr != nil {
		// Error occured during validation
		return valerr
	}
	// check if self invite is getting sent
	if invite.Admin == invite.For {
		return errors.BadRequest("Invalid Gang Invite")
	}
	// Send notification to the receiver if active
	go func() {
		data := entity.SSEData{
			Data: invite,
			Type: "gangInvite",
			To:   invite.For,
		}
		s.sseService.GetOrSetEvent(ctx).Message <- data
	}()
	return s.gangRepo.SendGangInvite(ctx, s.logger, invite)
}

func (s service) acceptganginvite(ctx context.Context, user entity.User, invite entity.GangInvite) error {
	valerr := s.validateGangData(ctx, invite)
	if valerr != nil {
		// Error occured during validation
		return valerr
	}
	// check if self invite is getting sent
	if invite.Admin == invite.For {
		return errors.BadRequest("Invalid Gang Invite")
	}
	// Send notification to the gang page
	go func() {
		user.Password = ""
		data := entity.SSEData{
			Data: user,
			Type: "gangJoin",
			To:   invite.Admin,
		}
		s.sseService.GetOrSetEvent(ctx).Message <- data
	}()
	return s.gangRepo.AcceptGangInvite(ctx, s.logger, invite)
}

func (s service) leavegang(ctx context.Context, boot entity.GangExit) error {
	joinedGang, dberr := s.gangRepo.GetJoinedGang(ctx, s.logger, boot.Member)
	if dberr != nil {
		// Error in GetJoinedGang()
		return dberr
	}
	boot.Name = joinedGang.Name
	boot.Key = "gang:" + joinedGang.Admin
	dberr = s.gangRepo.LeaveGang(ctx, s.logger, boot)
	if dberr != nil {
		// Error in bootmember()
		return dberr
	}
	// Send notification to gang members
	members, dberr := s.gangRepo.GetGangMembers(ctx, s.logger, joinedGang.Admin)
	if dberr == nil {
		for _, member := range members {
			go func(member string) {
				data := entity.SSEData{
					Data: boot.Member,
					Type: "gangLeave",
					To:   member,
				}
				s.sseService.GetOrSetEvent(ctx).Message <- data
			}(member)
		}
	}
	return nil
}

func (s service) rejectganginvite(ctx context.Context, invite entity.GangInvite) error {
	valerr := s.validateGangData(ctx, invite)
	if valerr != nil {
		// Error occured during validation
		return valerr
	}
	// check if self invite is getting sent
	if invite.Admin == invite.For {
		return errors.BadRequest("Invalid Gang Invite")
	}
	return s.gangRepo.DelGangInvite(ctx, s.logger, invite)
}

func (s service) bootmember(ctx context.Context, admin string, boot entity.GangExit) error {
	valerr := s.validateGangData(ctx, boot)
	if valerr != nil {
		// Error occured during validation
		return valerr
	}
	// Send notification to gang members
	members, _ := s.gangRepo.GetGangMembers(ctx, s.logger, admin)
	dberr := s.gangRepo.LeaveGang(ctx, s.logger, boot)
	if dberr != nil {
		// Error in LeaveGang()
		return dberr
	}
	// Send notification to the kicked member
	go func() {
		data := entity.SSEData{
			Data: boot,
			Type: "gangBoot",
			To:   boot.Member,
		}
		s.sseService.GetOrSetEvent(ctx).Message <- data
	}()
	for _, member := range members {
		go func(member string) {
			data := entity.SSEData{
				Data: boot.Member,
				Type: "gangLeave",
				To:   member,
			}
			s.sseService.GetOrSetEvent(ctx).Message <- data
		}(member)
	}
	return nil
}

func (s service) delgang(ctx context.Context, admin string) error {
	// Send notification to gang members
	members, _ := s.gangRepo.GetGangMembers(ctx, s.logger, admin)
	dberr := s.gangRepo.DelGang(ctx, s.logger, admin)
	if dberr != nil {
		// Error in DelGang()
		return dberr
	}
	for _, member := range members {
		go func(member string) {
			data := entity.SSEData{
				Data: nil,
				Type: "gangDelete",
				To:   member,
			}
			s.sseService.GetOrSetEvent(ctx).Message <- data
		}(member)
	}
	return nil
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
