package app

import "github.com/shijl0925/go-toolkits/gormx"

// IUserRepo extends the base gormx repository with the User model.
type IUserRepo interface {
	gormx.IBaseRepo[User]
}

type userRepo struct {
	gormx.BaseRepo[User]
}

// NewUserRepo creates a new IUserRepo instance.
func NewUserRepo() IUserRepo {
	return &userRepo{}
}
