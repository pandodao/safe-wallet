package output

import (
	"context"
	"database/sql"
	"errors"

	sq "github.com/Masterminds/squirrel"
	"github.com/pandodao/generic"
	"github.com/pandodao/safe-wallet/core"
	"github.com/shopspring/decimal"
	"github.com/tsenart/nap"
)

func New(db *nap.DB) core.OutputStore {
	return &store{db: db}
}

type store struct {
	db *nap.DB
}

func (s *store) getMaxAssignOffset(ctx context.Context) (uint64, error) {
	b := sq.Select("MAX(offset)").From("assigns")
	row := b.RunWith(s.db).QueryRowContext(ctx)

	var seq uint64
	if err := row.Scan(&seq); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}

	return seq, nil
}

func (s *store) GetOffset(ctx context.Context) (uint64, error) {
	b := sq.Select("sequence").
		From("outputs").
		OrderBy("sequence DESC").
		Limit(1)
	row := b.RunWith(s.db).QueryRowContext(ctx)

	var seq uint64
	if err := row.Scan(&seq); err == nil {
		return seq, nil
	} else if errors.Is(err, sql.ErrNoRows) {
		return s.getMaxAssignOffset(ctx)
	} else {
		return 0, err
	}
}

func saveOutput(ctx context.Context, tx *sql.Tx, output *core.Output) error {
	b := sq.Insert("outputs").
		Options("IGNORE").
		Columns("sequence", "created_at", "hash", "`index`", "user_id", "asset_id", "amount").
		Values(output.Sequence, output.CreatedAt, output.Hash.String(), output.Index, output.UserID, output.AssetID, output.Amount)

	_, err := b.RunWith(tx).ExecContext(ctx)
	return err
}

func (s *store) Save(ctx context.Context, outputs []*core.Output) error {
	tx := generic.Must(s.db.Begin())
	defer tx.Rollback()

	for _, output := range outputs {
		if err := saveOutput(ctx, tx, output); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *store) List(ctx context.Context, userID string, offset uint64, limit int) ([]*core.Output, error) {
	b := sq.Select(scanColumns...).
		From("outputs").
		Where("sequence >= ?", offset).
		Limit(uint64(limit)).
		OrderBy("sequence")

	if userID != "" {
		b = b.Where("user_id = ?", userID)
	}

	rows, err := b.RunWith(s.db).QueryContext(ctx)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var outputs []*core.Output
	for rows.Next() {
		var output core.Output
		if err := scanOutput(rows, &output); err != nil {
			return nil, err
		}

		outputs = append(outputs, &output)
	}

	return outputs, nil
}

func (s *store) ListTarget(ctx context.Context, userID, assetID string, offset uint64, target decimal.Decimal, limit int) ([]*core.Output, error) {
	b := sq.Select(scanColumns...).
		From("outputs").
		Where("user_id = ? AND asset_id = ? AND sequence > ?", userID, assetID, offset).
		OrderBy("sequence").
		Limit(uint64(limit))
	rows, err := b.RunWith(s.db).QueryContext(ctx)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var (
		outputs []*core.Output
		sum     decimal.Decimal
	)

	for rows.Next() {
		var output core.Output
		if err := scanOutput(rows, &output); err != nil {
			return nil, err
		}

		outputs = append(outputs, &output)
		if sum = sum.Add(output.Amount); sum.GreaterThanOrEqual(target) {
			break
		}
	}

	return outputs, nil
}

func (s *store) ListRange(ctx context.Context, userID, assetID string, from, to uint64) ([]*core.Output, error) {
	b := sq.Select(scanColumns...).
		From("outputs").
		Where("user_id = ? AND asset_id = ? AND sequence >= ? AND sequence <= ?", userID, assetID, from, to).
		OrderBy("sequence")
	rows, err := b.RunWith(s.db).QueryContext(ctx)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var outputs []*core.Output
	for rows.Next() {
		var output core.Output
		if err := scanOutput(rows, &output); err != nil {
			return nil, err
		}

		outputs = append(outputs, &output)
	}

	return outputs, nil
}

func (s *store) Delete(ctx context.Context, seq uint64) error {
	b := sq.Delete("outputs").Where("sequence = ?", seq)
	_, err := b.RunWith(s.db).ExecContext(ctx)
	return err
}

func (s *store) SumBalances(ctx context.Context, userID, assetID string) ([]*core.Balance, error) {
	b := sq.Select("outputs.user_id", "outputs.asset_id", "SUM(outputs.amount)", "COUNT(*)").
		From("outputs").
		LeftJoin("assigns ON outputs.asset_id = assigns.asset_id AND outputs.user_id = assigns.user_id").
		Where("outputs.sequence > COALESCE(assigns.offset,0)").
		GroupBy("outputs.user_id", "outputs.asset_id")

	if userID != "" {
		b = b.Where("outputs.user_id = ?", userID)

		if assetID != "" {
			b = b.Where("outputs.asset_id = ?", assetID)
		}
	}

	rows, err := b.RunWith(s.db).QueryContext(ctx)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var balances []*core.Balance
	for rows.Next() {
		var balance core.Balance
		if err := rows.Scan(&balance.UserID, &balance.AssetID, &balance.Amount, &balance.Count); err != nil {
			return nil, err
		}

		balances = append(balances, &balance)
	}

	return balances, nil
}
