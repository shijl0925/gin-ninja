package orm

import "github.com/gin-gonic/gin"

// Begin starts a request-scoped transaction if one is not already active.
func Begin(c *gin.Context) (*gorm.DB, error) {
	if tx, ok := currentTx(c); ok {
		return tx.WithContext(c.Request.Context()), nil
	}

	tx := GetBaseDB(c).WithContext(c.Request.Context()).Begin()
	if tx.Error != nil {
		return nil, tx.Error
	}
	c.Set(txContextKey, tx)
	return tx, nil
}

// Commit commits the active request-scoped transaction.
func Commit(c *gin.Context) error {
	tx, ok := currentTx(c)
	if !ok {
		return nil
	}
	clearContextKey(c, txContextKey)
	return tx.Commit().Error
}

// Rollback rolls back the active request-scoped transaction.
func Rollback(c *gin.Context) error {
	tx, ok := currentTx(c)
	if !ok {
		return nil
	}
	clearContextKey(c, txContextKey)
	return tx.Rollback().Error
}

// InTransaction reports whether a request-scoped transaction is active.
func InTransaction(c *gin.Context) bool {
	_, ok := currentTx(c)
	return ok
}

// WithTransaction executes fn in a request-scoped transaction.
// Nested calls reuse the active transaction.
func WithTransaction(c *gin.Context, fn func() error) (err error) {
	if fn == nil {
		return nil
	}
	if InTransaction(c) {
		return fn()
	}

	if _, err = Begin(c); err != nil {
		return err
	}
	defer func() {
		if r := recover(); r != nil {
			_ = Rollback(c)
			panic(r)
		}
	}()

	if err = fn(); err != nil {
		_ = Rollback(c)
		return err
	}
	return Commit(c)
}

func currentTx(c *gin.Context) (*gorm.DB, bool) {
	if v, ok := c.Get(txContextKey); ok {
		if tx, ok := v.(*gorm.DB); ok {
			return tx, true
		}
	}
	return nil, false
}

func clearContextKey(c *gin.Context, key string) {
	if c.Keys == nil {
		return
	}
	delete(c.Keys, key)
}
