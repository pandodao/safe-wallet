package transfer

import (
	"context"
	"fmt"

	sq "github.com/Masterminds/squirrel"
)

type Assign struct {
	AssetID  string
	Offset   uint64
	Transfer string
}

func insertAssign(ctx context.Context, e execer, assign *Assign) error {
	b := sq.Insert("assigns").
		Columns("asset_id", "offset", "transfer").
		Values(assign.AssetID, assign.Offset, assign.Transfer)
	stmt, args := b.MustSql()
	_, err := e.ExecContext(ctx, stmt, args...)
	return err
}

func updateAssign(ctx context.Context, e execer, assign *Assign, previousOffset uint64) error {
	if previousOffset == 0 {
		return insertAssign(ctx, e, assign)
	}

	b := sq.Update("assigns").
		Set("offset", assign.Offset).
		Set("transfer", assign.Transfer).
		Where("asset_id = ? AND offset = ?", assign.AssetID, previousOffset)

	stmt, args := b.MustSql()
	r, err := e.ExecContext(ctx, stmt, args...)
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
