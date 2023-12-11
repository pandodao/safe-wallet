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

func (s *store) GetOffset(ctx context.Context) (uint64, error) {
	b := sq.Select("sequence").
		From("outputs").
		OrderBy("sequence DESC").
		Limit(1)
	stmt, args := b.MustSql()
	row := s.db.QueryRowContext(ctx, stmt, args...)

	var seq uint64
	if err := row.Scan(&seq); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}

	return seq, nil
}

func save(ctx context.Context, tx *sql.Tx, output *core.Output) error {
	b := sq.Insert("outputs").
		Options("IGNORE").
		Columns("sequence", "created_at", "hash", "`index`", "asset_id", "amount").
		Values(output.Sequence, output.CreatedAt, output.Hash.String(), output.Index, output.AssetID, output.Amount)
	stmt, args := b.MustSql()
	_, err := tx.ExecContext(ctx, stmt, args...)
	return err
}

func (s *store) Save(ctx context.Context, outputs []*core.Output) error {
	tx := generic.Must(s.db.Begin())
	defer tx.Rollback()

	for _, output := range outputs {
		if err := save(ctx, tx, output); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *store) List(ctx context.Context, offset uint64, assetID string, target decimal.Decimal, limit int) ([]*core.Output, error) {
	b := sq.Select(scanColumns...).
		From("outputs").
		Where("asset_id = ? AND sequence > ?", assetID, offset).
		OrderBy("sequence").
		Limit(uint64(limit))
	stmt, args := b.MustSql()
	rows, err := s.db.QueryContext(ctx, stmt, args...)
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

func (s *store) ListRange(ctx context.Context, assetID string, from, to uint64) ([]*core.Output, error) {
	b := sq.Select(scanColumns...).
		From("outputs").
		Where("asset_id = ? AND sequence >= ? AND sequence <= ?", assetID, from, to).
		OrderBy("sequence")
	stmt, args := b.MustSql()
	rows, err := s.db.QueryContext(ctx, stmt, args...)
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

func (s *store) SumBalance(ctx context.Context, asset string) (*core.Balance, error) {
	b := sq.Select("outputs.asset_id", "SUM(outputs.amount)").
		From("outputs").
		LeftJoin("assigns ON outputs.asset_id = assigns.asset_id").
		Where("outputs.asset_id = ?", asset).
		Where("outputs.sequence > COALESCE(assigns.offset,0)").
		GroupBy("outputs.asset_id")
	stmt, args := b.MustSql()
	row := s.db.QueryRowContext(ctx, stmt, args...)

	balance := core.Balance{AssetID: asset}
	if err := row.Scan(&balance.AssetID, &balance.Amount); !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	return &balance, nil
}

func (s *store) SumBalances(ctx context.Context) ([]*core.Balance, error) {
	b := sq.Select("outputs.asset_id", "SUM(outputs.amount)").
		From("outputs").
		LeftJoin("assigns ON outputs.asset_id = assigns.asset_id").
		Where("outputs.sequence > COALESCE(assigns.offset,0)").
		GroupBy("outputs.asset_id")

	stmt, args := b.MustSql()
	rows, err := s.db.QueryContext(ctx, stmt, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var balances []*core.Balance
	for rows.Next() {
		var balance core.Balance
		if err := rows.Scan(&balance.AssetID, &balance.Amount); err != nil {
			return nil, err
		}

		balances = append(balances, &balance)
	}

	return balances, nil
}
