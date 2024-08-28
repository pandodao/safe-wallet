package wallet

import (
	"context"
	"crypto/rand"

	"github.com/fox-one/mixin-sdk-go/v2"
	"github.com/fox-one/mixin-sdk-go/v2/mixinnet"
	"github.com/pandodao/safe-wallet/core"
)

type service struct {
	client *mixin.Client
}

func New(client *mixin.Client) core.WalletService {
	return &service{client: client}
}

func (s *service) Create(ctx context.Context, label string) (*core.Wallet, error) {
	// create user
	_, keystore, err := s.client.CreateUser(ctx, mixin.GenerateEd25519Key(), label)
	if err != nil {
		return nil, err
	}

	client, err := mixin.NewFromKeystore(keystore)
	if err != nil {
		return nil, err
	}

	// set tip pin
	pin := mixinnet.GenerateKey(rand.Reader)
	if err := client.ModifyPin(ctx, "", pin.Public().String()); err != nil {
		return nil, err
	}

	// set spend key
	spendKey := mixinnet.GenerateKey(rand.Reader)
	if _, err := client.SafeMigrate(ctx, spendKey.String(), pin.String()); err != nil {
		return nil, err
	}

	return &core.Wallet{
		UserID:     keystore.ClientID,
		Label:      label,
		SessionID:  keystore.SessionID,
		PrivateKey: keystore.PrivateKey,
		PinToken:   keystore.PinToken,
		Pin:        pin.String(),
		SpendKey:   spendKey.String(),
	}, nil
}
