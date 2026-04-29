package ninja

import (
	"fmt"

	"github.com/gin-gonic/gin"
)

// TransactionHandlers configures request-scoped transaction behavior for a
// NinjaAPI instance.
type TransactionHandlers struct {
	Begin           func(*gin.Context) error
	Commit          func(*gin.Context) error
	Rollback        func(*gin.Context) error
	WithTransaction func(*gin.Context, func() error) error
}

func errTransactionUnavailable() error {
	return fmt.Errorf("transaction helpers are unavailable; configure ninja.Config.TransactionHandlers")
}

func apiTransactionHandlers(api *NinjaAPI) (
	begin func(*gin.Context) error,
	commit func(*gin.Context) error,
	rollback func(*gin.Context) error,
	withTransaction func(*gin.Context, func() error) error,
) {
	if api == nil || api.transactions == nil {
		return nil, nil, nil, nil
	}
	handlers := api.transactions
	return handlers.Begin, handlers.Commit, handlers.Rollback, handlers.WithTransaction
}
