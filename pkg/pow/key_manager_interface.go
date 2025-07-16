package pow

import "time"

// KeyManager is the interface for HMAC key management
type KeyManager interface {
	// GetCurrentKey returns the current signing key
	GetCurrentKey() []byte
	
	// GetKeys returns both current and previous keys for verification
	GetKeys() (current, previous []byte)
	
	// RotateKeys generates a new key and moves current to previous
	RotateKeys() error
	
	// GetRotationAge returns how long since the last key rotation
	GetRotationAge() time.Duration
}