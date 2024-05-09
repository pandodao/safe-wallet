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

func (s *service) submit(ctx context.Context, signedBy string) error {
	req, err := s.client.SafeReadMultisigRequests(ctx, signedBy)
	if err != nil {
		return err
	}

	req, err = s.client.SafeCreateMultisigRequest(ctx, &mixin.SafeTransactionRequestInput{
		RequestID:      req.RequestID,
		RawTransaction: req.RawTransaction,
	})

	if err != nil {
		return err
	}

	tx, err := mixinnet.TransactionFromRaw(req.RawTransaction)
	if err != nil {
		return err
	}

	if err := mixin.SafeSignTransaction(tx, s.spendKey, req.Views, 0); err != nil {
		return err
	}

	raw, err := tx.DumpData()
	if err != nil {
		return err
	}

	if _, err := s.client.SafeSubmitTransactionRequest(ctx, &mixin.SafeTransactionRequestInput{
		RequestID:      req.RequestID,
		RawTransaction: hex.EncodeToString(raw),
	}); err != nil {
		return err
	}

	return nil
}

func (s *service) ReadState(ctx context.Context, output *core.Output) (string, error) {
	utxo, err := s.client.SafeReadUtxoByHash(ctx, output.Hash, output.Index)
	if err != nil {
		return "", err
	}

	if utxo.State == mixin.SafeUtxoStateSigned {
		if err := s.submit(ctx, utxo.SignedBy); err != nil {
			return "", err
		}
	}

	return string(utxo.State), nil
}
