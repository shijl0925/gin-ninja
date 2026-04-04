package app

import (
	ninja "github.com/shijl0925/gin-ninja"
	"github.com/shijl0925/gin-ninja/pagination"
)

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
	Token   string `json:"token"`
	Expires int    `json:"expires_in"` // seconds
	UserID  uint   `json:"user_id"`
	Name    string `json:"name"`
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

// ListUsersInput holds query parameters for listing users.
type ListUsersInput struct {
	pagination.PageInput
	Search  string `form:"search"   filter:"name,like"    description:"Filter by name (partial match)"`
	IsAdmin *bool  `form:"is_admin" filter:"is_admin,eq" description:"Filter by admin flag"`
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

// ---------------------------------------------------------------------------
// Feature demo schemas
// ---------------------------------------------------------------------------

// RequestMetaInput demonstrates cookie/header/query binding with defaults.
type RequestMetaInput struct {
	Session string `cookie:"session"        default:"guest-session" description:"Session cookie value"`
	TraceID string `header:"X-Trace-ID"     default:"trace-demo"    description:"Trace identifier header"`
	Lang    string `form:"lang"             default:"zh-CN"         description:"Preferred language query parameter"`
	Verbose bool   `form:"verbose"          default:"false"         description:"Verbose output flag"`
}

// RequestMetaOutput echoes the effective request metadata after binding.
type RequestMetaOutput struct {
	Session string `json:"session"`
	TraceID string `json:"trace_id"`
	Lang    string `json:"lang"`
	Verbose bool   `json:"verbose"`
}

// SlowDemoOutput is the response body for the timeout demo endpoint.
type SlowDemoOutput struct {
	Status string `json:"status"`
}

// LimitedDemoOutput is the response body for the rate-limit demo endpoint.
type LimitedDemoOutput struct {
	Status string `json:"status"`
}

// HiddenDemoOutput is the response body for a route excluded from OpenAPI.
type HiddenDemoOutput struct {
	Status string `json:"status"`
}

// FeatureItemOut is the item schema used by the paginated demo endpoint.
type FeatureItemOut struct {
	Code    string `json:"code"`
	Title   string `json:"title"`
	Enabled bool   `json:"enabled"`
}

// FeatureListInput is the request schema for the paginated demo endpoint.
type FeatureListInput struct {
	pagination.PageInput
	Search string `form:"search" default:"demo" description:"Optional feature search term"`
}

// UploadSingleInput demonstrates single-file multipart binding with form fields.
type UploadSingleInput struct {
	Title string              `form:"title" binding:"required" description:"File title"`
	File  *ninja.UploadedFile `file:"file"  binding:"required" description:"Single uploaded file"`
}

// UploadManyInput demonstrates multi-file multipart binding with extra form fields.
type UploadManyInput struct {
	Category string                `form:"category" binding:"required" description:"Upload category"`
	Files    []*ninja.UploadedFile `file:"files"    binding:"required" description:"Uploaded files"`
}

// UploadDemoOutput echoes uploaded file metadata.
type UploadDemoOutput struct {
	Title     string   `json:"title,omitempty"`
	Category  string   `json:"category,omitempty"`
	Filename  string   `json:"filename,omitempty"`
	Size      int64    `json:"size,omitempty"`
	FileCount int      `json:"file_count"`
	Names     []string `json:"names,omitempty"`
}
