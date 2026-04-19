package app

import "github.com/shijl0925/go-toolkits/gormx"

type IUserRepo interface {
	gormx.IBaseRepo[User]
}

type userRepo struct {
	gormx.BaseRepo[User]
}

// NewUserRepo creates a concrete user repository instance.
func NewUserRepo() IUserRepo {
	return &userRepo{}
}
