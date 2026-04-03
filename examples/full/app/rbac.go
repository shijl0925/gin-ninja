package app

import (
	"errors"
	"sort"

	ninja "github.com/shijl0925/gin-ninja"
	"github.com/shijl0925/gin-ninja/orm"
	"github.com/shijl0925/go-toolkits/gormx"
	"gorm.io/gorm"
)

const (
	PermissionUserList   = "System:User:List"
	PermissionUserDetail = "System:User:Detail"
	PermissionUserCreate = "System:User:Create"
	PermissionUserEdit   = "System:User:Edit"
	PermissionUserDelete = "System:User:Delete"
)

type subjectInfo struct {
	User        User
	Roles       []string
	Permissions []string
}

// SeedRBAC seeds a small RBAC dataset inspired by go-vben-admin:
// roles own permission codes and users are granted roles.
func SeedRBAC(db *gorm.DB) error {
	permissions := []Permission{
		{Name: "List users", Code: PermissionUserList, Description: "View the user list"},
		{Name: "View user detail", Code: PermissionUserDetail, Description: "View a single user"},
		{Name: "Create user", Code: PermissionUserCreate, Description: "Create a user"},
		{Name: "Edit user", Code: PermissionUserEdit, Description: "Update a user"},
		{Name: "Delete user", Code: PermissionUserDelete, Description: "Delete a user"},
	}
	for _, permission := range permissions {
		if err := firstOrCreateByCode(db, &permission); err != nil {
			return err
		}
	}

	roleSeeds := []struct {
		Role  Role
		Codes []string
	}{
		{Role: Role{Name: "Super Admin", Code: "SUPER_ADMIN", Description: "Owns every user permission"}, Codes: []string{
			PermissionUserList, PermissionUserDetail, PermissionUserCreate, PermissionUserEdit, PermissionUserDelete,
		}},
		{Role: Role{Name: "User Manager", Code: "USER_MANAGER", Description: "Can manage users except deletion"}, Codes: []string{
			PermissionUserList, PermissionUserDetail, PermissionUserCreate, PermissionUserEdit,
		}},
		{Role: Role{Name: "Auditor", Code: "AUDITOR", Description: "Read-only access to users"}, Codes: []string{
			PermissionUserList, PermissionUserDetail,
		}},
	}

	for _, seed := range roleSeeds {
		role := seed.Role
		if err := firstOrCreateByCode(db, &role); err != nil {
			return err
		}
		var granted []Permission
		if err := db.Where("code IN ?", seed.Codes).Find(&granted).Error; err != nil {
			return err
		}
		if err := db.Model(&role).Association("Permissions").Replace(granted); err != nil {
			return err
		}
	}

	users := []struct {
		Name    string
		Email   string
		Age     int
		IsAdmin bool
		Role    string
	}{
		{Name: "Admin", Email: "admin@example.com", Age: 30, IsAdmin: true, Role: "SUPER_ADMIN"},
		{Name: "Manager", Email: "manager@example.com", Age: 28, IsAdmin: false, Role: "USER_MANAGER"},
		{Name: "Auditor", Email: "auditor@example.com", Age: 26, IsAdmin: false, Role: "AUDITOR"},
	}

	for _, seedUser := range users {
		var user User
		err := db.Where("email = ?", seedUser.Email).First(&user).Error
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			user = User{
				Name:     seedUser.Name,
				Email:    seedUser.Email,
				Password: hashPassword("password123"),
				Age:      seedUser.Age,
				IsAdmin:  seedUser.IsAdmin,
			}
			if err := db.Create(&user).Error; err != nil {
				return err
			}
		case err != nil:
			return err
		}

		var role Role
		if err := db.Where("code = ?", seedUser.Role).First(&role).Error; err != nil {
			return err
		}
		if err := db.Model(&user).Association("Roles").Replace([]Role{role}); err != nil {
			return err
		}
	}

	return nil
}

// ResolvePermissions loads the current user's effective permission codes.
func ResolvePermissions(ctx *ninja.Context) ([]string, error) {
	info, err := loadSubjectInfo(ctx, ctx.GetUserID())
	if err != nil {
		return nil, err
	}
	return info.Permissions, nil
}

// GetCurrentSubject returns the authenticated user's current RBAC context.
func GetCurrentSubject(ctx *ninja.Context, _ *struct{}) (*CurrentSubjectOut, error) {
	info, err := loadSubjectInfo(ctx, ctx.GetUserID())
	if err != nil {
		return nil, err
	}
	return &CurrentSubjectOut{
		User:        toUserOut(info.User),
		Roles:       info.Roles,
		Permissions: info.Permissions,
	}, nil
}

func loadSubjectInfo(ctx *ninja.Context, userID uint) (*subjectInfo, error) {
	if userID == 0 {
		return nil, ninja.ErrUnauthorized
	}

	var user User
	var db *gorm.DB
	if ctx != nil {
		db = orm.GetDB(ctx.Context).WithContext(ctx.Request.Context())
	} else {
		db = gormx.GetDb()
	}
	if err := db.Preload("Roles.Permissions").First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ninja.ErrUnauthorized
		}
		return nil, err
	}

	roleSet := make(map[string]struct{})
	permissionSet := make(map[string]struct{})
	for _, role := range user.Roles {
		roleSet[role.Code] = struct{}{}
		for _, permission := range role.Permissions {
			permissionSet[permission.Code] = struct{}{}
		}
	}

	roles := sortedKeys(roleSet)
	permissions := sortedKeys(permissionSet)
	return &subjectInfo{User: user, Roles: roles, Permissions: permissions}, nil
}

func sortedKeys(set map[string]struct{}) []string {
	items := make([]string, 0, len(set))
	for key := range set {
		items = append(items, key)
	}
	sort.Strings(items)
	return items
}

func firstOrCreateByCode[T interface{ GetCode() string }](db *gorm.DB, item T) error {
	return db.Where("code = ?", item.GetCode()).FirstOrCreate(item).Error
}

func (r *Role) GetCode() string       { return r.Code }
func (p *Permission) GetCode() string { return p.Code }
