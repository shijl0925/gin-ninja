package app

import "github.com/shijl0925/go-toolkits/gormx"

type userRepo struct {
	gormx.BaseRepo[User]
}

// NewUserRepo creates a concrete user repository instance.
func NewUserRepo() *userRepo {
	return &userRepo{}
}
