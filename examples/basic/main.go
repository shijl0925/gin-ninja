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
	"strings"

	ginpkg "github.com/gin-gonic/gin"
	ninja "github.com/shijl0925/gin-ninja"
	"github.com/shijl0925/gin-ninja/middleware"
	"github.com/shijl0925/gin-ninja/orm"
	"github.com/shijl0925/gin-ninja/pagination"
	gormdriver "gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var runBasicMain = run
var fatalBasic = func(v ...any) { log.Fatal(v...) }
var basicDB *gorm.DB

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
// Data access
// ---------------------------------------------------------------------------

// ---------------------------------------------------------------------------
// Handlers
// ---------------------------------------------------------------------------

func toOut(u User) UserOut { return UserOut{u.ID, u.Name, u.Email, u.Age} }

func userDB(ctx *ninja.Context) *gorm.DB {
	if ctx != nil && ctx.Context != nil {
		return orm.WithContext(ctx.Context)
	}
	return basicDB
}

func loadUserByID(db *gorm.DB, id uint) (User, error) {
	var user User
	if err := db.Where("id = ?", id).First(&user).Error; err != nil {
		return User{}, err
	}
	return user, nil
}

func escapeLike(value string) string {
	replacer := strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`)
	return replacer.Replace(value)
}

func listUsers(ctx *ninja.Context, in *ListUsersInput) (*pagination.Page[UserOut], error) {
	db := userDB(ctx)
	query := db.Model(&User{})
	if in.Search != "" {
		query = query.Where("name LIKE ? ESCAPE '\\'", "%"+escapeLike(strings.TrimSpace(in.Search))+"%")
	}
	countQuery := query.Session(&gorm.Session{})
	var total int64
	if err := countQuery.Count(&total).Error; err != nil {
		return nil, err
	}
	var items []User
	if err := query.Limit(in.GetSize()).Offset(in.Offset()).Find(&items).Error; err != nil {
		return nil, err
	}
	out := make([]UserOut, len(items))
	for i, v := range items {
		out[i] = toOut(v)
	}
	return pagination.NewPage(out, total, in.PageInput), nil
}

func getUser(ctx *ninja.Context, in *GetUserInput) (*UserOut, error) {
	db := userDB(ctx)
	u, err := loadUserByID(db, in.UserID)
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
	if err := db.Create(u).Error; err != nil {
		return nil, err
	}
	out := toOut(*u)
	return &out, nil
}

func updateUser(ctx *ninja.Context, in *UpdateUserInput) (*UserOut, error) {
	db := userDB(ctx)
	if _, err := loadUserByID(db, in.UserID); err != nil {
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
		if err := db.Model(&User{}).Where("id = ?", in.UserID).Updates(upd).Error; err != nil {
			return nil, err
		}
	}
	u, err := loadUserByID(db, in.UserID)
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
	return db.Model(&User{}).Where("id = ?", in.UserID).Delete(&User{}).Error
}

func initDB(dsn string) (*gorm.DB, error) {
	db, err := gorm.Open(gormdriver.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}
	if err := db.AutoMigrate(&User{}); err != nil {
		return nil, err
	}
	basicDB = db
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
