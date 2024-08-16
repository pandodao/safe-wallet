package loader

import (
	"context"

	"github.com/fox-one/mixin-sdk-go/v2"
	"github.com/fox-one/mixin-sdk-go/v2/mixinnet"
	"github.com/pandodao/safe-wallet/core"
	"github.com/pandodao/safe-wallet/service/output"
	"github.com/pandodao/safe-wallet/service/transfer"
)

type loader struct {
	wallets  core.WalletStore
	client   *mixin.Client
	spendKey mixinnet.Key
}

func New(wallets core.WalletStore, client *mixin.Client, spendKey mixinnet.Key) core.ServiceLoader {
	return &loader{client: client, spendKey: spendKey, wallets: wallets}
}

func (s *loader) loadWallet(ctx context.Context, userID string) (*mixin.Client, mixinnet.Key, error) {
	if userID == s.client.ClientID {
		return s.client, s.spendKey, nil
	}

	wallet, err := s.wallets.Find(ctx, userID)
	if err != nil {
		return nil, mixinnet.Key{}, err
	}

	client, err := mixin.NewFromKeystore(&mixin.Keystore{
		ClientID:   wallet.UserID,
		SessionID:  wallet.SessionID,
		PrivateKey: wallet.PrivateKey,
	})

	if err != nil {
		return nil, mixinnet.Key{}, err
	}

	spendKey, err := mixinnet.KeyFromString(wallet.SpendKey)
	if err != nil {
		return nil, mixinnet.Key{}, err
	}

	return client, spendKey, nil
}

func (s *loader) LoadOutput(ctx context.Context, userID string) (core.OutputService, error) {
	client, _, err := s.loadWallet(ctx, userID)
	if err != nil {
		return nil, err
	}

	return output.New(client), nil
}

func (s *loader) LoadTransfer(ctx context.Context, userID string) (core.TransferService, error) {
	client, spendKey, err := s.loadWallet(ctx, userID)
	if err != nil {
		return nil, err
	}

	return transfer.New(client, spendKey), nil
}
