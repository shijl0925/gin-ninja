package app

import (
	"errors"
	"time"

	ninja "github.com/shijl0925/gin-ninja"
	"github.com/shijl0925/gin-ninja/filter"
	"github.com/shijl0925/gin-ninja/middleware"
	"github.com/shijl0925/gin-ninja/pagination"
	"github.com/shijl0925/go-toolkits/gormx"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var userSortSchema = pagination.NewSortSchema("id", "name", "email", "age", "is_admin", "created_at")

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

// Register creates a new user account without requiring authentication.
func Register(ctx *ninja.Context, in *RegisterInput) (*UserOut, error) {
	repo := NewUserRepo()

	query, u := gormx.NewQuery[User]()
	query.Eq(&u.Email, in.Email)
	_, err := repo.SelectOneByOpts(query.ToOptions()...)
	switch {
	case err == nil:
		return nil, ninja.NewErrorWithCode(409, "EMAIL_ALREADY_EXISTS", "email already registered")
	case !errors.Is(err, gorm.ErrRecordNotFound):
		return nil, err
	}

	return createUser(repo, in.Name, in.Email, in.Password, in.Age)
}

// ---------------------------------------------------------------------------
// Users CRUD
// ---------------------------------------------------------------------------

// ListUsers returns a paginated list of users.
func ListUsers(ctx *ninja.Context, in *ListUsersInput) (*pagination.Page[UserOut], error) {
	repo := resolveUserRepo(in.Repo)
	query, _ := gormx.NewQuery[User]()

	filterOpts, err := filter.BuildOptions(in)
	if err != nil {
		return nil, ninja.NewErrorWithCode(400, "BAD_FILTER", err.Error())
	}
	if err := pagination.ApplySort(query, in.PageInput, userSortSchema); err != nil {
		return nil, ninja.NewErrorWithCode(400, "BAD_SORT", err.Error())
	}

	opts := append(filterOpts, query.ToOptions()...)
	items, total, err := repo.SelectPage(in.GetPage(), in.GetSize(), opts...)
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
	u, err := resolveUserRepo(in.Repo).SelectOneById(int(in.UserID))
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
	return createUser(resolveUserRepo(in.Repo), in.Name, in.Email, in.Password, in.Age)
}

// UpdateUser updates an existing user's fields.
func UpdateUser(ctx *ninja.Context, in *UpdateUserInput) (*UserOut, error) {
	repo := resolveUserRepo(in.Repo)
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
	if err := resolveUserRepo(in.Repo).DeleteById(int(in.UserID)); err != nil {
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

func createUser(repo IUserRepo, name, email, password string, age int) (*UserOut, error) {
	u := &User{
		Name:     name,
		Email:    email,
		Password: hashPassword(password),
		Age:      age,
	}
	if err := repo.Insert(u); err != nil {
		return nil, err
	}
	out := toUserOut(*u)
	return &out, nil
}

func resolveUserRepo(repo IUserRepo) IUserRepo {
	if repo != nil {
		return repo
	}
	return NewUserRepo()
}
