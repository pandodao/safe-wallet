package core

import (
	"context"
	"time"

	"github.com/fox-one/mixin-sdk-go/v2/mixinnet"
	"github.com/shopspring/decimal"
)

type Output struct {
	Sequence  uint64          `json:"sequence,omitempty"`
	CreatedAt time.Time       `json:"created_at"`
	Hash      mixinnet.Hash   `json:"hash,omitempty"`
	Index     uint8           `json:"index,omitempty"`
	AssetID   string          `json:"asset_id,omitempty"`
	Amount    decimal.Decimal `json:"amount"`
}

type Balance struct {
	AssetID string          `json:"asset_id,omitempty"`
	Amount  decimal.Decimal `json:"amount"`
}

type OutputStore interface {
	GetOffset(ctx context.Context) (uint64, error)
	Save(ctx context.Context, outputs []*Output) error
	List(ctx context.Context, offset uint64, assetID string, target decimal.Decimal, limit int) ([]*Output, error)
	ListRange(ctx context.Context, assetID string, from, to uint64) ([]*Output, error)
	Clean(ctx context.Context, assetID string, offset uint64) error
	SumBalance(ctx context.Context, asset string) (*Balance, error)
	SumBalances(ctx context.Context) ([]*Balance, error)
}

type OutputService interface {
	Pull(ctx context.Context, offset uint64, limit int) ([]*Output, error)
	ListRange(ctx context.Context, assetID string, from, to uint64) ([]*Output, error)
}
