// Structure of User Model in Popcorn.

package entity

// Tags: json -> serialization, redis -> serialize from db output, valid -> validations
type User struct {
	Username string `json:"username" redis:"username" valid:"required,type(string),printableascii,stringlength(5|20),nospace~username:No spaces allowed here"`
	Password string `json:"password" redis:"password" valid:"required,type(string),minstringlength(5),pwdstrength~password:At least 1 letter and 1 number is mandatory"`
}