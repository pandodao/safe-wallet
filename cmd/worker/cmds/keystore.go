package cmds

import (
	"github.com/pandodao/safe-wallet/core"
)

type Keystore struct {
	ClientID   string `json:"client_id"`
	SessionID  string `json:"session_id"`
	PrivateKey string `json:"private_key"`
	PinToken   string `json:"pin_token"`
	Pin        string `json:"pin"`
	SpendKey   string `json:"spend_key"`
	Label      string `json:"label"`
}

func keystoreFromWallet(wallet *core.Wallet) *Keystore {
	return &Keystore{
		ClientID:   wallet.UserID,
		SessionID:  wallet.SessionID,
		PrivateKey: wallet.PrivateKey,
		PinToken:   wallet.PinToken,
		Pin:        wallet.Pin,
		SpendKey:   wallet.SpendKey,
		Label:      wallet.Label,
	}
}
