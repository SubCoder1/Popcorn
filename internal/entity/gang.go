// Structure of Gang Model in Popcorn.

package entity

// Information structure of Gangs in Popcorn.
// Saved in DB as gang:<Gang.Admin>.
type Gang struct {
	// Admin of the gang, i.e., the creator who has the control of doing anything in the Gang.
	Admin string `json:"gang_admin,omitempty" redis:"gang_admin" valid:"-"`
	// Name of the gang, can be a duplicate.
	Name string `json:"gang_name" redis:"gang_name" valid:"required,type(string),printableascii,stringlength(5|20),gangname_custom~gang_name:Invalid Gang Name"`
	// Passkey of the gang making it private.
	PassKey string `json:"gang_pass_key" redis:"gang_pass_key" valid:"required,type(string),stringlength(5|72),nospace~gang_pass_key:Cannot contain whitespace"`
	// Gang Member Limit, minimum 2 and maximum 10.
	Limit uint `json:"gang_member_limit" redis:"gang_member_limit" valid:"required,range(2|10)"`
	// Consider this as a Foreign key to 'GangMembersList' struct, which keeps a list of all the members currently in this gang.
	MembersListKey string `json:"gang_members_key,omitempty" redis:"gang_members_key" valid:"-"`
	// Gang Timestamp.
	Created int64 `json:"gang_created,omitempty" redis:"gang_created" valid:"-"`
	// Gang Content filename.
	ContentName string `json:"-" redis:"gang_content_name" valid:"-"`
	// Gang Content file ID.
	ContentID string `json:"-" redis:"gang_content_ID" valid:"-"`
	// Gang Content URL.
	ContentURL string `json:"gang_content_url" redis:"gang_content_url" valid:"url,optional"`
	// Gang Screen Share.
	ContentScreenShare bool `json:"gang_screen_share" redis:"gang_screen_share" valid:"-"`
	// Gang Stream status.
	Streaming bool `json:"-" redis:"gang_streaming" valid:"-"`
	// Gang invite hash which decyphers to gang:<Gang.Admin>:<Gang.Name>.
	InviteHashCode string `json:"-" redis:"gang_invite_hashcode" valid:"-"`
}

// Response structure of Gangs in Popcorn, typically used in get methods.
// Used to send gang data to client.
type GangResponse struct {
	Admin              string `json:"gang_admin,omitempty" redis:"gang_admin"`
	Name               string `json:"gang_name" redis:"gang_name"`
	Limit              uint   `json:"gang_member_limit" redis:"gang_member_limit"`
	IsAdmin            bool   `json:"is_admin"`
	Count              int    `json:"gang_members_count"`
	Created            int64  `json:"gang_created,omitempty" redis:"gang_created"`
	ContentName        string `json:"gang_content_name" redis:"gang_content_name"`
	ContentID          string `json:"gang_content_ID" redis:"gang_content_ID"`
	ContentURL         string `json:"gang_content_url" redis:"gang_content_url"`
	ContentScreenShare bool   `json:"gang_screen_share" redis:"gang_screen_share"`
	Streaming          bool   `json:"gang_streaming" redis:"gang_streaming"`
	InviteHashCode     string `json:"gang_invite_hashcode" redis:"gang_invite_hashcode"`
}

// Saved in DB as gang-members:<members>.
type GangMembersList struct {
	// Stores the list of gang members currently in the gang.
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
type GangInvite struct {
	Admin          string `json:"gang_admin,omitempty" valid:"-"`
	Name           string `json:"gang_name" valid:"type(string),printableascii,stringlength(5|20),gangname_custom~gang_name:Invalid Gang Name"`
	For            string `json:"gang_invite_for,omitempty" valid:"type(string),printableascii,stringlength(5|30),username_custom~username:Invalid Username"`
	InviteHashCode string `json:"gang_invite_hashcode,omitempty" valid:"type(string),stringlength(10|730)"`
	CreatedTimeAgo int64  `json:"invite_sent_timeago,omitempty" valid:"-"`
}

// Used to bind and validate boot_member or leave_gang request.
type GangExit struct {
	Member string `json:"member_name" valid:"required,type(string),printableascii,stringlength(5|30),username_custom~username:Invalid Username"`
	Name   string `json:"gang_name" valid:"required,type(string),printableascii,stringlength(5|20),gangname_custom~gang_name:Invalid Gang Name"`
	Key    string `json:"-" valid:"-"`
	// leave request from user, boot request from gang admin
	Type string `json:"-" valid:"in(leave|boot)"`
}

// Used to bind and validate incoming gang conversations in Popcorn.
type GangMessage struct {
	Message string `json:"message" valid:"required,type(string)"`
}

type LivekitConfig struct {
	// Host url of livekit cloud
	Host string
	// api key required for livekit authentication
	ApiKey string
	// api secret required for livekit authentication
	ApiSecret string
	// identity who's trying to access livekit helpers
	Identity string
	// optional content file ID for uploading track
	Content string
	// optional livekit room name
	RoomName string
	// Livekit concurrent ingress limit
	MaxConcurrentIngressLimit int
	// Livekit max screenshare hours
	MaxScreenShareHours int
}
