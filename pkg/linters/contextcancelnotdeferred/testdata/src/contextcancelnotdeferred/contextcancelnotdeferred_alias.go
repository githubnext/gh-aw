package contextcancelnotdeferred

import (
	ctx "context"
	"time"
)

// flagged: aliased context import still resolves to context package
func BadWithCancelAliased(parent ctx.Context) error {
	c, cancel := ctx.WithCancel(parent) // want `context cancel function should be deferred immediately after context.WithCancel/WithTimeout/WithDeadline`
	_ = c
	cancel()
	return nil
}

// not flagged: defer used correctly with aliased import
func GoodWithTimeoutAliased(parent ctx.Context) error {
	c, cancel := ctx.WithTimeout(parent, time.Second)
	defer cancel()
	_ = c
	return nil
}
