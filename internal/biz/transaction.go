package biz

import "context"

// Transaction is the interface for managing database transactions.
// Defined in biz layer, implemented by data/infra layer.
type Transaction interface {
	InTx(context.Context, func(ctx context.Context) error) error
}
