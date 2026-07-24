package main

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"golang.org/x/crypto/pbkdf2"
)

const (
	// Encryption constants
	KeySize   = 32 // AES-256
	SaltSize  = 32 // 256 bits
	NonceSize = 12 // GCM standard nonce size
	Iterations = 100000 // PBKDF2 iterations
)

// EncryptedFileMetadata stores encryption metadata
type EncryptedFileMetadata struct {
	Salt          []byte `json:"salt"`
	Nonce         []byte `json:"nonce"`
	IsEncrypted   bool   `json:"is_encrypted"`
	OriginalSize  int64  `json:"original_size"`
	EncryptedSize int64  `json:"encrypted_size"`
}

// GenerateSalt creates a random salt for key derivation
func GenerateSalt() ([]byte, error) {
	salt := make([]byte, SaltSize)
	if _, err := rand.Read(salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}
	return salt, nil
}

// DeriveKey derives an encryption key from a passphrase using PBKDF2
func DeriveKey(passphrase string, salt []byte) []byte {
	return pbkdf2.Key([]byte(passphrase), salt, Iterations, KeySize, sha256.New)
}

// EncryptFile encrypts a file using AES-256-GCM
// Returns: encrypted file path, metadata, error
func EncryptFile(inputPath, passphrase string) (string, *EncryptedFileMetadata, error) {
	// Read original file
	plaintext, err := os.ReadFile(inputPath)
	if err != nil {
		return "", nil, fmt.Errorf("failed to read file: %w", err)
	}

	originalSize := int64(len(plaintext))

	// Generate salt and derive key
	salt, err := GenerateSalt()
	if err != nil {
		return "", nil, err
	}

	key := DeriveKey(passphrase, salt)

	// Create cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, NonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return "", nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	encryptedSize := int64(len(ciphertext))

	// Write encrypted file (overwrite original)
	if err := os.WriteFile(inputPath, ciphertext, 0644); err != nil {
		return "", nil, fmt.Errorf("failed to write encrypted file: %w", err)
	}

	metadata := &EncryptedFileMetadata{
		Salt:          salt,
		Nonce:         nonce,
		IsEncrypted:   true,
		OriginalSize:  originalSize,
		EncryptedSize: encryptedSize,
	}

	return inputPath, metadata, nil
}

// DecryptFile decrypts a file using AES-256-GCM
// Returns: decrypted data (in memory), error
func DecryptFile(filePath, passphrase string, metadata *EncryptedFileMetadata) ([]byte, error) {
	if !metadata.IsEncrypted {
		// Not encrypted, just read and return
		return os.ReadFile(filePath)
	}

	// Read encrypted file
	ciphertext, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read encrypted file: %w", err)
	}

	// Derive key from passphrase and salt
	key := DeriveKey(passphrase, metadata.Salt)

	// Create cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Decrypt
	plaintext, err := gcm.Open(nil, metadata.Nonce, ciphertext, nil)
	if err != nil {
		return nil, errors.New("decryption failed: invalid passphrase or corrupted file")
	}

	return plaintext, nil
}

// EncryptStream encrypts data from a reader and writes to a writer
// Useful for streaming large files
func EncryptStream(reader io.Reader, writer io.Writer, passphrase string) (*EncryptedFileMetadata, error) {
	// Read all data (for simplicity, can be optimized for streaming)
	plaintext, err := io.ReadAll(reader)
	if err != nil {
		return nil, fmt.Errorf("failed to read input: %w", err)
	}

	originalSize := int64(len(plaintext))

	// Generate salt and derive key
	salt, err := GenerateSalt()
	if err != nil {
		return nil, err
	}

	key := DeriveKey(passphrase, salt)

	// Create cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, NonceSize)
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt
	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)
	encryptedSize := int64(len(ciphertext))

	// Write encrypted data
	if _, err := writer.Write(ciphertext); err != nil {
		return nil, fmt.Errorf("failed to write encrypted data: %w", err)
	}

	metadata := &EncryptedFileMetadata{
		Salt:          salt,
		Nonce:         nonce,
		IsEncrypted:   true,
		OriginalSize:  originalSize,
		EncryptedSize: encryptedSize,
	}

	return metadata, nil
}

// DecryptStream decrypts data from a reader and writes to a writer
func DecryptStream(reader io.Reader, writer io.Writer, metadata *EncryptedFileMetadata, passphrase string) error {
	if !metadata.IsEncrypted {
		// Not encrypted, just copy
		_, err := io.Copy(writer, reader)
		return err
	}

	// Read encrypted data
	ciphertext, err := io.ReadAll(reader)
	if err != nil {
		return fmt.Errorf("failed to read encrypted data: %w", err)
	}

	// Derive key
	key := DeriveKey(passphrase, metadata.Salt)

	// Create cipher
	block, err := aes.NewCipher(key)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM: %w", err)
	}

	// Decrypt
	plaintext, err := gcm.Open(nil, metadata.Nonce, ciphertext, nil)
	if err != nil {
		return errors.New("decryption failed: invalid passphrase or corrupted file")
	}

	// Write decrypted data
	if _, err := writer.Write(plaintext); err != nil {
		return fmt.Errorf("failed to write decrypted data: %w", err)
	}

	return nil
}

// GenerateRoomPassphrase generates a secure random passphrase for a room
// This is optional - rooms can use PIN as passphrase or generate separate one
func GenerateRoomPassphrase() (string, error) {
	bytes := make([]byte, 32) // 256 bits
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate passphrase: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// EncryptFileInPlace encrypts a file in the uploads directory
// Used during file upload process
func EncryptFileInPlace(filename, passphrase string) (*EncryptedFileMetadata, error) {
	filePath := filepath.Join(UploadDir, filename)
	_, metadata, err := EncryptFile(filePath, passphrase)
	if err != nil {
		return nil, err
	}
	return metadata, nil
}

// ValidatePassphrase checks if a passphrase can decrypt a file
func ValidatePassphrase(filePath string, metadata *EncryptedFileMetadata, passphrase string) bool {
	if !metadata.IsEncrypted {
		return true // No encryption, no validation needed
	}

	_, err := DecryptFile(filePath, passphrase, metadata)
	return err == nil
}
