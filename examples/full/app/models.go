package app

import "gorm.io/gorm"

// User is the GORM domain model.
type User struct {
	gorm.Model
	Name     string `gorm:"column:name;not null"               json:"name"`
	Email    string `gorm:"column:email;type:varchar(255);uniqueIndex;not null" json:"email"`
	Password string `gorm:"column:password;not null"           json:"-"` // never serialised
	Age      int    `gorm:"column:age"                         json:"age"`
	IsAdmin  bool   `gorm:"column:is_admin;default:false"      json:"is_admin"`
	Roles    []Role `gorm:"many2many:user_roles;"              json:"-"`
	RoleIDs  []uint `gorm:"-"                                  json:"role_ids"`
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

func (u *User) AfterFind(*gorm.DB) error {
	u.syncRoleIDs()
	return nil
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
