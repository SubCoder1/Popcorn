// Structure of Gang Model in Popcorn.

package entity

// Saved in DB as gang:<this.Admin>
type Gang struct {
	Admin          string `json:"gang_admin,omitempty" redis:"gang_admin" valid:"-"`
	Name           string `json:"gang_name" redis:"gang_name" valid:"required,type(string),printableascii,stringlength(5|20),nospaceonly~gang_name:Gang Name cannot contain only spaces"`
	PassKey        string `json:"gang_pass_key" redis:"pass_key" valid:"required,type(string),minstringlength(5)"`
	Limit          uint   `json:"gang_member_limit" redis:"gang_limit" valid:"required,range(2|10)"`
	MembersListKey string `json:"gang_members_key,omitempty" valid:"-"`
	Created        int64  `json:"gang_created,omitempty" redis:"gang_created" valid:"-"`
}

// Saved in DB as gang-members:<gang.Admin>
type GangMembersList struct {
	MembersList []string `json:"gang_member_list,omitempty" valid:"-"`
}
