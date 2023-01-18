// Service layer of the internal package authentication.

package auth

import (
	"Popcorn/internal/entity"
	"Popcorn/internal/errors"
	"Popcorn/pkg/log"
	"context"
	"strings"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"

	"Popcorn/internal/user"

	"golang.org/x/crypto/bcrypt"
)

// Service layer of internal package auth which encapsulates authentication logic of Popcorn.
type Service interface {
	// Registers an user in Popcorn with valid user credentials
	register(ctx context.Context, user entity.User) (map[string]any, error)
	// Logs-in an user into Popcorn with valid user credentials
	login(ctx context.Context, user entity.User) (map[string]any, error)
	// Logs-out an user from Popcorn
	logout(ctx context.Context) error
	// Generates a fresh JWT for an user in Popcorn
	refreshtoken(ctx context.Context, username string) (map[string]any, error)
}

// Object of this will be passed around from main to routers to API.
// Helps to access the service layer interface and call methods.
// Also helps to pass objects to be used from outer layer.
type service struct {
	accSigningKey string
	refSigningKey string
	userRepo      user.Repository
	authRepo      Repository
	logger        log.Logger
}

// Helps to access the service layer interface and call methods. Service object is passed from main.
func NewService(accSigningKey string, refSigningKey string, userRepo user.Repository, authRepo Repository, logger log.Logger) Service {
	return service{accSigningKey, refSigningKey, userRepo, authRepo, logger}
}

func (s service) register(ctx context.Context, ue entity.User) (map[string]any, error) {
	token := make(map[string]any)

	// Validate the received user data which is serialized to entity.User struct
	valerr := s.validateUserData(ctx, ue)
	if valerr != nil {
		// Error occured during validation
		return token, valerr
	}
	// Trim full_name before saving
	ue.FullName = strings.Trim(ue.FullName, " ")

	// Check for user availability against user.Username
	available, dberr := s.userRepo.Exists(ctx, s.logger, ue.Username)
	if dberr != nil {
		// Error occured in Exists()
		return token, dberr
	} else if available {
		// User by the received username is already available in the platform
		valerr := errors.New("username:username is already taken")
		return token, errors.GenerateValidationErrorResponse([]error{valerr})
	}

	// users is a global key in db used to store current total number of users in Popcorn
	// Increment users by 1 and use that value as userID
	dberr = s.userRepo.IncrTotal(ctx, s.logger)
	if dberr != nil {
		// Error occured in IncrTotal()
		return token, dberr
	}

	// Hash user password and save the credentials in the user object
	hasheduserpwd, hasherr := s.generatePwDHash(ctx, ue.Password)
	if hasherr != nil {
		return token, errors.InternalServerError("")
	}
	ue.Password = hasheduserpwd

	// Set user's profile pic
	ue.ProfilePic = ue.SelectProfilePic()

	// Save the user in the DB
	_, dberr = s.userRepo.Set(ctx, s.logger, ue, true, false)
	if dberr != nil {
		// Error occured in Set()
		return token, dberr
	}

	// Generate JWT for the newly created user
	userJWTData, jwterr := s.createToken(ctx, ue.Username)
	if jwterr != nil {
		// Error during generating user's jwtData
		return token, jwterr
	}
	// Save generated tokens with expiration into the DB
	dberr = s.authRepo.SetToken(ctx, s.logger, userJWTData)
	if dberr != nil {
		// Error during saving user's JWT
		return token, dberr
	}

	token["access_token"] = userJWTData.AccessToken
	token["refresh_token"] = userJWTData.RefreshToken
	token["access_token_exp"] = time.Now().Add(time.Hour * 4)
	token["access_token_maxAge"] = (4 * 60) * 60
	token["refresh_token_exp"] = time.Now().Add(time.Hour * 24 * 7)
	token["refresh_token_maxAge"] = ((24 * 7) * 60) * 60
	return token, nil
}

func (s service) login(ctx context.Context, request entity.User) (map[string]any, error) {
	token := make(map[string]any)

	// Validate the received user data which is serialized to entity.User struct
	valerr := s.validateUserData(ctx, request)
	if valerr != nil {
		// Error occured during validation
		return token, valerr
	}

	// Check if user is available in Popcorn
	available, dberr := s.userRepo.Exists(ctx, s.logger, request.Username)
	if dberr != nil {
		// Error occured in Exists()
		return token, dberr
	} else if !available {
		// User by the received username is not available in the platform
		return token, errors.Unauthorized("Username or Password is incorrect")
	}

	// Fetch user's password hash from DB and validate against incoming password
	user, dberr := s.userRepo.Get(ctx, s.logger, request.Username)
	if dberr != nil {
		// Error occured in Get()
		return token, dberr
	} else if !s.verifyPwDHash(ctx, request.Password, user.Password) {
		// Invalid password
		return token, errors.Unauthorized("Username or Password is incorrect")
	}

	// Set user's profile pic
	user.ProfilePic = user.SelectProfilePic()

	// Save the user in the DB
	_, dberr = s.userRepo.Set(ctx, s.logger, user, true, true)
	if dberr != nil {
		// Error occured in Set()
		return token, dberr
	}

	// Generate JWT for the newly created user
	userJWTData, jwterr := s.createToken(ctx, user.Username)
	if jwterr != nil {
		// Error during generating user's jwtData
		return token, jwterr
	}
	// Save generated tokens with expiration into the DB
	dberr = s.authRepo.SetToken(ctx, s.logger, userJWTData)
	if dberr != nil {
		// Error during saving user's JWT
		return token, dberr
	}

	token["access_token"] = userJWTData.AccessToken
	token["refresh_token"] = userJWTData.RefreshToken
	token["access_token_exp"] = time.Now().Add(time.Hour * 4)
	token["access_token_maxAge"] = (4 * 60) * 60
	token["refresh_token_exp"] = time.Now().Add(time.Hour * 24 * 7)
	token["refresh_token_maxAge"] = ((24 * 7) * 60) * 60
	return token, nil
}

func (s service) logout(ctx context.Context) error {
	userAccToken := ctx.Value("access_token")
	if userAccToken == nil {
		// access_token or refresh_token missing from context
		return errors.InternalServerError("")
	}
	// Delete user's access_token from the DB
	dberr := s.authRepo.DelToken(ctx, s.logger, userAccToken.(string))
	if dberr != nil {
		// Error in DelToken
		return dberr
	}
	return nil
}

func (s service) refreshtoken(ctx context.Context, username string) (map[string]any, error) {
	token := make(map[string]any)
	// Create fresh JWT for user
	userJWTData, jwterr := s.createToken(ctx, username)
	if jwterr != nil {
		// Error during generating user's jwtData
		return token, errors.InternalServerError("")
	}
	// Save generated tokens with expiration into the DB
	dberr := s.authRepo.SetToken(ctx, s.logger, userJWTData)
	if dberr != nil {
		// Error during saving user's JWT
		return token, dberr
	}
	token["access_token"] = userJWTData.AccessToken
	token["refresh_token"] = userJWTData.RefreshToken
	token["access_token_exp"] = time.Now().Add(time.Hour * 4)
	token["access_token_maxAge"] = (4 * 60) * 60
	token["refresh_token_exp"] = time.Now().Add(time.Hour * 24 * 7)
	token["refresh_token_maxAge"] = ((24 * 7) * 60) * 60
	return token, nil
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
func (s service) verifyPwDHash(ctx context.Context, password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

type JWTdata struct {
	Username        string `json:"username"`
	AccessToken     string `json:"access_token"`
	AccTokenExp     int64  `json:"access_token_expiry"`
	AccessTokenUUID string `json:"access_token_uuid"`
	RefreshToken    string `json:"refresh_token"`
	RefTokenExp     int64  `json:"refresh_token_expiry"`
	RefTokenUUID    string `json:"refresh_token_uuid"`
}

// Helper to generate a JWT for an user given the claims data.
func (s service) generateJWT(ctx context.Context, claims jwt.Claims, signingKey string) (string, error) {
	token, jwterr := jwt.NewWithClaims(jwt.SigningMethodHS256, claims).SignedString([]byte(signingKey))
	if jwterr != nil {
		s.logger.Error().Err(jwterr).Msg("Error occured during JWT generation")
		return "", errors.InternalServerError("")
	}
	return token, nil
}

// Helper to create and return jwtData for an user with userID passed as param.
func (s service) createToken(ctx context.Context, username string) (*JWTdata, error) {
	jd := &JWTdata{}
	var jwterr error

	jd.Username = username
	jd.AccessTokenUUID = uuid.NewString()
	jd.AccTokenExp = time.Now().Add(time.Hour * 4).Unix()
	jd.RefTokenUUID = uuid.NewString()
	jd.RefTokenExp = time.Now().Add(time.Hour * 24 * 7).Unix()

	// Generate AccessToken using above data as claims
	// Pass AccessTokenSigningKey fetched from env to service
	jd.AccessToken, jwterr = s.generateJWT(ctx, jwt.MapClaims{
		"authorized":        true,
		"access_token_uuid": jd.AccessTokenUUID,
		"username":          username,
		"exp":               jd.AccTokenExp,
	}, s.accSigningKey)
	if jwterr != nil {
		// Error in generateJWT
		return nil, jwterr
	}
	// Generate RefreshToken using above data as claims
	// Pass RefreshTokenSigningKey fetched from env to service
	jd.RefreshToken, jwterr = s.generateJWT(ctx, jwt.MapClaims{
		"refresh_token_uuid": jd.RefTokenUUID,
		"username":           username,
		"exp":                jd.RefTokenExp,
	}, s.refSigningKey)
	if jwterr != nil {
		// Error in generateJWT
		return nil, jwterr
	}

	return jd, nil
}
