package main

import (
	"github.com/google/wire"
	"github.com/pandodao/safe-wallet/worker/cashier"
	"github.com/pandodao/safe-wallet/worker/syncer"
)

var workerSet = wire.NewSet(
	cashier.New,
	syncer.New,
)
