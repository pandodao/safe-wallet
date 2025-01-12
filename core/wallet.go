package core

import "context"

type Wallet struct {
	UserID     string `json:"user_id"`
	Label      string `json:"label"`
	SessionID  string `json:"session_id"`
	PrivateKey string `json:"private_key"`
	PinToken   string `json:"pin_token"`
	Pin        string `json:"pin"`
	SpendKey   string `json:"spend_key"`
}

type WalletStore interface {
	Create(ctx context.Context, wallet *Wallet) error
	Find(ctx context.Context, userID string) (*Wallet, error)
	List(ctx context.Context) ([]*Wallet, error)
}

type WalletService interface {
	Create(ctx context.Context, label string) (*Wallet, error)
}

type ServiceLoader interface {
	LoadOutput(ctx context.Context, userID string) (OutputService, error)
	LoadTransfer(ctx context.Context, userID string) (TransferService, error)
}
