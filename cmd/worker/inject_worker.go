package main

import (
	"github.com/fox-one/mixin-sdk-go/v2"
	"github.com/google/wire"
	"github.com/pandodao/safe-wallet/worker/cashier"
	"github.com/pandodao/safe-wallet/worker/cleaner"
	"github.com/pandodao/safe-wallet/worker/syncer"
	"github.com/spf13/viper"
)

var workerSet = wire.NewSet(
	cashier.New,
	syncer.New,
	provideCleanerConfig,
	cleaner.New,
)

func provideCleanerConfig(v *viper.Viper, ks *mixin.Keystore) cleaner.Config {
	v.SetDefault("cleaner.capacity", 512)

	return cleaner.Config{
		Capacity: v.GetInt("cleaner.capacity"),
	}
}
