package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/carlmjohnson/versioninfo"
	"github.com/pandodao/safe-wallet/cmd/worker/cmds"
	"github.com/pandodao/safe-wallet/worker/cashier"
	"github.com/pandodao/safe-wallet/worker/cleaner"
	"github.com/pandodao/safe-wallet/worker/syncer"
	"github.com/spf13/viper"
	"golang.org/x/sync/errgroup"
)

var (
	opt struct {
		config  string
		debug   bool
		version bool
	}

	version = "0.0.1-src"
	commit  = versioninfo.Short()
)

func main() {
	flag.StringVar(&opt.config, "config", "config.yaml", "config file path")
	flag.BoolVar(&opt.debug, "debug", false, "debug mode")
	flag.BoolVar(&opt.version, "version", false, "show version")
	flag.Parse()

	flag.Args()

	if opt.version {
		fmt.Printf("safe-wallet worker\n\tversion: %s\n\tcommit: %s\n", version, commit)
		return
	}

	ctx, cancel := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer cancel()

	v := initViper()
	logger := initLogger()

	app, cleanup, err := setupApp(v, logger)
	if err != nil {
		logger.Error("setup failed", "err", err)
		return
	}

	defer cleanup()

	if args := flag.Args(); len(args) > 0 {
		logger.Info("running command", "args", args)
		if err := app.cmds.Run(ctx, args); err != nil {
			logger.Error("command failed", "err", err)
		}
		return
	}

	logger.Info("safe wallet worker launched", "version", version, "commit", commit)

	var g errgroup.Group

	g.Go(func() error {
		return app.syncer.Run(ctx)
	})

	g.Go(func() error {
		return app.cashier.Run(ctx)
	})

	g.Go(func() error {
		return app.cleaner.Run(ctx)
	})

	if err := g.Wait(); err != nil {
		logger.Error("worker exit", "err", err)
	}
}

type app struct {
	cmds    *cmds.Cmd
	syncer  *syncer.Syncer
	cashier *cashier.Cashier
	cleaner *cleaner.Cleaner
	logger  *slog.Logger
}

func initLogger() *slog.Logger {
	level := slog.LevelInfo
	if opt.debug {
		level = slog.LevelDebug
	}

	handler := slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: level})
	return slog.New(handler)
}

func initViper() *viper.Viper {
	v := viper.New()
	v.SetConfigFile(opt.config)
	v.SetConfigType("yaml")
	v.AutomaticEnv()
	if err := v.ReadInConfig(); err != nil {
		log.Panicln(err)
	}

	return v
}
