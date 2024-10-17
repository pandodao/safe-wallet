// Code generated by Wire. DO NOT EDIT.

//go:generate go run github.com/google/wire/cmd/wire
//go:build !wireinject
// +build !wireinject

package main

import (
	"github.com/pandodao/safe-wallet/cmd/worker/cmds"
	"github.com/pandodao/safe-wallet/service/loader"
	"github.com/pandodao/safe-wallet/service/output"
	output2 "github.com/pandodao/safe-wallet/store/output"
	"github.com/pandodao/safe-wallet/store/property"
	"github.com/pandodao/safe-wallet/store/transfer"
	"github.com/pandodao/safe-wallet/store/wallet"
	"github.com/pandodao/safe-wallet/worker/cashier"
	"github.com/pandodao/safe-wallet/worker/cleaner"
	"github.com/pandodao/safe-wallet/worker/syncer"
	"github.com/spf13/viper"
	"log/slog"
)

// Injectors from wire.go:

func setupApp(v *viper.Viper, logger *slog.Logger) (app, func(), error) {
	db, cleanup, err := provideDB(v)
	if err != nil {
		return app{}, nil, err
	}
	keystore := provideKeystore(v)
	v2, err := provideEncryptKey(keystore)
	if err != nil {
		cleanup()
		return app{}, nil, err
	}
	walletStore := wallet.New(db, v2)
	cmd := &cmds.Cmd{
		Wallets: walletStore,
	}
	client, err := provideMixinClient(keystore)
	if err != nil {
		cleanup()
		return app{}, nil, err
	}
	outputService := output.New(client)
	outputStore := output2.New(db)
	propertyStore := property.New(db)
	syncerSyncer := syncer.New(outputService, outputStore, propertyStore, logger)
	transferStore := transfer.New(db)
	key, err := provideSpendKey(v, client)
	if err != nil {
		cleanup()
		return app{}, nil, err
	}
	serviceLoader := loader.New(walletStore, client, key)
	cashierCashier := cashier.New(outputStore, transferStore, serviceLoader, logger)
	config := provideCleanerConfig(v, keystore)
	cleanerCleaner := cleaner.New(outputStore, transferStore, outputService, logger, config)
	mainApp := app{
		cmds:    cmd,
		syncer:  syncerSyncer,
		cashier: cashierCashier,
		cleaner: cleanerCleaner,
		logger:  logger,
	}
	return mainApp, func() {
		cleanup()
	}, nil
}
