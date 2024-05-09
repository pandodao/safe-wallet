package cleaner

import (
	"context"
	"log/slog"
	"time"

	"github.com/asaskevich/govalidator"
	"github.com/fox-one/mixin-sdk-go/v2"
	"github.com/google/uuid"
	"github.com/pandodao/safe-wallet/core"
	"github.com/zyedidia/generic/mapset"
)

type Config struct {
	Receiver *mixin.MixAddress
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
	outputz core.OutputService,
	transfers core.TransferStore,
	logger *slog.Logger,
	cfg Config,
) *Cleaner {
	if _, err := govalidator.ValidateStruct(cfg); err != nil {
		panic(err)
	}

	return &Cleaner{
		outputs:   outputs,
		outputz:   outputz,
		transfers: transfers,
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
		// assets 保存轮询到出现 unspent utxo 的 asset，需要跳过
		assets = mapset.New[string]()
		offset uint64
	)

	for {
		const limit = 500
		outputs, err := w.outputs.List(ctx, offset, limit)
		if err != nil {
			w.logger.Error("outputs.List", "err", err)
			return err
		}

		if len(outputs) == 0 {
			break
		}

		for _, output := range outputs {
			offset = output.Sequence + 1

			if assets.Has(output.AssetID) {
				continue
			}

			state, err := w.outputz.ReadState(ctx, output)
			if err != nil {
				w.logger.Error("outputz.ReadState", "err", err)
				return err
			}

			switch mixin.SafeUtxoState(state) {
			case mixin.SafeUtxoStateSpent:
				if err := w.outputs.Delete(ctx, output.Sequence); err != nil {
					w.logger.Error("outputs.Delete", "seq", output.Sequence, "err", err)
					return err
				}
			default:
				assets.Put(output.AssetID)
			}
		}
	}

	return w.mergeOutputs(ctx)
}

// mergeOutputs 尝试将比较碎的币主动合并
func (w *Cleaner) mergeOutputs(ctx context.Context) error {
	balances, err := w.outputs.SumBalances(ctx)
	if err != nil {
		w.logger.Error("outputs.SumBalances", "err", err)
		return err
	}

	for _, b := range balances {
		if b.Count <= w.cfg.Capacity {
			continue
		}

		offset, err := w.transfers.GetAssignOffset(ctx, b.AssetID)
		if err != nil {
			w.logger.Error("transfers.GetAssignOffset", "err", err)
			return err
		}

		const limit = 256
		outputs, err := w.outputs.ListTarget(ctx, offset, b.AssetID, b.Amount, limit)
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
			Opponent:  w.cfg.Receiver,
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
