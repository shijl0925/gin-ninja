// Package orm provides a thin integration layer between gin-ninja and the
// go-toolkits/gormx ORM library (https://github.com/shijl0925/go-toolkits).
//
// Quick start:
//
//	import (
//	    "github.com/shijl0925/gin-ninja/orm"
//	    "gorm.io/driver/sqlite"
//	    "gorm.io/gorm"
//	)
//
//	func main() {
//	    db, _ := gorm.Open(sqlite.Open("app.db"), &gorm.Config{})
//	    orm.Init(db)
//
//	    api := ninja.New(ninja.Config{Title: "My API"})
//	    api.Engine().Use(orm.Middleware(db))
//	    // ...
//	}
package orm

import (
	"github.com/gin-gonic/gin"
	"github.com/shijl0925/go-toolkits/gormx"
	"gorm.io/gorm"
)

const dbContextKey = "gin_ninja_db"
const txContextKey = "gin_ninja_db_tx"

// Init initialises the global gormx database instance.
// This is equivalent to calling gormx.Init directly.
func Init(db *gorm.DB) {
	gormx.Init(db)
}

// Middleware returns a gin middleware that stores the given *gorm.DB in the
// request context under a known key.  Use GetDB to retrieve it in handlers.
//
//	api.Engine().Use(orm.Middleware(db))
func Middleware(db *gorm.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Set(dbContextKey, db)
		c.Next()
	}
}

func baseDB(c *gin.Context) *gorm.DB {
	if v, ok := c.Get(dbContextKey); ok {
		if db, ok := v.(*gorm.DB); ok {
			return db
		}
	}
	return gormx.GetDb()
}

// GetDB retrieves the *gorm.DB stored in the gin context by Middleware.
// If none was stored (e.g. the middleware was not registered), it falls back
// to the global gormx instance.
func GetDB(c *gin.Context) *gorm.DB {
	if v, ok := c.Get(txContextKey); ok {
		if db, ok := v.(*gorm.DB); ok {
			return db
		}
	}
	return baseDB(c)
}

// GetBaseDB retrieves the non-transactional request database.
func GetBaseDB(c *gin.Context) *gorm.DB {
	return baseDB(c)
}

// WithContext returns a *gorm.DB scoped to the request context, enabling
// proper request-scoped tracing and cancellation.
//
//	db := orm.WithContext(c)
//	db.Find(&users)
func WithContext(c *gin.Context) *gorm.DB {
	return GetDB(c).WithContext(c.Request.Context())
}
