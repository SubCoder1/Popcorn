// Structure of User Model in Popcorn.

package entity

import (
	"math/rand"
	"time"
)

// Saved in DB as user:<gang_name>
type User struct {
	Username   string `json:"username" redis:"username" valid:"required,type(string),printableascii,stringlength(5|30),username_custom~username:Invalid Username"`
	FullName   string `json:"full_name" redis:"full_name" valid:"required,type(string),stringlength(5|30),ascii,fullname_custom~full_name:Invalid Fullname"`
	Password   string `json:"password,omitempty" redis:"password" valid:"required,type(string),stringlength(5|730),nospace~password:Cannot contain whitespace,pwdstrength~password:At least 1 letter and 1 number is mandatory"`
	ProfilePic string `json:"user_profile_pic,omitempty" redis:"user_profile_pic" valid:"-"`
}

// Used to bind and validate user_login request
type UserLogin struct {
	Username string `json:"username" valid:"required,type(string),printableascii,stringlength(5|30),username_custom~username:Invalid Username"`
	Password string `json:"password" valid:"required,type(string),minstringlength(5),nospace~password:Cannot contain whitespace,pwdstrength~password:At least 1 letter and 1 number is mandatory"`
}

// Used to validate search_user request
type UserSearch struct {
	Username string `valid:"required,type(string),printableascii,stringlength(1|30),username_custom~username:Invalid Username"`
	Cursor   int    `valid:"-"`
}

// Randomly sets user's profile pic during login/register
func (u User) SelectProfilePic() string {
	profiles := []string{
		"alien.png",
		"batman.png",
		"cyclops.png",
		"dead.png",
		"devil.png",
		"doll.png",
		"dracula.png",
		"frankenstein.png",
		"ghost.png",
		"grim-reaper.png",
		"joker.png",
		"mummy.png",
		"murderer.png",
		"ninja.png",
		"orc.png",
		"pirate.png",
		"prisoner.png",
		"robber.png",
		"thief.png",
		"witch.png",
		"zombie.png",
		"spiderman.png",
		"thanos.png",
		"bane.png",
	}
	r := rand.New(rand.NewSource(time.Now().Unix())) // initialize global pseudo random generator
	return profiles[r.Intn(len(profiles))]
}
