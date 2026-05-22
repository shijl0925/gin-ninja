package orm

import "testing"

func TestTransactionHandlersIncludesRollbackIntegration(t *testing.T) {
	handlers := TransactionHandlers()
	if handlers == nil {
		t.Fatal("expected transaction handlers")
	}
	if handlers.Begin == nil || handlers.Commit == nil || handlers.Rollback == nil || handlers.WithTransaction == nil {
		t.Fatalf("expected all transaction handlers to be populated: %+v", handlers)
	}
}

func TestRollbackTxFromGinContextRollsBackActiveTransaction(t *testing.T) {
	c, db := newTxContext(t)
	name := "integration-rollback-user"

	if err := beginTxFromGinContext(c); err != nil {
		t.Fatalf("beginTxFromGinContext: %v", err)
	}
	if !InTransaction(c) {
		t.Fatal("expected transaction to be active")
	}
	if err := WithContext(c).Create(&txUser{Name: name}).Error; err != nil {
		t.Fatalf("Create: %v", err)
	}
	if err := rollbackTxFromGinContext(c); err != nil {
		t.Fatalf("rollbackTxFromGinContext: %v", err)
	}
	if InTransaction(c) {
		t.Fatal("expected transaction to be cleared after rollback")
	}

	var count int64
	if err := db.Model(&txUser{}).Where("name = ?", name).Count(&count).Error; err != nil {
		t.Fatalf("count: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected rollback to leave 0 rows, got %d", count)
	}
}
