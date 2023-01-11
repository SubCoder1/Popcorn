// Service layer of the internal package gang.

package gang

import (
	"Popcorn/internal/user"
	"Popcorn/pkg/log"
)

// Service layer of internal package gang which encapsulates gang CRUD logic of Popcorn.
type Service interface {
}

// Object of this will be passed around from main to routers to API.
// Helps to access the service layer interface and call methods.
// Also helps to pass objects to be used from outer layer.
type service struct {
	gangRepo Repository
	userRepo user.Repository
	logger   log.Logger
}

// Helps to access the service layer interface and call methods. Service object is passed from main.
func NewService(gangRepo Repository, userRepo user.Repository, logger log.Logger) Service {
	return service{gangRepo, userRepo, logger}
}
