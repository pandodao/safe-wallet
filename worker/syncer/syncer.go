package syncer

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/pandodao/safe-wallet/core"
)

const (
	propertySyncOffset = "sync_offset"
)

func New(
	outputz core.OutputService,
	outputs core.OutputStore,
	properties core.PropertyStore,
	logger *slog.Logger,
) *Syncer {
	return &Syncer{
		outputz:    outputz,
		outputs:    outputs,
		properties: properties,
		logger:     logger.With("worker", "syncer"),
	}
}

type Syncer struct {
	outputz    core.OutputService
	outputs    core.OutputStore
	properties core.PropertyStore
	logger     *slog.Logger
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
	var offset uint64
	if err := w.properties.Get(ctx, propertySyncOffset, &offset); err != nil {
		w.logger.Error("properties.Get", "err", err)
		return err
	}

	const limit = 500
	outputs, nextOffset, err := w.outputz.Pull(ctx, offset, limit)
	if err != nil {
		w.logger.Error("outputz.Pull", "err", err)
		return err
	}

	if len(outputs) > 0 {
		w.logger.Info("list new outputs", "count", len(outputs), "offset", offset)

		if err := w.outputs.Save(ctx, outputs); err != nil {
			w.logger.Error("outputs.Save", "err", err)
			return err
		}
	}

	if nextOffset <= offset {
		return fmt.Errorf("no new outputs")
	}

	if err := w.properties.Set(ctx, propertySyncOffset, nextOffset); err != nil {
		w.logger.Error("properties.Set", "err", err)
		return err
	}

	return nil
}
