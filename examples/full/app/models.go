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

// Project is a related admin demo model owned by a user.
type Project struct {
	gorm.Model
	Title   string `gorm:"column:title;not null"               json:"title"`
	Summary string `gorm:"column:summary;type:text"            json:"summary"`
	OwnerID uint   `gorm:"column:owner_id;not null;index"      json:"owner_id"`
	Owner   User   `gorm:"foreignKey:OwnerID"                  json:"-"`
}
