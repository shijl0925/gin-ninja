package app

import "gorm.io/gorm"

type IUserRepo interface {
	SelectPage(page, size int, db *gorm.DB) ([]User, int64, error)
	SelectOneByID(id uint, db *gorm.DB) (User, error)
	SelectOneByEmail(email string, db *gorm.DB) (User, error)
	Insert(item *User, db *gorm.DB) error
	UpdateByID(id uint, updates map[string]interface{}, db *gorm.DB) error
	DeleteByID(id uint, db *gorm.DB) error
}

type userRepo struct{}

// NewUserRepo returns an implementation of IUserRepo.
func NewUserRepo() IUserRepo {
	return &userRepo{}
}

func (r *userRepo) SelectPage(page, size int, db *gorm.DB) ([]User, int64, error) {
	if db == nil {
		return nil, 0, gorm.ErrInvalidDB
	}
	var total int64
	if err := db.Session(&gorm.Session{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var users []User
	if err := db.Offset((page - 1) * size).Limit(size).Find(&users).Error; err != nil {
		return nil, 0, err
	}
	return users, total, nil
}

func (r *userRepo) SelectOneByID(id uint, db *gorm.DB) (User, error) {
	if db == nil {
		return User{}, gorm.ErrInvalidDB
	}
	var user User
	err := db.First(&user, id).Error
	return user, err
}

func (r *userRepo) SelectOneByEmail(email string, db *gorm.DB) (User, error) {
	if db == nil {
		return User{}, gorm.ErrInvalidDB
	}
	var user User
	err := db.Where("email = ?", email).First(&user).Error
	return user, err
}

func (r *userRepo) Insert(item *User, db *gorm.DB) error {
	if db == nil {
		return gorm.ErrInvalidDB
	}
	return db.Create(item).Error
}

func (r *userRepo) UpdateByID(id uint, updates map[string]interface{}, db *gorm.DB) error {
	if db == nil {
		return gorm.ErrInvalidDB
	}
	result := db.Model(&User{}).Where("id = ?", id).Updates(updates)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (r *userRepo) DeleteByID(id uint, db *gorm.DB) error {
	if db == nil {
		return gorm.ErrInvalidDB
	}
	result := db.Delete(&User{}, id)
	if result.Error != nil {
		return result.Error
	}
	if result.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
