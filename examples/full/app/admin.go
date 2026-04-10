package app

import (
	"strings"

	ninja "github.com/shijl0925/gin-ninja"
	admin "github.com/shijl0925/gin-ninja/admin"
	"gorm.io/gorm"
)

// NewAdminSite returns the example admin site mounted by the full demo app.
func NewAdminSite() *admin.Site {
	site := admin.NewSite(admin.WithPermissionChecker(requireAuthenticatedAdmin))
	site.MustRegister(&admin.Resource{
		Name:         "users",
		Label:        "Users",
		Path:         "/users",
		Model:        User{},
		ListFields:   []string{"id", "name", "email", "age", "is_admin", "createdAt", "updatedAt"},
		DetailFields: []string{"id", "name", "email", "age", "is_admin", "createdAt", "updatedAt"},
		CreateFields: []string{"name", "email", "password", "age", "is_admin"},
		UpdateFields: []string{"name", "email", "password", "age", "is_admin"},
		FilterFields: []string{"is_admin", "age", "createdAt"},
		SortFields:   []string{"id", "name", "email", "age", "is_admin", "createdAt", "updatedAt"},
		SearchFields: []string{"name", "email"},
		FieldOptions: map[string]admin.FieldOptions{
			"email":    {Component: "email"},
			"password": {Component: "password", Create: boolPtr(true), Update: boolPtr(true)},
		},
		BeforeCreate: func(ctx *ninja.Context, values map[string]any) error {
			return normalizeAdminUserValues(values, true)
		},
		BeforeUpdate: func(ctx *ninja.Context, current any, values map[string]any) error {
			return normalizeAdminUserValues(values, false)
		},
	})
	site.MustRegister(&admin.Resource{
		Name:         "projects",
		Label:        "Projects",
		Path:         "/projects",
		Model:        Project{},
		ListFields:   []string{"id", "title", "owner_id", "createdAt", "updatedAt"},
		DetailFields: []string{"id", "title", "summary", "owner_id", "createdAt", "updatedAt"},
		CreateFields: []string{"title", "summary", "owner_id"},
		UpdateFields: []string{"title", "summary", "owner_id"},
		SearchFields: []string{"title", "summary"},
		SortFields:   []string{"id", "title", "owner_id", "createdAt", "updatedAt"},
		FieldOptions: map[string]admin.FieldOptions{
			"owner_id": {
				Relation: &admin.RelationOptions{
					Resource:     "users",
					ValueField:   "id",
					LabelField:   "name",
					SearchFields: []string{"name", "email"},
				},
			},
		},
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

func boolPtr(v bool) *bool { return &v }
