package transfer

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	"github.com/lib/pq"
	"github.com/pandodao/generic"
	"github.com/pandodao/safe-wallet/core"
	"github.com/tsenart/nap"
)

func New(db *nap.DB) core.TransferStore {
	return &store{db: db}
}

type store struct {
	db *nap.DB
}

func insert(ctx context.Context, e execer, transfer *core.Transfer) error {
	opponents := pq.StringArray(transfer.Opponent.Members())
	threshold := transfer.Opponent.Threshold
	b := sq.Insert("transfers").
		Columns("trace_id", "status", "asset_id", "amount", "memo", "opponents", "threshold", "output_from", "output_to").
		Values(transfer.TraceID, transfer.Status, transfer.AssetID, transfer.Amount, transfer.Memo, opponents, threshold, transfer.AssignRange[0], transfer.AssignRange[1])
	stmt, args := b.MustSql()
	_, err := e.ExecContext(ctx, stmt, args...)
	return err
}

func update(ctx context.Context, e execer, transfer *core.Transfer, to core.TransferStatus) error {
	b := sq.Update("transfers").
		Set("status", to).
		Set("output_from", transfer.AssignRange[0]).
		Set("output_to", transfer.AssignRange[1]).
		Where("id = ? AND status == ?", transfer.ID, transfer.Status)
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

	return err
}

func (s *store) Create(ctx context.Context, transfer *core.Transfer) error {
	return insert(ctx, s.db, transfer)
}

func (s *store) Assign(ctx context.Context, transfer *core.Transfer, previousOffset uint64) error {
	tx := generic.Must(s.db.Begin())
	defer tx.Rollback()

	if err := updateAssign(ctx, tx, &Assign{
		AssetID:  transfer.AssetID,
		Offset:   transfer.AssignRange[1],
		Transfer: transfer.TraceID,
	}, previousOffset); err != nil {
		return err
	}

	if transfer.ID == 0 {
		transfer.Status = core.TransferStatusAssigned
		return insert(ctx, tx, transfer)
	}

	return update(ctx, tx, transfer, core.TransferStatusAssigned)
}

func (s *store) UpdateStatus(ctx context.Context, transfer *core.Transfer, to core.TransferStatus) error {
	return update(ctx, s.db, transfer, to)
}

func (s *store) FindTrace(ctx context.Context, traceID string) (*core.Transfer, error) {
	b := sq.Select(scanColumns...).
		From("transfers").
		Where("trace_id = ?", traceID)
	stmt, args := b.MustSql()
	row := s.db.QueryRowContext(ctx, stmt, args...)
	var transfer core.Transfer
	if err := scanTransfer(row, &transfer); err != nil {
		return nil, err
	}

	return &transfer, nil
}

func (s *store) ListStatus(ctx context.Context, status core.TransferStatus, limit int) ([]*core.Transfer, error) {
	b := sq.Select(scanColumns...).
		From("transfers").
		Where("status = ?", status).
		OrderBy("id").
		Limit(uint64(limit))
	stmt, args := b.MustSql()
	rows, err := s.db.QueryContext(ctx, stmt, args...)
	if err != nil {
		return nil, err
	}

	defer rows.Close()

	var transfers []*core.Transfer
	for rows.Next() {
		var transfer core.Transfer
		if err := scanTransfer(rows, &transfer); err != nil {
			return nil, err
		}

		transfers = append(transfers, &transfer)
	}

	return transfers, nil
}

func (s *store) GetAssignOffset(ctx context.Context, assetID string) (uint64, error) {
	b := sq.Select("offset").
		From("assigns").
		Where("asset_id = ?", assetID)
	stmt, args := b.MustSql()
	row := s.db.QueryRowContext(ctx, stmt, args...)
	var offset uint64
	if err := row.Scan(&offset); !errors.Is(err, sql.ErrNoRows) {
		return 0, err
	}

	return offset, nil
}
