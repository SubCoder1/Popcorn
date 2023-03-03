// Structure of Gang Model in Popcorn.

package entity

// Information structure of Gangs in Popcorn.
// Saved in DB as gang:<this.Admin>.
type Gang struct {
	Admin          string `json:"gang_admin,omitempty" redis:"gang_admin" valid:"-"`
	Name           string `json:"gang_name" redis:"gang_name" valid:"required,type(string),printableascii,stringlength(5|20),gangname_custom~gang_name:Invalid Gang Name"`
	PassKey        string `json:"gang_pass_key" redis:"gang_pass_key" valid:"required,type(string),stringlength(5|730),nospace~gang_pass_key:Cannot contain whitespace"`
	Limit          uint   `json:"gang_member_limit" redis:"gang_member_limit" valid:"required,range(2|10)"`
	MembersListKey string `json:"gang_members_key,omitempty" redis:"gang_members_key" valid:"-"`
	Created        int64  `json:"gang_created,omitempty" redis:"gang_created" valid:"-"`
}

// Response structure of Gangs in Popcorn, typically used in get methods.
// Used to send gang data to client.
type GangResponse struct {
	Admin          string `json:"gang_admin,omitempty" redis:"gang_admin"`
	Name           string `json:"gang_name" redis:"gang_name"`
	Limit          uint   `json:"gang_member_limit" redis:"gang_member_limit"`
	IsAdmin        bool   `json:"is_admin"`
	Count          int    `json:"gang_members_count"`
	Created        int64  `json:"gang_created,omitempty" redis:"gang_created"`
	CreatedTimeAgo string `json:"gang_created_timeago"`
}

// Saved in DB as gang-members:<members>.
type GangMembersList struct {
	MembersList []string `json:"gang_member_list,omitempty" redis:"gang_member_list" valid:"-"`
}

// Used to bind and validate join_gang request.
type GangJoin struct {
	Admin   string `json:"gang_admin" valid:"required,type(string),printableascii,stringlength(5|30),username_custom~admin:No spaces allowed here"`
	Name    string `json:"gang_name" valid:"required,type(string),printableascii,stringlength(5|20),gangname_custom~gang_name:Invalid Gang Name"`
	Key     string `json:"-" valid:"-"`
	PassKey string `json:"gang_pass_key" valid:"required,type(string),stringlength(5|730),nospace~gang_pass_key:Cannot contain whitespace"`
}

// Used to bind and validate search_gang request.
type GangSearch struct {
	Name   string `valid:"required,type(string),printableascii,stringlength(1|20),gangname_custom~Name:Invalid Gang Name"`
	Cursor int    `valid:"-"`
}

// Information structure of Gang invitation in Popcorn.
// Gang-invites are stored in user's gang-invites:<username> DB set.
// GangInvite is stored in the format <GangInvite.Admin>:<GangInvite.Name>:<Created_UNIX_Timestamp>
// Created_UNIX_Timestamp is converted into GangInvite.CreatedTimeAgo
type GangInvite struct {
	Admin          string `json:"gang_admin,omitempty" valid:"-"`
	Name           string `json:"gang_name" valid:"required,type(string),printableascii,stringlength(5|20),gangname_custom~gang_name:Invalid Gang Name"`
	For            string `json:"gang_invite_for,omitempty" valid:"required,type(string),printableascii,stringlength(5|30),username_custom~username:Invalid Username"`
	CreatedTimeAgo string `json:"invite_sent_timeago,omitempty" valid:"-"`
}

// Used to bind and validate boot_member or leave_gang request.
type GangExit struct {
	Member string `json:"member_name" valid:"required,type(string),printableascii,stringlength(5|20),username_custom~username:Invalid Username"`
	Name   string `json:"gang_name" valid:"required,type(string),printableascii,stringlength(5|20),gangname_custom~gang_name:Invalid Gang Name"`
	Key    string `json:"-" valid:"-"`
	Type   string `json:"-" valid:"in(leave|boot)"` // leave request from user, boot request from gang admin
}
