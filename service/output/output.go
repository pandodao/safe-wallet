package output

import (
	"context"

	"github.com/fox-one/mixin-sdk-go/v2"
	"github.com/pandodao/safe-wallet/core"
)

func New(client *mixin.Client) core.OutputService {
	return &service{
		client: client,
	}
}

type service struct {
	client *mixin.Client
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
