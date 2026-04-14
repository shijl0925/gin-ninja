package app

import (
	"fmt"
	"strings"

	ninja "github.com/shijl0925/gin-ninja"
	admin "github.com/shijl0925/gin-ninja/admin"
	"github.com/shijl0925/gin-ninja/orm"
	"gorm.io/gorm"
)

// NewAdminSite returns the example admin site mounted by the full demo app.
func NewAdminSite() *admin.Site {
	site := admin.NewSite(admin.WithPermissionChecker(requireAuthenticatedAdmin))
	site.MustRegisterModel(&admin.ModelResource{
		Model:        User{},
		ListFields:   []string{"id", "name", "email", "age", "is_admin", "createdAt", "updatedAt"},
		DetailFields: []string{"id", "name", "email", "age", "is_admin", "role_ids", "createdAt", "updatedAt"},
		CreateFields: []string{"name", "email", "password", "age", "is_admin", "role_ids"},
		UpdateFields: []string{"name", "email", "password", "age", "is_admin", "role_ids"},
		FilterFields: []string{"is_admin", "age", "createdAt"},
		SortFields:   []string{"id", "name", "email", "age", "is_admin", "createdAt", "updatedAt"},
		SearchFields: []string{"name", "email"},
		QueryScope: func(ctx *ninja.Context, db *gorm.DB) *gorm.DB {
			return db.Preload("Roles")
		},
		FieldOptions: map[string]admin.FieldOptions{
			"email":    {Component: "email"},
			"password": {Component: "password", Create: boolPtr(true), Update: boolPtr(true)},
			"role_ids": {
				Label: "Roles",
				Relation: &admin.RelationOptions{
					Resource:     "roles",
					ValueField:   "id",
					LabelField:   "name",
					SearchFields: []string{"name", "code", "remark"},
				},
			},
		},
		BeforeCreate: func(ctx *ninja.Context, values map[string]any) error {
			return normalizeAdminUserValues(values, true)
		},
		AfterCreate: func(ctx *ninja.Context, created any) error {
			user, ok := created.(*User)
			if !ok {
				return nil
			}
			return syncAdminUserRoles(ctx, user, user.RoleIDs)
		},
		BeforeUpdate: func(ctx *ninja.Context, current any, values map[string]any) error {
			if err := normalizeAdminUserValues(values, false); err != nil {
				return err
			}
			user, ok := current.(*User)
			if !ok {
				return nil
			}
			roleIDs, exists := values["role_ids"]
			if !exists {
				return nil
			}
			delete(values, "role_ids")
			normalizedRoleIDs, err := normalizeAdminRoleIDs(roleIDs)
			if err != nil {
				return err
			}
			return syncAdminUserRoles(ctx, user, normalizedRoleIDs)
		},
		AfterUpdate: func(ctx *ninja.Context, current any) error {
			user, ok := current.(*User)
			if !ok {
				return nil
			}
			if err := orm.WithContext(ctx.Context).Preload("Roles").First(user, user.ID).Error; err != nil {
				return err
			}
			user.syncRoleIDs()
			return nil
		},
	})
	site.MustRegisterModel(&admin.ModelResource{
		Model:        Role{},
		ListFields:   []string{"id", "name", "code", "status", "createdAt", "updatedAt"},
		DetailFields: []string{"id", "name", "code", "status", "remark", "createdAt", "updatedAt"},
		CreateFields: []string{"name", "code", "status", "remark"},
		UpdateFields: []string{"name", "code", "status", "remark"},
		FilterFields: []string{"status", "name", "code"},
		SortFields:   []string{"id", "name", "code", "status", "createdAt", "updatedAt"},
		SearchFields: []string{"name", "code", "remark"},
	})
	site.MustRegisterModel(&admin.ModelResource{
		Model:        Project{},
		ListFields:   []string{"id", "title", "owner_id", "createdAt", "updatedAt"},
		DetailFields: []string{"id", "title", "summary", "owner_id", "createdAt", "updatedAt"},
		CreateFields: []string{"title", "summary", "owner_id"},
		UpdateFields: []string{"title", "summary", "owner_id"},
		FilterFields: []string{"id"},
		SearchFields: []string{"title", "summary"},
		SortFields:   []string{"id", "title", "owner_id", "createdAt", "updatedAt"},
		RowPermissions: admin.RowPermissionFunc(func(ctx *ninja.Context, action admin.Action, resource *admin.Resource, db *gorm.DB) *gorm.DB {
			return db.Where("owner_id = ?", ctx.GetUserID())
		}),
	})
	return site
}

func requireAuthenticatedAdmin(ctx *ninja.Context, action admin.Action, resource *admin.Resource) error {
	if ctx.GetUserID() == 0 {
		return ninja.UnauthorizedError()
	}
	return nil
}

func normalizeAdminUserValues(values map[string]any, requirePassword bool) error {
	if name, ok := values["name"].(string); ok {
		values["name"] = strings.TrimSpace(name)
	}
	if email, ok := values["email"].(string); ok {
		values["email"] = strings.TrimSpace(strings.ToLower(email))
	}
	if roleIDs, ok := values["role_ids"]; ok {
		normalized, err := normalizeAdminRoleIDs(roleIDs)
		if err != nil {
			return err
		}
		values["role_ids"] = normalized
	}

	password, hasPassword := values["password"]
	if !hasPassword {
		if requirePassword {
			return ninja.NewErrorWithCode(400, "BAD_REQUEST", "field \"password\" is required")
		}
		return nil
	}
	passwordText, ok := password.(string)
	if !ok {
		return ninja.NewErrorWithCode(400, "BAD_REQUEST", "field \"password\" must be a string")
	}
	if len(passwordText) < 8 {
		return ninja.NewErrorWithCode(400, "BAD_REQUEST", "field \"password\" must be at least 8 characters")
	}
	values["password"] = hashPassword(passwordText)
	return nil
}

func normalizeAdminRoleIDs(raw any) ([]uint, error) {
	if raw == nil {
		return []uint{}, nil
	}
	var roleIDs []uint
	switch ids := raw.(type) {
	case []uint:
		roleIDs = append(roleIDs, ids...)
	case []int:
		for _, id := range ids {
			if id < 0 {
				return nil, ninja.NewErrorWithCode(400, "BAD_REQUEST", "field \"role_ids\" must not contain negative values")
			}
			roleIDs = append(roleIDs, uint(id))
		}
	case []float64:
		for _, id := range ids {
			if id < 0 || id != float64(uint(id)) {
				return nil, ninja.NewErrorWithCode(400, "BAD_REQUEST", "field \"role_ids\" must contain whole unsigned integers")
			}
			roleIDs = append(roleIDs, uint(id))
		}
	case []any:
		for _, item := range ids {
			switch id := item.(type) {
			case uint:
				roleIDs = append(roleIDs, id)
			case int:
				if id < 0 {
					return nil, ninja.NewErrorWithCode(400, "BAD_REQUEST", "field \"role_ids\" must not contain negative values")
				}
				roleIDs = append(roleIDs, uint(id))
			case float64:
				if id < 0 || id != float64(uint(id)) {
					return nil, ninja.NewErrorWithCode(400, "BAD_REQUEST", "field \"role_ids\" must contain whole unsigned integers")
				}
				roleIDs = append(roleIDs, uint(id))
			default:
				return nil, ninja.NewErrorWithCode(400, "BAD_REQUEST", "field \"role_ids\" must be an array of unsigned integers")
			}
		}
	default:
		return nil, ninja.NewErrorWithCode(400, "BAD_REQUEST", "field \"role_ids\" must be an array of unsigned integers")
	}
	seen := make(map[uint]struct{}, len(roleIDs))
	normalized := make([]uint, 0, len(roleIDs))
	for _, id := range roleIDs {
		if id == 0 {
			return nil, ninja.NewErrorWithCode(400, "BAD_REQUEST", "field \"role_ids\" must not contain zero")
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		normalized = append(normalized, id)
	}
	return normalized, nil
}

func syncAdminUserRoles(ctx *ninja.Context, user *User, roleIDs []uint) error {
	if user == nil {
		return nil
	}
	db := orm.WithContext(ctx.Context)
	if len(roleIDs) == 0 {
		if err := db.Model(user).Association("Roles").Replace([]Role{}); err != nil {
			return err
		}
		user.Roles = nil
		user.RoleIDs = []uint{}
		return nil
	}

	var roles []Role
	if err := db.Where("id IN ?", roleIDs).Find(&roles).Error; err != nil {
		return err
	}
	roleByID := make(map[uint]Role, len(roles))
	for _, role := range roles {
		roleByID[role.ID] = role
	}
	orderedRoles := make([]Role, 0, len(roleIDs))
	for _, id := range roleIDs {
		role, ok := roleByID[id]
		if !ok {
			return ninja.NewErrorWithCode(400, "BAD_REQUEST", fmt.Sprintf("role %d does not exist", id))
		}
		orderedRoles = append(orderedRoles, role)
	}
	if err := db.Model(user).Association("Roles").Replace(orderedRoles); err != nil {
		return err
	}
	user.Roles = orderedRoles
	user.RoleIDs = append([]uint(nil), roleIDs...)
	return nil
}

func boolPtr(v bool) *bool { return &v }
