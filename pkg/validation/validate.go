// Handles all sorts of custom data validations happening in Popcorn.

package validation

import (
	"strings"
	"unicode"

	"github.com/asaskevich/govalidator"
)

// This function registers custom validation tags to be used as annotations in struct.
// After registering and adding the annotation, govalidator.ValidateStruct will trigger the validation.
func RegisterCustomValidations() {
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
}
