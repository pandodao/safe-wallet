package main

import (
	"github.com/google/wire"
	"github.com/pandodao/safe-wallet/store/db"
	"github.com/pandodao/safe-wallet/store/output"
	"github.com/pandodao/safe-wallet/store/transfer"
	"github.com/pandodao/safe-wallet/store/wallet"
	"github.com/spf13/viper"
	"github.com/tsenart/nap"
)

var storeSet = wire.NewSet(
	provideDB,
	output.New,
	transfer.New,
	wallet.New,
)

func provideDB(v *viper.Viper) (*nap.DB, func(), error) {
	v.SetDefault("db.driver", "mysql")

	driver := v.GetString("db.driver")
	dsn := v.GetString("db.dsn")

	for _, replica := range v.GetStringSlice("db.replicas") {
		dsn += ";" + replica
	}

	conn, err := nap.Open(driver, dsn)
	if err != nil {
		return nil, nil, err
	}

	if err := db.Migrate(conn.Master()); err != nil {
		return nil, nil, err
	}

	return conn, func() { _ = conn.Close() }, nil
}
