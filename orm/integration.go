package orm

import (
	"errors"

	"github.com/gin-gonic/gin"
	ninja "github.com/shijl0925/gin-ninja"
	"gorm.io/gorm"
)

func init() {
	ninja.RegisterTransactionHandlers(beginTxFromGinContext, commitTxFromGinContext, rollbackTxFromGinContext, WithTransaction)
}

func beginTxFromGinContext(c *gin.Context) error {
	_, err := Begin(c)
	return err
}

func commitTxFromGinContext(c *gin.Context) error {
	return Commit(c)
}

func rollbackTxFromGinContext(c *gin.Context) error {
	return Rollback(c)
}

// BeginTx starts a request-scoped transaction for a ninja context.
func BeginTx(c *ninja.Context) error {
	if c == nil {
		return nil
	}
	return beginTxFromGinContext(c.Context)
}

// CommitTx commits the active request-scoped transaction for a ninja context.
func CommitTx(c *ninja.Context) error {
	if c == nil {
		return nil
	}
	return commitTxFromGinContext(c.Context)
}

// RollbackTx rolls back the active request-scoped transaction for a ninja context.
func RollbackTx(c *ninja.Context) error {
	if c == nil {
		return nil
	}
	return rollbackTxFromGinContext(c.Context)
}

// RegisterDefaultErrorMappers installs the standard GORM-to-ninja error mappings on a specific API instance.
func RegisterDefaultErrorMappers(api *ninja.NinjaAPI) {
	if api == nil {
		return
	}
	api.RegisterErrorMapper(func(err error) error {
		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			return ninja.NotFoundError()
		case errors.Is(err, gorm.ErrDuplicatedKey):
			return ninja.ConflictError()
		default:
			return nil
		}
	})
}
