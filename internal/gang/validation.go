// All custom validations related to gang entity in Popcorn are defined here.

package gang

import (
	"Popcorn/pkg/log"
	"context"
	"regexp"

	"github.com/asaskevich/govalidator"
)

func RegisterCustomValidations(ctx context.Context, logger log.Logger) {
	// Gang name validation.
	// Gang name can only contain letters, numbers, underscore, periods and spaces.
	govalidator.TagMap["gangname_custom"] = govalidator.Validator(func(str string) bool {
		pattern := regexp.MustCompile("[^a-zA-Z0-9_. ]")
		return !pattern.MatchString(str) && !govalidator.HasWhitespaceOnly(str)
	})

	logger.WithCtx(ctx).Info().Msg("Successfully registered gang related custom validations.")
}
