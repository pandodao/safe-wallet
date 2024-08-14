package transfer

import (
	"context"
	"database/sql"
	"fmt"

	sq "github.com/Masterminds/squirrel"
)

type Assign struct {
	UserID   string
	AssetID  string
	Offset   uint64
	Transfer string
}

func insertAssign(ctx context.Context, tx *sql.Tx, assign *Assign) error {
	b := sq.Insert("assigns").
		Columns("user_id", "asset_id", "offset", "transfer").
		Values(assign.UserID, assign.AssetID, assign.Offset, assign.Transfer)
	_, err := b.RunWith(tx).ExecContext(ctx)
	return err
}

func updateAssign(ctx context.Context, tx *sql.Tx, assign *Assign, previousOffset uint64) error {
	if previousOffset == 0 {
		return insertAssign(ctx, tx, assign)
	}

	b := sq.Update("assigns").
		Set("offset", assign.Offset).
		Set("transfer", assign.Transfer).
		Where("user_id = ? AND asset_id = ? AND offset = ?", assign.UserID, assign.AssetID, previousOffset)

	r, err := b.RunWith(tx).ExecContext(ctx)
	if err != nil {
		return err
	}

	n, err := r.RowsAffected()
	if err != nil {
		return err
	}

	if n == 0 {
		return fmt.Errorf("optimistic lock failed")
	}

	return nil
}
