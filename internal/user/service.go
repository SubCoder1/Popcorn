// Handles of all the REST API endpoints related to User Model.

package user

import (
	"Popcorn/pkg/validation"
	"net/http"

	"github.com/asaskevich/govalidator"
	"github.com/gin-gonic/gin"
)

// Handler function for User Registration on Popcorn.
func RegisterUser() gin.HandlerFunc {
	return func(gctx *gin.Context) {
		var user User

		// Bind received data into User struct
		if binderr := gctx.BindJSON(&user); binderr != nil {
			gctx.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{"error": binderr.Error()})
			return
		}

		// Check for user validity and availability
		_, err := govalidator.ValidateStruct(user)
		if err != nil {
			errs := err.(govalidator.Errors).Errors()
			gctx.JSON(http.StatusBadRequest, gin.H{
				"errors": validation.GenerateValidationErrorJSON(errs),
			})
			return
		}
		//available, dberr := db.CheckUserAvailibility(user.Username)

		// Generate JWT Token for created user

		gctx.JSON(http.StatusOK, gin.H{"token": "success"})
	}
}
