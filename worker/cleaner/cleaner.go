package cleaner

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/fox-one/mixin-sdk-go/v2"
	"github.com/google/uuid"
	"github.com/pandodao/safe-wallet/core"
	"github.com/zyedidia/generic/mapset"
)

type Config struct {
	Capacity int `valid:"required"`
}

type Cleaner struct {
	outputs   core.OutputStore
	outputz   core.OutputService
	transfers core.TransferStore
	logger    *slog.Logger
	cfg       Config
}

func New(
	outputs core.OutputStore,
	transfers core.TransferStore,
	outputz core.OutputService,
	logger *slog.Logger,
	cfg Config,
) *Cleaner {
	if _, err := govalidator.ValidateStruct(cfg); err != nil {
		panic(err)
	}

	return &Cleaner{
		outputs:   outputs,
		transfers: transfers,
		outputz:   outputz,
		logger:    logger.With("worker", "cleaner"),
		cfg:       cfg,
	}
}

func (w *Cleaner) Run(ctx context.Context) error {
	w.logger.Info("cleaner start")

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
			_ = w.run(ctx)
		}
	}
}

func (w *Cleaner) run(ctx context.Context) error {
	var (
		offset uint64
		assets = mapset.New[string]()
	)

	for {
		const limit = 500
		outputs, err := w.outputs.List(ctx, "", offset, limit)
		if err != nil {
			w.logger.Error("outputs.List", "err", err)
			return err
		}

		if len(outputs) == 0 {
			break
		}

		for _, output := range outputs {
			offset = output.Sequence + 1

			key := fmt.Sprintf("%s-%s", output.UserID, output.AssetID)
			if assets.Has(key) {
				continue
			}

			outputs, _, err := w.outputz.Pull(ctx, output.Sequence, 1)
			if err != nil {
				w.logger.Error("outputz.Pull", "err", err)
				return err
			}

			// output 被返回说明该 output 是 unspent 的
			if len(outputs) > 0 && outputs[0].Sequence == output.Sequence {
				assets.Put(key)
				continue
			}

			if err := w.outputs.Delete(ctx, output.Sequence); err != nil {
				w.logger.Error("outputs.Delete", "err", err)
				return err
			}
		}
	}

	return w.mergeOutputs(ctx)
}

// mergeOutputs 尝试将比较碎的币主动合并
func (w *Cleaner) mergeOutputs(ctx context.Context) error {
	balances, err := w.outputs.SumBalances(ctx, "", "")
	if err != nil {
		w.logger.Error("outputs.SumBalances", "err", err)
		return err
	}

	for _, b := range balances {
		if b.Count <= w.cfg.Capacity {
			continue
		}

		offset, err := w.transfers.GetAssignOffset(ctx, b.UserID, b.AssetID)
		if err != nil {
			w.logger.Error("transfers.GetAssignOffset", "err", err)
			return err
		}

		const limit = 256
		outputs, err := w.outputs.ListTarget(ctx, b.UserID, b.AssetID, offset, b.Amount, limit)
		if err != nil {
			w.logger.Error("outputs.ListTarget", "err", err)
			return err
		}

		if len(outputs) < limit {
			continue
		}

		t := &core.Transfer{
			CreatedAt: time.Now(),
			TraceID:   uuid.NewString(),
			Status:    core.TransferStatusPending,
			AssetID:   b.AssetID,
			Memo:      "auto merge",
			Opponent:  mixin.RequireNewMixAddress([]string{b.UserID}, 1),
		}

		t.AssignRange[0] = outputs[0].Sequence
		for _, output := range outputs {
			t.Amount = t.Amount.Add(output.Amount)
			t.AssignRange[1] = output.Sequence
		}

		if err := w.transfers.Assign(ctx, t, offset); err != nil {
			w.logger.Error("transfers.Assign", "err", err)
			return err
		}
	}

	return nil
}
