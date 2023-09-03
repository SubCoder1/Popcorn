// Gang content streaming services are all pasted here.

package gang

import (
	"Popcorn/internal/entity"
	"Popcorn/internal/errors"
	"Popcorn/internal/sse"
	"Popcorn/internal/user"
	"Popcorn/pkg/cleanup"
	"Popcorn/pkg/log"
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
	"time"

	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go"
)

var UPLOAD_PATH string = os.Getenv("UPLOAD_DIR")

type LivekitConfig struct {
	// Host url of livekit cloud
	Host string
	// api key required for livekit authentication
	ApiKey string
	// api secret required for livekit authentication
	ApiSecret string
	// identity who's trying to access livekit helpers
	Identity string
	// optional content file ID for uploading track
	Content string
	// optional livekit room name
	RoomName string
}

// Helper to fetch livekit room access token to be used by clients.
func getStreamToken(ctx context.Context, logger log.Logger, gangRepo Repository, userRepo user.Repository, config LivekitConfig) (string, error) {
	// Verify if user has joined any gang
	gang, dberr := gangRepo.GetJoinedGang(ctx, logger, config.Identity)
	if dberr != nil {
		// Error occured in GetJoinedGang()
		return "", dberr
	} else if gang.Admin == "" {
		// No joined gang, check if user has created one
		gang, dberr = gangRepo.GetGang(ctx, logger, "gang:"+config.Identity, config.Identity, false)
		if dberr != nil {
			// Error occured in GetGang()
			return "", dberr
		} else if gang.Admin == "" {
			// User has not created or joined a gang
			return "", errors.BadRequest("user must create or join a gang")
		}
	}
	// fetch from DB if user has an unexpired token already saved
	streaming_token := userRepo.GetStreamingToken(ctx, logger, config.Identity)
	if len(streaming_token) != 0 {
		return streaming_token, nil
	}

	yes, no := true, false
	at := auth.NewAccessToken(config.ApiKey, config.ApiSecret)
	grant := &auth.VideoGrant{
		RoomJoin:       true,
		RoomAdmin:      no,
		Room:           "room:" + gang.Admin,
		RoomCreate:     no,
		RoomList:       no,
		RoomRecord:     no,
		Recorder:       no,
		CanPublish:     &no,
		CanSubscribe:   &yes,
		CanPublishData: &no,
		IngressAdmin:   no,
	}
	at.AddGrant(grant).
		SetIdentity(config.Identity).
		SetValidFor(time.Hour * 3)

	streaming_token, err := at.ToJWT()
	if err != nil {
		logger.Error().Err(err).Msg("Error occured during fetching livekit client access token")
		return "", errors.InternalServerError("")
	}
	// Save the newly created streaming_token
	go userRepo.AddStreamingToken(ctx, logger, config.Identity, streaming_token)

	return streaming_token, err
}

// Helper to create a livekit room to be used for content streaming in Popcorn gangs.
func createStreamRoom(ctx context.Context, logger log.Logger, gang_limit uint32, config LivekitConfig) error {
	roomClient := lksdk.NewRoomServiceClient(config.Host, config.ApiKey, config.ApiSecret)

	roomList, rerr := roomClient.ListRooms(ctx, &livekit.ListRoomsRequest{Names: []string{config.RoomName}})
	if rerr != nil {
		// Error occured in livekit CreateRoom()
		logger.WithCtx(ctx).Error().Err(rerr).Msg("Error occured while creating room in livekit.ListRooms()")
		return errors.InternalServerError("")
	}
	if len(roomList.Rooms) == 0 {
		_, rerr := roomClient.CreateRoom(ctx, &livekit.CreateRoomRequest{
			Name:            config.RoomName,
			MaxParticipants: gang_limit,
			EmptyTimeout:    10800,
		})
		if rerr != nil {
			// Error occured in livekit CreateRoom()
			logger.WithCtx(ctx).Error().Err(rerr).Msg("Error occured while creating room in livekit.createStreamRoom()")
			return errors.InternalServerError("")
		}
		logger.WithCtx(ctx).Info().Msgf("Created livekit room for %s", config.RoomName)
	}

	return rerr
}

// Helper to delete room, triggered during delGang request from admin.
func deleteStreamRoom(ctx context.Context, logger log.Logger, config LivekitConfig) error {
	roomClient := lksdk.NewRoomServiceClient(config.Host, config.ApiKey, config.ApiSecret)

	_, rerr := roomClient.DeleteRoom(ctx, &livekit.DeleteRoomRequest{Room: config.RoomName})
	if rerr != nil {
		// Error occured in livekit.DeleteRoom()
		logger.WithCtx(ctx).Error().Err(rerr).Msgf("Couldn't delete room - %s", config.RoomName)
		return errors.InternalServerError("")
	}

	return rerr
}

// Helper to remove an user from the stream.
// Triggered during leave gang or booting a member.
func RemoveGangMemberFromStream(ctx context.Context, logger log.Logger, config LivekitConfig, member string) error {
	roomClient := lksdk.NewRoomServiceClient(config.Host, config.ApiKey, config.ApiSecret)
	_, rerr := roomClient.RemoveParticipant(ctx, &livekit.RoomParticipantIdentity{
		Room:     config.RoomName,
		Identity: member,
	})
	if rerr != nil {
		// Error occured in RemoveParticipant()
		logger.WithCtx(ctx).Error().Err(rerr).Msg("Error occured during removing member in livekit.RemoveParticipant()")
		return errors.InternalServerError("")
	}
	return nil
}

// Helper to create and return an IngressClient.
func createIngressClient(ctx context.Context, logger log.Logger, config LivekitConfig) *lksdk.IngressClient {
	return lksdk.NewIngressClient(config.Host, config.ApiKey, config.ApiSecret)
}

// Helper to start streaming gang content via livekit ingress and ffmpeg.
func ingressStreamContent(ctx context.Context, logger log.Logger, sseService sse.Service, gangRepo Repository, config LivekitConfig) error {
	ingressClient := createIngressClient(ctx, logger, config)

	// Delete existing ingress with same roomname
	ingerr := deleteIngress(ctx, logger, ingressClient, config.RoomName)
	if ingerr != nil {
		// Error occured in deleteIngress()
		return ingerr
	}

	// Create a new ingress request
	ingressRequest := &livekit.CreateIngressRequest{
		Name:                "ingress:" + config.Identity,
		RoomName:            config.RoomName,
		ParticipantIdentity: "gang_admin",
		ParticipantName:     config.Identity,
		Video: &livekit.IngressVideoOptions{
			EncodingOptions: &livekit.IngressVideoOptions_Preset{
				Preset: livekit.IngressVideoEncodingPreset_H264_1080P_30FPS_3_LAYERS,
			},
		},
		Audio: &livekit.IngressAudioOptions{
			EncodingOptions: &livekit.IngressAudioOptions_Preset{
				Preset: livekit.IngressAudioEncodingPreset_OPUS_MONO_64KBS,
			},
		},
	}
	info, ingerr := ingressClient.CreateIngress(ctx, ingressRequest)
	if ingerr != nil {
		// Error in CreateIngress()
		logger.WithCtx(ctx).Error().Err(ingerr).Msg("Error occured during the execution of livekit.CreateIngress()")
		return errors.InternalServerError("")
	}

	ffmpegCmd := exec.Command(
		"ffmpeg",
		"-re",
		"-i", UPLOAD_PATH+config.Content,
		"-c:v", "libx264",
		"-loglevel", "error",
		"-stats",
		"-preset:v", "veryfast",
		"-b:v", "3M",
		"-profile:v", "high",
		"-c:a", "aac",
		"-b:a", "128k",
		"-f", "flv",
		fmt.Sprintf("%s/%s", info.GetUrl(), info.GetStreamKey()),
	)
	// Signal to close running ffmpeg process on server shutdown
	go func() {
		s := make(chan os.Signal, 1)
		signal.Notify(s, syscall.SIGINT, syscall.SIGTERM, syscall.SIGHUP)
		closing_signal := <-s
		err := ffmpegCmd.Process.Signal(closing_signal)
		if err != nil {
			// Error occured during force-closing ffmpeg process
			logger.WithCtx(ctx).Error().Err(err).Msg("Error occured during force-closing ffmpeg process")
		} else {
			updateAfterStreamEnds(ctx, logger, sseService, gangRepo, ingressClient, config)
		}
	}()
	// Start the stream process in a separate goroutine
	go func() {
		output, execerr := ffmpegCmd.CombinedOutput()
		if execerr != nil {
			logger.WithCtx(ctx).Error().Err(execerr).Msgf("Failed to run ffmpeg command - %s", string(output))
		}
		updateAfterStreamEnds(ctx, logger, sseService, gangRepo, ingressClient, config)
	}()
	// Another goroutine to handle user triggered force-close of this stream
	go func() {
		streamRecords[config.RoomName] = make(close_stream_signal, 1)
		<-streamRecords[config.RoomName]
		err := ffmpegCmd.Process.Kill()
		if err != nil {
			// Error during killing ffmpeg process
			logger.WithCtx(ctx).Error().Err(err).Msgf("Error during killing off ffmpeg process for %s", config.RoomName)
		}
		close(streamRecords[config.RoomName])
		delete(streamRecords, config.RoomName)
	}()
	return nil
}

// Helper to delete already built livekit ingress.
func deleteIngress(ctx context.Context, logger log.Logger, client *lksdk.IngressClient, roomName string) error {
	ingressList, ingerr := client.ListIngress(ctx, &livekit.ListIngressRequest{RoomName: roomName})
	if ingerr != nil {
		// Error occured in ListIngress()
		logger.WithCtx(ctx).Error().Err(ingerr).Msg("Error occured during listing ingress via livekit.ListIngress()")
		return errors.InternalServerError("")
	}
	for _, ing := range ingressList.Items {
		_, ingerr = client.DeleteIngress(ctx, &livekit.DeleteIngressRequest{IngressId: ing.IngressId})
		if ingerr != nil {
			logger.WithCtx(ctx).Error().Err(ingerr).Msgf("Error occured while deleting ingress - %s via livekit.DeleteIngress()", ing)
		}
	}
	return nil
}

// Helper to update content data after stream process finishes.
func updateAfterStreamEnds(ctx context.Context, logger log.Logger, sseService sse.Service, gangRepo Repository,
	ingressClient *lksdk.IngressClient, config LivekitConfig) {
	logger.WithCtx(ctx).Info().Msgf("Stream ended for content %s | %s", config.Content, config.RoomName)
	// Delete ingress
	deleteIngress(ctx, logger, ingressClient, config.RoomName)
	// Delete gang content files
	cleanup.DeleteContentFiles(config.Content, logger)
	// Erase gang content data
	gangRepo.UpdateGangContentData(ctx, logger, config.Identity, "", "", false)
	// Notify the members that stream has stopped
	members, _ := gangRepo.GetGangMembers(ctx, logger, config.Identity)
	for _, member := range members {
		go func(member string) {
			data := entity.SSEData{
				Data: nil,
				Type: "gangEndContent",
				To:   member,
			}
			sseService.GetOrSetEvent(ctx).Message <- data
		}(member)
	}
}
