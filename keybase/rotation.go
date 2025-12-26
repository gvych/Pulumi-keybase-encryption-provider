package keybase

import (
	"context"
	"fmt"
	"time"

	"github.com/keybase/saltpack"
	"github.com/pulumi/pulumi-keybase-encryption/keybase/cache"
	"github.com/pulumi/pulumi-keybase-encryption/keybase/crypto"
)

// KeyRotationDetector detects when messages were encrypted with retired keys
type KeyRotationDetector struct {
	cacheManager *cache.Manager
}

// KeyRotationInfo contains information about a key rotation event
type KeyRotationInfo struct {
	// ReceiverKeyRetired indicates if the receiver key used is no longer current
	ReceiverKeyRetired bool
	
	// SenderKeyRetired indicates if the sender key used is no longer current
	SenderKeyRetired bool
	
	// ReceiverUsername is the username of the receiver (if known)
	ReceiverUsername string
	
	// SenderUsername is the username of the sender (if known and not anonymous)
	SenderUsername string
	
	// DecryptedAt is when the message was decrypted
	DecryptedAt time.Time
	
	// MessageKeyInfo is the original decryption metadata
	MessageKeyInfo *saltpack.MessageKeyInfo
	
	// NeedsReEncryption indicates if the message should be re-encrypted
	NeedsReEncryption bool
	
	// RetirementReason describes why the key is considered retired
	RetirementReason string
}

// NewKeyRotationDetector creates a new key rotation detector
func NewKeyRotationDetector(cacheManager *cache.Manager) *KeyRotationDetector {
	return &KeyRotationDetector{
		cacheManager: cacheManager,
	}
}

// DetectRotation checks if a decrypted message used retired keys
//
// This method compares the keys used during decryption against the current
// keys configured for the recipients. If the keys don't match, it indicates
// a key rotation has occurred.
//
// Parameters:
//   - ctx: Context for API calls to fetch current keys
//   - messageInfo: The MessageKeyInfo returned from decryption
//   - configuredRecipients: The current list of recipient usernames
//
// Returns:
//   - KeyRotationInfo with details about any detected rotation
//   - Error if key verification fails
func (d *KeyRotationDetector) DetectRotation(
	ctx context.Context,
	messageInfo *saltpack.MessageKeyInfo,
	configuredRecipients []string,
) (*KeyRotationInfo, error) {
	if messageInfo == nil {
		return nil, fmt.Errorf("messageInfo cannot be nil")
	}
	
	info := &KeyRotationInfo{
		MessageKeyInfo: messageInfo,
		DecryptedAt:    time.Now(),
	}
	
	// Check if the receiver key is still current
	// We do this by comparing the receiver key against all current recipient keys
	if err := d.checkReceiverKeyRotation(ctx, info, configuredRecipients); err != nil {
		return info, fmt.Errorf("failed to check receiver key rotation: %w", err)
	}
	
	// Check if the sender key is still current (if not anonymous)
	if !messageInfo.SenderIsAnon {
		if err := d.checkSenderKeyRotation(ctx, info); err != nil {
			return info, fmt.Errorf("failed to check sender key rotation: %w", err)
		}
	}
	
	// Determine if re-encryption is needed
	info.NeedsReEncryption = info.ReceiverKeyRetired || info.SenderKeyRetired
	
	return info, nil
}

// checkReceiverKeyRotation verifies if the receiver key is still current
func (d *KeyRotationDetector) checkReceiverKeyRotation(
	ctx context.Context,
	info *KeyRotationInfo,
	configuredRecipients []string,
) error {
	if len(configuredRecipients) == 0 {
		// No recipients configured - can't verify
		return nil
	}
	
	// Fetch current public keys for all recipients
	currentKeys, err := d.cacheManager.GetPublicKeys(ctx, configuredRecipients)
	if err != nil {
		return fmt.Errorf("failed to fetch current recipient keys: %w", err)
	}
	
	// Get the receiver's public key from the secret key used for decryption
	receiverPublicKey := info.MessageKeyInfo.ReceiverKey.GetPublicKey()
	
	// Check if the receiver's current key matches any of the configured recipients
	keyFound := false
	for _, userKey := range currentKeys {
		// Parse the current public key
		currentPublicKey, err := crypto.ParseKeybasePublicKey(userKey.PublicKey)
		if err != nil {
			// Try parsing from KeyID
			keyID, parseErr := crypto.ParseKeybaseKeyID(userKey.KeyID)
			if parseErr != nil {
				continue
			}
			
			if len(keyID) >= 32 {
				currentPublicKey, err = crypto.CreatePublicKey(keyID[len(keyID)-32:])
				if err != nil {
					continue
				}
			}
		}
		
		// Compare keys
		if crypto.KeysEqual(receiverPublicKey, currentPublicKey) {
			keyFound = true
			info.ReceiverUsername = userKey.Username
			break
		}
	}
	
	if !keyFound {
		info.ReceiverKeyRetired = true
		info.RetirementReason = "receiver key no longer matches any configured recipient"
	}
	
	return nil
}

// checkSenderKeyRotation verifies if the sender key is still current
func (d *KeyRotationDetector) checkSenderKeyRotation(
	ctx context.Context,
	info *KeyRotationInfo,
) error {
	// For sender key rotation detection, we would need to:
	// 1. Determine the sender's username (not directly available from MessageKeyInfo)
	// 2. Fetch the sender's current public key from Keybase API
	// 3. Compare it with the sender key used in the message
	//
	// However, MessageKeyInfo only provides the sender's public key, not their username.
	// In a production system, you might maintain a mapping of public keys to usernames,
	// or require senders to include their username in a separate metadata field.
	//
	// For now, we'll mark this as a limitation that requires manual verification.
	
	// Note: This is a simplified implementation
	// A full implementation would require additional metadata or a key-to-username mapping
	info.SenderKeyRetired = false // Cannot determine without additional context
	
	return nil
}

// ReEncryptionRequest represents a request to re-encrypt data with new keys
type ReEncryptionRequest struct {
	// Plaintext is the decrypted data to re-encrypt
	Plaintext []byte
	
	// NewRecipients is the list of new recipient usernames
	// If empty, uses the currently configured recipients
	NewRecipients []string
	
	// RotationInfo is the original rotation detection info
	RotationInfo *KeyRotationInfo
}

// ReEncryptionResult contains the result of a re-encryption operation
type ReEncryptionResult struct {
	// Ciphertext is the newly encrypted data
	Ciphertext []byte
	
	// Recipients is the list of recipients used for re-encryption
	Recipients []string
	
	// ReEncryptedAt is when the re-encryption occurred
	ReEncryptedAt time.Time
	
	// PreviousRotationInfo is the rotation info that triggered re-encryption
	PreviousRotationInfo *KeyRotationInfo
}

// MigrationResult contains the result of migrating a single encrypted value
type MigrationResult struct {
	// Plaintext is the decrypted data
	Plaintext []byte
	
	// NewCiphertext is the re-encrypted data (nil if no rotation detected)
	NewCiphertext []byte
	
	// RotationDetected indicates if key rotation was detected
	RotationDetected bool
	
	// RotationInfo contains details about the rotation
	RotationInfo *KeyRotationInfo
	
	// Error is any error that occurred during migration
	Error error
}
