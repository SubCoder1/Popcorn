// Structure of User Model in Popcorn.

package entity

// Tags: json -> serialization, redis -> serialize from db output, valid -> validations
type User struct {
	ID       uint64 `json:"id" redis:"id" valid:"-"`
	Username string `json:"username" redis:"username" valid:"required,type(string),printableascii,stringlength(5|20),nospace~username:No spaces allowed here"`
	FullName string `json:"full_name" redis:"full_name" valid:"required,type(string),stringlength(5|30),ascii,alphawithspace~full_name:Couldn't validate Full Name"`
	Password string `json:"password" redis:"password" valid:"required,type(string),minstringlength(5),pwdstrength~password:At least 1 letter and 1 number is mandatory"`
}
