package cleaner

import (
	"context"
	"log/slog"
	"time"

	"github.com/pandodao/safe-wallet/core"
	"github.com/zyedidia/generic/mapset"
)

type Cleaner struct {
	outputs core.OutputStore
	outputz core.OutputService
	logger  *slog.Logger
}

func New(
	outputs core.OutputStore,
	outputz core.OutputService,
	logger *slog.Logger,
) *Cleaner {
	return &Cleaner{
		outputs: outputs,
		outputz: outputz,
		logger:  logger.With("worker", "cleaner"),
	}
}

func (w *Cleaner) Run(ctx context.Context) error {
	w.logger.Info("cleaner start")

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

func (w *Cleaner) run(ctx context.Context) error {
	assets := mapset.New[string]()

	var seq uint64
	for {
		outputs, err := w.outputz.Pull(ctx, seq+1, 500)
		if err != nil {
			w.logger.Error("outputz.Pull", "err", err)
			return err
		}

		if len(outputs) == 0 {
			break
		}

		for _, output := range outputs {
			seq = output.Sequence

			if assets.Has(output.AssetID) {
				continue
			}

			w.logger.Debug("clean outputs", "asset", output.AssetID, "offset", output.Sequence)
			if err := w.outputs.Clean(ctx, output.AssetID, output.Sequence); err != nil {
				w.logger.Error("outputs.Clean", "err", err)
				return err
			}

			assets.Put(output.AssetID)
		}
	}

	return nil
}
