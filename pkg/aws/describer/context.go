package describer

import (
	"context"
)

var (
	key describeContextKey = "describe_ctx"
)

type describeContextKey string

type DescribeContext struct {
	AccountID string
	Region    string
	Partition string
}

func WithDescribeContext(ctx context.Context, describeCtx DescribeContext) context.Context {
	return context.WithValue(ctx, key, describeCtx)
}

func GetDescribeContext(ctx context.Context) DescribeContext {
	describe, ok := ctx.Value(key).(DescribeContext)
	if !ok {
		panic("context key not found")
	}
	return describe
}
