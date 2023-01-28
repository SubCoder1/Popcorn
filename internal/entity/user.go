// Structure of User Model in Popcorn.

package entity

import (
	"math/rand"
	"time"
)

// Saved in DB as user:<gang_name>
type User struct {
	Username   string `json:"username" redis:"username" valid:"required,type(string),printableascii,stringlength(5|20),username_custom~username:Invalid Username"`
	FullName   string `json:"full_name,omitempty" redis:"full_name" valid:"type(string),stringlength(5|30),ascii,fullname_custom~full_name:Invalid Fullname,optional"`
	Password   string `json:"password,omitempty" redis:"password" valid:"required,type(string),minstringlength(5),pwdstrength~password:At least 1 letter and 1 number is mandatory"`
	ProfilePic string `json:"user_profile_pic,omitempty" redis:"user_profile_pic" valid:"-"`
}

// Used to validate search_user request
type UserSearch struct {
	Username string `valid:"required,type(string),printableascii,stringlength(1|20),username_custom~username:Invalid Username"`
	Cursor   int    `valid:"-"`
}

// Randomly sets user's profile pic during login/register
func (u User) SelectProfilePic() string {
	profiles := []string{
		"cyclops.png",
		"dead-cat.png",
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
		"thief.png",
		"witch.png",
		"zombie.png",
	}
	rand.Seed(time.Now().Unix()) // initialize global pseudo random generator
	return profiles[rand.Intn(len(profiles))]
}
