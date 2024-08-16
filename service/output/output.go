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

func (s *service) Pull(ctx context.Context, offset uint64, limit int) ([]*core.Output, uint64, error) {
	utxos, err := s.client.SafeListUtxos(ctx, mixin.SafeListUtxoOption{
		Members:           []string{s.client.ClientID},
		Threshold:         1,
		Offset:            offset,
		Limit:             limit,
		Order:             "ASC",
		IncludeSubWallets: true,
	})
	if err != nil {
		return nil, 0, err
	}

	var outputs []*core.Output
	for _, utxo := range utxos {
		offset = utxo.Sequence + 1

		if utxo.State != mixin.SafeUtxoStateUnspent {
			continue
		}

		outputs = append(outputs, &core.Output{
			Sequence:  utxo.Sequence,
			CreatedAt: utxo.CreatedAt,
			Hash:      utxo.TransactionHash,
			Index:     utxo.OutputIndex,
			UserID:    utxo.Receivers[0],
			AssetID:   utxo.AssetID,
			Amount:    utxo.Amount,
		})
	}

	return outputs, offset, nil
}

func (s *service) ListRange(ctx context.Context, assetID string, from, to uint64) ([]*core.Output, error) {
	utxos, err := s.client.SafeListUtxos(ctx, mixin.SafeListUtxoOption{
		Members:   []string{s.client.ClientID},
		Threshold: 1,
		Offset:    from,
		Limit:     500,
		Order:     "ASC",
		Asset:     assetID,
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
