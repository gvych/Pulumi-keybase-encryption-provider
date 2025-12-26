package crypto

import (
	"encoding/hex"
	"fmt"

	"github.com/keybase/saltpack"
)

// MessageInfo contains information about a decrypted Saltpack message
type MessageInfo struct {
	// ReceiverKID is the key identifier of the recipient key used for decryption
	ReceiverKID []byte
	
	// ReceiverKIDHex is the hex-encoded receiver key identifier
	ReceiverKIDHex string
	
	// SenderKID is the key identifier of the sender (nil if anonymous)
	SenderKID []byte
	
	// SenderKIDHex is the hex-encoded sender key identifier (empty if anonymous)
	SenderKIDHex string
	
	// IsAnonymousSender indicates if the sender is anonymous
	IsAnonymousSender bool
	
	// ReceiverIndex is the index of the recipient in the receivers list (0-based)
	// This indicates which recipient slot was used for decryption
	ReceiverIndex int
}

// ParseMessageKeyInfo extracts information from saltpack.MessageKeyInfo
// This function deserializes the message format to identify recipients and
// determine which recipient key was used for decryption.
//
// The MessageKeyInfo structure contains:
// - ReceiverKey: The secret key of the recipient that was used for decryption
// - SenderKey: The public key of the sender (nil if anonymous)
// - SenderIsAnon: Whether the sender is anonymous
// - ReceiverIsAnon: Whether the receiver is anonymous
//
// Returns:
//   - MessageInfo with parsed details
//   - Error if parsing fails
func ParseMessageKeyInfo(info *saltpack.MessageKeyInfo) (*MessageInfo, error) {
	if info == nil {
		return nil, fmt.Errorf("MessageKeyInfo is nil")
	}
	
	messageInfo := &MessageInfo{}
	
	// Extract receiver key information
	// ReceiverKey is the BoxSecretKey that was used to decrypt, so we get its public key
	if info.ReceiverKey != nil {
		receiverPublicKey := info.ReceiverKey.GetPublicKey()
		if receiverPublicKey != nil {
			messageInfo.ReceiverKID = receiverPublicKey.ToKID()
			messageInfo.ReceiverKIDHex = hex.EncodeToString(messageInfo.ReceiverKID)
		}
	} else {
		return nil, fmt.Errorf("ReceiverKey is nil in MessageKeyInfo")
	}
	
	// Extract sender key information (may be nil for anonymous sender)
	// Use the SenderIsAnon field for explicit anonymous check
	messageInfo.IsAnonymousSender = info.SenderIsAnon
	
	if !info.SenderIsAnon && info.SenderKey != nil {
		messageInfo.SenderKID = info.SenderKey.ToKID()
		messageInfo.SenderKIDHex = hex.EncodeToString(messageInfo.SenderKID)
	}
	
	// Note: ReceiverIndex would need to be extracted from the message header
	// The saltpack.MessageKeyInfo doesn't directly expose this, but it's available
	// in the internal message header structure. For now, we set it to -1 to indicate unknown.
	messageInfo.ReceiverIndex = -1
	
	return messageInfo, nil
}

// GetReceiverKeyID extracts the receiver key ID from MessageKeyInfo
// This is a convenience function for the common use case of just needing
// the key ID of the recipient that decrypted the message.
func GetReceiverKeyID(info *saltpack.MessageKeyInfo) ([]byte, error) {
	if info == nil {
		return nil, fmt.Errorf("MessageKeyInfo is nil")
	}
	
	if info.ReceiverKey == nil {
		return nil, fmt.Errorf("ReceiverKey is nil")
	}
	
	// ReceiverKey is a BoxSecretKey, get its public key
	receiverPublicKey := info.ReceiverKey.GetPublicKey()
	if receiverPublicKey == nil {
		return nil, fmt.Errorf("failed to get public key from ReceiverKey")
	}
	
	return receiverPublicKey.ToKID(), nil
}

// GetSenderKeyID extracts the sender key ID from MessageKeyInfo
// Returns nil if the sender is anonymous (no error in that case)
func GetSenderKeyID(info *saltpack.MessageKeyInfo) []byte {
	if info == nil || info.SenderIsAnon || info.SenderKey == nil {
		return nil
	}
	
	return info.SenderKey.ToKID()
}

// IsAnonymousSender checks if the message was sent anonymously
func IsAnonymousSender(info *saltpack.MessageKeyInfo) bool {
	if info == nil {
		return true // Treat nil as anonymous
	}
	
	return info.SenderIsAnon
}

// VerifySender checks if the message was sent by a specific sender key
// Returns true if the sender matches, false otherwise
// Returns false for anonymous messages
func VerifySender(info *saltpack.MessageKeyInfo, expectedSenderKey saltpack.BoxPublicKey) bool {
	if info == nil || info.SenderIsAnon || info.SenderKey == nil || expectedSenderKey == nil {
		return false
	}
	
	senderKID := info.SenderKey.ToKID()
	expectedKID := expectedSenderKey.ToKID()
	
	if len(senderKID) != len(expectedKID) {
		return false
	}
	
	// Compare key IDs byte by byte
	for i := range senderKID {
		if senderKID[i] != expectedKID[i] {
			return false
		}
	}
	
	return true
}

// VerifyReceiver checks if the message was decrypted by a specific receiver key
// Returns true if the receiver matches, false otherwise
func VerifyReceiver(info *saltpack.MessageKeyInfo, expectedReceiverKey saltpack.BoxPublicKey) bool {
	if info == nil || info.ReceiverKey == nil || expectedReceiverKey == nil {
		return false
	}
	
	// ReceiverKey is a BoxSecretKey, get its public key
	receiverPublicKey := info.ReceiverKey.GetPublicKey()
	if receiverPublicKey == nil {
		return false
	}
	
	receiverKID := receiverPublicKey.ToKID()
	expectedKID := expectedReceiverKey.ToKID()
	
	if len(receiverKID) != len(expectedKID) {
		return false
	}
	
	// Compare key IDs byte by byte
	for i := range receiverKID {
		if receiverKID[i] != expectedKID[i] {
			return false
		}
	}
	
	return true
}

// FormatKeyID formats a key ID as a human-readable hex string
// This is useful for logging and debugging
func FormatKeyID(kid []byte) string {
	if len(kid) == 0 {
		return "<empty>"
	}
	
	hexStr := hex.EncodeToString(kid)
	
	// For very long key IDs (>64 hex chars, i.e., >32 bytes), show first and last 8 characters
	// Standard Curve25519 keys are 32 bytes (64 hex chars), so we show them in full
	if len(hexStr) > 64 {
		return fmt.Sprintf("%s...%s", hexStr[:8], hexStr[len(hexStr)-8:])
	}
	
	return hexStr
}

// MessageInfoString returns a human-readable string representation of MessageInfo
func MessageInfoString(info *MessageInfo) string {
	if info == nil {
		return "<nil MessageInfo>"
	}
	
	result := "MessageInfo{\n"
	result += fmt.Sprintf("  ReceiverKID: %s\n", FormatKeyID(info.ReceiverKID))
	
	if info.IsAnonymousSender {
		result += "  Sender: <anonymous>\n"
	} else {
		result += fmt.Sprintf("  SenderKID: %s\n", FormatKeyID(info.SenderKID))
	}
	
	if info.ReceiverIndex >= 0 {
		result += fmt.Sprintf("  ReceiverIndex: %d\n", info.ReceiverIndex)
	} else {
		result += "  ReceiverIndex: <unknown>\n"
	}
	
	result += "}"
	
	return result
}
