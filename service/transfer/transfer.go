package transfer

import (
	"context"
	"encoding/hex"
	"fmt"
	"sync"

	"github.com/fox-one/mixin-sdk-go/v2"
	"github.com/fox-one/mixin-sdk-go/v2/mixinnet"
	"github.com/pandodao/generic"
	"github.com/pandodao/safe-wallet/core"
	"github.com/shopspring/decimal"
	"github.com/zyedidia/generic/cache"
)

func New(client *mixin.Client, key mixinnet.Key) core.TransferService {
	return &service{
		client:   client,
		spendKey: key,
		hashes:   cache.New[string, mixinnet.Hash](256),
	}
}

type service struct {
	client   *mixin.Client
	spendKey mixinnet.Key

	hashes *cache.Cache[string, mixinnet.Hash]
	mux    sync.RWMutex
}

func (s *service) getAssetHash(ctx context.Context, assetID string) (mixinnet.Hash, error) {
	s.mux.RLock()
	v, ok := s.hashes.Get(assetID)
	s.mux.RUnlock()

	if ok {
		return v, nil
	}

	asset, err := s.client.SafeReadAsset(ctx, assetID)
	if err != nil {
		return v, err
	}

	v, err = mixinnet.HashFromString(asset.KernelAssetID)
	if err != nil {
		return v, err
	}

	s.mux.Lock()
	s.hashes.Put(assetID, v)
	s.mux.Unlock()

	return v, nil
}

func (s *service) Spend(ctx context.Context, transfer *core.Transfer, outputs []*core.Output) error {
	if s.client.ClientID != transfer.UserID {
		panic("transfer user id not match")
	}

	assetHash, err := s.getAssetHash(ctx, transfer.AssetID)
	if err != nil {
		return err
	}

	var (
		utxos []*mixin.SafeUtxo
		sum   decimal.Decimal
	)

	for _, output := range outputs {
		if output.UserID != transfer.UserID {
			panic("output user id not match")
		}

		utxos = append(utxos, &mixin.SafeUtxo{
			TransactionHash:    output.Hash,
			OutputIndex:        output.Index,
			KernelAssetID:      assetHash,
			Amount:             output.Amount,
			Receivers:          []string{output.UserID},
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
		return fmt.Errorf("make transaction failed: %w", err)
	}

	// prepare transaction
	req, err := s.client.SafeCreateTransactionRequest(ctx, &mixin.SafeTransactionRequestInput{
		RequestID:      transfer.TraceID,
		RawTransaction: generic.Must(tx.Dump()),
	})

	if err != nil {
		return fmt.Errorf("create transaction request failed: %w", err)
	}

	// sign transaction
	if err := mixin.SafeSignTransaction(tx, s.spendKey, req.Views, 0); err != nil {
		return fmt.Errorf("sign transaction failed: %w", err)
	}

	// submit transaction
	if _, err := s.client.SafeSubmitTransactionRequest(ctx, &mixin.SafeTransactionRequestInput{
		RequestID:      transfer.TraceID,
		RawTransaction: hex.EncodeToString(generic.Must(tx.DumpData())),
	}); err != nil {
		return fmt.Errorf("submit transaction failed: %w", err)
	}

	return nil
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
