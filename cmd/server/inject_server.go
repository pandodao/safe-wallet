package main

import (
	"fmt"
	"net/http"

	"github.com/fox-one/mixin-sdk-go/v2"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/google/wire"
	"github.com/pandodao/safe-wallet/handler/api"
	"github.com/pandodao/safe-wallet/handler/hc"
	"github.com/pandodao/safe-wallet/handler/rpc"
	"github.com/rs/cors"
	"github.com/spf13/viper"
)

var serverSet = wire.NewSet(
	provideRpcConfig,
	rpc.New,
	api.New,
	provideServer,
)

func provideRpcConfig(ks *mixin.Keystore) rpc.Config {
	viper.SetDefault("rpc.prefix", "/twirp")

	return rpc.Config{
		ClientID:      ks.ClientID,
		Prefix:        viper.GetString("rpc.prefix"),
		BlockedAssets: viper.GetStringSlice("rpc.blocked_assets"),
	}
}

func provideServer(apiHandler *api.Server, rpcHandler *rpc.Server) *http.Server {
	m := chi.NewMux()
	m.Use(middleware.RealIP)
	m.Use(middleware.Logger)
	m.Use(middleware.Recoverer)
	m.Use(cors.AllowAll().Handler)

	m.Mount("/api", apiHandler.Handler())
	m.Mount(rpcHandler.Handler())
	m.Mount("/hc", hc.Handler(version))

	return &http.Server{
		Addr:    fmt.Sprintf(":%d", opt.port),
		Handler: m,
	}
}
