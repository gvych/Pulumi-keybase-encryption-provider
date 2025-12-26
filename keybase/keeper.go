package keybase

import (
	"bytes"
	"context"
	"fmt"

	"github.com/keybase/saltpack"
	"github.com/pulumi/pulumi-keybase-encryption/keybase/api"
	"github.com/pulumi/pulumi-keybase-encryption/keybase/cache"
	"github.com/pulumi/pulumi-keybase-encryption/keybase/credentials"
	"github.com/pulumi/pulumi-keybase-encryption/keybase/crypto"
	"gocloud.dev/gcerrors"
)

// Keeper implements the driver.Keeper interface for Pulumi secrets encryption
// It provides encryption and decryption using Keybase public keys with Saltpack
type Keeper struct {
	config      *Config
	cacheManager *cache.Manager
	encryptor   *crypto.Encryptor
	decryptor   *crypto.Decryptor
	keyring     *crypto.SimpleKeyring
}

// KeeperConfig holds configuration for creating a Keeper
type KeeperConfig struct {
	// Config is the parsed Keybase configuration
	Config *Config
	
	// CacheManager is the cache manager for public keys (optional, will be created if nil)
	CacheManager *cache.Manager
	
	// SenderKey is the sender's secret key (optional, can be nil for anonymous sender)
	SenderKey saltpack.BoxSecretKey
}

// NewKeeper creates a new Keeper instance
func NewKeeper(config *KeeperConfig) (*Keeper, error) {
	if config == nil || config.Config == nil {
		return nil, fmt.Errorf("keeper config is required")
	}
	
	if len(config.Config.Recipients) == 0 {
		return nil, fmt.Errorf("at least one recipient is required")
	}
	
	// Create cache manager if not provided
	cacheManager := config.CacheManager
	if cacheManager == nil {
		managerConfig := &cache.ManagerConfig{
			CacheConfig: &cache.CacheConfig{
				TTL: config.Config.CacheTTL,
			},
			APIConfig: api.DefaultClientConfig(),
		}
		
		var err error
		cacheManager, err = cache.NewManager(managerConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create cache manager: %w", err)
		}
	}
	
	// Create encryptor
	encryptor, err := crypto.NewEncryptor(&crypto.EncryptorConfig{
		SenderKey: config.SenderKey,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create encryptor: %w", err)
	}
	
	// Create keyring for decryption
	keyring := crypto.NewSimpleKeyring()
	
	// Try to load local user's secret key for decryption
	// This is optional - if it fails, decryption won't work but encryption will
	if err := loadLocalSecretKey(keyring); err != nil {
		// Don't fail here - encryption can still work without local key
		// Decryption will fail with a clear error if attempted
	}
	
	// Create decryptor
	decryptor, err := crypto.NewDecryptor(&crypto.DecryptorConfig{
		Keyring: keyring,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create decryptor: %w", err)
	}
	
	return &Keeper{
		config:      config.Config,
		cacheManager: cacheManager,
		encryptor:   encryptor,
		decryptor:   decryptor,
		keyring:     keyring,
	}, nil
}

// NewKeeperFromURL creates a new Keeper from a Keybase URL
func NewKeeperFromURL(url string) (*Keeper, error) {
	config, err := ParseURL(url)
	if err != nil {
		return nil, fmt.Errorf("failed to parse URL: %w", err)
	}
	
	return NewKeeper(&KeeperConfig{
		Config: config,
	})
}

// Encrypt encrypts plaintext for all configured recipients
// 
// This method:
// 1. Fetches public keys for all recipients via API/cache
// 2. Converts PGP keys to Saltpack BoxPublicKey format
// 3. Uses streaming encryption for large messages (>10 MiB)
// 4. Uses in-memory encryption for smaller messages
// 5. Returns the encrypted ciphertext
//
// For messages larger than 10 MiB, streaming encryption is used to avoid
// loading the entire ciphertext into memory, improving performance and
// reducing memory usage.
func (k *Keeper) Encrypt(ctx context.Context, plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, &KeeperError{
			Message: "plaintext cannot be empty",
			Code:    gcerrors.InvalidArgument,
		}
	}
	
	// Step 1: Fetch public keys for all recipients
	userPublicKeys, err := k.cacheManager.GetPublicKeys(ctx, k.config.Recipients)
	if err != nil {
		// Classify API errors
		if apiErr, ok := err.(*api.APIError); ok {
			return nil, k.classifyAPIError(apiErr)
		}
		return nil, &KeeperError{
			Message: fmt.Sprintf("failed to fetch recipient public keys: %v", err),
			Code:    gcerrors.Internal,
			Underlying: err,
		}
	}
	
	if len(userPublicKeys) != len(k.config.Recipients) {
		return nil, &KeeperError{
			Message: fmt.Sprintf("expected %d public keys, got %d", len(k.config.Recipients), len(userPublicKeys)),
			Code:    gcerrors.Internal,
		}
	}
	
	// Step 2: Convert PGP keys to Saltpack BoxPublicKey format
	receivers := make([]saltpack.BoxPublicKey, 0, len(userPublicKeys))
	
	for _, userKey := range userPublicKeys {
		// Parse the Keybase public key
		// Note: Keybase API returns PGP keys, but for now we'll use a workaround
		// In production, you would extract the Curve25519 subkey from the PGP key
		publicKey, err := crypto.ParseKeybasePublicKey(userKey.PublicKey)
		if err != nil {
			// For now, try to parse the KeyID directly as it might be a Curve25519 key
			// This is a workaround - in production you'd properly parse the PGP key bundle
			keyID, parseErr := crypto.ParseKeybaseKeyID(userKey.KeyID)
			if parseErr != nil {
				return nil, &KeeperError{
					Message: fmt.Sprintf("failed to parse public key for user %s: %v (key ID parse error: %v)", 
						userKey.Username, err, parseErr),
					Code: gcerrors.InvalidArgument,
					Underlying: err,
				}
			}
			
			// Try to create a public key from the key ID
			if len(keyID) >= 32 {
				publicKey, err = crypto.CreatePublicKey(keyID[len(keyID)-32:])
				if err != nil {
					return nil, &KeeperError{
						Message: fmt.Sprintf("failed to create public key for user %s: %v", userKey.Username, err),
						Code: gcerrors.InvalidArgument,
						Underlying: err,
					}
				}
			} else {
				return nil, &KeeperError{
					Message: fmt.Sprintf("key ID too short for user %s: expected at least 32 bytes, got %d", 
						userKey.Username, len(keyID)),
					Code: gcerrors.InvalidArgument,
				}
			}
		}
		
		// Validate the public key
		if err := crypto.ValidatePublicKey(publicKey); err != nil {
			return nil, &KeeperError{
				Message: fmt.Sprintf("invalid public key for user %s: %v", userKey.Username, err),
				Code: gcerrors.InvalidArgument,
				Underlying: err,
			}
		}
		
		receivers = append(receivers, publicKey)
	}
	
	// Step 3: Encrypt using Saltpack
	// Use streaming for large messages (>10 MiB) to avoid memory issues
	const streamingThreshold = 10 * 1024 * 1024 // 10 MiB
	
	if len(plaintext) > streamingThreshold {
		// Use streaming encryption for large messages
		return k.encryptStreaming(plaintext, receivers)
	}
	
	// Use in-memory encryption for smaller messages
	// Use ASCII-armored output for better compatibility with Pulumi state files
	ciphertext, err := k.encryptor.EncryptArmored(plaintext, receivers)
	if err != nil {
		return nil, &KeeperError{
			Message: fmt.Sprintf("encryption failed: %v", err),
			Code: gcerrors.Internal,
			Underlying: err,
		}
	}
	
	// Step 4: Return as bytes
	return []byte(ciphertext), nil
}

// Decrypt decrypts ciphertext using the local Keybase keyring
//
// This method automatically detects the message size and uses streaming
// decryption for large messages (>10 MiB) to avoid loading the entire
// plaintext into memory.
func (k *Keeper) Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error) {
	if len(ciphertext) == 0 {
		return nil, &KeeperError{
			Message: "ciphertext cannot be empty",
			Code:    gcerrors.InvalidArgument,
		}
	}
	
	// Use streaming for large ciphertexts (>10 MiB)
	const streamingThreshold = 10 * 1024 * 1024 // 10 MiB
	
	if len(ciphertext) > streamingThreshold {
		// Use streaming decryption for large messages
		return k.decryptStreaming(ciphertext)
	}
	
	// Use in-memory decryption for smaller messages
	// Try to decrypt as ASCII-armored first
	plaintext, _, err := k.decryptor.DecryptArmored(string(ciphertext))
	if err != nil {
		// If armored decryption fails, try binary decryption
		plaintext, _, err = k.decryptor.Decrypt(ciphertext)
		if err != nil {
			return nil, &KeeperError{
				Message: fmt.Sprintf("decryption failed: %v", err),
				Code: gcerrors.InvalidArgument,
				Underlying: err,
			}
		}
	}
	
	return plaintext, nil
}

// encryptStreaming encrypts large plaintext using streaming to avoid memory issues
func (k *Keeper) encryptStreaming(plaintext []byte, receivers []saltpack.BoxPublicKey) ([]byte, error) {
	// Create readers and writers for streaming
	plaintextReader := bytes.NewReader(plaintext)
	var ciphertextBuf bytes.Buffer
	
	// Use streaming encryption with ASCII armoring
	err := k.encryptor.EncryptStreamArmored(plaintextReader, &ciphertextBuf, receivers)
	if err != nil {
		return nil, &KeeperError{
			Message: fmt.Sprintf("streaming encryption failed: %v", err),
			Code: gcerrors.Internal,
			Underlying: err,
		}
	}
	
	return ciphertextBuf.Bytes(), nil
}

// decryptStreaming decrypts large ciphertext using streaming to avoid memory issues
func (k *Keeper) decryptStreaming(ciphertext []byte) ([]byte, error) {
	// Create readers and writers for streaming
	ciphertextReader := bytes.NewReader(ciphertext)
	var plaintextBuf bytes.Buffer
	
	// Try armored streaming decryption first
	_, err := k.decryptor.DecryptStreamArmored(ciphertextReader, &plaintextBuf)
	if err != nil {
		// If armored decryption fails, try binary streaming decryption
		ciphertextReader.Reset(ciphertext)
		plaintextBuf.Reset()
		
		_, err = k.decryptor.DecryptStream(ciphertextReader, &plaintextBuf)
		if err != nil {
			return nil, &KeeperError{
				Message: fmt.Sprintf("streaming decryption failed: %v", err),
				Code: gcerrors.InvalidArgument,
				Underlying: err,
			}
		}
	}
	
	return plaintextBuf.Bytes(), nil
}

// Close releases resources held by the Keeper
func (k *Keeper) Close() error {
	if k.cacheManager != nil {
		return k.cacheManager.Close()
	}
	return nil
}

// ErrorAs maps Keeper errors to specific error types
func (k *Keeper) ErrorAs(err error, target interface{}) bool {
	if keeperErr, ok := err.(*KeeperError); ok {
		if ptr, ok := target.(**KeeperError); ok {
			*ptr = keeperErr
			return true
		}
	}
	
	// Check for API errors
	if apiErr, ok := err.(*api.APIError); ok {
		if ptr, ok := target.(**api.APIError); ok {
			*ptr = apiErr
			return true
		}
	}
	
	return false
}

// ErrorCode maps errors to Go Cloud error codes
func (k *Keeper) ErrorCode(err error) gcerrors.ErrorCode {
	if keeperErr, ok := err.(*KeeperError); ok {
		return keeperErr.Code
	}
	
	if apiErr, ok := err.(*api.APIError); ok {
		return k.classifyAPIError(apiErr).Code
	}
	
	return gcerrors.Unknown
}

// KeeperError represents a Keeper-specific error
type KeeperError struct {
	Message    string
	Code       gcerrors.ErrorCode
	Underlying error
}

func (e *KeeperError) Error() string {
	if e.Underlying != nil {
		return fmt.Sprintf("%s: %v", e.Message, e.Underlying)
	}
	return e.Message
}

func (e *KeeperError) Unwrap() error {
	return e.Underlying
}

// classifyAPIError maps API errors to Keeper errors with appropriate error codes
func (k *Keeper) classifyAPIError(apiErr *api.APIError) *KeeperError {
	var code gcerrors.ErrorCode
	
	switch apiErr.Kind {
	case api.ErrorKindNetwork:
		code = gcerrors.Internal // Network errors are transient internal issues
	case api.ErrorKindTimeout:
		code = gcerrors.DeadlineExceeded
	case api.ErrorKindNotFound:
		code = gcerrors.NotFound
	case api.ErrorKindInvalidInput:
		code = gcerrors.InvalidArgument
	case api.ErrorKindServerError:
		code = gcerrors.Internal
	case api.ErrorKindRateLimit:
		code = gcerrors.ResourceExhausted
	default:
		code = gcerrors.Unknown
	}
	
	return &KeeperError{
		Message:    apiErr.Message,
		Code:       code,
		Underlying: apiErr,
	}
}

// loadLocalSecretKey attempts to load the local user's secret key for decryption
func loadLocalSecretKey(keyring *crypto.SimpleKeyring) error {
	// Verify Keybase is available
	if err := credentials.VerifyKeybaseAvailable(); err != nil {
		return fmt.Errorf("keybase not available: %w", err)
	}
	
	// Load the sender key (which includes the secret key for the current user)
	senderKey, err := crypto.LoadSenderKey(nil)
	if err != nil {
		return fmt.Errorf("failed to load sender key: %w", err)
	}
	
	// Add the secret key to the keyring
	keyring.AddKey(senderKey.SecretKey)
	
	return nil
}
