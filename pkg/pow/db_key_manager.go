package pow

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/pbkdf2"
	generated "world-of-wisdom/internal/database/generated"
)

// DBKeyManager handles HMAC key generation, storage, and rotation using database
type DBKeyManager struct {
	mu          sync.RWMutex
	db          *pgxpool.Pool
	queries     *generated.Queries
	currentKey  []byte
	previousKey []byte
	rotatedAt   time.Time
	version     int
	
	// Encryption key derived from master secret
	encryptionKey []byte
}

// NewDBKeyManager creates a new database-backed key manager
func NewDBKeyManager(db *pgxpool.Pool, masterSecret string) (*DBKeyManager, error) {
	// Derive encryption key from master secret
	salt := []byte("wow-hmac-key-encryption")
	encryptionKey := pbkdf2.Key([]byte(masterSecret), salt, 10000, 32, sha256.New)
	
	km := &DBKeyManager{
		db:            db,
		queries:       generated.New(),
		encryptionKey: encryptionKey,
	}

	// Try to load existing keys from database
	if err := km.loadKeys(); err != nil {
		// If no keys exist, generate new ones
		if err.Error() == "no active key found" {
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
func (km *DBKeyManager) GetCurrentKey() []byte {
	km.mu.RLock()
	defer km.mu.RUnlock()
	
	key := make([]byte, len(km.currentKey))
	copy(key, km.currentKey)
	return key
}

// GetKeys returns both current and previous keys for verification
func (km *DBKeyManager) GetKeys() (current, previous []byte) {
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
func (km *DBKeyManager) RotateKeys() error {
	km.mu.Lock()
	defer km.mu.Unlock()

	// Generate new key
	newKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, newKey); err != nil {
		return fmt.Errorf("failed to generate new key: %w", err)
	}

	// Encrypt keys
	encryptedCurrent, err := km.encrypt(newKey)
	if err != nil {
		return fmt.Errorf("failed to encrypt current key: %w", err)
	}
	
	var encryptedPrevious pgtype.Text
	if km.currentKey != nil {
		prev, err := km.encrypt(km.currentKey)
		if err != nil {
			return fmt.Errorf("failed to encrypt previous key: %w", err)
		}
		encryptedPrevious = pgtype.Text{String: prev, Valid: true}
	}

	// Start transaction
	ctx := context.Background()
	tx, err := km.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}
	defer tx.Rollback(ctx)

	// Deactivate current keys
	if err := km.queries.DeactivateHMACKeys(ctx, tx); err != nil {
		return fmt.Errorf("failed to deactivate current keys: %w", err)
	}

	// Create new key record
	metadata := map[string]interface{}{
		"rotated_from_version": km.version,
		"rotation_reason":      "scheduled",
	}
	metadataJSON, _ := json.Marshal(metadata)
	
	newVersion := km.version + 1
	_, err = km.queries.CreateHMACKey(ctx, tx, generated.CreateHMACKeyParams{
		KeyVersion:            int32(newVersion),
		EncryptedKey:         encryptedCurrent,
		PreviousEncryptedKey: encryptedPrevious,
		Metadata:             metadataJSON,
	})
	if err != nil {
		return fmt.Errorf("failed to create new key: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	// Update in-memory keys
	km.previousKey = km.currentKey
	km.currentKey = newKey
	km.rotatedAt = time.Now()
	km.version = newVersion

	return nil
}

// loadKeys loads keys from database
func (km *DBKeyManager) loadKeys() error {
	ctx := context.Background()
	
	// Get active key
	keyRecord, err := km.queries.GetActiveHMACKey(ctx, km.db)
	if err != nil {
		return fmt.Errorf("no active key found")
	}

	// Decrypt current key
	currentKey, err := km.decrypt(keyRecord.EncryptedKey)
	if err != nil {
		return fmt.Errorf("failed to decrypt current key: %w", err)
	}

	km.currentKey = currentKey
	km.rotatedAt = keyRecord.RotatedAt.Time
	km.version = int(keyRecord.KeyVersion)

	// Decrypt previous key if exists
	if keyRecord.PreviousEncryptedKey.Valid && keyRecord.PreviousEncryptedKey.String != "" {
		previousKey, err := km.decrypt(keyRecord.PreviousEncryptedKey.String)
		if err != nil {
			return fmt.Errorf("failed to decrypt previous key: %w", err)
		}
		km.previousKey = previousKey
	}

	return nil
}

// generateAndSaveKeys generates initial keys and saves them to database
func (km *DBKeyManager) generateAndSaveKeys() error {
	key := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return fmt.Errorf("failed to generate key: %w", err)
	}

	// Encrypt key
	encryptedKey, err := km.encrypt(key)
	if err != nil {
		return fmt.Errorf("failed to encrypt key: %w", err)
	}

	// Save to database
	ctx := context.Background()
	metadata := map[string]interface{}{
		"initial_key": true,
		"created_by":  "system",
	}
	metadataJSON, _ := json.Marshal(metadata)
	
	_, err = km.queries.CreateHMACKey(ctx, km.db, generated.CreateHMACKeyParams{
		KeyVersion:   1,
		EncryptedKey: encryptedKey,
		Metadata:     metadataJSON,
	})
	if err != nil {
		return fmt.Errorf("failed to save key to database: %w", err)
	}

	km.currentKey = key
	km.rotatedAt = time.Now()
	km.version = 1

	return nil
}

// encrypt encrypts data using AES-GCM
func (km *DBKeyManager) encrypt(data []byte) (string, error) {
	block, err := aes.NewCipher(km.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// decrypt decrypts data using AES-GCM
func (km *DBKeyManager) decrypt(encrypted string) ([]byte, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(km.encryptionKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

// GetRotationAge returns how long since the last key rotation
func (km *DBKeyManager) GetRotationAge() time.Duration {
	km.mu.RLock()
	defer km.mu.RUnlock()
	return time.Since(km.rotatedAt)
}