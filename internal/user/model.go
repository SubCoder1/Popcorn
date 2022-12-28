// Structure of User Model in Popcorn.

package user

type User struct {
	ID       string `json:"id,omitempty"`
	Username string `json:"username" validate:"required,ascii,min=5,max=20,excludesall=' '"`
	Password string `json:"password" validate:"required,min=6"`
}
