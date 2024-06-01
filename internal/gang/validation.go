// All custom validations related to gang entity in Popcorn are defined here.

package gang

import (
	"Popcorn/internal/entity"
	"Popcorn/internal/errors"
	"Popcorn/pkg/log"
	"context"
	"regexp"

	"github.com/asaskevich/govalidator"
)

func RegisterCustomValidationTags(ctx context.Context, logger log.Logger) {
	// Gang name validation.
	// Gang name can only contain letters, numbers, underscore, periods and spaces.
	govalidator.TagMap["gangname_custom"] = govalidator.Validator(func(str string) bool {
		pattern := regexp.MustCompile("[^a-zA-Z0-9_. ]")
		return !pattern.MatchString(str) && !govalidator.HasWhitespaceOnly(str)
	})

	logger.WithCtx(ctx).Info().Msg("Successfully registered gang related custom validations.")
}

func canUpdateGangContentRelatedData(existingGangData entity.GangResponse, gang *entity.Gang) bool {
	return !existingGangData.Streaming && !((existingGangData.ContentID != "" && gang.ContentURL != "") ||
		(existingGangData.ContentID != "" && gang.ContentScreenShare) ||
		(existingGangData.ContentURL != "" && gang.ContentScreenShare) ||
		(existingGangData.ContentURL != "" && gang.ContentID != "") ||
		(existingGangData.ContentScreenShare && gang.ContentID != "") ||
		(existingGangData.ContentScreenShare && gang.ContentURL != ""))
}

func validateGangData(_ context.Context, gang interface{}) error {
	_, valerr := govalidator.ValidateStruct(gang)
	if valerr != nil {
		valerr := valerr.(govalidator.Errors).Errors()
		return errors.GenerateValidationErrorResponse(valerr)
	}
	return nil
}

func validateGangInviteData(ctx context.Context, inviteData *entity.GangInvite) error {
	// Either invite hashcode should be there in the request json or the default body
	if len(inviteData.InviteHashCode) == 0 && len(inviteData.Name) == 0 {
		return errors.BadRequest("Blank join request")
	}
	return validateGangData(ctx, inviteData)
}
