//go:build wireinject
// +build wireinject

package main

import (
	"log/slog"

	"github.com/google/wire"
	"github.com/spf13/viper"
)

func setupApp(v *viper.Viper, logger *slog.Logger) (app, func(), error) {
	panic(wire.Build(
		storeSet,
		serviceSet,
		workerSet,
		wire.Struct(new(app), "*"),
	))
}
