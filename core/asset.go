package core

import (
	"context"

	"github.com/fox-one/mixin-sdk-go/v2/mixinnet"
)

type Asset struct {
	ID      string        `json:"id,omitempty"`
	Hash    mixinnet.Hash `json:"hash,omitempty"`
	ChainID string        `json:"chain_id,omitempty"`
	Symbol  string        `json:"symbol,omitempty"`
	Name    string        `json:"name,omitempty"`
	Logo    string        `json:"logo,omitempty"`
}

type AssetService interface {
	Find(ctx context.Context, id string) (*Asset, error)
	FindHash(ctx context.Context, hash mixinnet.Hash) (*Asset, error)
}
