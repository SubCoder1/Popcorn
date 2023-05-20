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
		gangGroup.POST("/delete_content", delContent(gangService, logger))
	}
}

// createGang returns a handler which takes care of creating gangs in Popcorn.
func createGang(gangService Service, logger log.Logger) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		var gang entity.Gang

		// Serialize received data into User struct
		if binderr := gctx.ShouldBindJSON(&gang); binderr != nil {
			// Error occured during serialization
			logger.WithCtx(gctx).Error().Err(binderr).Msg("Binding error occured with User struct.")
			gctx.JSON(http.StatusUnprocessableEntity, errors.UnprocessableEntity(""))
			return
		}

		// Fetch username from context which will be used as the gang admin
		user, ok := gctx.Value("User").(entity.User)
		if !ok {
			// Type assertion error
			logger.WithCtx(gctx).Error().Msg("Type assertion error in createGang")
			gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
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
				gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
			}
			gctx.JSON(err.Status, err)
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
			logger.WithCtx(gctx).Error().Err(binderr).Msg("Binding error occured with User struct.")
			gctx.JSON(http.StatusUnprocessableEntity, errors.UnprocessableEntity(""))
			return
		}

		// Fetch username from context which will be used as the gang admin
		user, ok := gctx.Value("User").(entity.User)
		if !ok {
			// Type assertion error
			logger.WithCtx(gctx).Error().Msg("Type assertion error in createGang")
			gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
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
				gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
			}
			gctx.JSON(err.Status, err)
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
			gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
		}
		data, canCreate, canJoin, err := gangService.getgang(gctx, user.Username)
		if err != nil {
			// Error occured, might be validation or server error
			err, ok := err.(errors.ErrorResponse)
			if !ok {
				// Type assertion error
				gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
			}
			gctx.JSON(err.Status, err)
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
			gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
		}
		invites, err := gangService.getganginvites(gctx, user.Username)

		if err != nil {
			// Error occured, might be validation or server error
			err, ok := err.(errors.ErrorResponse)
			if !ok {
				// Type assertion error
				gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
			}
			gctx.JSON(err.Status, err)
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
			gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
		}
		membersList, err := gangService.getgangmembers(gctx, user.Username)
		if err != nil {
			// Error occured, might be validation or server error
			err, ok := err.(errors.ErrorResponse)
			if !ok {
				// Type assertion error
				gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
			}
			gctx.JSON(err.Status, err)
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
			gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
		}
		var gangKey entity.GangJoin
		// Serialize received data into GangKey struct
		if binderr := gctx.ShouldBindJSON(&gangKey); binderr != nil {
			// Error occured during serialization
			gctx.JSON(http.StatusUnprocessableEntity, errors.UnprocessableEntity(""))
			return
		}
		// Set gang-key which is of format gang:<gang_admin>
		gangKey.Key = "gang:" + gangKey.Admin
		err := gangService.joingang(gctx, user, gangKey)
		if err != nil {
			// Error occured, might be validation or server error
			err, ok := err.(errors.ErrorResponse)
			if !ok {
				// Type assertion error
				gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
			}
			gctx.JSON(err.Status, err)
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
			gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
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
				gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
			}
			gctx.JSON(err.Status, err)
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
			logger.WithCtx(gctx).Error().Msg("Type assertion error in get_gang")
			gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
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
				gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
			}
			gctx.JSON(err.Status, err)
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
			gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
		}
		var gangInvite entity.GangInvite
		// Serialize received data into GangInvite struct
		if binderr := gctx.ShouldBindJSON(&gangInvite); binderr != nil {
			// Error occured during serialization
			gctx.JSON(http.StatusUnprocessableEntity, errors.UnprocessableEntity(""))
			return
		}
		// User should be the admin here
		gangInvite.Admin = user.Username
		// Set CreatedTimeAgo to now
		gangInvite.CreatedTimeAgo = time.Now().Unix()
		err := gangService.sendganginvite(gctx, gangInvite)
		if err != nil {
			// Error occured, might be validation or server error
			err, ok := err.(errors.ErrorResponse)
			if !ok {
				// Type assertion error
				gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
			}
			gctx.JSON(err.Status, err)
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
			gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
		}
		var gangInvite entity.GangInvite
		// Serialize received data into GangInvite struct
		if binderr := gctx.ShouldBindJSON(&gangInvite); binderr != nil {
			// Error occured during serialization
			gctx.JSON(http.StatusUnprocessableEntity, errors.UnprocessableEntity(""))
			return
		}
		gangInvite.For = user.Username
		err := gangService.acceptganginvite(gctx, user, gangInvite)
		if err != nil {
			// Error occured, might be validation or server error
			err, ok := err.(errors.ErrorResponse)
			if !ok {
				// Type assertion error
				gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
			}
			gctx.JSON(err.Status, err)
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
			gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
		}
		var gangInvite entity.GangInvite
		// Serialize received data into GangInvite struct
		if binderr := gctx.ShouldBindJSON(&gangInvite); binderr != nil {
			// Error occured during serialization
			gctx.JSON(http.StatusUnprocessableEntity, errors.UnprocessableEntity(""))
			return
		}
		gangInvite.For = user.Username
		err := gangService.rejectganginvite(gctx, gangInvite)
		if err != nil {
			// Error occured, might be validation or server error
			err, ok := err.(errors.ErrorResponse)
			if !ok {
				// Type assertion error
				gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
			}
			gctx.JSON(err.Status, err)
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
			logger.WithCtx(gctx).Error().Msg("Type assertion error in bootMemberFromGang")
			gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
		}
		var boot entity.GangExit
		if binderr := gctx.ShouldBindJSON(&boot); binderr != nil {
			// Error occured during serialization
			gctx.JSON(http.StatusUnprocessableEntity, errors.UnprocessableEntity(""))
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
				gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
			}
			gctx.JSON(err.Status, err)
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
			logger.WithCtx(gctx).Error().Msg("Type assertion error in bootMemberFromGang")
			gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
		}
		err := gangService.delgang(gctx, user.Username)
		if err != nil {
			// Error occured, might be validation or server error
			err, ok := err.(errors.ErrorResponse)
			if !ok {
				// Type assertion error
				gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
			}
			gctx.JSON(err.Status, err)
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
			logger.WithCtx(gctx).Error().Msg("Type assertion error in bootMemberFromGang")
			gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
		}
		var msg entity.GangMessage
		if binderr := gctx.ShouldBindJSON(&msg); binderr != nil {
			// Error occured during serialization
			gctx.JSON(http.StatusUnprocessableEntity, errors.UnprocessableEntity(""))
			return
		}
		err := gangService.sendmessage(gctx, msg, user)
		if err != nil {
			// Error occured, might be validation or server error
			err, ok := err.(errors.ErrorResponse)
			if !ok {
				// Type assertion error
				gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
			}
			gctx.JSON(err.Status, err)
			return
		}
		gctx.Status(http.StatusOK)
	}
}

// delContent returns a handler which takes care of removing uploaded content.
func delContent(gangService Service, logger log.Logger) gin.HandlerFunc {
	return func(gctx *gin.Context) {
		// Fetch username from context which will be used as the deletecontent service
		user, ok := gctx.Value("User").(entity.User)
		if !ok {
			// Type assertion error
			logger.WithCtx(gctx).Error().Msg("Type assertion error in bootMemberFromGang")
			gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
		}
		gangName := struct {
			GangName string `json:"gang_name"`
		}{}
		if binderr := gctx.ShouldBindJSON(&gangName); binderr != nil {
			// Error occured during serialization
			gctx.JSON(http.StatusUnprocessableEntity, errors.UnprocessableEntity(""))
			return
		}
		err := gangService.deletecontent(gctx, user.Username, gangName.GangName)
		if err != nil {
			// Error occured, might be validation or server error
			err, ok := err.(errors.ErrorResponse)
			if !ok {
				// Type assertion error
				gctx.JSON(http.StatusInternalServerError, errors.InternalServerError(""))
			}
			gctx.JSON(err.Status, err)
			return
		}
		gctx.Status(http.StatusOK)
	}
}
