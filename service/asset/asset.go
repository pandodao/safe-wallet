package asset

import (
	"context"
	"sync"

	"github.com/fox-one/mixin-sdk-go/v2"
	"github.com/fox-one/mixin-sdk-go/v2/mixinnet"
	"github.com/pandodao/generic"
	"github.com/pandodao/safe-wallet/core"
	"github.com/zyedidia/generic/cache"
)

func New(client *mixin.Client) core.AssetService {
	return &service{
		client: client,
		cache:  cache.New[string, *core.Asset](1024),
	}
}

type service struct {
	client *mixin.Client

	cache *cache.Cache[string, *core.Asset]
	mux   sync.Mutex
}

func (s *service) Find(ctx context.Context, id string) (*core.Asset, error) {
	s.mux.Lock()
	v, ok := s.cache.Get(id)
	s.mux.Unlock()
	if ok {
		return v, nil
	}

	asset, err := s.client.SafeReadAsset(ctx, id)
	if err != nil {
		return nil, err
	}

	v = &core.Asset{
		ID:      asset.AssetID,
		Hash:    generic.Must(mixinnet.HashFromString(asset.KernelAssetID)),
		ChainID: asset.ChainID,
		Symbol:  asset.Symbol,
		Name:    asset.Name,
		Logo:    asset.IconURL,
	}

	s.mux.Lock()
	s.cache.Put(v.ID, v)
	s.cache.Put(v.Hash.String(), v)
	s.mux.Unlock()

	return v, nil
}

func (s *service) FindHash(ctx context.Context, hash mixinnet.Hash) (*core.Asset, error) {
	return s.Find(ctx, hash.String())
}
