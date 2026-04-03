package app

import "gorm.io/gorm"

// User is the GORM domain model.
type User struct {
	gorm.Model
	Name     string `gorm:"column:name;not null"               json:"name"`
	Email    string `gorm:"column:email;uniqueIndex;not null"   json:"email"`
	Password string `gorm:"column:password;not null"           json:"-"` // never serialised
	Age      int    `gorm:"column:age"                         json:"age"`
	IsAdmin  bool   `gorm:"column:is_admin;default:false"      json:"is_admin"`
	Roles    []Role `gorm:"many2many:user_roles;"              json:"-"`
}

// Role is a simple RBAC role model used by the full example.
type Role struct {
	gorm.Model
	Name        string       `gorm:"column:name;not null"                    json:"name"`
	Code        string       `gorm:"column:code;uniqueIndex;not null"        json:"code"`
	Description string       `gorm:"column:description"                      json:"description"`
	Permissions []Permission `gorm:"many2many:role_permissions;"             json:"-"`
}

// Permission is a simple RBAC permission model identified by a stable code.
type Permission struct {
	gorm.Model
	Name        string `gorm:"column:name;not null"                    json:"name"`
	Code        string `gorm:"column:code;uniqueIndex;not null"        json:"code"`
	Description string `gorm:"column:description"                      json:"description"`
}
