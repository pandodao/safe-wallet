package main

import (
	"context"

	"github.com/fox-one/mixin-sdk-go/v2"
	"github.com/fox-one/mixin-sdk-go/v2/mixinnet"
	"github.com/google/wire"
	"github.com/pandodao/safe-wallet/service/loader"
	"github.com/pandodao/safe-wallet/service/output"
	"github.com/spf13/viper"
)

var serviceSet = wire.NewSet(
	provideKeystore,
	provideMixinClient,
	provideSpendKey,
	output.New,
	loader.New,
)

func provideKeystore(v *viper.Viper) *mixin.Keystore {
	return &mixin.Keystore{
		ClientID:          v.GetString("dapp.client_id"),
		SessionID:         v.GetString("dapp.session_id"),
		PrivateKey:        v.GetString("dapp.private_key"),
		SessionPrivateKey: v.GetString("dapp.session_private_key"),
	}
}

func provideMixinClient(ks *mixin.Keystore) (*mixin.Client, error) {
	return mixin.NewFromKeystore(ks)
}

func provideSpendKey(v *viper.Viper, client *mixin.Client) (mixinnet.Key, error) {
	user, err := client.UserMe(context.Background())
	if err != nil {
		return mixinnet.Key{}, err
	}

	s := v.GetString("dapp.spend_key")
	return mixinnet.ParseKeyWithPub(s, user.SpendPublicKey)
}
