package keybase

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/keybase/saltpack"
	"gocloud.dev/gcerrors"
)

// Example_decryptionErrorHandling demonstrates comprehensive error handling for decryption
func Example_decryptionErrorHandling() {
	// Create a keeper (in real use, this would be properly configured)
	keeper := &Keeper{
		config: DefaultConfig(),
	}

	// Example 1: No decryption key error
	err1 := saltpack.ErrNoDecryptionKey
	code1 := keeper.ErrorCode(err1)
	fmt.Printf("ErrNoDecryptionKey maps to: %v\n", code1 == gcerrors.NotFound)

	// Example 2: Bad ciphertext error
	err2 := saltpack.ErrBadCiphertext(5)
	code2 := keeper.ErrorCode(err2)
	fmt.Printf("ErrBadCiphertext maps to: %v\n", code2 == gcerrors.InvalidArgument)

	// Example 3: Context timeout
	err3 := context.DeadlineExceeded
	code3 := keeper.ErrorCode(err3)
	fmt.Printf("DeadlineExceeded maps to: %v\n", code3 == gcerrors.DeadlineExceeded)

	// Example 4: Context canceled
	err4 := context.Canceled
	code4 := keeper.ErrorCode(err4)
	fmt.Printf("Canceled maps to: %v\n", code4 == gcerrors.Canceled)

	// Output:
	// ErrNoDecryptionKey maps to: true
	// ErrBadCiphertext maps to: true
	// DeadlineExceeded maps to: true
	// Canceled maps to: true
}

// Example_extractSaltpackError demonstrates extracting underlying Saltpack errors
func Example_extractSaltpackError() {
	keeper := &Keeper{
		config: DefaultConfig(),
	}

	// Create a KeeperError wrapping a Saltpack error
	saltpackErr := saltpack.ErrBadCiphertext(3)
	keeperErr := &KeeperError{
		Message:    "decryption failed",
		Code:       gcerrors.InvalidArgument,
		Underlying: saltpackErr,
	}

	// Extract the underlying Saltpack error
	var extracted saltpack.ErrBadCiphertext
	if keeper.ErrorAs(keeperErr, &extracted) {
		fmt.Printf("Extracted error at packet: %d\n", extracted)
	}

	// Output:
	// Extracted error at packet: 3
}

// Example_decryptionWithTimeout demonstrates timeout handling
func Example_decryptionWithTimeout() {
	keeper := &Keeper{
		config: DefaultConfig(),
	}

	// Create a context with a very short timeout
	ctx, cancel := context.WithTimeout(context.Background(), 1*time.Nanosecond)
	defer cancel()

	// Wait for the context to expire
	time.Sleep(10 * time.Millisecond)

	// Try to decrypt (will fail due to timeout)
	_, err := keeper.Decrypt(ctx, []byte("some ciphertext"))
	if err != nil {
		code := keeper.ErrorCode(err)
		isTimeout := code == gcerrors.DeadlineExceeded || errors.Is(err, context.DeadlineExceeded)
		fmt.Printf("Timeout detected: %v\n", isTimeout)
	}

	// Output:
	// Timeout detected: true
}

// Example_errorMessages demonstrates detailed error messages
func Example_errorMessages() {
	keeper := &Keeper{
		config: DefaultConfig(),
	}

	// Test different error types and their messages
	testCases := []struct {
		name string
		err  error
	}{
		{"No decryption key", saltpack.ErrNoDecryptionKey},
		{"Bad ciphertext", saltpack.ErrBadCiphertext(1)},
		{"Bad tag", saltpack.ErrBadTag(2)},
	}

	for _, tc := range testCases {
		keeperErr := keeper.classifyDecryptionError(tc.err)
		fmt.Printf("%s: %s\n", tc.name, keeperErr.Code)
	}

	// Output:
	// No decryption key: NotFound
	// Bad ciphertext: InvalidArgument
	// Bad tag: InvalidArgument
}
