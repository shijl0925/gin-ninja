package app

import (
	"errors"
	"time"

	ninja "github.com/shijl0925/gin-ninja"
	"github.com/shijl0925/gin-ninja/filter"
	"github.com/shijl0925/gin-ninja/middleware"
	"github.com/shijl0925/gin-ninja/order"
	"github.com/shijl0925/gin-ninja/orm"
	"github.com/shijl0925/gin-ninja/pagination"
	"github.com/shijl0925/gin-ninja/settings"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Auth
// ---------------------------------------------------------------------------

func userDB(ctx *ninja.Context) *gorm.DB {
	if ctx != nil && ctx.Context != nil {
		return orm.WithContext(ctx.Context)
	}
	return nil
}

// LoginHandler returns a login handler bound to explicit JWT settings.
func LoginHandler(cfg settings.JWTConfig) func(*ninja.Context, *LoginInput) (*LoginOutput, error) {
	return func(ctx *ninja.Context, in *LoginInput) (*LoginOutput, error) {
		return login(ctx, in, cfg)
	}
}

func login(ctx *ninja.Context, in *LoginInput, cfg settings.JWTConfig) (*LoginOutput, error) {
	db := userDB(ctx)
	repo := NewUserRepo()

	user, err := repo.SelectOneByEmail(in.Email, db)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ninja.NewError(401, "invalid email or password")
		}
		return nil, err
	}

	if !checkPassword(user.Password, in.Password) {
		return nil, ninja.NewError(401, "invalid email or password")
	}

	token, err := middleware.GenerateTokenWithConfig(user.ID, user.Name, cfg)
	if err != nil {
		return nil, err
	}

	return &LoginOutput{
		Token:   token,
		Expires: int(cfg.ExpireDuration() / time.Second),
		UserID:  user.ID,
		Name:    user.Name,
	}, nil
}

// CurrentUser returns the authenticated user's profile and token metadata.
func CurrentUser(ctx *ninja.Context, _ *CurrentUserInput) (*CurrentUserOutput, error) {
	userID := ctx.GetUserID()
	if userID == 0 {
		return nil, ninja.UnauthorizedError()
	}

	db := userDB(ctx)
	var user User
	if err := db.Preload("Roles").First(&user, userID).Error; err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ninja.NewError(401, "authenticated user no longer exists")
		}
		return nil, err
	}

	out := &CurrentUserOutput{
		UserID:  user.ID,
		Name:    user.Name,
		Email:   user.Email,
		IsAdmin: user.IsAdmin,
		Roles:   make([]AuthRoleOut, 0, len(user.Roles)),
	}
	for _, role := range user.Roles {
		out.Roles = append(out.Roles, AuthRoleOut{
			ID:     role.ID,
			Name:   role.Name,
			Code:   role.Code,
			Status: role.Status,
		})
	}
	if claims := middleware.GetClaims(ctx.Context); claims != nil {
		out.Issuer = claims.Issuer
		if claims.ExpiresAt != nil {
			out.ExpiresAt = claims.ExpiresAt.Time.Format(time.RFC3339)
		}
		if claims.IssuedAt != nil {
			out.IssuedAt = claims.IssuedAt.Time.Format(time.RFC3339)
		}
	}
	return out, nil
}

// Register creates a new user account without requiring authentication.
func Register(ctx *ninja.Context, in *RegisterInput) (*UserOut, error) {
	db := userDB(ctx)
	repo := NewUserRepo()

	_, err := repo.SelectOneByEmail(in.Email, db)
	switch {
	case err == nil:
		return nil, ninja.NewError(409, "email already registered")
	case !errors.Is(err, gorm.ErrRecordNotFound):
		return nil, err
	}

	user, err := createUser(repo, db, in.Name, in.Email, in.Password, in.Age)
	if err != nil {
		return nil, err
	}
	out := toUserOut(*user)
	return &out, nil
}

// ---------------------------------------------------------------------------
// Users CRUD
// ---------------------------------------------------------------------------

// ListUsers returns a paginated list of users.
func ListUsers(ctx *ninja.Context, in *ListUsersInput) (*pagination.Page[UserOut], error) {
	db := userDB(ctx)
	repo := NewUserRepo()
	query := db.Model(&User{})

	query, err := filter.ApplyDB(query, in)
	if err != nil {
		return nil, ninja.NewError(400, err.Error())
	}
	query, err = order.ApplyDB(query, in)
	if err != nil {
		return nil, ninja.NewError(400, err.Error())
	}

	items, total, err := repo.SelectPage(in.GetPage(), in.GetSize(), query)
	if err != nil {
		return nil, err
	}

	return pagination.NewPage(toUserOuts(items), total, in.PageInput), nil
}

// GetUser retrieves a single user by ID.
func GetUser(ctx *ninja.Context, in *GetUserInput) (*UserOut, error) {
	db := userDB(ctx)
	repo := NewUserRepo()
	u, err := repo.SelectOneByID(in.UserID, db)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ninja.NotFoundError()
		}
		return nil, err
	}
	out := toUserOut(u)
	return &out, nil
}

// CreateUser creates a new user.
func CreateUser(ctx *ninja.Context, in *CreateUserInput) (*UserOut, error) {
	db := userDB(ctx)
	repo := NewUserRepo()
	user, err := createUser(repo, db, in.Name, in.Email, in.Password, in.Age)
	if err != nil {
		return nil, err
	}
	out := toUserOut(*user)
	return &out, nil
}

// UpdateUser updates an existing user's fields.
func UpdateUser(ctx *ninja.Context, in *UpdateUserInput) (*UserOut, error) {
	db := userDB(ctx)
	repo := NewUserRepo()
	_, err := repo.SelectOneByID(in.UserID, db)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ninja.NotFoundError()
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
		if err := repo.UpdateByID(in.UserID, updates, db); err != nil {
			return nil, err
		}
	}

	u, err := repo.SelectOneByID(in.UserID, db)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ninja.NotFoundError()
		}
		return nil, err
	}
	out := toUserOut(u)
	return &out, nil
}

// DeleteUser removes a user by ID.
func DeleteUser(ctx *ninja.Context, in *DeleteUserInput) error {
	db := userDB(ctx)
	repo := NewUserRepo()
	if err := repo.DeleteByID(in.UserID, db); err != nil {
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

func createUser(repo IUserRepo, db *gorm.DB, name, email, password string, age int) (*User, error) {
	u := &User{
		Name:     name,
		Email:    email,
		Password: hashPassword(password),
		Age:      age,
	}
	if err := repo.Insert(u, db); err != nil {
		return nil, err
	}
	return u, nil
}
