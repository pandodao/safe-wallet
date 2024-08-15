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
	UserID    string          `json:"user_id,omitempty"`
	AssetID   string          `json:"asset_id,omitempty"`
	Amount    decimal.Decimal `json:"amount"`
}

type Balance struct {
	UserID  string          `json:"user_id,omitempty"`
	AssetID string          `json:"asset_id,omitempty"`
	Amount  decimal.Decimal `json:"amount"`
	Count   int             `json:"count"`
}

type OutputStore interface {
	GetOffset(ctx context.Context) (uint64, error)
	Save(ctx context.Context, outputs []*Output) error
	List(ctx context.Context, userID string, offset uint64, limit int) ([]*Output, error)
	ListTarget(ctx context.Context, userID, assetID string, offset uint64, target decimal.Decimal, limit int) ([]*Output, error)
	ListRange(ctx context.Context, userID, assetID string, from, to uint64) ([]*Output, error)
	Delete(ctx context.Context, sequence uint64) error
	SumBalances(ctx context.Context, userID, assetID string) ([]*Balance, error)
}

type OutputService interface {
	Pull(ctx context.Context, offset uint64, limit int) ([]*Output, error)
	ListRange(ctx context.Context, assetID string, from, to uint64) ([]*Output, error)
	ReadState(ctx context.Context, output *Output) (string, error)
}
