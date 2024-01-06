package output

import (
	"context"
	"encoding/hex"

	"github.com/fox-one/mixin-sdk-go/v2"
	"github.com/fox-one/mixin-sdk-go/v2/mixinnet"
	"github.com/pandodao/safe-wallet/core"
)

func New(client *mixin.Client, key mixinnet.Key) core.OutputService {
	return &service{
		client:   client,
		spendKey: key,
	}
}

type service struct {
	client   *mixin.Client
	spendKey mixinnet.Key
}

func (s *service) Pull(ctx context.Context, offset uint64, limit int) ([]*core.Output, error) {
	utxos, err := s.client.SafeListUtxos(ctx, mixin.SafeListUtxoOption{
		Members:   []string{s.client.ClientID},
		Threshold: 1,
		Offset:    offset,
		Limit:     limit,
		Order:     "ASC",
		State:     mixin.SafeUtxoStateUnspent,
	})
	if err != nil {
		return nil, err
	}

	outputs := make([]*core.Output, len(utxos))
	for i, utxo := range utxos {
		outputs[i] = &core.Output{
			Sequence:  utxo.Sequence,
			CreatedAt: utxo.CreatedAt,
			Hash:      utxo.TransactionHash,
			Index:     utxo.OutputIndex,
			AssetID:   utxo.AssetID,
			Amount:    utxo.Amount,
		}
	}

	return outputs, nil
}

func (s *service) ListRange(ctx context.Context, assetID string, from, to uint64) ([]*core.Output, error) {
	asset, err := s.client.SafeReadAsset(ctx, assetID)
	if err != nil {
		return nil, err
	}

	utxos, err := s.client.SafeListUtxos(ctx, mixin.SafeListUtxoOption{
		Members:   []string{s.client.ClientID},
		Threshold: 1,
		Offset:    from,
		Limit:     500,
		Order:     "ASC",
		Asset:     asset.KernelAssetID,
	})
	if err != nil {
		return nil, err
	}

	var outputs []*core.Output
	for _, utxo := range utxos {
		if utxo.AssetID != assetID {
			continue
		}

		if utxo.Sequence > to {
			break
		}

		outputs = append(outputs, &core.Output{
			Sequence:  utxo.Sequence,
			CreatedAt: utxo.CreatedAt,
			Hash:      utxo.TransactionHash,
			Index:     utxo.OutputIndex,
			AssetID:   utxo.AssetID,
			Amount:    utxo.Amount,
		})
	}

	return outputs, nil
}

func (s *service) FlushSigned(ctx context.Context) (int, error) {
	utxos, err := s.client.SafeListUtxos(ctx, mixin.SafeListUtxoOption{
		Members:   []string{s.client.ClientID},
		Threshold: 1,
		Limit:     1,
		Order:     "ASC",
		State:     mixin.SafeUtxoStateSigned,
	})
	if err != nil || len(utxos) == 0 {
		return 0, err
	}

	utxo := utxos[0]
	req, err := s.client.SafeReadMultisigRequests(ctx, utxo.SignedBy)
	if err != nil {
		return 0, err
	}

	req, err = s.client.SafeCreateMultisigRequest(ctx, &mixin.SafeTransactionRequestInput{
		RequestID:      req.RequestID,
		RawTransaction: req.RawTransaction,
	})

	if err != nil {
		return 0, err
	}

	tx, err := mixinnet.TransactionFromRaw(req.RawTransaction)
	if err != nil {
		return 0, err
	}

	if err := mixin.SafeSignTransaction(tx, s.spendKey, req.Views, 0); err != nil {
		return 0, err
	}

	raw, err := tx.DumpData()
	if err != nil {
		return 0, err
	}

	if _, err := s.client.SafeSubmitTransactionRequest(ctx, &mixin.SafeTransactionRequestInput{
		RequestID:      req.RequestID,
		RawTransaction: hex.EncodeToString(raw),
	}); err != nil {
		return 0, err
	}

	return 1, nil
}
