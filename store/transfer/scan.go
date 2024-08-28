package transfer

import (
	"context"
	"database/sql"

	"github.com/fox-one/mixin-sdk-go/v2"
	"github.com/lib/pq"
	"github.com/pandodao/safe-wallet/core"
)

type querier interface {
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
}

type scanner interface {
	Scan(dest ...interface{}) error
}

var scanColumns = []string{
	"id",
	"created_at",
	"trace_id",
	"status",
	"user_id",
	"asset_id",
	"amount",
	"memo",
	"opponents",
	"threshold",
	"output_from",
	"output_to",
}

func scanTransfer(scanner scanner, transfer *core.Transfer) error {
	var (
		opponents pq.StringArray
		threshold uint8
		memo      sql.NullString
	)

	if err := scanner.Scan(
		&transfer.ID,
		&transfer.CreatedAt,
		&transfer.TraceID,
		&transfer.Status,
		&transfer.UserID,
		&transfer.AssetID,
		&transfer.Amount,
		&memo,
		&opponents,
		&threshold,
		&transfer.AssignRange[0],
		&transfer.AssignRange[1],
	); err != nil {
		return err
	}

	transfer.Memo = memo.String
	transfer.Opponent = mixin.RequireNewMixAddress(opponents, threshold)
	return nil
}
