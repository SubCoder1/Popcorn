// All custom validations related to gang entity in Popcorn are defined here.

package user

import (
	"Popcorn/pkg/log"
	"context"
	"unicode"

	"github.com/asaskevich/govalidator"
)

func RegisterCustomValidations(ctx context.Context, logger log.Logger) {
	// This custom validation checks for user-set password strength.
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

	logger.WithCtx(ctx).Info().Msg("Successfully registered user related custom validations.")
}
