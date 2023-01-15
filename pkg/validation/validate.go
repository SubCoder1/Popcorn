// Handles all sorts of custom data validations happening in Popcorn.

package validation

import (
	"context"
	"regexp"
	"strings"
	"unicode"

	"Popcorn/pkg/log"

	"github.com/asaskevich/govalidator"
)

// This function registers custom validation tags to be used as annotations in struct.
// After registering and adding the annotation, govalidator.ValidateStruct will trigger the validation.
func RegisterCustomValidations(ctx context.Context, logger log.Logger) {
	// This custom validation checks if there's any spaces in the input string.
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
		return hasChar && hasNum
	})
	// This custom validation validates user full name.
	// Only Alphabets with whitespace is allowed.
	govalidator.TagMap["alphawithspace"] = govalidator.Validator(func(full_name string) bool {
		r := regexp.MustCompile("^[a-zA-Z ]*$")
		return r.MatchString(full_name)
	})
	logger.WithCtx(ctx).Info().Msg("Successfully registered custom validations.")
}
