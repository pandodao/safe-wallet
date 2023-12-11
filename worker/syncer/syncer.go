package syncer

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/pandodao/safe-wallet/core"
)

func New(
	outputz core.OutputService,
	outputs core.OutputStore,
	logger *slog.Logger,
) *Syncer {
	return &Syncer{
		outputz: outputz,
		outputs: outputs,
		logger:  logger.With("worker", "syncer"),
	}
}

type Syncer struct {
	outputz core.OutputService
	outputs core.OutputStore
	logger  *slog.Logger
}

func (w *Syncer) Run(ctx context.Context) error {
	w.logger.Info("syncer start")

	for {
		dur := time.Second
		if w.run(ctx) == nil {
			dur = 500 * time.Millisecond
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(dur):
		}
	}
}

func (w *Syncer) run(ctx context.Context) error {
	offset, err := w.outputs.GetOffset(ctx)
	if err != nil {
		w.logger.Error("outputs.GetOffset", "err", err)
		return err
	}

	const limit = 500
	outputs, err := w.outputz.Pull(ctx, offset+1, limit)
	if err != nil {
		w.logger.Error("outputz.Pull", "err", err)
		return err
	}

	if len(outputs) == 0 {
		return fmt.Errorf("outputs dry")
	}

	w.logger.Info("list new outputs", "count", len(outputs), "offset", offset)

	if err := w.outputs.Save(ctx, outputs); err != nil {
		w.logger.Error("outputs.Save", "err", err)
		return err
	}

	tail := outputs[len(outputs)-1]
	w.logger.Info("outputs update", "seq", tail.Sequence, "at", tail.CreatedAt)
	return nil
}
