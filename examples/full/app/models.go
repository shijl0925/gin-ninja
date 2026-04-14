package app

import (
	"fmt"
	"strings"

	ninja "github.com/shijl0925/gin-ninja"
	"gorm.io/gorm"
)

// User is the GORM domain model.
type User struct {
	gorm.Model
	Name     string `gorm:"column:name;not null"               json:"name"`
	Email    string `gorm:"column:email;type:varchar(255);uniqueIndex;not null" json:"email"`
	Password string `gorm:"column:password;not null"           json:"-" admin:"component:password;create;update;readonly:false"` // never serialised
	Age      int    `gorm:"column:age"                         json:"age"`
	IsAdmin  bool   `gorm:"column:is_admin;default:false"      json:"is_admin"`
	Roles    []Role `gorm:"many2many:user_roles;"              json:"-"`
	RoleIDs  []uint `gorm:"-"                                  json:"role_ids" admin:"label:Roles;relation:roles"`
}

type Role struct {
	gorm.Model
	Status int    `json:"status" gorm:"column:status;type:integer;default:1;"`
	Remark string `json:"remark" gorm:"column:remark;type:longtext;"`
	Name   string `json:"name" gorm:"column:name;type:varchar(64);not null;"`
	Code   string `json:"code" gorm:"column:code;type:varchar(64);"`
}

// Project is a related admin demo model owned by a user.
type Project struct {
	gorm.Model
	Title   string `gorm:"column:title;not null"               json:"title"`
	Summary string `gorm:"column:summary;type:text"            json:"summary"`
	OwnerID uint   `gorm:"column:owner_id;not null;index"      json:"owner_id"`
	Owner   User   `gorm:"foreignKey:OwnerID"                  json:"-"`
}

type userRole struct {
	UserID uint `gorm:"column:user_id"`
	RoleID uint `gorm:"column:role_id"`
}

func (u *User) AfterFind(*gorm.DB) error {
	u.syncRoleIDs()
	return nil
}

func (u *User) BeforeSave(*gorm.DB) error {
	u.Name = strings.TrimSpace(u.Name)
	u.Email = strings.TrimSpace(strings.ToLower(u.Email))
	if strings.TrimSpace(u.Password) == "" || isHashedPassword(u.Password) {
		return nil
	}
	if len(u.Password) < 8 {
		return ninja.NewErrorWithCode(400, "BAD_REQUEST", "field \"password\" must be at least 8 characters")
	}
	u.Password = hashPassword(u.Password)
	return nil
}

func (u *User) AfterSave(tx *gorm.DB) error {
	return syncUserRoles(tx, u, u.RoleIDs)
}

func (u *User) syncRoleIDs() {
	if len(u.Roles) == 0 {
		u.RoleIDs = nil
		return
	}
	ids := make([]uint, 0, len(u.Roles))
	for _, role := range u.Roles {
		ids = append(ids, role.ID)
	}
	u.RoleIDs = ids
}

func isHashedPassword(password string) bool {
	return strings.HasPrefix(password, "$2")
}

func syncUserRoles(tx *gorm.DB, user *User, roleIDs []uint) error {
	if user == nil {
		return nil
	}
	if roleIDs == nil {
		return nil
	}
	normalized := make([]uint, 0, len(roleIDs))
	seen := make(map[uint]struct{}, len(roleIDs))
	for _, id := range roleIDs {
		if id == 0 {
			return ninja.NewErrorWithCode(400, "BAD_REQUEST", "field \"role_ids\" must not contain zero")
		}
		if _, exists := seen[id]; exists {
			continue
		}
		seen[id] = struct{}{}
		normalized = append(normalized, id)
	}
	if len(normalized) == 0 {
		if err := tx.Where("user_id = ?", user.ID).Delete(&userRole{}).Error; err != nil {
			return err
		}
		user.Roles = nil
		user.RoleIDs = []uint{}
		return nil
	}

	var roles []Role
	if err := tx.Where("id IN ?", normalized).Find(&roles).Error; err != nil {
		return err
	}
	roleByID := make(map[uint]Role, len(roles))
	for _, role := range roles {
		roleByID[role.ID] = role
	}
	orderedRoles := make([]Role, 0, len(normalized))
	for _, id := range normalized {
		role, ok := roleByID[id]
		if !ok {
			return ninja.NewErrorWithCode(400, "BAD_REQUEST", fmt.Sprintf("role %d does not exist", id))
		}
		orderedRoles = append(orderedRoles, role)
	}
	if err := tx.Where("user_id = ?", user.ID).Delete(&userRole{}).Error; err != nil {
		return err
	}
	links := make([]userRole, 0, len(normalized))
	for _, id := range normalized {
		links = append(links, userRole{UserID: user.ID, RoleID: id})
	}
	if err := tx.Create(&links).Error; err != nil {
		return err
	}
	user.Roles = orderedRoles
	user.RoleIDs = normalized
	return nil
}
