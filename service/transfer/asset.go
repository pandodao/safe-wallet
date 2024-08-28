package transfer

import (
	"context"

	"github.com/fox-one/mixin-sdk-go/v2"
	lru "github.com/hashicorp/golang-lru/v2"
)

var assets, _ = lru.New[string, *mixin.SafeAsset](256)

func (s *service) getAsset(ctx context.Context, assetID string) (*mixin.SafeAsset, error) {
	if v, ok := assets.Get(assetID); ok {
		return v, nil
	}

	asset, err := s.client.SafeReadAsset(ctx, assetID)
	if err != nil {
		return nil, err
	}

	assets.Add(assetID, asset)
	return asset, nil
}
