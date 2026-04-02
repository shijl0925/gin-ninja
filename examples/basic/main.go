// Package main demonstrates a complete gin-ninja application backed by SQLite.
//
// Run:
//
//	go run ./examples/basic
//
// Then visit:
//   - http://localhost:8080/docs           – Swagger UI
//   - http://localhost:8080/openapi.json   – raw OpenAPI spec
//   - http://localhost:8080/api/v1/users   – list users
package main

import (
	"errors"
	"log"
	"net/http"

	ginpkg "github.com/gin-gonic/gin"
	ninja "github.com/shijl0925/gin-ninja"
	"github.com/shijl0925/gin-ninja/orm"
	"github.com/shijl0925/gin-ninja/pagination"
	"github.com/shijl0925/go-toolkits/gormx"
	gormdriver "gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Domain model
// ---------------------------------------------------------------------------

// User is the GORM model.
type User struct {
	gorm.Model
	Name  string `gorm:"column:name;not null"  json:"name"`
	Email string `gorm:"column:email;uniqueIndex;not null" json:"email"`
	Age   int    `gorm:"column:age"            json:"age"`
}

// ---------------------------------------------------------------------------
// Schemas (request / response)
// ---------------------------------------------------------------------------

// UserOut is the public representation of a user returned by the API.
type UserOut struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

// ListUsersInput holds query parameters for listing users.
type ListUsersInput struct {
	pagination.PageInput
	Search string `form:"search" description:"Filter users by name"`
}

// GetUserInput holds the path parameter for retrieving a single user.
type GetUserInput struct {
	UserID uint `path:"id" binding:"required"`
}

// CreateUserInput is the request body for creating a user.
type CreateUserInput struct {
	Name  string `json:"name"  binding:"required"       description:"Full name"`
	Email string `json:"email" binding:"required,email" description:"Email address"`
	Age   int    `json:"age"   binding:"omitempty,min=0,max=150"`
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

// ---------------------------------------------------------------------------
// Repository
// ---------------------------------------------------------------------------

type IUserRepo interface {
	gormx.IBaseRepo[User]
}

type userRepo struct {
	gormx.BaseRepo[User]
}

func newUserRepo() IUserRepo { return &userRepo{} }

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

func toUserOut(u User) UserOut {
	return UserOut{ID: u.ID, Name: u.Name, Email: u.Email, Age: u.Age}
}

func listUsers(ctx *ninja.Context, input *ListUsersInput) (*pagination.Page[UserOut], error) {
	repo := newUserRepo()

	query, u := gormx.NewQuery[User]()
	if input.Search != "" {
		query.Like(&u.Name, "%"+input.Search+"%")
	}
	query.Limit(input.GetSize()).Offset(input.Offset())

	items, err := repo.SelectListByOpts(query.ToOptions()...)
	if err != nil {
		return nil, err
	}

	countQuery, cu := gormx.NewQuery[User]()
	if input.Search != "" {
		countQuery.Like(&cu.Name, "%"+input.Search+"%")
	}
	total, err := repo.SelectCount(countQuery.ToOptions()...)
	if err != nil {
		return nil, err
	}

	out := make([]UserOut, len(items))
	for i, u := range items {
		out[i] = toUserOut(u)
	}
	return pagination.NewPage(out, total, input.PageInput), nil
}

func getUser(ctx *ninja.Context, input *GetUserInput) (*UserOut, error) {
	repo := newUserRepo()
	u, err := repo.SelectOneById(int(input.UserID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ninja.ErrNotFound
		}
		return nil, err
	}
	out := toUserOut(u)
	return &out, nil
}

func createUser(ctx *ninja.Context, input *CreateUserInput) (*UserOut, error) {
	u := &User{Name: input.Name, Email: input.Email, Age: input.Age}
	repo := newUserRepo()
	if err := repo.Insert(u); err != nil {
		return nil, err
	}
	out := toUserOut(*u)
	return &out, nil
}

func updateUser(ctx *ninja.Context, input *UpdateUserInput) (*UserOut, error) {
	repo := newUserRepo()
	u, err := repo.SelectOneById(int(input.UserID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ninja.ErrNotFound
		}
		return nil, err
	}

	updates := map[string]interface{}{}
	if input.Name != "" {
		updates["name"] = input.Name
	}
	if input.Email != "" {
		updates["email"] = input.Email
	}
	if input.Age != 0 {
		updates["age"] = input.Age
	}
	if err := repo.UpdateById(int(u.ID), updates); err != nil {
		return nil, err
	}
	u, _ = repo.SelectOneById(int(u.ID))
	out := toUserOut(u)
	return &out, nil
}

func deleteUser(ctx *ninja.Context, input *DeleteUserInput) error {
	repo := newUserRepo()
	if err := repo.DeleteById(int(input.UserID)); err != nil {
		return err
	}
	return nil
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func main() {
	// Open SQLite database.
	db, err := gorm.Open(gormdriver.Open("users.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("failed to connect database:", err)
	}
	if err := db.AutoMigrate(&User{}); err != nil {
		log.Fatal("auto migrate failed:", err)
	}

	// Initialise the global gormx DB.
	orm.Init(db)

	// Build the API.
	api := ninja.New(ninja.Config{
		Title:       "Gin Ninja Example",
		Version:     "1.0.0",
		Description: "A sample API built with gin-ninja",
		Prefix:      "/api/v1",
	})

	// Attach ORM middleware so handlers can call orm.GetDB(ctx).
	api.Engine().Use(orm.Middleware(db))

	// Create the users router.
	usersRouter := ninja.NewRouter("/users", ninja.WithTags("Users"))

	ninja.Get(usersRouter, "/", listUsers,
		ninja.Summary("List users"),
		ninja.Description("Returns a paginated list of users"),
	)
	ninja.Get(usersRouter, "/:id", getUser,
		ninja.Summary("Get user"),
	)
	ninja.Post(usersRouter, "/", createUser,
		ninja.Summary("Create user"),
	)
	ninja.Put(usersRouter, "/:id", updateUser,
		ninja.Summary("Update user"),
	)
	ninja.Delete(usersRouter, "/:id", deleteUser,
		ninja.Summary("Delete user"),
	)

	api.AddRouter(usersRouter)

	// Health-check endpoint registered directly on the gin engine.
	api.Engine().GET("/health", func(c *ginpkg.Context) {
		c.JSON(http.StatusOK, ginpkg.H{"status": "ok"})
	})

	log.Println("Starting server on :8080")
	log.Println("Docs: http://localhost:8080/docs")
	if err := api.Run(":8080"); err != nil {
		log.Fatal(err)
	}
}
