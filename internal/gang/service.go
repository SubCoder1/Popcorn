// Service layer of the internal package gang.

package gang

import (
	"Popcorn/internal/entity"
	"Popcorn/internal/errors"
	"Popcorn/internal/sse"
	"Popcorn/internal/user"
	"Popcorn/pkg/cleanup"
	"Popcorn/pkg/log"
	"context"
	"time"

	"github.com/asaskevich/govalidator"
	"golang.org/x/crypto/bcrypt"
)

// Service layer of internal package gang which encapsulates gang CRUD logic of Popcorn.
type Service interface {
	// Creates a gang in Popcorn
	creategang(ctx context.Context, gang *entity.Gang) error
	// Updates a gang in Popcorn
	updategang(ctx context.Context, gang *entity.Gang) error
	// Get user created or joined gang data in Popcorn
	getgang(ctx context.Context, username string) (interface{}, bool, bool, error)
	// Get gang invites received by user in Popcorn
	getganginvites(ctx context.Context, username string) ([]entity.GangInvite, error)
	// Get list of gang members of user created gang in Popcorn
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
	// send incoming message to gang members
	sendmessage(ctx context.Context, msg entity.GangMessage, user entity.User) error
	// get livekit stream token needed for streaming content
	fetchstreamtoken(ctx context.Context, username string) (string, error)
	// livestream gang content to all of the gang members
	playcontent(ctx context.Context, admin string) error
	// stop ongoing gang livestream
	stopcontent(ctx context.Context, admin string) error
}

// Object of this will be passed around from main to routers to API.
// Helps to access the service layer interface and call methods.
// Also helps to pass objects to be used from outer layer.
type service struct {
	livekit_config LivekitConfig
	gangRepo       Repository
	userRepo       user.Repository
	sseService     sse.Service
	logger         log.Logger
}

// Instance of stream records used as an helper to close stream
type close_stream_signal chan bool

var streamRecords map[string]close_stream_signal

// Helps to access the service layer interface and call methods. Service object is passed from main.
func NewService(livekit_conf LivekitConfig, gangRepo Repository, userRepo user.Repository, sseService sse.Service, logger log.Logger) Service {
	streamRecords = map[string]close_stream_signal{}
	return service{livekit_conf, gangRepo, userRepo, sseService, logger}
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

	// Create livekit room first
	s.livekit_config.Identity = gang.Admin
	s.livekit_config.RoomName = "room:" + gang.Admin
	rerr := createStreamRoom(ctx, s.logger, uint32(gang.Limit), s.livekit_config)
	if rerr != nil {
		// Error occured in createStreamRoom()
		return rerr
	}

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
	} else if len(gang.PassKey) >= 5 {
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
	if dberr != nil {
		// Error in SetOrUpdateGang()
		return dberr
	}
	// Send notifications to gang Members about the updates
	members, _ := s.gangRepo.GetGangMembers(ctx, s.logger, gang.Admin)
	for _, member := range members {
		member := member
		go func() {
			data := entity.SSEData{
				Data: nil,
				Type: "gangUpdate",
				To:   member,
			}
			s.sseService.GetOrSetEvent(ctx).Message <- data
		}()
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
	if gangData.Admin != "" {
		s.livekit_config.Identity = gangData.Admin
		s.livekit_config.RoomName = "room:" + gangData.Admin
		rerr := createStreamRoom(ctx, s.logger, uint32(gangData.Limit), s.livekit_config)
		if rerr != nil {
			// Error occured in createStreamRoom()
			return entity.GangResponse{}, canCreate, canJoin, rerr
		}
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
	members, _ := s.gangRepo.GetGangMembers(ctx, s.logger, joinGangData.Admin)
	for _, member := range members {
		go func(member string) {
			data := entity.SSEData{
				Data: user,
				Type: "gangJoin",
				To:   member,
			}
			s.sseService.GetOrSetEvent(ctx).Message <- data
		}(member)
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
	// Check if the user who's accepting the invite is him/herself an admin
	// If so, then check further if he/she is currently streaming any content
	// close the content streaming process first (if found)
	gangKey := "gang:" + user.Username
	gang, dberr := s.gangRepo.GetGang(ctx, s.logger, gangKey, user.Username, false)
	if dberr != nil {
		// Error in GetGang()
		return dberr
	}
	if gang.IsAdmin && gang.Streaming {
		// Kill the streaming process
		s.stopcontent(ctx, gang.Admin)
	}

	dberr = s.gangRepo.AcceptGangInvite(ctx, s.logger, invite)
	if dberr != nil {
		// Error in AcceptGangInvite()
		return dberr
	}
	// Send notification to the gang page
	members, _ := s.gangRepo.GetGangMembers(ctx, s.logger, invite.Admin)
	user.Password = ""
	for _, member := range members {
		go func(member string) {
			data := entity.SSEData{
				Data: user,
				Type: "gangJoin",
				To:   member,
			}
			s.sseService.GetOrSetEvent(ctx).Message <- data
		}(member)
	}
	return nil
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
	// Remove member from ongoing stream
	s.livekit_config.RoomName = "room:" + admin
	rerr := RemoveGangMemberFromStream(ctx, s.logger, s.livekit_config, boot.Member)
	if rerr != nil {
		// Error in RemoveGangMemberFromStream()
		return rerr
	}
	// Send notification to gang members
	members, dberr := s.gangRepo.GetGangMembers(ctx, s.logger, admin)
	if dberr != nil {
		// Error in GetGangMembers()
		return dberr
	}
	dberr = s.gangRepo.LeaveGang(ctx, s.logger, boot)
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
	gangKey := "gang:" + admin
	oldGangData, dberr := s.gangRepo.GetGang(ctx, s.logger, gangKey, admin, false)
	if dberr != nil {
		// Error occured in GetGang()
		return dberr
	} else if oldGangData.Admin == "" {
		// Gang not found
		return errors.NotFound("user must create a gang")
	}

	// Delete livekit room
	s.livekit_config.RoomName = "room:" + admin
	rerr := deleteStreamRoom(ctx, s.logger, s.livekit_config)
	if rerr != nil {
		// Error occured in deleteStreamRoom()
		return rerr
	}

	// Delete uploaded gang contents
	go cleanup.DeleteContentFiles("./uploads/"+oldGangData.ContentID, s.logger)

	dberr = s.gangRepo.DelGang(ctx, s.logger, admin)
	if dberr != nil {
		// Error in DelGang()
		return dberr
	}
	// Send notification to gang members
	members, _ := s.gangRepo.GetGangMembers(ctx, s.logger, admin)
	for _, member := range members {
		s.userRepo.DelStreamingToken(ctx, s.logger, member)
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

func (s service) sendmessage(ctx context.Context, msg entity.GangMessage, user entity.User) error {
	valerr := s.validateGangData(ctx, msg)
	if valerr != nil {
		// Error occured during validation
		return valerr
	}
	// get gang key to fetch the list of gang members using GetGang or GetJoinedGang
	gang, dberr := s.gangRepo.GetGang(ctx, s.logger, "gang:"+user.Username, user.Username, true)
	if dberr != nil {
		// Error in GetGang()
		return dberr
	} else if (gang == entity.GangResponse{}) {
		// check using getJoinedGang
		gang, dberr = s.gangRepo.GetJoinedGang(ctx, s.logger, user.Username)
		if dberr != nil {
			// Error in GetJoinedGang()
			return dberr
		} else if (gang == entity.GangResponse{}) {
			return errors.BadRequest("user needs to create or join a gang")
		}
	}
	members, dberr := s.gangRepo.GetGangMembers(ctx, s.logger, gang.Admin)
	if dberr != nil {
		// Error in GetGangMembers()
		return dberr
	}
	// Send received message to members
	for _, member := range members {
		if user.Username != member {
			go func(member string) {
				// Don't send this message to the sender
				data := entity.SSEData{
					Data: struct {
						Text string `json:"text"`
						User struct {
							Username   string `json:"username"`
							ProfilePic string `json:"user_profile_pic"`
						} `json:"user"`
					}{msg.Message, struct {
						Username   string `json:"username"`
						ProfilePic string `json:"user_profile_pic"`
					}{user.Username, user.ProfilePic}},
					Type: "gangMessage",
					To:   member,
				}
				s.sseService.GetOrSetEvent(ctx).Message <- data
			}(member)
		}
	}
	return nil
}

func (s service) fetchstreamtoken(ctx context.Context, username string) (string, error) {
	s.livekit_config.Identity = username
	return getStreamToken(ctx, s.logger, s.gangRepo, s.userRepo, s.livekit_config)
}

func (s service) playcontent(ctx context.Context, admin string) error {
	gangKey := "gang:" + admin
	gang, dberr := s.gangRepo.GetGang(ctx, s.logger, gangKey, admin, false)
	if dberr != nil {
		// Error occured in GetGang()
		return dberr
	} else if gang.Admin == "" {
		// Not an admin
		return errors.BadRequest("user needs to create a gang")
	} else if gang.Streaming {
		// Already streaming
		return errors.BadRequest("content is already streaming")
	}
	// getting the members list early
	// coz if failure occurs here, no point of publishing content
	members, dberr := s.gangRepo.GetGangMembers(ctx, s.logger, admin)
	if dberr != nil {
		// Error occured in GetGangMembers()
		return dberr
	}
	// set gang.Streaming flag to true
	dberr = s.gangRepo.UpdateGangContentData(ctx, s.logger, admin, gang.ContentName, gang.ContentID, true)
	if dberr != nil {
		// Error occured in UpdateGangContentData()
		return dberr
	}
	// Send notification to gang members
	for _, member := range members {
		go func(member string) {
			data := entity.SSEData{
				Data: nil,
				Type: "gangPlayContent",
				To:   member,
			}
			s.sseService.GetOrSetEvent(ctx).Message <- data
		}(member)
	}
	// Publish encoded content files into livekit cloud
	s.livekit_config.Content = gang.ContentID
	s.livekit_config.RoomName = "room:" + admin
	s.livekit_config.Identity = admin
	perr := ingressStreamContent(ctx, s.logger, s.sseService, s.gangRepo, s.livekit_config)
	if perr != nil {
		// Error occured in publishStreamContent()
		return perr
	}
	return nil
}

func (s service) stopcontent(ctx context.Context, admin string) error {
	gangKey := "gang:" + admin
	gang, dberr := s.gangRepo.GetGang(ctx, s.logger, gangKey, admin, false)
	if dberr != nil {
		// Error occured in GetGang()
		return dberr
	} else if gang.Admin == "" {
		// Not an admin
		return errors.BadRequest("user needs to create a gang")
	} else if !gang.Streaming {
		// Not streaming
		return errors.BadRequest("content is not being streamed")
	}

	s.livekit_config.RoomName = "room:" + admin
	if stream, ok := streamRecords[s.livekit_config.RoomName]; ok {
		stream <- true
	} else {
		s.logger.WithCtx(ctx).Warn().Msgf("Couldn't find streamRecords for %s", s.livekit_config.RoomName)
		ingressClient := createIngressClient(ctx, s.logger, s.livekit_config)
		updateAfterStreamEnds(ctx, s.logger, s.sseService, s.gangRepo, ingressClient, s.livekit_config)
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
		s.logger.WithCtx(ctx).Error().Err(err).Msg("Error occured during Password encryption")
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
