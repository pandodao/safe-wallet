package main

import (
	"crypto/sha256"
	"encoding/hex"
	"io"

	"github.com/fox-one/mixin-sdk-go/v2"
	"github.com/fox-one/mixin-sdk-go/v2/mixinnet"
	"github.com/google/wire"
	"github.com/pandodao/safe-wallet/store/db"
	"github.com/pandodao/safe-wallet/store/output"
	"github.com/pandodao/safe-wallet/store/property"
	"github.com/pandodao/safe-wallet/store/transfer"
	"github.com/pandodao/safe-wallet/store/wallet"
	"github.com/spf13/viper"
	"github.com/tsenart/nap"
)

var storeSet = wire.NewSet(
	provideDB,
	output.New,
	transfer.New,
	property.New,
	provideEncryptKey,
	wallet.New,
)

func provideEncryptKey(keystore *mixin.Keystore) ([]byte, error) {
	h := sha256.New()
	io.WriteString(h, keystore.ClientID)
	io.WriteString(h, keystore.SessionID)

	if keystore.PrivateKey != "" {
		io.WriteString(h, keystore.PrivateKey)
	} else if keystore.SessionPrivateKey != "" {
		io.WriteString(h, keystore.SessionPrivateKey)
	}

	seed := hex.EncodeToString(h.Sum(nil))
	key, err := mixinnet.KeyFromSeed(seed)
	if err != nil {
		return nil, err
	}

	return key[:], nil
}

func provideDB(v *viper.Viper) (*nap.DB, func(), error) {
	v.SetDefault("db.driver", "mysql")

	driver := v.GetString("db.driver")
	dsn := v.GetString("db.dsn")
	conn, err := nap.Open(driver, dsn)
	if err != nil {
		return nil, nil, err
	}

	if err := db.Migrate(conn.Master()); err != nil {
		return nil, nil, err
	}

	return conn, func() { _ = conn.Close() }, nil
}
