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
		Icon:         "users",
		Group:        "Identity",
		Description:  "Manage application users, profile fields, admin flags, and assigned roles.",
		Order:        10,
		Model:        User{},
		Preloads:     []string{"Roles"},
		ListFields:   []string{"id", "name", "email", "age", "is_admin", "createdAt", "updatedAt"},
		DetailFields: []string{"id", "name", "email", "age", "is_admin", "role_ids", "createdAt", "updatedAt"},
		CreateFields: []string{"name", "email", "password", "age", "is_admin", "role_ids"},
		UpdateFields: []string{"name", "email", "password", "age", "is_admin", "role_ids"},
		FilterFields: []string{"is_admin", "age", "createdAt"},
		SortFields:   []string{"id", "name", "email", "age", "is_admin", "createdAt", "updatedAt"},
		SearchFields: []string{"name", "email"},
		FieldOptions: map[string]admin.FieldOptions{
			"name":     {Placeholder: "Full name", Help: "Shown in admin tables and relation selectors."},
			"email":    {Placeholder: "user@example.com"},
			"password": {Placeholder: "Leave blank when editing to keep the current password."},
			"role_ids": {Help: "Search roles and select one or more memberships.", Width: "full"},
		},
	})
	site.MustRegisterModel(&admin.ModelResource{
		Icon:         "shield",
		Group:        "Identity",
		Description:  "Define role names, codes, status values, and operational notes.",
		Order:        20,
		Model:        Role{},
		ListFields:   []string{"id", "name", "code", "status", "createdAt", "updatedAt"},
		DetailFields: []string{"id", "name", "code", "status", "remark", "createdAt", "updatedAt"},
		CreateFields: []string{"name", "code", "status", "remark"},
		UpdateFields: []string{"name", "code", "status", "remark"},
		FilterFields: []string{"status", "name", "code"},
		SortFields:   []string{"id", "name", "code", "status", "createdAt", "updatedAt"},
		SearchFields: []string{"name", "code", "remark"},
		FieldOptions: map[string]admin.FieldOptions{
			"name":   {Placeholder: "Administrators"},
			"code":   {Placeholder: "admin"},
			"remark": {Placeholder: "Internal role notes", Width: "full"},
		},
	})
	site.MustRegisterModel(&admin.ModelResource{
		Icon:         "briefcase",
		Group:        "Delivery",
		Description:  "Review projects scoped to the signed-in owner and maintain ownership metadata.",
		Order:        30,
		Model:        Project{},
		ListFields:   []string{"id", "title", "owner_id", "createdAt", "updatedAt"},
		DetailFields: []string{"id", "title", "summary", "owner_id", "createdAt", "updatedAt"},
		CreateFields: []string{"title", "summary", "owner_id"},
		UpdateFields: []string{"title", "summary", "owner_id"},
		FilterFields: []string{"id"},
		SearchFields: []string{"title", "summary"},
		SortFields:   []string{"id", "title", "owner_id", "createdAt", "updatedAt"},
		FieldOptions: map[string]admin.FieldOptions{
			"title":    {Placeholder: "Project title", Width: "full"},
			"summary":  {Placeholder: "Short project summary", Width: "full"},
			"owner_id": {Help: "Only users visible to the admin API can be selected as owners."},
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
