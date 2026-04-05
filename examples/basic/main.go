// Package main demonstrates a minimal gin-ninja application.
//
// Run:
//
//	go run ./examples/basic
//
// Then visit:
//   - http://localhost:8080/docs
//   - http://localhost:8080/openapi.json
package main

import (
	"errors"
	"log"
	"net/http"

	ginpkg "github.com/gin-gonic/gin"
	ninja "github.com/shijl0925/gin-ninja"
	"github.com/shijl0925/gin-ninja/middleware"
	"github.com/shijl0925/gin-ninja/orm"
	"github.com/shijl0925/gin-ninja/pagination"
	"github.com/shijl0925/go-toolkits/gormx"
	gormdriver "gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// ---------------------------------------------------------------------------
// Model
// ---------------------------------------------------------------------------

type User struct {
	gorm.Model
	Name  string `gorm:"column:name;not null"               json:"name"`
	Email string `gorm:"column:email;uniqueIndex;not null"   json:"email"`
	Age   int    `gorm:"column:age"                         json:"age"`
}

// ---------------------------------------------------------------------------
// Schemas
// ---------------------------------------------------------------------------

type UserOut struct {
	ID    uint   `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
	Age   int    `json:"age"`
}

type ListUsersInput struct {
	pagination.PageInput
	Search string `form:"search"`
}

type GetUserInput struct {
	UserID uint `path:"id" binding:"required"`
}

type CreateUserInput struct {
	Name  string `json:"name"  binding:"required"`
	Email string `json:"email" binding:"required,email"`
	Age   int    `json:"age"   binding:"omitempty,min=0,max=150"`
}

type UpdateUserInput struct {
	UserID uint   `path:"id" binding:"required"`
	Name   string `json:"name"  binding:"omitempty"`
	Email  string `json:"email" binding:"omitempty,email"`
	Age    int    `json:"age"   binding:"omitempty,min=0,max=150"`
}

type DeleteUserInput struct {
	UserID uint `path:"id" binding:"required"`
}

// ---------------------------------------------------------------------------
// Repository
// ---------------------------------------------------------------------------

type IUserRepo interface{ gormx.IBaseRepo[User] }
type userRepo struct{ gormx.BaseRepo[User] }

func newUserRepo() IUserRepo { return &userRepo{} }

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

func toOut(u User) UserOut { return UserOut{u.ID, u.Name, u.Email, u.Age} }

func listUsers(ctx *ninja.Context, in *ListUsersInput) (*pagination.Page[UserOut], error) {
	r := newUserRepo()
	q, u := gormx.NewQuery[User]()
	cq, cu := gormx.NewQuery[User]()
	if in.Search != "" {
		q.Like(&u.Name, in.Search)
		cq.Like(&cu.Name, in.Search)
	}
	q.Limit(in.GetSize()).Offset(in.Offset())
	items, _ := r.SelectListByOpts(q.ToOptions()...)
	total, _ := r.SelectCount(cq.ToOptions()...)
	out := make([]UserOut, len(items))
	for i, v := range items {
		out[i] = toOut(v)
	}
	return pagination.NewPage(out, total, in.PageInput), nil
}

func getUser(ctx *ninja.Context, in *GetUserInput) (*UserOut, error) {
	u, err := newUserRepo().SelectOneById(int(in.UserID))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ninja.NotFoundError()
		}
		return nil, err
	}
	out := toOut(u)
	return &out, nil
}

func createUser(ctx *ninja.Context, in *CreateUserInput) (*UserOut, error) {
	u := &User{Name: in.Name, Email: in.Email, Age: in.Age}
	if err := newUserRepo().Insert(u); err != nil {
		return nil, err
	}
	out := toOut(*u)
	return &out, nil
}

func updateUser(ctx *ninja.Context, in *UpdateUserInput) (*UserOut, error) {
	r := newUserRepo()
	if _, err := r.SelectOneById(int(in.UserID)); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ninja.NotFoundError()
		}
		return nil, err
	}
	upd := map[string]interface{}{}
	if in.Name != "" {
		upd["name"] = in.Name
	}
	if in.Email != "" {
		upd["email"] = in.Email
	}
	if in.Age != 0 {
		upd["age"] = in.Age
	}
	if len(upd) > 0 {
		r.UpdateById(int(in.UserID), upd) //nolint:errcheck
	}
	u, _ := r.SelectOneById(int(in.UserID))
	out := toOut(u)
	return &out, nil
}

func deleteUser(ctx *ninja.Context, in *DeleteUserInput) error {
	return newUserRepo().DeleteById(int(in.UserID))
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func main() {
	db, err := gorm.Open(gormdriver.Open("users.db"), &gorm.Config{})
	if err != nil {
		log.Fatal("db:", err)
	}
	db.AutoMigrate(&User{}) //nolint:errcheck
	orm.Init(db)

	api := ninja.New(ninja.Config{
		Title:             "Gin Ninja Basic Example",
		Version:           "1.0.0",
		Prefix:            "/api/v1",
		DisableGinDefault: true,
	})

	// Attach infrastructure middleware.
	api.UseGin(
		middleware.RequestID(),
		middleware.CORS(nil),
		orm.Middleware(db),
	)

	r := ninja.NewRouter("/users", ninja.WithTags("Users"))
	ninja.Get(r, "/", listUsers, ninja.Summary("List users"))
	ninja.Get(r, "/:id", getUser, ninja.Summary("Get user"))
	ninja.Post(r, "/", createUser, ninja.Summary("Create user"))
	ninja.Put(r, "/:id", updateUser, ninja.Summary("Update user"))
	ninja.Delete(r, "/:id", deleteUser, ninja.Summary("Delete user"))
	api.AddRouter(r)

	api.Engine().GET("/health", func(c *ginpkg.Context) {
		c.JSON(http.StatusOK, ginpkg.H{"status": "ok"})
	})

	log.Println("Docs: http://localhost:8080/docs")
	log.Fatal(api.Run(":8080"))
}
