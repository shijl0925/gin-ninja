package app

import (
	"fmt"
	"sync"

	ninja "github.com/shijl0925/gin-ninja"
	"github.com/shijl0925/gin-ninja/pagination"
)

const (
	usersV2CacheNamespace = "users:v2"
)

var usersV2Cache struct {
	mu          sync.RWMutex
	invalidator *ninja.CacheInvalidator
}

// ConfigureUsersV2Cache wires the cache store used by the versioned CRUD cache demo.
func ConfigureUsersV2Cache(store ninja.ResponseCacheStore) {
	usersV2Cache.mu.Lock()
	defer usersV2Cache.mu.Unlock()
	usersV2Cache.invalidator = ninja.NewCacheInvalidator(store)
}

// UsersV2DetailCacheKey pins detail responses to a stable key that write operations can delete directly.
func UsersV2DetailCacheKey(ctx *ninja.Context) string {
	if ctx == nil {
		return ""
	}
	return usersV2DetailCacheKeyByID(ctx.Param("id"))
}

// ListUsersV2 reuses the regular users query logic behind the v2 routes.
func ListUsersV2(ctx *ninja.Context, in *ListUsersInput) (*pagination.Page[User], error) {
	return ListUsers(ctx, in)
}

// GetUserV2 reuses the regular detail query logic behind the cached detail demo.
func GetUserV2(ctx *ninja.Context, in *GetUserInput) (*User, error) {
	return GetUser(ctx, in)
}

// CreateUserV2 creates a user and invalidates cached list responses.
func CreateUserV2(ctx *ninja.Context, in *CreateUserInput) (*User, error) {
	out, err := CreateUser(ctx, in)
	if err != nil {
		return nil, err
	}
	return out, nil
}

// UpdateUserV2 updates a user and invalidates cached list/detail responses.
func UpdateUserV2(ctx *ninja.Context, in *UpdateUserInput) (*User, error) {
	out, err := UpdateUser(ctx, in)
	if err != nil {
		return nil, err
	}
	invalidateUsersV2UserCache(in.UserID)
	return out, nil
}

// DeleteUserV2 deletes a user and invalidates cached list/detail responses.
func DeleteUserV2(ctx *ninja.Context, in *DeleteUserInput) error {
	if err := DeleteUser(ctx, in); err != nil {
		return err
	}
	invalidateUsersV2UserCache(in.UserID)
	return nil
}

func usersV2DetailCacheKeyByID(id interface{}) string {
	return fmt.Sprintf("%s:detail:%v", usersV2CacheNamespace, id)
}

func invalidateUsersV2UserCache(userID uint) {
	invalidator := usersV2CacheInvalidator()
	if invalidator == nil {
		return
	}
	invalidator.Delete(usersV2DetailCacheKeyByID(userID))
}

func usersV2CacheInvalidator() *ninja.CacheInvalidator {
	usersV2Cache.mu.RLock()
	defer usersV2Cache.mu.RUnlock()
	return usersV2Cache.invalidator
}
