package wallet

import (
	"context"

	sq "github.com/Masterminds/squirrel"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/pandodao/safe-wallet/core"
	"github.com/tsenart/nap"
)

func New(db *nap.DB) core.WalletStore {
	wallets, err := lru.New[string, *core.Wallet](256)
	if err != nil {
		panic(err)
	}

	return &walletStore{
		db:      db,
		wallets: wallets,
	}

}

type walletStore struct {
	db      *nap.DB
	wallets *lru.Cache[string, *core.Wallet]
}

var columns = []string{"user_id", "label", "session_id", "pin_token", "pin", "private_key", "spend_key"}

func (s *walletStore) Create(ctx context.Context, wallet *core.Wallet) error {
	b := sq.Insert("wallets").
		Columns(columns...).
		Values(wallet.UserID, wallet.Label, wallet.SessionID, wallet.PinToken, wallet.Pin, wallet.PrivateKey, wallet.SpendKey)

	_, err := b.RunWith(s.db).ExecContext(ctx)
	return err
}

func (s *walletStore) Find(ctx context.Context, userID string) (*core.Wallet, error) {
	if w, ok := s.wallets.Get(userID); ok {
		return w, nil
	}

	w, err := s.find(ctx, userID)
	if err != nil {
		return nil, err
	}

	s.wallets.Add(userID, w)
	return w, nil
}

func (s *walletStore) find(ctx context.Context, userID string) (*core.Wallet, error) {
	b := sq.Select(columns...).From("wallets").Where(sq.Eq{"user_id": userID})
	row := b.RunWith(s.db).QueryRowContext(ctx)
	var wallet core.Wallet
	if err := row.Scan(&wallet.UserID, &wallet.Label, &wallet.SessionID, &wallet.PinToken, &wallet.Pin, &wallet.PrivateKey, &wallet.SpendKey); err != nil {
		return nil, err
	}

	return &wallet, nil
}
