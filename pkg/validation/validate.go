// Handles all sorts of custom data validations and post-validation responses happening in Popcorn.

package validation

import (
	"strings"
	"unicode"

	"github.com/asaskevich/govalidator"
)

// Standard for Validation-error responses
type validationResponse struct {
	Param   string `json:"param"`   // Parameter or Field
	Message string `json:"message"` // Issue in Field
}

func CustomValidationTags() {
	// This custom validation checks if there's any spaces in the input string.
	// Currently being used in the UserModel.
	govalidator.TagMap["nospace"] = govalidator.Validator(func(str string) bool {
		return !strings.Contains(str, " ")
	})
	// This custom validation checks for password strength.
	// Only checks for 1 letter and 1 number, nothing too complicated.
	govalidator.TagMap["pwdstrength"] = govalidator.Validator(func(pwd string) bool {
		hasChar, hasNum := false, false
		for _, char := range pwd {
			if unicode.IsLetter(char) {
				hasChar = true
			}
			if unicode.IsNumber(char) {
				hasNum = true
			}
			if hasChar && hasNum {
				break
			}
		}
		return !(hasChar || hasNum)
	})
}

// Scans through set of validation errors found by govalidator
// Generates a slice of serializable validationResponse
func GenerateValidationErrorJSON(errs []error) []validationResponse {
	// govalidator returns array of errors -> Param:Message
	// We split the error from ":"
	resp := []validationResponse{}
	for _, err := range errs {
		e := strings.Split(err.Error(), ":")
		resp = append(
			resp, validationResponse{
				Param:   e[0],
				Message: strings.TrimSpace(e[1]),
			},
		)
	}
	return resp
}
