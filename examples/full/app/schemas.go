package app

import "github.com/shijl0925/gin-ninja/pagination"

// ---------------------------------------------------------------------------
// Auth schemas
// ---------------------------------------------------------------------------

// LoginInput is the request body for POST /auth/login.
type LoginInput struct {
	Email    string `json:"email"    binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

// RegisterInput is the request body for POST /auth/register.
type RegisterInput struct {
	Name     string `json:"name"     binding:"required"        description:"Full name"`
	Email    string `json:"email"    binding:"required,email"  description:"Email address"`
	Password string `json:"password" binding:"required,min=8"  description:"Password (min 8 chars)"`
	Age      int    `json:"age"      binding:"omitempty,min=0,max=150"`
}

// LoginOutput is the response body for POST /auth/login.
type LoginOutput struct {
	Token       string   `json:"token"`
	Expires     int      `json:"expires_in"` // seconds
	UserID      uint     `json:"user_id"`
	Name        string   `json:"name"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

// ---------------------------------------------------------------------------
// User schemas
// ---------------------------------------------------------------------------

// UserOut is the public representation of a user.
type UserOut struct {
	ID      uint   `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	Age     int    `json:"age"`
	IsAdmin bool   `json:"is_admin"`
}

// CurrentSubjectOut is the RBAC summary for the current authenticated user.
type CurrentSubjectOut struct {
	User        UserOut   `json:"user"`
	Roles       []string  `json:"roles"`
	Permissions []string  `json:"permissions"`
}

// ListUsersInput holds query parameters for listing users.
type ListUsersInput struct {
	pagination.PageInput
	Search  string `form:"search"   description:"Filter by name (partial match)"`
	IsAdmin *bool  `form:"is_admin" description:"Filter by admin flag"`
}

// GetUserInput holds the path parameter for retrieving a single user.
type GetUserInput struct {
	UserID uint `path:"id" binding:"required"`
}

// CreateUserInput is the request body for creating a user.
type CreateUserInput struct {
	Name     string `json:"name"     binding:"required"        description:"Full name"`
	Email    string `json:"email"    binding:"required,email"  description:"Email address"`
	Password string `json:"password" binding:"required,min=8"  description:"Password (min 8 chars)"`
	Age      int    `json:"age"      binding:"omitempty,min=0,max=150"`
}

// UpdateUserInput combines a path param with a JSON body.
type UpdateUserInput struct {
	UserID uint `path:"id" binding:"required"`

	Name  string `json:"name"  binding:"omitempty"`
	Email string `json:"email" binding:"omitempty,email"`
	Age   int    `json:"age"   binding:"omitempty,min=0,max=150"`
}

// DeleteUserInput holds the path parameter for deleting a user.
type DeleteUserInput struct {
	UserID uint `path:"id" binding:"required"`
}

// toUserOut converts a domain User to the public UserOut representation.
func toUserOut(u User) UserOut {
	return UserOut{
		ID:      u.ID,
		Name:    u.Name,
		Email:   u.Email,
		Age:     u.Age,
		IsAdmin: u.IsAdmin,
	}
}
