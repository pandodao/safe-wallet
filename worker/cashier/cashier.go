package cashier

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/pandodao/safe-wallet/core"
	"golang.org/x/sync/errgroup"
)

func New(
	outputs core.OutputStore,
	outputz core.OutputService,
	transfers core.TransferStore,
	transferz core.TransferService,
	logger *slog.Logger,
) *Cashier {
	return &Cashier{
		outputs:   outputs,
		outputz:   outputz,
		transfers: transfers,
		transferz: transferz,
		logger:    logger.With("worker", "cashier"),
	}
}

type Cashier struct {
	outputs   core.OutputStore
	outputz   core.OutputService
	transfers core.TransferStore
	transferz core.TransferService
	logger    *slog.Logger
}

func (w *Cashier) Run(ctx context.Context) error {
	w.logger.Info("cashier start")

	for {
		dur := 500 * time.Millisecond
		if w.run(ctx) == nil {
			dur = 200 * time.Millisecond
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(dur):
		}
	}
}

func (w *Cashier) run(ctx context.Context) error {
	const limit = 64
	transfers, err := w.transfers.ListStatus(ctx, core.TransferStatusAssigned, limit)
	if err != nil {
		w.logger.Error("transfers.ListStatus", "err", err)
		return err
	}

	if len(transfers) == 0 {
		return fmt.Errorf("assigned transfers dry")
	}

	var g errgroup.Group
	g.SetLimit(10)

	for idx := range transfers {
		transfer := transfers[idx]
		g.Go(func() error {
			return w.handleTransfer(ctx, transfer)
		})
	}

	return g.Wait()
}

func (w *Cashier) handleTransfer(ctx context.Context, transfer *core.Transfer) error {
	logger := w.logger.With("transfer", transfer.TraceID)

	logger.Info("handle transfer", "asset", transfer.AssetID, "amount", transfer.Amount)

	outputs, err := w.outputs.ListRange(ctx, transfer.UserID, transfer.AssetID, transfer.AssignRange[0], transfer.AssignRange[1])
	if err != nil {
		logger.Error("outputs.ListRange", "err", err)
		return err
	}

	if len(outputs) == 0 {
		outputs, err = w.outputz.ListRange(ctx, transfer.AssetID, transfer.AssignRange[0], transfer.AssignRange[1])
		if err != nil {
			logger.Error("outputz.ListRange", "err", err)
			return err
		}
	}

	if len(outputs) == 0 {
		logger.Error("spend outputs dry", "from", transfer.AssignRange[0], "to", transfer.AssignRange[1])
		return fmt.Errorf("spend outputs dry")
	}

	logger.Debug("assigned outputs loaded", "count", len(outputs))

	if err := w.transferz.Spend(ctx, transfer, outputs); err != nil {
		logger.Error("transferz.Spend", "err", err)
		return err
	}

	logger.Debug("transfer spend done")

	if err := w.transfers.UpdateStatus(ctx, transfer, core.TransferStatusHandled); err != nil {
		logger.Error("transfers.UpdateStatus", "err", err)
		return err
	}

	logger.Debug("transfer status updated")
	return nil
}
