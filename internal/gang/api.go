// Exposes all of the REST APIs related to Gang creation in Popcorn.

package gang

import (
	"Popcorn/internal/entity"
	"Popcorn/internal/errors"
	"Popcorn/pkg/log"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
)

// Registers all of the REST API handlers related to internal package gang onto the gin server.
func APIHandlers(router *gin.Engine, gangService Service, authWithAcc gin.HandlerFunc, logger log.Logger) {
	gangGroup := router.Group("/api/gang", authWithAcc)
	{
		gangGroup.GET("/search", searchGang(gangService, logger))
		gangGroup.GET("/get", getGang(gangService, logger))
		gangGroup.GET("/get/invites", getGangInvites(gangService, logger))
		gangGroup.GET("/get/gang_members", getGangMembers(gangService, logger))
		gangGroup.POST("/create", createGang(gangService, logger))
		gangGroup.POST("/update", updateGang(gangService, logger))
		gangGroup.POST("/join", joinGang(gangService, logger))
		gangGroup.POST("/leave", leaveGang(gangService, logger))
		gangGroup.POST("/send_invite", sendInvite(gangService, logger))
		gangGroup.POST("/accept_invite", acceptInvite(gangService, logger))
		gangGroup.POST("/reject_invite", rejectInvite(gangService, logger))
		gangGroup.POST("/boot_member", bootMember(gangService, logger))
		gangGroup.POST("/delete", delGang(gangService, logger))
		gangGroup.POST("/send_msg", sendMessage(gangService, logger))
		gangGroup.POST("/get_token", fetchStreamToken(gangService, logger))
		gangGroup.POST("/play", playContent(gangService, logger))
		gangGroup.POST("/stop", stopContent(gangService, logger))
	}
}

// createGang returns a handler which takes care of creating gangs in Popcorn.
func createGang(gangService Service, logger log.Logger) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		var gang entity.Gang

		// Serialize received data into User struct
		if binderr := gctx.ShouldBindJSON(&gang); binderr != nil {
			// Error occured during serialization
			logger.WithCtx(gctx).Error().Err(binderr).Msg("Binding error occured with Gang struct")
			gctx.AbortWithStatusJSON(http.StatusUnprocessableEntity, errors.UnprocessableEntity(""))
			return
		}

		// Fetch username from context which will be used as the gang admin
		user, ok := gctx.Value("User").(entity.User)
		if !ok {
			// Type assertion error
			logger.WithCtx(gctx).Error().Msg("Type assertion error in createGang")
			gctx.AbortWithStatusJSON(http.StatusInternalServerError, errors.InternalServerError(""))
			return
		}
		gang.Admin = user.Username

		// Apply the service logic for Create Gang in Popcorn
		err := gangService.creategang(gctx, &gang)
		if err != nil {
			// Error occured, might be validation or server error
			err, ok := err.(errors.ErrorResponse)
			if !ok {
				// Type assertion error
				gctx.AbortWithStatusJSON(http.StatusInternalServerError, errors.InternalServerError(""))
				return
			}
			gctx.AbortWithStatusJSON(err.Status, err)
			return
		}
		gctx.Status(http.StatusOK)
	}
}

// updateGang returns a handler which takes care of updating user created gang in Popcorn.
func updateGang(gangService Service, logger log.Logger) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		var gang entity.Gang

		// Serialize received data into User struct
		if binderr := gctx.ShouldBindJSON(&gang); binderr != nil {
			// Error occured during serialization
			logger.WithCtx(gctx).Error().Err(binderr).Msg("Binding error occured with Gang struct")
			gctx.AbortWithStatusJSON(http.StatusUnprocessableEntity, errors.UnprocessableEntity(""))
			return
		}

		// Fetch username from context which will be used as the gang admin
		user, ok := gctx.Value("User").(entity.User)
		if !ok {
			// Type assertion error
			logger.WithCtx(gctx).Error().Msg("Type assertion error in updateGang")
			gctx.AbortWithStatusJSON(http.StatusInternalServerError, errors.InternalServerError(""))
			return
		}
		gang.Admin = user.Username

		// Apply the service logic for Update Gang in Popcorn
		err := gangService.updategang(gctx, &gang)
		if err != nil {
			// Error occured, might be validation or server error
			err, ok := err.(errors.ErrorResponse)
			if !ok {
				// Type assertion error
				gctx.AbortWithStatusJSON(http.StatusInternalServerError, errors.InternalServerError(""))
				return
			}
			gctx.AbortWithStatusJSON(err.Status, err)
			return
		}
		gctx.Status(http.StatusOK)
	}
}

// getGang returns a handler which takes care of getting user created or joined gangs in Popcorn.
func getGang(gangService Service, logger log.Logger) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		// Fetch username from context which will be used in getgang service
		user, ok := gctx.Value("User").(entity.User)
		if !ok {
			// Type assertion error
			logger.WithCtx(gctx).Error().Msg("Type assertion error in getGang")
			gctx.AbortWithStatusJSON(http.StatusInternalServerError, errors.InternalServerError(""))
		}
		data, canCreate, canJoin, err := gangService.getgang(gctx, user.Username)
		if err != nil {
			// Error occured, might be validation or server error
			err, ok := err.(errors.ErrorResponse)
			if !ok {
				// Type assertion error
				gctx.AbortWithStatusJSON(http.StatusInternalServerError, errors.InternalServerError(""))
				return
			}
			gctx.AbortWithStatusJSON(err.Status, err)
			return
		}

		gctx.JSON(http.StatusOK, gin.H{
			"gang":          data,
			"canCreateGang": canCreate,
			"canJoinGang":   canJoin,
		})
	}
}

// getGangInvites returns a handler which takes care of getting received gang invites in Popcorn.
func getGangInvites(gangService Service, logger log.Logger) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		// Fetch username from context which will be used in getganginvites service
		user, ok := gctx.Value("User").(entity.User)
		if !ok {
			// Type assertion error
			logger.WithCtx(gctx).Error().Msg("Type assertion error in getGangInvites")
			gctx.AbortWithStatusJSON(http.StatusInternalServerError, errors.InternalServerError(""))
		}
		invites, err := gangService.getganginvites(gctx, user.Username)

		if err != nil {
			// Error occured, might be validation or server error
			err, ok := err.(errors.ErrorResponse)
			if !ok {
				// Type assertion error
				gctx.AbortWithStatusJSON(http.StatusInternalServerError, errors.InternalServerError(""))
				return
			}
			gctx.AbortWithStatusJSON(err.Status, err)
			return
		}

		gctx.JSON(http.StatusOK, gin.H{
			"invites": invites,
		})
	}
}

// getGangMembers returns a handler which takes care of getting a list of all the gang members in Popcorn.
func getGangMembers(gangService Service, logger log.Logger) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		// Fetch username from context which will be used in getgangmembers service
		user, ok := gctx.Value("User").(entity.User)
		if !ok {
			// Type assertion error
			logger.WithCtx(gctx).Error().Msg("Type assertion error in getGangMembers")
			gctx.AbortWithStatusJSON(http.StatusInternalServerError, errors.InternalServerError(""))
		}
		membersList, err := gangService.getgangmembers(gctx, user.Username)
		if err != nil {
			// Error occured, might be validation or server error
			err, ok := err.(errors.ErrorResponse)
			if !ok {
				// Type assertion error
				gctx.AbortWithStatusJSON(http.StatusInternalServerError, errors.InternalServerError(""))
				return
			}
			gctx.AbortWithStatusJSON(err.Status, err)
			return
		}
		gctx.JSON(http.StatusOK, gin.H{
			"members": membersList,
		})
	}
}

// joinGang returns a handler which takes care of joining a gang in Popcorn.
func joinGang(gangService Service, logger log.Logger) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		// Fetch username from context which will be used as the joingang service
		user, ok := gctx.Value("User").(entity.User)
		if !ok {
			// Type assertion error
			logger.WithCtx(gctx).Error().Msg("Type assertion error in joinGang")
			gctx.AbortWithStatusJSON(http.StatusInternalServerError, errors.InternalServerError(""))
		}
		var joinData entity.GangJoin
		// Serialize received data into GangKey struct
		if binderr := gctx.ShouldBindJSON(&joinData); binderr != nil {
			// Error occured during serialization
			gctx.AbortWithStatusJSON(http.StatusUnprocessableEntity, errors.UnprocessableEntity(""))
			return
		}
		// Set gang-key which is of format gang:<gang_admin>
		joinData.Key = "gang:" + joinData.Admin
		err := gangService.joingang(gctx, user, joinData)
		if err != nil {
			// Error occured, might be validation or server error
			err, ok := err.(errors.ErrorResponse)
			if !ok {
				// Type assertion error
				gctx.AbortWithStatusJSON(http.StatusInternalServerError, errors.InternalServerError(""))
				return
			}
			gctx.AbortWithStatusJSON(err.Status, err)
			return
		}
		gctx.Status(http.StatusOK)
	}
}

// leaveGang returns a handler which takes care of leaving a gang in Popcorn.
func leaveGang(gangService Service, logger log.Logger) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		// Fetch username from context which will be used as the leavegang service
		user, ok := gctx.Value("User").(entity.User)
		if !ok {
			// Type assertion error
			logger.WithCtx(gctx).Error().Msg("Type assertion error in leaveGang")
			gctx.AbortWithStatusJSON(http.StatusInternalServerError, errors.InternalServerError(""))
		}
		var boot entity.GangExit
		boot.Member = user.Username
		boot.Type = "leave"
		err := gangService.leavegang(gctx, boot)
		if err != nil {
			// Error occured, might be validation or server error
			err, ok := err.(errors.ErrorResponse)
			if !ok {
				// Type assertion error
				gctx.AbortWithStatusJSON(http.StatusInternalServerError, errors.InternalServerError(""))
				return
			}
			gctx.AbortWithStatusJSON(err.Status, err)
			return
		}
		gctx.Status(http.StatusOK)
	}
}

// searchGang returns a handler which takes care of gang search in Popcorn.
func searchGang(gangService Service, logger log.Logger) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		// Fetch username from context which will be used in searchgang service
		user, ok := gctx.Value("User").(entity.User)
		if !ok {
			// Type assertion error
			logger.WithCtx(gctx).Error().Msg("Type assertion error in searchGang")
			gctx.AbortWithStatusJSON(http.StatusInternalServerError, errors.InternalServerError(""))
		}

		var query entity.GangSearch
		gang_name := gctx.DefaultQuery("gang_name", "")
		cursor, converr := strconv.Atoi(gctx.DefaultQuery("cursor", "0"))
		if converr != nil || cursor < 0 || cursor > 1000 {
			// Invalid cursor input
			gctx.Status(http.StatusBadRequest)
			return
		}
		// bind data into query struct
		query.Name = gang_name
		query.Cursor = cursor

		response, newCursor, err := gangService.searchgang(gctx, query, user.Username)
		if err != nil {
			// Error occured, might be validation or server error
			err, ok := err.(errors.ErrorResponse)
			if !ok {
				// Type assertion error
				gctx.AbortWithStatusJSON(http.StatusInternalServerError, errors.InternalServerError(""))
				return
			}
			gctx.AbortWithStatusJSON(err.Status, err)
			return
		}
		gctx.JSON(http.StatusOK, gin.H{
			"result": response,
			"page":   newCursor,
		})
	}
}

// sendInvite returns a handler which takes care of sending gang invite in Popcorn.
func sendInvite(gangService Service, logger log.Logger) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		// Fetch username from context which will be used as the sendinvite service
		user, ok := gctx.Value("User").(entity.User)
		if !ok {
			// Type assertion error
			logger.WithCtx(gctx).Error().Msg("Type assertion error in sendInvite")
			gctx.AbortWithStatusJSON(http.StatusInternalServerError, errors.InternalServerError(""))
		}
		var gangInvite entity.GangInvite
		// Serialize received data into GangInvite struct
		if binderr := gctx.ShouldBindJSON(&gangInvite); binderr != nil {
			// Error occured during serialization
			gctx.AbortWithStatusJSON(http.StatusUnprocessableEntity, errors.UnprocessableEntity(""))
			return
		}
		// User should be the admin here
		gangInvite.Admin = user.Username
		// Set CreatedTimeAgo to now
		gangInvite.CreatedTimeAgo = time.Now().Unix()
		gangInvite.InviteHashCode = "NOTREQUIRED"
		err := gangService.sendganginvite(gctx, gangInvite)
		if err != nil {
			// Error occured, might be validation or server error
			err, ok := err.(errors.ErrorResponse)
			if !ok {
				// Type assertion error
				gctx.AbortWithStatusJSON(http.StatusInternalServerError, errors.InternalServerError(""))
				return
			}
			gctx.AbortWithStatusJSON(err.Status, err)
			return
		}
		gctx.Status(http.StatusOK)
	}
}

// acceptInvite returns a handler which takes care of accepting gang invite in Popcorn.
func acceptInvite(gangService Service, logger log.Logger) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		// Fetch username from context which will be used as the acceptinvite service
		user, ok := gctx.Value("User").(entity.User)
		if !ok {
			// Type assertion error
			logger.WithCtx(gctx).Error().Msg("Type assertion error in acceptInvite")
			gctx.AbortWithStatusJSON(http.StatusInternalServerError, errors.InternalServerError(""))
		}
		var gangInvite entity.GangInvite
		// Serialize received data into GangInvite struct
		if binderr := gctx.ShouldBindJSON(&gangInvite); binderr != nil {
			// Error occured during serialization
			gctx.AbortWithStatusJSON(http.StatusUnprocessableEntity, errors.UnprocessableEntity(""))
			return
		}
		gangInvite.For = user.Username
		err := gangService.acceptganginvite(gctx, user, gangInvite)
		if err != nil {
			// Error occured, might be validation or server error
			err, ok := err.(errors.ErrorResponse)
			if !ok {
				// Type assertion error
				gctx.AbortWithStatusJSON(http.StatusInternalServerError, errors.InternalServerError(""))
				return
			}
			gctx.AbortWithStatusJSON(err.Status, err)
			return
		}
		gctx.Status(http.StatusOK)
	}
}

// rejectInvite returns a handler which takes care of rejecting gang invite in Popcorn.
func rejectInvite(gangService Service, logger log.Logger) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		// Fetch username from context which will be used as the rejectinvite service
		user, ok := gctx.Value("User").(entity.User)
		if !ok {
			// Type assertion error
			logger.WithCtx(gctx).Error().Msg("Type assertion error in rejectInvite")
			gctx.AbortWithStatusJSON(http.StatusInternalServerError, errors.InternalServerError(""))
			return
		}
		var gangInvite entity.GangInvite
		// Serialize received data into GangInvite struct
		if binderr := gctx.ShouldBindJSON(&gangInvite); binderr != nil {
			// Error occured during serialization
			gctx.AbortWithStatusJSON(http.StatusUnprocessableEntity, errors.UnprocessableEntity(""))
			return
		}
		gangInvite.For = user.Username
		gangInvite.InviteHashCode = "NOTREQUIRED"
		err := gangService.rejectganginvite(gctx, gangInvite)
		if err != nil {
			// Error occured, might be validation or server error
			err, ok := err.(errors.ErrorResponse)
			if !ok {
				// Type assertion error
				gctx.AbortWithStatusJSON(http.StatusInternalServerError, errors.InternalServerError(""))
				return
			}
			gctx.AbortWithStatusJSON(err.Status, err)
			return
		}
		gctx.Status(http.StatusOK)
	}
}

// bootMember returns a handler which takes care of booting member from a gang in Popcorn.
func bootMember(gangService Service, logger log.Logger) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		// Fetch username from context which will be used as the bootmember service
		user, ok := gctx.Value("User").(entity.User)
		if !ok {
			// Type assertion error
			logger.WithCtx(gctx).Error().Msg("Type assertion error in bootMember")
			gctx.AbortWithStatusJSON(http.StatusInternalServerError, errors.InternalServerError(""))
			return
		}
		var boot entity.GangExit
		if binderr := gctx.ShouldBindJSON(&boot); binderr != nil {
			// Error occured during serialization
			gctx.AbortWithStatusJSON(http.StatusUnprocessableEntity, errors.UnprocessableEntity(""))
			return
		}
		boot.Key = "gang:" + user.Username
		boot.Type = "boot"
		err := gangService.bootmember(gctx, user.Username, boot)
		if err != nil {
			// Error occured, might be validation or server error
			err, ok := err.(errors.ErrorResponse)
			if !ok {
				// Type assertion error
				gctx.AbortWithStatusJSON(http.StatusInternalServerError, errors.InternalServerError(""))
				return
			}
			gctx.AbortWithStatusJSON(err.Status, err)
			return
		}
		gctx.Status(http.StatusOK)
	}
}

// delGang returns a handler which takes care of deleting a gang from Popcorn before expiry.
func delGang(gangService Service, logger log.Logger) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		// Fetch username from context which will be used as the delgang service
		user, ok := gctx.Value("User").(entity.User)
		if !ok {
			// Type assertion error
			logger.WithCtx(gctx).Error().Msg("Type assertion error in delGang")
			gctx.AbortWithStatusJSON(http.StatusInternalServerError, errors.InternalServerError(""))
			return
		}
		err := gangService.delgang(gctx, user.Username)
		if err != nil {
			// Error occured, might be validation or server error
			err, ok := err.(errors.ErrorResponse)
			if !ok {
				// Type assertion error
				gctx.AbortWithStatusJSON(http.StatusInternalServerError, errors.InternalServerError(""))
				return
			}
			gctx.AbortWithStatusJSON(err.Status, err)
			return
		}
		gctx.Status(http.StatusOK)
	}
}

// sendMessage returns a handler which takes care of broadcasting message to gang members.
func sendMessage(gangService Service, logger log.Logger) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		// Fetch username from context which will be used as the bootmember service
		user, ok := gctx.Value("User").(entity.User)
		if !ok {
			// Type assertion error
			logger.WithCtx(gctx).Error().Msg("Type assertion error in sendMessage")
			gctx.AbortWithStatusJSON(http.StatusInternalServerError, errors.InternalServerError(""))
			return
		}
		var msg entity.GangMessage
		if binderr := gctx.ShouldBindJSON(&msg); binderr != nil {
			// Error occured during serialization
			gctx.AbortWithStatusJSON(http.StatusUnprocessableEntity, errors.UnprocessableEntity(""))
			return
		}
		err := gangService.sendmessage(gctx, msg, user)
		if err != nil {
			// Error occured, might be validation or server error
			err, ok := err.(errors.ErrorResponse)
			if !ok {
				// Type assertion error
				gctx.AbortWithStatusJSON(http.StatusInternalServerError, errors.InternalServerError(""))
				return
			}
			gctx.AbortWithStatusJSON(err.Status, err)
			return
		}
		gctx.Status(http.StatusOK)
	}
}

// fetchStreamToken returns a handler which takes care of getting livekit token for streaming.
func fetchStreamToken(gangService Service, logger log.Logger) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		// Fetch username from context which will be used in getgang service
		user, ok := gctx.Value("User").(entity.User)
		if !ok {
			// Type assertion error
			logger.WithCtx(gctx).Error().Msg("Type assertion error in fetchStreamToken")
			gctx.AbortWithStatusJSON(http.StatusInternalServerError, errors.InternalServerError(""))
			return
		}
		token, err := gangService.fetchstreamtoken(gctx, user.Username)
		if err != nil {
			// Error occured, might be validation or server error
			err, ok := err.(errors.ErrorResponse)
			if !ok {
				// Type assertion error
				gctx.AbortWithStatusJSON(http.StatusInternalServerError, errors.InternalServerError(""))
				return
			}
			gctx.AbortWithStatusJSON(err.Status, err)
			return
		}
		gctx.JSON(http.StatusOK, gin.H{"stream_token": token})
	}
}

// playContent returns a handler which takes care of live streaming content to gang members.
func playContent(gangService Service, logger log.Logger) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		// Fetch username from context which will be used as the playcontent service
		user, ok := gctx.Value("User").(entity.User)
		if !ok {
			// Type assertion error
			logger.WithCtx(gctx).Error().Msg("Type assertion error in playContent")
			gctx.AbortWithStatusJSON(http.StatusInternalServerError, errors.InternalServerError(""))
			return
		}
		err := gangService.playcontent(gctx, user.Username)
		if err != nil {
			// Error occured, might be validation or server error
			err, ok := err.(errors.ErrorResponse)
			if !ok {
				// Type assertion error
				gctx.AbortWithStatusJSON(http.StatusInternalServerError, errors.InternalServerError(""))
				return
			}
			gctx.AbortWithStatusJSON(err.Status, err)
			return
		}
		gctx.Status(http.StatusOK)
	}
}

// stopContent returns a handler which takes care of stopping an ongoing gang stream.
func stopContent(gangService Service, logger log.Logger) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		// Fetch username from context which will be used as the playcontent service
		user, ok := gctx.Value("User").(entity.User)
		if !ok {
			// Type assertion error
			logger.WithCtx(gctx).Error().Msg("Type assertion error in stopContent")
			gctx.AbortWithStatusJSON(http.StatusInternalServerError, errors.InternalServerError(""))
			return
		}
		err := gangService.stopcontent(gctx, user.Username)
		if err != nil {
			// Error occured, might be validation or server error
			err, ok := err.(errors.ErrorResponse)
			if !ok {
				// Type assertion error
				gctx.AbortWithStatusJSON(http.StatusInternalServerError, errors.InternalServerError(""))
				return
			}
			gctx.AbortWithStatusJSON(err.Status, err)
			return
		}
		gctx.Status(http.StatusOK)
	}
}
