package app

import (
	"errors"
	"time"

	ninja "github.com/shijl0925/gin-ninja"
	"github.com/shijl0925/gin-ninja/middleware"
	"github.com/shijl0925/gin-ninja/pagination"
	"github.com/shijl0925/go-toolkits/gormx"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Auth
// ---------------------------------------------------------------------------

// Login validates credentials and returns a signed JWT token.
func Login(ctx *ninja.Context, in *LoginInput) (*LoginOutput, error) {
	repo := NewUserRepo()

	query, u := gormx.NewQuery[User]()
	query.Eq(&u.Email, in.Email)
	user, err := repo.SelectOneByOpts(query.ToOptions()...)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ninja.NewErrorWithCode(401, "INVALID_CREDENTIALS", "invalid email or password")
		}
		return nil, err
	}

	if !checkPassword(user.Password, in.Password) {
		return nil, ninja.NewErrorWithCode(401, "INVALID_CREDENTIALS", "invalid email or password")
	}

	token, err := middleware.GenerateToken(user.ID, user.Name)
	if err != nil {
		return nil, err
	}

	return &LoginOutput{
		Token:   token,
		Expires: int(24 * time.Hour / time.Second),
		UserID:  user.ID,
		Name:    user.Name,
	}, nil
}

// ---------------------------------------------------------------------------
// Users CRUD
// ---------------------------------------------------------------------------

// ListUsers returns a paginated list of users.
func ListUsers(ctx *ninja.Context, in *ListUsersInput) (*pagination.Page[UserOut], error) {
	repo := NewUserRepo()

	query, u := gormx.NewQuery[User]()
	if in.Search != "" {
		query.Like(&u.Name, "%"+in.Search+"%")
	}

	countQuery, cu := gormx.NewQuery[User]()
	if in.Search != "" {
		countQuery.Like(&cu.Name, "%"+in.Search+"%")
	}

	query.Limit(in.GetSize()).Offset(in.Offset())

	items, err := repo.SelectListByOpts(query.ToOptions()...)
	if err != nil {
		return nil, err
	}
	total, err := repo.SelectCount(countQuery.ToOptions()...)
	if err != nil {
		return nil, err
	}

	out := make([]UserOut, len(items))
	for i, item := range items {
		out[i] = toUserOut(item)
	}
	return pagination.NewPage(out, total, in.PageInput), nil
}

// GetUser retrieves a single user by ID.
func GetUser(ctx *ninja.Context, in *GetUserInput) (*UserOut, error) {
	repo := NewUserRepo()
	u, err := repo.SelectOneById(int(in.UserID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ninja.ErrNotFound
		}
		return nil, err
	}
	out := toUserOut(u)
	return &out, nil
}

// CreateUser creates a new user.
func CreateUser(ctx *ninja.Context, in *CreateUserInput) (*UserOut, error) {
	u := &User{
		Name:     in.Name,
		Email:    in.Email,
		Password: hashPassword(in.Password),
		Age:      in.Age,
	}
	repo := NewUserRepo()
	if err := repo.Insert(u); err != nil {
		return nil, err
	}
	out := toUserOut(*u)
	return &out, nil
}

// UpdateUser updates an existing user's fields.
func UpdateUser(ctx *ninja.Context, in *UpdateUserInput) (*UserOut, error) {
	repo := NewUserRepo()
	_, err := repo.SelectOneById(int(in.UserID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ninja.ErrNotFound
		}
		return nil, err
	}

	updates := map[string]interface{}{}
	if in.Name != "" {
		updates["name"] = in.Name
	}
	if in.Email != "" {
		updates["email"] = in.Email
	}
	if in.Age != 0 {
		updates["age"] = in.Age
	}
	if len(updates) > 0 {
		if err := repo.UpdateById(int(in.UserID), updates); err != nil {
			return nil, err
		}
	}

	u, _ := repo.SelectOneById(int(in.UserID))
	out := toUserOut(u)
	return &out, nil
}

// DeleteUser removes a user by ID.
func DeleteUser(ctx *ninja.Context, in *DeleteUserInput) error {
	repo := NewUserRepo()
	if err := repo.DeleteById(int(in.UserID)); err != nil {
		return err
	}
	return nil
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// hashPassword creates a bcrypt hash of the given password.
func hashPassword(password string) string {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		// bcrypt.GenerateFromPassword only errors on invalid cost parameters;
		// DefaultCost is always valid, so this branch should never be reached.
		panic("bcrypt: " + err.Error())
	}
	return string(hash)
}

// checkPassword compares a bcrypt hash with a candidate password.
func checkPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}
