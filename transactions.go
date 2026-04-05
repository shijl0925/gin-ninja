package ninja

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

var (
	contextBeginTx    func(*gin.Context) error
	contextCommitTx   func(*gin.Context) error
	contextRollbackTx func(*gin.Context) error
	contextWithTx     func(*gin.Context, func() error) error
)

// RegisterTransactionHandlers configures the deprecated Context transaction helpers.
func RegisterTransactionHandlers(begin, commit, rollback func(*gin.Context) error, withTransaction func(*gin.Context, func() error) error) {
	contextBeginTx = begin
	contextCommitTx = commit
	contextRollbackTx = rollback
	contextWithTx = withTransaction
}

func errTransactionUnavailable() error {
	return fmt.Errorf("transaction helpers are unavailable; import github.com/shijl0925/gin-ninja/orm and use orm.BeginTx/CommitTx/RollbackTx")
}
