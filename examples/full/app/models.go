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
}
