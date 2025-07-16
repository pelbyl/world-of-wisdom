package pow

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"
)

// FileKeyManager handles HMAC key generation, storage, and rotation using file storage
type FileKeyManager struct {
	mu          sync.RWMutex
	currentKey  []byte
	previousKey []byte
	rotatedAt   time.Time
	keyPath     string
}

// KeyData represents the persisted key structure
type KeyData struct {
	CurrentKey  string    `json:"current_key"`
	PreviousKey string    `json:"previous_key,omitempty"`
	RotatedAt   time.Time `json:"rotated_at"`
	Version     int       `json:"version"`
}

// NewFileKeyManager creates a new file-based key manager with persistent storage
func NewFileKeyManager(keyPath string) (*FileKeyManager, error) {
	km := &FileKeyManager{
		keyPath: keyPath,
	}

	// Try to load existing keys
	if err := km.loadKeys(); err != nil {
		// If no keys exist, generate new ones
		if os.IsNotExist(err) {
			if err := km.generateAndSaveKeys(); err != nil {
				return nil, fmt.Errorf("failed to generate initial keys: %w", err)
			}
		} else {
			return nil, fmt.Errorf("failed to load keys: %w", err)
		}
	}

	return km, nil
}

// GetCurrentKey returns the current signing key
func (km *FileKeyManager) GetCurrentKey() []byte {
	km.mu.RLock()
	defer km.mu.RUnlock()
	
	key := make([]byte, len(km.currentKey))
	copy(key, km.currentKey)
	return key
}

// GetKeys returns both current and previous keys for verification
func (km *FileKeyManager) GetKeys() (current, previous []byte) {
	km.mu.RLock()
	defer km.mu.RUnlock()
	
	current = make([]byte, len(km.currentKey))
	copy(current, km.currentKey)
	
	if km.previousKey != nil {
		previous = make([]byte, len(km.previousKey))
		copy(previous, km.previousKey)
	}
	
	return current, previous
}

// RotateKeys generates a new key and moves current to previous
func (km *FileKeyManager) RotateKeys() error {
	km.mu.Lock()
	defer km.mu.Unlock()

	// Generate new key
	newKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, newKey); err != nil {
		return fmt.Errorf("failed to generate new key: %w", err)
	}

	// Rotate keys
	km.previousKey = km.currentKey
	km.currentKey = newKey
	km.rotatedAt = time.Now()

	// Save to disk
	return km.saveKeys()
}

// loadKeys loads keys from persistent storage
func (km *FileKeyManager) loadKeys() error {
	data, err := os.ReadFile(km.keyPath)
	if err != nil {
		return err
	}

	var keyData KeyData
	if err := json.Unmarshal(data, &keyData); err != nil {
		return fmt.Errorf("failed to unmarshal key data: %w", err)
	}

	// Decode base64 keys
	currentKey, err := base64.StdEncoding.DecodeString(keyData.CurrentKey)
	if err != nil {
		return fmt.Errorf("failed to decode current key: %w", err)
	}

	km.currentKey = currentKey
	km.rotatedAt = keyData.RotatedAt

	if keyData.PreviousKey != "" {
		previousKey, err := base64.StdEncoding.DecodeString(keyData.PreviousKey)
		if err != nil {
			return fmt.Errorf("failed to decode previous key: %w", err)
		}
		km.previousKey = previousKey
	}

	return nil
}

// saveKeys persists keys to storage
func (km *FileKeyManager) saveKeys() error {
	keyData := KeyData{
		CurrentKey: base64.StdEncoding.EncodeToString(km.currentKey),
		RotatedAt:  km.rotatedAt,
		Version:    1,
	}

	if km.previousKey != nil {
		keyData.PreviousKey = base64.StdEncoding.EncodeToString(km.previousKey)
	}

	data, err := json.MarshalIndent(keyData, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal key data: %w", err)
	}

	// Write to temporary file first
	tmpPath := km.keyPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write key file: %w", err)
	}

	// Atomic rename
	if err := os.Rename(tmpPath, km.keyPath); err != nil {
		return fmt.Errorf("failed to rename key file: %w", err)
	}

	return nil
}

// generateAndSaveKeys generates initial keys and saves them
func (km *FileKeyManager) generateAndSaveKeys() error {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return fmt.Errorf("failed to generate key: %w", err)
	}

	km.currentKey = key
	km.rotatedAt = time.Now()

	return km.saveKeys()
}

// GetRotationAge returns how long since the last key rotation
func (km *FileKeyManager) GetRotationAge() time.Duration {
	km.mu.RLock()
	defer km.mu.RUnlock()
	return time.Since(km.rotatedAt)
}