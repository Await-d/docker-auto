package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"

	"golang.org/x/crypto/bcrypt"
)

const (
	// DefaultBcryptCost is the default cost for bcrypt hashing
	DefaultBcryptCost = 12
	// MinPasswordLength is the minimum password length
	MinPasswordLength = 8
	// RandomStringCharset is the character set for random string generation
	RandomStringCharset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
)

// HashPassword hashes a password using bcrypt
func HashPassword(password string) (string, error) {
	if len(password) < MinPasswordLength {
		return "", fmt.Errorf("password must be at least %d characters long", MinPasswordLength)
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), DefaultBcryptCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}

	return string(hash), nil
}

// VerifyPassword verifies a password against its hash
func VerifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

// GenerateRandomString generates a random string of specified length
func GenerateRandomString(length int) string {
	if length <= 0 {
		return ""
	}

	b := make([]byte, length)
	charsetLen := len(RandomStringCharset)

	for i := range b {
		randomIndex := make([]byte, 1)
		rand.Read(randomIndex)
		b[i] = RandomStringCharset[int(randomIndex[0])%charsetLen]
	}

	return string(b)
}

// GenerateSecureRandomBytes generates cryptographically secure random bytes
func GenerateSecureRandomBytes(length int) ([]byte, error) {
	if length <= 0 {
		return nil, fmt.Errorf("length must be positive")
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}

	return bytes, nil
}

// GenerateSecureRandomString generates a base64-encoded secure random string
func GenerateSecureRandomString(length int) (string, error) {
	bytes, err := GenerateSecureRandomBytes(length)
	if err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(bytes), nil
}

// deriveKey derives a 32-byte key from a password using SHA-256
func deriveKey(password string) []byte {
	hash := sha256.Sum256([]byte(password))
	return hash[:]
}

// EncryptSensitiveData encrypts sensitive data using AES-GCM
func EncryptSensitiveData(data, key string) (string, error) {
	if data == "" {
		return "", fmt.Errorf("data cannot be empty")
	}
	if key == "" {
		return "", fmt.Errorf("encryption key cannot be empty")
	}

	// Derive key from password
	derivedKey := deriveKey(key)

	// Create AES cipher
	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Generate nonce
	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	// Encrypt data
	ciphertext := gcm.Seal(nonce, nonce, []byte(data), nil)

	// Encode to base64
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

// DecryptSensitiveData decrypts sensitive data using AES-GCM
func DecryptSensitiveData(encrypted, key string) (string, error) {
	if encrypted == "" {
		return "", fmt.Errorf("encrypted data cannot be empty")
	}
	if key == "" {
		return "", fmt.Errorf("decryption key cannot be empty")
	}

	// Decode from base64
	ciphertext, err := base64.StdEncoding.DecodeString(encrypted)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	// Derive key from password
	derivedKey := deriveKey(key)

	// Create AES cipher
	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return "", fmt.Errorf("failed to create AES cipher: %w", err)
	}

	// Create GCM mode
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}

	// Extract nonce
	if len(ciphertext) < gcm.NonceSize() {
		return "", fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:gcm.NonceSize()], ciphertext[gcm.NonceSize():]

	// Decrypt data
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt data: %w", err)
	}

	return string(plaintext), nil
}

// ValidatePasswordStrength validates password strength
func ValidatePasswordStrength(password string) error {
	if len(password) < MinPasswordLength {
		return fmt.Errorf("password must be at least %d characters long", MinPasswordLength)
	}

	hasUpper := false
	hasLower := false
	hasDigit := false
	hasSpecial := false

	for _, char := range password {
		switch {
		case char >= 'A' && char <= 'Z':
			hasUpper = true
		case char >= 'a' && char <= 'z':
			hasLower = true
		case char >= '0' && char <= '9':
			hasDigit = true
		case char >= 32 && char <= 126: // printable ASCII characters
			if !((char >= 'A' && char <= 'Z') || (char >= 'a' && char <= 'z') || (char >= '0' && char <= '9')) {
				hasSpecial = true
			}
		}
	}

	if !hasUpper {
		return fmt.Errorf("password must contain at least one uppercase letter")
	}
	if !hasLower {
		return fmt.Errorf("password must contain at least one lowercase letter")
	}
	if !hasDigit {
		return fmt.Errorf("password must contain at least one digit")
	}
	if !hasSpecial {
		return fmt.Errorf("password must contain at least one special character")
	}

	return nil
}

// HashSHA256 creates a SHA-256 hash of the input string
func HashSHA256(input string) string {
	hash := sha256.Sum256([]byte(input))
	return fmt.Sprintf("%x", hash)
}

// GenerateAPIKey generates a secure API key
func GenerateAPIKey() (string, error) {
	// Generate 32 random bytes (256 bits)
	bytes, err := GenerateSecureRandomBytes(32)
	if err != nil {
		return "", fmt.Errorf("failed to generate API key: %w", err)
	}

	// Encode as base64 URL-safe string
	return base64.URLEncoding.EncodeToString(bytes), nil
}

// CompareHash compares a plain text string with its SHA-256 hash
func CompareHash(plaintext, hash string) bool {
	return HashSHA256(plaintext) == hash
}

// EncryptRegistryCredentials encrypts registry credentials using the provided key
func EncryptRegistryCredentials(username, password, encryptionKey string) (encryptedUsername, encryptedPassword string, err error) {
	if username == "" && password == "" {
		return "", "", fmt.Errorf("username and password cannot both be empty")
	}

	if username != "" {
		encryptedUsername, err = EncryptSensitiveData(username, encryptionKey)
		if err != nil {
			return "", "", fmt.Errorf("failed to encrypt username: %w", err)
		}
	}

	if password != "" {
		encryptedPassword, err = EncryptSensitiveData(password, encryptionKey)
		if err != nil {
			return "", "", fmt.Errorf("failed to encrypt password: %w", err)
		}
	}

	return encryptedUsername, encryptedPassword, nil
}

// DecryptRegistryCredentials decrypts registry credentials using the provided key
func DecryptRegistryCredentials(encryptedUsername, encryptedPassword, encryptionKey string) (username, password string, err error) {
	if encryptedUsername == "" && encryptedPassword == "" {
		return "", "", fmt.Errorf("encrypted username and password cannot both be empty")
	}

	if encryptedUsername != "" {
		username, err = DecryptSensitiveData(encryptedUsername, encryptionKey)
		if err != nil {
			return "", "", fmt.Errorf("failed to decrypt username: %w", err)
		}
	}

	if encryptedPassword != "" {
		password, err = DecryptSensitiveData(encryptedPassword, encryptionKey)
		if err != nil {
			return "", "", fmt.Errorf("failed to decrypt password: %w", err)
		}
	}

	return username, password, nil
}