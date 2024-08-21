package wallet

import (
	"crypto/rand"
	"testing"

	"github.com/fox-one/mixin-sdk-go/v2/mixinnet"
)

func TestEncryptDecrypt(t *testing.T) {
	// Generate a random key
	key := mixinnet.GenerateKey(rand.Reader) // 256-bit key

	testCases := []struct {
		name      string
		plaintext string
	}{
		{"Empty string", ""},
		{"Short text", "Hello, World!"},
		{"Long text", "This is a longer text that we'll use to test encryption and decryption of larger data sets."},
		{"Special characters", "!@#$%^&*()_+{}[]|\\:;\"'<>,.?/~`"},
		{"mixinnet.Key", string(mixinnet.GenerateKey(rand.Reader).String())},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Encrypt the plaintext
			ciphertext, err := encrypt(key[:], tc.plaintext)
			if err != nil {
				t.Fatalf("Encryption failed: %v", err)
			}

			// Decrypt the ciphertext
			decrypted, err := decrypt(key[:], ciphertext)
			if err != nil {
				t.Fatalf("Decryption failed: %v", err)
			}

			// Compare the decrypted text with the original plaintext
			if decrypted != tc.plaintext {
				t.Errorf("Decrypted text does not match original plaintext. Got %q, want %q", decrypted, tc.plaintext)
			}
		})
	}
}

func TestDecryptInvalidInput(t *testing.T) {
	key := make([]byte, 32)
	_, err := rand.Read(key)
	if err != nil {
		t.Fatalf("Failed to generate random key: %v", err)
	}

	testCases := []struct {
		name       string
		ciphertext string
	}{
		{"Empty string", ""},
		{"Invalid base64", "This is not base64!"},
		{"Too short after base64 decode", "aGVsbG8="}, // "hello" in base64
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := decrypt(key, tc.ciphertext)
			if err == nil {
				t.Error("Expected an error, but got nil")
			}
		})
	}
}
