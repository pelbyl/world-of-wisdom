package pow

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
)

// BinaryChallenge represents a compact binary format for challenges
// Format: [Version:1][Algorithm:1][Difficulty:1][Timestamp:8][ExpiresAt:8]
//         [Seed:16][Nonce:8][Signature:32][Argon2Params:10] (optional)
type BinaryChallenge struct {
	header     [3]byte   // version, algorithm, difficulty
	timestamps [16]byte  // timestamp + expiresAt (8 bytes each)
	seed       [16]byte  // truncated seed
	nonce      [8]byte   // random nonce
	signature  [32]byte  // HMAC-SHA256
	argon2     [10]byte  // optional Argon2 params (t:4, m:4, p:1, l:1)
}

// ChallengeFormat represents supported challenge formats
type ChallengeFormat string

const (
	FormatJSON   ChallengeFormat = "json"
	FormatBinary ChallengeFormat = "binary"
)

// AlgorithmType represents the PoW algorithm type for binary encoding
type AlgorithmType byte

const (
	AlgorithmSHA256 AlgorithmType = 0x01
	AlgorithmArgon2 AlgorithmType = 0x02
)

// ToBinary converts a SecureChallenge to binary format
func (c *SecureChallenge) ToBinary() ([]byte, error) {
	var bc BinaryChallenge
	
	// Header: version, algorithm, difficulty
	bc.header[0] = c.Version
	
	switch c.Algorithm {
	case "sha256":
		bc.header[1] = byte(AlgorithmSHA256)
	case "argon2":
		bc.header[1] = byte(AlgorithmArgon2)
	default:
		return nil, fmt.Errorf("unsupported algorithm for binary format: %s", c.Algorithm)
	}
	
	if c.Difficulty < 1 || c.Difficulty > 6 {
		return nil, fmt.Errorf("invalid difficulty for binary format: %d", c.Difficulty)
	}
	bc.header[2] = byte(c.Difficulty)
	
	// Timestamps
	binary.BigEndian.PutUint64(bc.timestamps[0:8], uint64(c.Timestamp))
	binary.BigEndian.PutUint64(bc.timestamps[8:16], uint64(c.ExpiresAt))
	
	// Seed (truncate or pad to 16 bytes)
	seedBytes, err := hex.DecodeString(c.Seed)
	if err != nil {
		return nil, fmt.Errorf("invalid seed hex: %w", err)
	}
	if len(seedBytes) > 16 {
		copy(bc.seed[:], seedBytes[:16])
	} else {
		copy(bc.seed[:], seedBytes)
	}
	
	// Nonce
	nonceBytes, err := hex.DecodeString(c.Nonce)
	if err != nil {
		return nil, fmt.Errorf("invalid nonce hex: %w", err)
	}
	if len(nonceBytes) > 8 {
		copy(bc.nonce[:], nonceBytes[:8])
	} else {
		copy(bc.nonce[:], nonceBytes)
	}
	
	// Signature
	signature, err := base64.StdEncoding.DecodeString(c.Signature)
	if err != nil {
		return nil, fmt.Errorf("invalid signature base64: %w", err)
	}
	if len(signature) != 32 {
		return nil, fmt.Errorf("signature must be 32 bytes, got %d", len(signature))
	}
	copy(bc.signature[:], signature)
	
	// Base structure (75 bytes: 3+16+16+8+32)
	result := make([]byte, 75)
	copy(result[0:3], bc.header[:])
	copy(result[3:19], bc.timestamps[:])
	copy(result[19:35], bc.seed[:])
	copy(result[35:43], bc.nonce[:])
	copy(result[43:75], bc.signature[:])
	
	// Add Argon2 parameters if needed
	if c.Algorithm == "argon2" && c.Argon2Params != nil {
		binary.BigEndian.PutUint32(bc.argon2[0:4], c.Argon2Params.Time)
		binary.BigEndian.PutUint32(bc.argon2[4:8], c.Argon2Params.Memory)
		bc.argon2[8] = byte(c.Argon2Params.Threads)
		bc.argon2[9] = byte(c.Argon2Params.KeyLength)
		
		// Extend result to include Argon2 params
		result = append(result, bc.argon2[:]...)
	}
	
	return result, nil
}

// FromBinary creates a SecureChallenge from binary data
func SecureChallengeFromBinary(data []byte, clientID string) (*SecureChallenge, error) {
	if len(data) < 75 {
		return nil, fmt.Errorf("binary data too short: expected at least 75 bytes, got %d", len(data))
	}
	
	challenge := &SecureChallenge{
		ClientID: clientID,
	}
	
	// Parse header
	challenge.Version = data[0]
	algorithmByte := AlgorithmType(data[1])
	challenge.Difficulty = int(data[2])
	
	switch algorithmByte {
	case AlgorithmSHA256:
		challenge.Algorithm = "sha256"
	case AlgorithmArgon2:
		challenge.Algorithm = "argon2"
	default:
		return nil, fmt.Errorf("unknown algorithm type: %d", algorithmByte)
	}
	
	// Parse timestamps
	challenge.Timestamp = int64(binary.BigEndian.Uint64(data[3:11]))
	challenge.ExpiresAt = int64(binary.BigEndian.Uint64(data[11:19]))
	
	// Parse seed
	challenge.Seed = hex.EncodeToString(data[19:35])
	
	// Parse nonce
	challenge.Nonce = hex.EncodeToString(data[35:43])
	
	// Parse signature
	challenge.Signature = base64.StdEncoding.EncodeToString(data[43:75])
	
	// Parse Argon2 parameters if present
	if challenge.Algorithm == "argon2" {
		if len(data) < 85 {
			return nil, fmt.Errorf("binary data too short for Argon2 challenge: expected at least 85 bytes, got %d", len(data))
		}
		
		challenge.Argon2Params = &Argon2Params{
			Time:      binary.BigEndian.Uint32(data[75:79]),
			Memory:    binary.BigEndian.Uint32(data[79:83]),
			Threads:   uint8(data[83]),
			KeyLength: uint32(data[84]),
		}
	}
	
	return challenge, nil
}

// ChallengeEncoder handles encoding challenges in different formats
type ChallengeEncoder struct {
	defaultFormat ChallengeFormat
}

// NewChallengeEncoder creates a new challenge encoder
func NewChallengeEncoder(defaultFormat ChallengeFormat) *ChallengeEncoder {
	return &ChallengeEncoder{
		defaultFormat: defaultFormat,
	}
}

// Encode encodes a challenge in the specified format
func (e *ChallengeEncoder) Encode(challenge *SecureChallenge, format ChallengeFormat) ([]byte, error) {
	switch format {
	case FormatJSON:
		return json.Marshal(challenge)
	case FormatBinary:
		return challenge.ToBinary()
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// Decode decodes a challenge from the specified format
func (e *ChallengeEncoder) Decode(data []byte, format ChallengeFormat, clientID string) (*SecureChallenge, error) {
	switch format {
	case FormatJSON:
		var challenge SecureChallenge
		if err := json.Unmarshal(data, &challenge); err != nil {
			return nil, fmt.Errorf("failed to decode JSON challenge: %w", err)
		}
		if clientID != "" {
			challenge.ClientID = clientID
		}
		return &challenge, nil
	case FormatBinary:
		return SecureChallengeFromBinary(data, clientID)
	default:
		return nil, fmt.Errorf("unsupported format: %s", format)
	}
}

// AutoDetectFormat attempts to detect the challenge format
func (e *ChallengeEncoder) AutoDetectFormat(data []byte) ChallengeFormat {
	// Try to detect JSON first
	if len(data) > 0 && (data[0] == '{' || bytes.TrimSpace(data)[0] == '{') {
		return FormatJSON
	}
	
	// Assume binary for other data
	return FormatBinary
}

// ChallengeTransport handles challenge transmission over network connections
type ChallengeTransport struct {
	encoder *ChallengeEncoder
}

// NewChallengeTransport creates a new challenge transport
func NewChallengeTransport() *ChallengeTransport {
	return &ChallengeTransport{
		encoder: NewChallengeEncoder(FormatJSON),
	}
}

// SendChallenge sends a challenge over a network connection
func (t *ChallengeTransport) SendChallenge(conn net.Conn, challenge *SecureChallenge, format ChallengeFormat) error {
	data, err := t.encoder.Encode(challenge, format)
	if err != nil {
		return fmt.Errorf("failed to encode challenge: %w", err)
	}
	
	// Send format indicator (1 byte) + length (4 bytes) + data
	formatByte := byte(0)
	switch format {
	case FormatJSON:
		formatByte = 1
	case FormatBinary:
		formatByte = 2
	}
	
	// Create packet: [format:1][length:4][data:n]
	packet := make([]byte, 5+len(data))
	packet[0] = formatByte
	binary.BigEndian.PutUint32(packet[1:5], uint32(len(data)))
	copy(packet[5:], data)
	
	_, err = conn.Write(packet)
	if err != nil {
		return fmt.Errorf("failed to send challenge: %w", err)
	}
	
	return nil
}

// ReceiveChallenge receives a challenge from a network connection
func (t *ChallengeTransport) ReceiveChallenge(conn net.Conn, clientID string) (*SecureChallenge, ChallengeFormat, error) {
	// Read header (format + length)
	header := make([]byte, 5)
	_, err := conn.Read(header)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read challenge header: %w", err)
	}
	
	formatByte := header[0]
	dataLength := binary.BigEndian.Uint32(header[1:5])
	
	if dataLength > 10*1024 { // 10KB max
		return nil, "", fmt.Errorf("challenge data too large: %d bytes", dataLength)
	}
	
	// Read challenge data
	data := make([]byte, dataLength)
	_, err = conn.Read(data)
	if err != nil {
		return nil, "", fmt.Errorf("failed to read challenge data: %w", err)
	}
	
	// Determine format
	var format ChallengeFormat
	switch formatByte {
	case 1:
		format = FormatJSON
	case 2:
		format = FormatBinary
	default:
		return nil, "", fmt.Errorf("unknown format byte: %d", formatByte)
	}
	
	// Decode challenge
	challenge, err := t.encoder.Decode(data, format, clientID)
	if err != nil {
		return nil, "", fmt.Errorf("failed to decode challenge: %w", err)
	}
	
	return challenge, format, nil
}

// CompressedChallengeTransport handles compressed challenge transmission
type CompressedChallengeTransport struct {
	*ChallengeTransport
	compressionEnabled bool
}

// NewCompressedChallengeTransport creates a transport with optional compression
func NewCompressedChallengeTransport(enableCompression bool) *CompressedChallengeTransport {
	return &CompressedChallengeTransport{
		ChallengeTransport: NewChallengeTransport(),
		compressionEnabled: enableCompression,
	}
}

// GetFormatStats returns statistics about format efficiency
func GetFormatStats(challenge *SecureChallenge) (map[string]any, error) {
	jsonData, err := json.Marshal(challenge)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal JSON: %w", err)
	}
	
	binaryData, err := challenge.ToBinary()
	if err != nil {
		return nil, fmt.Errorf("failed to create binary: %w", err)
	}
	
	jsonSize := len(jsonData)
	binarySize := len(binaryData)
	
	return map[string]any{
		"json_size":           jsonSize,
		"binary_size":         binarySize,
		"compression_ratio":   float64(binarySize) / float64(jsonSize),
		"space_saved_bytes":   jsonSize - binarySize,
		"space_saved_percent": (1.0 - float64(binarySize)/float64(jsonSize)) * 100,
	}, nil
}

// ValidateProtocolCompatibility checks if a challenge is compatible with a specific protocol version
func ValidateProtocolCompatibility(challenge *SecureChallenge, protocolVersion uint8) error {
	if challenge.Version > protocolVersion {
		return fmt.Errorf("challenge version %d not supported by protocol version %d", 
			challenge.Version, protocolVersion)
	}
	
	return nil
}

// MigrationHelper helps migrate between challenge formats
type MigrationHelper struct {
	encoder *ChallengeEncoder
}

// NewMigrationHelper creates a new migration helper
func NewMigrationHelper() *MigrationHelper {
	return &MigrationHelper{
		encoder: NewChallengeEncoder(FormatJSON),
	}
}

// ConvertFormat converts a challenge from one format to another
func (m *MigrationHelper) ConvertFormat(data []byte, fromFormat, toFormat ChallengeFormat, clientID string) ([]byte, error) {
	// Decode from source format
	challenge, err := m.encoder.Decode(data, fromFormat, clientID)
	if err != nil {
		return nil, fmt.Errorf("failed to decode source format: %w", err)
	}
	
	// Encode to target format
	result, err := m.encoder.Encode(challenge, toFormat)
	if err != nil {
		return nil, fmt.Errorf("failed to encode target format: %w", err)
	}
	
	return result, nil
}