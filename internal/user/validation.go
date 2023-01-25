// All custom validations related to user entity in Popcorn are defined here.

package user

import (
	"Popcorn/pkg/log"
	"context"
	"regexp"
	"unicode"

	"github.com/asaskevich/govalidator"
)

func RegisterCustomValidations(ctx context.Context, logger log.Logger) {
	// Password strength validation.
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
	// Username validation.
	// Username can only contain letters, numbers, underscores & periods.
	govalidator.TagMap["username_custom"] = govalidator.Validator(func(str string) bool {
		pattern := regexp.MustCompile("[^a-zA-Z0-9_.]")
		return !pattern.MatchString(str)
	})

	// Fullname validation.
	// Fullname can only contain letters, numbers & spaces.
	govalidator.TagMap["fullname_custom"] = govalidator.Validator(func(str string) bool {
		pattern := regexp.MustCompile("[^a-zA-Z ]")
		return !pattern.MatchString(str) && !govalidator.HasWhitespaceOnly(str)
	})

	logger.WithCtx(ctx).Info().Msg("Successfully registered user related custom validations.")
}
