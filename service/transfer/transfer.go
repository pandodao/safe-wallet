package transfer

import (
	"context"
	"encoding/hex"

	"github.com/fox-one/mixin-sdk-go/v2"
	"github.com/fox-one/mixin-sdk-go/v2/mixinnet"
	"github.com/pandodao/generic"
	"github.com/pandodao/safe-wallet/core"
	"github.com/shopspring/decimal"
)

func New(assetz core.AssetService, client *mixin.Client, key mixinnet.Key) core.TransferService {
	return &service{
		client:   client,
		spendKey: key,
		assetz:   assetz,
	}
}

type service struct {
	client   *mixin.Client
	spendKey mixinnet.Key

	assetz core.AssetService
}

func (s *service) Find(ctx context.Context, traceID string) (*core.Transfer, error) {
	req, err := s.client.SafeReadTransactionRequest(ctx, traceID)
	if err != nil {
		return nil, err
	}

	receiver := req.Receivers[0]
	opponent := mixin.RequireNewMixAddress(receiver.Members, receiver.Threshold)

	tx := generic.Must(mixinnet.TransactionFromRaw(req.RawTransaction))
	amount := generic.Must(decimal.NewFromString(tx.Outputs[0].Amount.String()))

	transfer := &core.Transfer{
		CreatedAt: req.CreatedAt,
		TraceID:   req.RequestID,
		Status:    core.TransferStatusPending,
		AssetID:   req.Asset.String(),
		Amount:    amount,
		Memo:      string(generic.Must(hex.DecodeString(req.Extra))),
		Opponent:  opponent,
	}

	switch req.State {
	case mixin.SafeUtxoStateSigned:
		transfer.Status = core.TransferStatusAssigned
	case mixin.SafeUtxoStateSpent:
		transfer.Status = core.TransferStatusHandled
	}

	return transfer, nil
}

func (s *service) Spend(ctx context.Context, transfer *core.Transfer, outputs []*core.Output) error {
	asset, err := s.assetz.Find(ctx, transfer.AssetID)
	if err != nil {
		return err
	}

	var (
		utxos []*mixin.SafeUtxo
		sum   decimal.Decimal
	)

	for _, output := range outputs {
		utxos = append(utxos, &mixin.SafeUtxo{
			TransactionHash:    output.Hash,
			OutputIndex:        output.Index,
			Asset:              asset.Hash,
			Amount:             output.Amount,
			Receivers:          []string{s.client.ClientID},
			ReceiversThreshold: 1,
		})

		sum = sum.Add(output.Amount)
	}

	b := mixin.NewSafeTransactionBuilder(utxos)
	b.Hint = transfer.TraceID
	b.Memo = transfer.Memo

	var receivers []*mixin.TransactionOutput
	receivers = append(receivers, &mixin.TransactionOutput{
		Address: transfer.Opponent,
		Amount:  transfer.Amount,
	})

	remain := sum.Sub(transfer.Amount)
	n := min(int(remain.Div(transfer.Amount).Ceil().IntPart()), 3) // 0 - 3
	for _, amount := range splitChange(remain, n) {
		receivers = append(receivers, &mixin.TransactionOutput{
			Address: mixin.RequireNewMixAddress([]string{s.client.ClientID}, 1),
			Amount:  amount,
		})
	}

	tx, err := s.client.MakeTransaction(ctx, b, receivers)
	if err != nil {
		return err
	}

	// prepare transaction
	prepareReq, err := s.client.SafeCreateTransactionRequest(ctx, &mixin.SafeTransactionRequestInput{
		RequestID:      transfer.TraceID,
		RawTransaction: hex.EncodeToString(generic.Must(tx.DumpData())),
	})

	if err != nil {
		return err
	}

	if prepareReq.State == mixin.SafeUtxoStateSpent {
		return nil
	}

	// sign transaction
	if err := mixin.SafeSignTransaction(tx, s.spendKey, prepareReq.Views, 0); err != nil {
		return err
	}

	// submit transaction
	_, err = s.client.SafeSubmitTransactionRequest(ctx, &mixin.SafeTransactionRequestInput{
		RequestID:      transfer.TraceID,
		RawTransaction: hex.EncodeToString(generic.Must(tx.DumpData())),
	})

	return err
}

func splitChange(amount decimal.Decimal, n int) []decimal.Decimal {
	var (
		changes []decimal.Decimal
		remain  = amount
	)

	for i := 0; i < n-1; i++ {
		change := remain.Div(decimal.NewFromInt(2)).Truncate(8)
		if change.IsZero() {
			break
		}

		changes = append(changes, change)
		remain = remain.Sub(change)
	}

	if remain.IsPositive() {
		changes = append(changes, remain)
	}

	return changes
}
