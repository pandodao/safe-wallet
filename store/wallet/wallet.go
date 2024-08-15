package wallet

import (
	"context"

	"github.com/pandodao/safe-wallet/core"
	"github.com/tsenart/nap"
)

type walletService struct {
	db *nap.DB
}

func (s *walletService) Create(ctx context.Context,wallet *core.Wallet) error {
	
}
