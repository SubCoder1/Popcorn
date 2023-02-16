// All global custom validations in Popcorn are defined here.
// These validations are allowed to be used anywhere in the application.

package validations

import (
	"Popcorn/pkg/log"
	"context"

	"github.com/asaskevich/govalidator"
)

func RegisterCustomValidations(ctx context.Context, logger log.Logger) {
	// This global validation doesn't allow whitespace in input.
	govalidator.TagMap["nospace"] = govalidator.Validator(func(str string) bool {
		return !govalidator.HasWhitespace(str)
	})
}
