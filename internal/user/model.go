// Structure of User Model in Popcorn.

package user

type User struct {
	ID       string `json:"id" valid:"-"`
	Username string `json:"username" valid:"required,type(string),printableascii,stringlength(5|20),nospace~username:No spaces allowed here"`
	Password string `json:"password" valid:"required,minstringlength(5),pwdstrength~password:At least 1 letter and 1 number is mandatory"`
}
