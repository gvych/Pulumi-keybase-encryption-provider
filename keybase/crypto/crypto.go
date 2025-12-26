package crypto

import (
	"context"
	"fmt"
	"io"

	"github.com/keybase/saltpack"
)

// Encryptor handles encryption operations using Saltpack
type Encryptor struct {
	// Version is the Saltpack version to use
	Version *saltpack.Version
	
	// SenderKey is the sender's secret key (optional for encryption)
	SenderKey saltpack.BoxSecretKey
}

// Decryptor handles decryption operations using Saltpack
type Decryptor struct {
	// Keyring provides access to secret keys for decryption
	Keyring saltpack.Keyring
}

// EncryptorConfig holds configuration for the Encryptor
type EncryptorConfig struct {
	// Version specifies the Saltpack version (defaults to nil which uses Version2)
	Version *saltpack.Version
	
	// SenderKey is the sender's secret key (nil for anonymous sender)
	SenderKey saltpack.BoxSecretKey
}

// DecryptorConfig holds configuration for the Decryptor
type DecryptorConfig struct {
	// Keyring provides access to secret keys for decryption
	Keyring saltpack.Keyring
}

// NewEncryptor creates a new Encryptor with the given configuration
func NewEncryptor(config *EncryptorConfig) (*Encryptor, error) {
	if config == nil {
		config = &EncryptorConfig{}
	}
	
	version := config.Version
	// Set default version if not provided
	if version == nil {
		v := saltpack.Version2()
		version = &v
	}
	
	return &Encryptor{
		Version:   version,
		SenderKey: config.SenderKey,
	}, nil
}

// NewDecryptor creates a new Decryptor with the given configuration
func NewDecryptor(config *DecryptorConfig) (*Decryptor, error) {
	if config == nil || config.Keyring == nil {
		return nil, fmt.Errorf("keyring is required for decryption")
	}
	
	return &Decryptor{
		Keyring: config.Keyring,
	}, nil
}

// Encrypt encrypts plaintext for multiple recipients using Saltpack
// 
// Parameters:
//   - plaintext: The data to encrypt
//   - receivers: Public keys of the recipients (supports 1 to N recipients)
//
// Returns:
//   - Encrypted ciphertext in binary format
//   - Error if encryption fails
func (e *Encryptor) Encrypt(plaintext []byte, receivers []saltpack.BoxPublicKey) ([]byte, error) {
	if len(receivers) == 0 {
		return nil, fmt.Errorf("at least one receiver is required")
	}
	
	if len(plaintext) == 0 {
		return nil, fmt.Errorf("plaintext cannot be empty")
	}
	
	// Use Saltpack Seal to encrypt for multiple recipients
	ciphertext, err := saltpack.Seal(*e.Version, plaintext, e.SenderKey, receivers)
	if err != nil {
		return nil, fmt.Errorf("encryption failed: %w", err)
	}
	
	return ciphertext, nil
}

// EncryptArmored encrypts plaintext and returns ASCII-armored output
// This is recommended for storing in text files like Pulumi state files
func (e *Encryptor) EncryptArmored(plaintext []byte, receivers []saltpack.BoxPublicKey) (string, error) {
	if len(receivers) == 0 {
		return "", fmt.Errorf("at least one receiver is required")
	}
	
	if len(plaintext) == 0 {
		return "", fmt.Errorf("plaintext cannot be empty")
	}
	
	// Use Base62 armoring which is Saltpack's native format
	ciphertext, err := saltpack.EncryptArmor62Seal(*e.Version, plaintext, e.SenderKey, receivers, "")
	if err != nil {
		return "", fmt.Errorf("armored encryption failed: %w", err)
	}
	
	return ciphertext, nil
}

// EncryptStream encrypts data from a reader to a writer using streaming
// This is efficient for large files (>10 MiB)
func (e *Encryptor) EncryptStream(plaintext io.Reader, ciphertext io.Writer, receivers []saltpack.BoxPublicKey) error {
	if len(receivers) == 0 {
		return fmt.Errorf("at least one receiver is required")
	}
	
	// Create streaming encryptor
	stream, err := saltpack.NewEncryptStream(*e.Version, ciphertext, e.SenderKey, receivers)
	if err != nil {
		return fmt.Errorf("failed to create encrypt stream: %w", err)
	}
	
	// Copy plaintext to the encryption stream
	if _, err := io.Copy(stream, plaintext); err != nil {
		return fmt.Errorf("encryption stream failed: %w", err)
	}
	
	// Close the stream to finalize encryption
	if err := stream.Close(); err != nil {
		return fmt.Errorf("failed to close encrypt stream: %w", err)
	}
	
	return nil
}

// EncryptStreamArmored encrypts data from a reader to a writer using streaming with ASCII armoring
func (e *Encryptor) EncryptStreamArmored(plaintext io.Reader, ciphertext io.Writer, receivers []saltpack.BoxPublicKey) error {
	if len(receivers) == 0 {
		return fmt.Errorf("at least one receiver is required")
	}
	
	// Create streaming encryptor with Base62 armoring
	stream, err := saltpack.NewEncryptArmor62Stream(*e.Version, ciphertext, e.SenderKey, receivers, "")
	if err != nil {
		return fmt.Errorf("failed to create armored encrypt stream: %w", err)
	}
	
	// Copy plaintext to the encryption stream
	if _, err := io.Copy(stream, plaintext); err != nil {
		return fmt.Errorf("encryption stream failed: %w", err)
	}
	
	// Close the stream to finalize encryption
	if err := stream.Close(); err != nil {
		return fmt.Errorf("failed to close encrypt stream: %w", err)
	}
	
	return nil
}

// Decrypt decrypts ciphertext using the configured keyring
// The keyring will automatically find the matching private key
//
// Returns:
//   - Plaintext
//   - MessageKeyInfo indicating which recipient key was used
//   - Error if decryption fails
func (d *Decryptor) Decrypt(ciphertext []byte) ([]byte, *saltpack.MessageKeyInfo, error) {
	if len(ciphertext) == 0 {
		return nil, nil, fmt.Errorf("ciphertext cannot be empty")
	}
	
	// Use Saltpack Open to decrypt
	// The keyring automatically finds the matching private key
	messageKeyInfo, plaintext, err := saltpack.Open(saltpack.CheckKnownMajorVersion, ciphertext, d.Keyring)
	if err != nil {
		return nil, nil, fmt.Errorf("decryption failed: %w", err)
	}
	
	return plaintext, messageKeyInfo, nil
}

// DecryptArmored decrypts ASCII-armored ciphertext
func (d *Decryptor) DecryptArmored(armoredCiphertext string) ([]byte, *saltpack.MessageKeyInfo, error) {
	if armoredCiphertext == "" {
		return nil, nil, fmt.Errorf("armored ciphertext cannot be empty")
	}
	
	// Dearmor and decrypt in one step
	messageKeyInfo, plaintext, _, err := saltpack.Dearmor62DecryptOpen(saltpack.CheckKnownMajorVersion, armoredCiphertext, d.Keyring)
	if err != nil {
		return nil, nil, fmt.Errorf("armored decryption failed: %w", err)
	}
	
	return plaintext, messageKeyInfo, nil
}

// DecryptStream decrypts data from a reader to a writer using streaming
func (d *Decryptor) DecryptStream(ciphertext io.Reader, plaintext io.Writer) (*saltpack.MessageKeyInfo, error) {
	// Create streaming decryptor
	messageKeyInfo, plaintextReader, err := saltpack.NewDecryptStream(saltpack.CheckKnownMajorVersion, ciphertext, d.Keyring)
	if err != nil {
		return nil, fmt.Errorf("failed to create decrypt stream: %w", err)
	}
	
	// Copy ciphertext through the decryption stream
	if _, err := io.Copy(plaintext, plaintextReader); err != nil {
		return nil, fmt.Errorf("decryption stream failed: %w", err)
	}
	
	return messageKeyInfo, nil
}

// DecryptStreamArmored decrypts ASCII-armored data from a reader to a writer using streaming
func (d *Decryptor) DecryptStreamArmored(armoredCiphertext io.Reader, plaintext io.Writer) (*saltpack.MessageKeyInfo, error) {
	// Create streaming decryptor for armored data
	messageKeyInfo, plaintextReader, _, err := saltpack.NewDearmor62DecryptStream(saltpack.CheckKnownMajorVersion, armoredCiphertext, d.Keyring)
	if err != nil {
		return nil, fmt.Errorf("failed to create armored decrypt stream: %w", err)
	}
	
	// Copy ciphertext through the decryption stream
	if _, err := io.Copy(plaintext, plaintextReader); err != nil {
		return nil, fmt.Errorf("decryption stream failed: %w", err)
	}
	
	return messageKeyInfo, nil
}

// EncryptWithContext encrypts plaintext with context support for cancellation
func (e *Encryptor) EncryptWithContext(ctx context.Context, plaintext []byte, receivers []saltpack.BoxPublicKey) ([]byte, error) {
	// Check if context is already cancelled
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context error before encryption: %w", err)
	}
	
	// Perform encryption (Saltpack doesn't have native context support,
	// but we check before and after for cancellation)
	result, err := e.Encrypt(plaintext, receivers)
	
	// Check if context was cancelled during encryption
	if err := ctx.Err(); err != nil {
		return nil, fmt.Errorf("context error during encryption: %w", err)
	}
	
	return result, err
}

// DecryptWithContext decrypts ciphertext with context support for cancellation
func (d *Decryptor) DecryptWithContext(ctx context.Context, ciphertext []byte) ([]byte, *saltpack.MessageKeyInfo, error) {
	// Check if context is already cancelled
	if err := ctx.Err(); err != nil {
		return nil, nil, fmt.Errorf("context error before decryption: %w", err)
	}
	
	// Perform decryption
	plaintext, info, err := d.Decrypt(ciphertext)
	
	// Check if context was cancelled during decryption
	if err := ctx.Err(); err != nil {
		return nil, nil, fmt.Errorf("context error during decryption: %w", err)
	}
	
	return plaintext, info, err
}
