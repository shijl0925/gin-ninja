// Package main demonstrates a minimal gin-ninja application.
//
// Run:
//
//	go run ./examples/basic
//
// Then visit:
//   - http://localhost:8080/
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

var runBasicMain = run
var fatalBasic = func(v ...any) { log.Fatal(v...) }

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

type UserRepoImpl struct{ gormx.BaseRepo[User] }

func newUserRepo() IUserRepo { return &UserRepoImpl{} }

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

func toOut(u User) UserOut { return UserOut{u.ID, u.Name, u.Email, u.Age} }

func userDB(ctx *ninja.Context) *gorm.DB {
	if ctx != nil && ctx.Context != nil {
		return orm.WithContext(ctx.Context)
	}
	return gormx.GetDb()
}

func listUsers(ctx *ninja.Context, in *ListUsersInput) (*pagination.Page[UserOut], error) {
	db := userDB(ctx)
	r := newUserRepo()
	q, u := gormx.NewQuery[User]()
	cq, cu := gormx.NewQuery[User]()
	if in.Search != "" {
		q.Like(&u.Name, in.Search)
		cq.Like(&cu.Name, in.Search)
	}
	q.Limit(in.GetSize()).Offset(in.Offset())
	items, _ := r.SelectListByOpts(append([]gormx.DBOption{gormx.UseDB(db)}, q.ToOptions()...)...)
	total, _ := r.SelectCount(append([]gormx.DBOption{gormx.UseDB(db)}, cq.ToOptions()...)...)
	out := make([]UserOut, len(items))
	for i, v := range items {
		out[i] = toOut(v)
	}
	return pagination.NewPage(out, total, in.PageInput), nil
}

func getUser(ctx *ninja.Context, in *GetUserInput) (*UserOut, error) {
	db := userDB(ctx)
	u, err := newUserRepo().SelectOneById(int(in.UserID), gormx.UseDB(db))
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
	db := userDB(ctx)
	u := &User{Name: in.Name, Email: in.Email, Age: in.Age}
	if err := newUserRepo().Insert(u, gormx.UseDB(db)); err != nil {
		return nil, err
	}
	out := toOut(*u)
	return &out, nil
}

func updateUser(ctx *ninja.Context, in *UpdateUserInput) (*UserOut, error) {
	db := userDB(ctx)
	r := newUserRepo()
	if _, err := r.SelectOneById(int(in.UserID), gormx.UseDB(db)); err != nil {
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
		if err := r.UpdateById(int(in.UserID), upd, gormx.UseDB(db)); err != nil {
			return nil, err
		}
	}
	u, err := r.SelectOneById(int(in.UserID), gormx.UseDB(db))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, ninja.NotFoundError()
		}
		return nil, err
	}
	out := toOut(u)
	return &out, nil
}

func deleteUser(ctx *ninja.Context, in *DeleteUserInput) error {
	db := userDB(ctx)
	return newUserRepo().DeleteById(int(in.UserID), gormx.UseDB(db))
}

func initDB(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(gormdriver.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&User{}); err != nil {
		return nil, err
	}
	orm.Init(db)
	return db, nil
}

func buildAPI(db *gorm.DB) *ninja.NinjaAPI {
	api := ninja.New(ninja.Config{
		Title:             "Gin Ninja Basic Example",
		Version:           "1.0.0",
		Prefix:            "/api/v1",
		DisableGinDefault: true,
	})

	// Attach infrastructure middleware.
	api.UseGin(
		ginpkg.Logger(),
		ginpkg.Recovery(),
		middleware.RequestID(),
		middleware.CORS(nil),
		orm.Middleware(db),
	)

	r := ninja.NewRouter("/users", ninja.WithTags("Users"))
	ninja.Get(r, "/", listUsers, ninja.Summary("List users"))
	ninja.Get(r, "/:id", getUser, ninja.Summary("Get user"))
	ninja.Post(r, "/", createUser, ninja.Summary("Create user"), ninja.WithTransaction())
	ninja.Put(r, "/:id", updateUser, ninja.Summary("Update user"), ninja.WithTransaction())
	ninja.Delete(r, "/:id", deleteUser, ninja.Summary("Delete user"), ninja.WithTransaction())
	api.AddRouter(r)

	api.Engine().GET("/health", func(c *ginpkg.Context) {
		c.JSON(http.StatusOK, ginpkg.H{"status": "ok"})
	})

	return api
}

func run(dsn, addr string) error {
	db, err := initDB(dsn)
	if err != nil {
		return err
	}
	api := buildAPI(db)

	log.Println("Docs: http://localhost:8080/docs")
	return api.Run(addr)
}

// ---------------------------------------------------------------------------
// Main
// ---------------------------------------------------------------------------

func main() {
	if err := runBasicMain("users.db", ":8080"); err != nil {
		fatalBasic(err)
	}
}
