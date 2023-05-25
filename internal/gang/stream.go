// Gang content streaming services are all pasted here.

package gang

import (
	"Popcorn/internal/entity"
	"Popcorn/internal/errors"
	"Popcorn/internal/user"
	"Popcorn/pkg/log"
	"context"
	"os"
	"time"

	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go"
)

var host string = os.Getenv("LIVEKIT_HOST")

// Helper to fetch livekit room access token to be used by clients.
func getStreamToken(ctx context.Context, logger log.Logger, gangRepo Repository, userRepo user.Repository, apiKey, apiSecret, username string) (string, error) {
	// Verify if user has joined any gang
	gang, dberr := gangRepo.GetJoinedGang(ctx, logger, username)
	if dberr != nil {
		// Error occured in GetJoinedGang()
		return "", dberr
	}
	// fetch from DB if user has an unexpired token already saved
	streaming_token := userRepo.GetStreamingToken(ctx, logger, username)
	if len(streaming_token) != 0 {
		return streaming_token, nil
	}

	at := auth.NewAccessToken(apiKey, apiSecret)
	grant := &auth.VideoGrant{
		RoomJoin:  true,
		RoomAdmin: false,
		Room:      gang.Admin + ":" + gang.Name,
	}
	at.AddGrant(grant).
		SetIdentity(username).
		SetValidFor(time.Hour)

	streaming_token, err := at.ToJWT()
	if err != nil {
		logger.Error().Err(err).Msg("Error occured during fetching livekit client access token")
		return streaming_token, errors.InternalServerError("")
	}
	// Save the newly created streaming_token
	go userRepo.AddStreamingToken(ctx, logger, username, streaming_token)

	return streaming_token, err
}

// Helper to create a livekit room to be used for content streaming in Popcorn gangs.
func createStreamRoom(ctx context.Context, logger log.Logger, gang entity.Gang, apiKey, apiSecret string) error {
	roomClient := lksdk.NewRoomServiceClient(host, apiKey, apiSecret)

	_, rerr := roomClient.CreateRoom(ctx, &livekit.CreateRoomRequest{
		Name:            "room:" + gang.Admin,
		MaxParticipants: uint32(gang.Limit),
	})
	if rerr != nil {
		// Error occured in livekit CreateRoom()
		logger.Error().Err(rerr).Msg("Error occured while creating room in gang.createStreamRoom()")
		return errors.InternalServerError("")
	}
	logger.WithCtx(ctx).Info().Msgf("Created livekit room for room:%s.", gang.Admin)

	return nil
}

// Helper to delete room, triggered during delGang request from admin.
func deleteStreamRoom(ctx context.Context, logger log.Logger, apiKey, apiSecret, roomName string) error {
	roomClient := lksdk.NewRoomServiceClient(host, apiKey, apiSecret)

	resp, rerr := roomClient.DeleteRoom(ctx, &livekit.DeleteRoomRequest{Room: roomName})
	if rerr != nil {
		// Error occured in livekit.DeleteRoom()
		logger.WithCtx(ctx).Error().Err(rerr).Msgf("Couldn't delete room - %s", roomName)
		return errors.InternalServerError("")
	}
	logger.WithCtx(ctx).Info().Msg(resp.String())

	return rerr
}
