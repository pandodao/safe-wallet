package core

import (
	"context"
	"time"

	"github.com/fox-one/mixin-sdk-go/v2"
	"github.com/shopspring/decimal"
)

type TransferStatus uint8

const (
	_ TransferStatus = iota
	TransferStatusPending
	TransferStatusAssigned
	TransferStatusHandled
)

//go:generate enumer -type=TransferStatus -trimprefix=TransferStatus -json

type Transfer struct {
	ID          uint64            `json:"id,omitempty"`
	CreatedAt   time.Time         `json:"created_at,omitempty"`
	TraceID     string            `json:"trace_id,omitempty"`
	Status      TransferStatus    `json:"state,omitempty"`
	UserID      string            `json:"user_id,omitempty"`
	AssetID     string            `json:"asset_id,omitempty"`
	Amount      decimal.Decimal   `json:"amount,omitempty"`
	Memo        string            `json:"memo,omitempty"`
	Opponent    *mixin.MixAddress `json:"opponent,omitempty"`
	AssignRange [2]uint64         `json:"assign_range,omitempty"`
}

type TransferStore interface {
	Create(ctx context.Context, transfer *Transfer) error
	Assign(ctx context.Context, transfer *Transfer, offset uint64) error
	UpdateStatus(ctx context.Context, transfer *Transfer, to TransferStatus) error
	FindTrace(ctx context.Context, traceID string) (*Transfer, error)
	ListStatus(ctx context.Context, status TransferStatus, limit int) ([]*Transfer, error)
	GetAssignOffset(ctx context.Context, userID, assetID string) (uint64, error)
}

type TransferService interface {
	Find(ctx context.Context, traceID string) (*Transfer, error)
	Spend(ctx context.Context, transfer *Transfer, outputs []*Output) error
}
