// Gang repository encapsulates the data access logic (interactions with the DB) related to Gang CRUD in Popcorn.

package gang

import "Popcorn/pkg/db"

type Repository interface {
}

// repository struct of gang Repository.
// Object of this will be passed around from main to internal.
// Helps to access the repository layer interface and call methods.
type repository struct {
	db *db.RedisDB
}

// Returns a new instance of gang repository for other packages to access its interface.
func NewRepository(dbwrp *db.RedisDB) Repository {
	return repository{db: dbwrp}
}
