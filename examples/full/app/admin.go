package app

import (
	ninja "github.com/shijl0925/gin-ninja"
	admin "github.com/shijl0925/gin-ninja/admin"
	"gorm.io/gorm"
)

// NewAdminSite returns the example admin site mounted by the full demo app.
func NewAdminSite() *admin.Site {
	site := admin.NewSite(admin.WithPermissionChecker(requireAuthenticatedAdmin))
	site.MustRegisterModel(&admin.ModelResource{
		Model:        User{},
		Preloads:     []string{"Roles"},
		ListFields:   []string{"id", "name", "email", "age", "is_admin", "createdAt", "updatedAt"},
		DetailFields: []string{"id", "name", "email", "age", "is_admin", "role_ids", "createdAt", "updatedAt"},
		CreateFields: []string{"name", "email", "password", "age", "is_admin", "role_ids"},
		UpdateFields: []string{"name", "email", "password", "age", "is_admin", "role_ids"},
		FilterFields: []string{"is_admin", "age", "createdAt"},
		SortFields:   []string{"id", "name", "email", "age", "is_admin", "createdAt", "updatedAt"},
		SearchFields: []string{"name", "email"},
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
