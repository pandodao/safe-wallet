package core

import "context"

type PropertyStore interface {
	Get(ctx context.Context, key string, value any) error
	Set(ctx context.Context, key string, value any) error
}
