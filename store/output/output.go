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

func (s *store) GetOffset(ctx context.Context, appID string) (uint64, error) {
	b := sq.Select("offset").From("apps").Where("id = ?", appID)
	row := b.RunWith(s.db).QueryRowContext(ctx)

	var seq uint64
	if err := row.Scan(&seq); err == nil {
		return seq, nil
	} else if errors.Is(err, sql.ErrNoRows) {
		return s.getOffsetFromOutputs(ctx, appID)
	} else {
		return 0, err
	}
}

func (s *store) getOffsetFromOutputs(ctx context.Context, appID string) (uint64, error) {
	b := sq.Select("sequence").
		From("outputs").
		Where("app_id = ?", appID).
		OrderBy("sequence DESC").
		Limit(1)
	row := b.RunWith(s.db).QueryRowContext(ctx)

	var seq uint64
	if err := row.Scan(&seq); err != nil && !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}

	return seq, nil
}

func saveOutput(ctx context.Context, tx *sql.Tx, output *core.Output) error {
	b := sq.Insert("outputs").
		Options("IGNORE").
		Columns("sequence", "created_at", "hash", "`index`", "user_id", "app_id", "asset_id", "amount").
		Values(output.Sequence, output.CreatedAt, output.Hash.String(), output.Index, output.UserID, output.AppID, output.AssetID, output.Amount)

	_, err := b.RunWith(tx).ExecContext(ctx)
	return err
}

func updateOffset(ctx context.Context, tx *sql.Tx, appID string, offset uint64) error {
	b := sq.Update("apps").Where("id = ?", appID).Set("offset", offset)
	result, err := b.RunWith(tx).ExecContext(ctx)
	if err != nil {
		return err
	}

	n, err := result.RowsAffected()
	if err != nil {
		return err
	}

	if n == 0 {
		return insertOffset(ctx, tx, appID, offset)
	}

	return nil
}

func insertOffset(ctx context.Context, tx *sql.Tx, appID string, offset uint64) error {
	b := sq.Insert("apps").
		Options("IGNORE").
		Columns("id", "sequence").
		Values(appID, offset)

	_, err := b.RunWith(tx).ExecContext(ctx)
	return err
}

func (s *store) Save(ctx context.Context, outputs []*core.Output) error {
	tx := generic.Must(s.db.Begin())
	defer tx.Rollback()

	offsets := map[string]uint64{}

	for _, output := range outputs {
		if err := saveOutput(ctx, tx, output); err != nil {
			return err
		}

		offsets[output.AppID] = output.Sequence
	}

	for id, offset := range offsets {
		if err := updateOffset(ctx, tx, id, offset); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func (s *store) List(ctx context.Context, userID string, offset uint64, limit int) ([]*core.Output, error) {
	b := sq.Select(scanColumns...).
		From("outputs").
		Where("user_id = ? AND sequence >= ?", userID, offset).
		Limit(uint64(limit)).
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

func (s *store) SumBalance(ctx context.Context, userID, assetID string) (*core.Balance, error) {
	b := sq.Select("SUM(outputs.amount)", "COUNT(*)").
		From("outputs").
		LeftJoin("assigns ON outputs.asset_id = assigns.asset_id AND outputs.user_id = assigns.user_id").
		Where("outputs.user_id = ? AND outputs.asset_id = ?", userID, assetID).
		Where("outputs.sequence > COALESCE(assigns.offset,0)")
	row := b.RunWith(s.db).QueryRowContext(ctx)

	balance := core.Balance{AssetID: assetID}
	if err := row.Scan(&balance.Amount, &balance.Count); !errors.Is(err, sql.ErrNoRows) {
		return nil, err
	}

	return &balance, nil
}

func (s *store) SumBalances(ctx context.Context, userID string) ([]*core.Balance, error) {
	b := sq.Select("outputs.asset_id", "SUM(outputs.amount)", "COUNT(*)").
		From("outputs").
		LeftJoin("assigns ON outputs.asset_id = assigns.asset_id AND outputs.user_id = assigns.user_id").
		Where("outputs.user_id = ?", userID).
		Where("outputs.sequence > COALESCE(assigns.offset,0)").
		GroupBy("outputs.asset_id")

	rows, err := b.RunWith(s.db).QueryContext(ctx)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var balances []*core.Balance
	for rows.Next() {
		var balance core.Balance
		if err := rows.Scan(&balance.AssetID, &balance.Amount, &balance.Count); err != nil {
			return nil, err
		}

		balances = append(balances, &balance)
	}

	return balances, nil
}
