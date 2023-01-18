// Structure of User Model in Popcorn.

package entity

import (
	"math/rand"
	"time"
)

// Tags: json -> serialization, redis -> serialize from db output, valid -> validations
type User struct {
	Username   string `json:"username" redis:"username" valid:"required,type(string),printableascii,stringlength(5|20),nospace~username:No spaces allowed here"`
	FullName   string `json:"full_name" redis:"full_name" valid:"type(string),stringlength(5|30),ascii,alphawithspace~full_name:Couldn't validate Full Name,optional"`
	Password   string `json:"password" redis:"password" valid:"required,type(string),minstringlength(5),pwdstrength~password:At least 1 letter and 1 number is mandatory"`
	ProfilePic string `json:"user_profile_pic,omitempty" redis:"user_profile_pic" valid:"-"`
}

// Randomly sets user's profile pic during login/register
func (u User) SelectProfilePic() string {
	profiles := []string{
		"alien.png",
		"death.png",
		"devil.png",
		"dracula.png",
		"frankenstein.png",
		"maniac.png",
		"mummy.png",
		"orc.png",
		"spider.png",
		"witch.png",
		"zombie.png",
	}
	rand.Seed(time.Now().Unix()) // initialize global pseudo random generator
	return profiles[rand.Intn(len(profiles))]
}
