// Structure of Gang Model in Popcorn.

package entity

// Saved in DB as gang:<this.Admin>
type Gang struct {
	Admin          string `json:"gang_admin,omitempty" redis:"gang_admin" valid:"-"`
	Name           string `json:"gang_name" redis:"gang_name" valid:"required,type(string),printableascii,stringlength(5|20),nospaceonly~gang_name:Gang Name cannot contain only spaces"`
	PassKey        string `json:"gang_pass_key" redis:"gang_pass_key" valid:"required,type(string),minstringlength(5)"`
	Limit          uint   `json:"gang_member_limit" redis:"gang_member_limit" valid:"required,range(2|10)"`
	MembersListKey string `json:"gang_members_key,omitempty" redis:"gang_members_key" valid:"-"`
	Created        int64  `json:"gang_created,omitempty" redis:"gang_created" valid:"-"`
}

// Saved in DB as gang-members:<members>
type GangMembersList struct {
	MembersList []string `json:"gang_member_list,omitempty" redis:"gang_member_list" valid:"-"`
}

// Used to bind and validate join_gang request
type GangKey struct {
	Admin string `json:"gang_admin,omitempty" valid:"required,type(string),printableascii,stringlength(5|20),nospace~username:No spaces allowed here"`
	Name  string `json:"gang_name" valid:"required,type(string),printableascii,stringlength(5|20),nospaceonly~gang_name:Gang Name cannot contain only spaces"`
	Key   string `json:"gang_key,omitempty" valid:"-"`
}

// Used to bind and validate search_gang request
type GangSearch struct {
	Name   string `valid:"required,type(string),printableascii,stringlength(1|20),nospaceonly~gang_name:Gang Name cannot contain only spaces"`
	Cursor int    `valid:"-"`
}
