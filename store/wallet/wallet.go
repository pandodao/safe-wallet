package wallet

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"

	sq "github.com/Masterminds/squirrel"
	lru "github.com/hashicorp/golang-lru/v2"
	"github.com/pandodao/safe-wallet/core"
	"github.com/tsenart/nap"
)

func New(db *nap.DB, encryptionKey []byte) core.WalletStore {
	cache, err := lru.New[string, *core.Wallet](256)
	if err != nil {
		panic(err)
	}

	if len(encryptionKey) != 32 {
		panic("encryption key must be 32 bytes long")
	}

	if bytes.Equal(encryptionKey, make([]byte, 32)) {
		panic("encryption key must not be all zeros")
	}

	return &walletStore{
		db:    db,
		cache: cache,
		key:   encryptionKey,
	}
}

type walletStore struct {
	db    *nap.DB
	cache *lru.Cache[string, *core.Wallet]
	key   []byte // AES encryption key
}

var columns = []string{"user_id", "label", "session_id", "pin_token", "pin", "private_key", "spend_key"}

func (s *walletStore) Create(ctx context.Context, wallet *core.Wallet) error {
	encryptedPin, err := encrypt(s.key, wallet.Pin)
	if err != nil {
		return fmt.Errorf("failed to encrypt PIN: %w", err)
	}

	encryptedSpendKey, err := encrypt(s.key, wallet.SpendKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt SpendKey: %w", err)
	}

	b := sq.Insert("wallets").
		Columns(columns...).
		Values(wallet.UserID, wallet.Label, wallet.SessionID, wallet.PinToken, encryptedPin, wallet.PrivateKey, encryptedSpendKey)

	_, err = b.RunWith(s.db).ExecContext(ctx)
	return err
}

func (s *walletStore) Find(ctx context.Context, userID string) (*core.Wallet, error) {
	if w, ok := s.cache.Get(userID); ok {
		return w, nil
	}

	w, err := s.find(ctx, userID)
	if err != nil {
		return nil, err
	}

	s.cache.Add(userID, w)
	return w, nil
}

func (s *walletStore) find(ctx context.Context, userID string) (*core.Wallet, error) {
	b := sq.Select(columns...).From("wallets").Where(sq.Eq{"user_id": userID})
	row := b.RunWith(s.db).QueryRowContext(ctx)
	var wallet core.Wallet
	var encryptedPin, encryptedSpendKey string
	err := row.Scan(&wallet.UserID, &wallet.Label, &wallet.SessionID, &wallet.PinToken, &encryptedPin, &wallet.PrivateKey, &encryptedSpendKey)
	if err != nil {
		return nil, err
	}

	wallet.Pin, err = decrypt(s.key, encryptedPin)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt PIN: %w", err)
	}

	wallet.SpendKey, err = decrypt(s.key, encryptedSpendKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt SpendKey: %w", err)
	}

	return &wallet, nil
}

// Encryption and decryption helper functions
func encrypt(key []byte, plaintext string) (string, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = rand.Read(nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func decrypt(key []byte, ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", err
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return "", fmt.Errorf("aes.NewCipher failed: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("cipher.NewGCM failed: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertextBytes := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, nil)
	if err != nil {
		return "", fmt.Errorf("gcm.Open failed: %w", err)
	}

	return string(plaintext), nil
}
