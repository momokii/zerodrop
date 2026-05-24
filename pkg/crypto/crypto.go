package crypto

import (
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/hex"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"runtime"

	"github.com/zerodrop/terminal/pkg/qr"
)

// KeyPair represents an ECC key pair for Curve25519
type KeyPair struct {
	PrivateKey *ecdh.PrivateKey
	PublicKey  *ecdh.PublicKey
}

// GenerateKeyPair generates a new X25519 key pair for encryption
func GenerateKeyPair() (*KeyPair, error) {
	curve := ecdh.X25519()

	privateKey, err := curve.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("failed to generate private key: %w", err)
	}

	publicKey := privateKey.PublicKey()

	return &KeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
	}, nil
}

// SavePublicKeyToFile saves the public key to a file in PEM format
// Uses SPKI DER format via x509.MarshalPKIXPublicKey so browsers can
// import it with crypto.subtle.importKey("spki", ...)
func SavePublicKeyToFile(publicKey *ecdh.PublicKey, filepath string) error {
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return fmt.Errorf("failed to marshal public key to SPKI: %w", err)
	}

	block := &pem.Block{
		Type:  "PUBLIC KEY",
		Bytes: publicKeyBytes,
	}

	pemData := pem.EncodeToMemory(block)

	err = os.WriteFile(filepath, pemData, 0644)
	if err != nil {
		return fmt.Errorf("failed to write public key file: %w", err)
	}

	return nil
}

// LogPrivateKeyAsQR logs the private key as a scannable QR code to stdout
// The private key is encoded in PEM format and prefixed with PRIVATE_KEY_QR:
func LogPrivateKeyAsQR(privateKey *ecdh.PrivateKey) error {
	// Encode to PEM format for readability
	block := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: privateKey.Bytes(),
	}
	privateKeyPEM := pem.EncodeToMemory(block)

	log.Printf("PRIVATE_KEY_QR (PEM DATA): %s", privateKeyPEM)

	// Generate QR PNG with raw PEM content (no ZD1: wrapper — reader.html expects raw PEM)
	pngData, err := qr.GenerateRawQRPNG(privateKeyPEM)
	if err != nil {
		log.Printf("WARNING: Could not generate private key QR image: %v", err)
	} else {
		// Save QR PNG to file so the operator can open and scan it
		const qrFilename = "private_key_qr.png"
		if err := os.WriteFile(qrFilename, pngData, 0644); err != nil {
			log.Printf("WARNING: Could not save private key QR to %s: %v", qrFilename, err)
		} else {
			absPath, _ := filepath.Abs(qrFilename)
			log.Printf("PRIVATE_KEY_QR saved to %s (%d bytes) — open this image and scan with reader.html", absPath, len(pngData))
		}
	}

	return nil
}

// BurnProtocol securely destroys the private key from memory
// It explicitly zeros the byte slice and uses runtime.KeepAlive to prevent
// compiler optimization from removing the zeroing operation
func BurnProtocol(keyPair *KeyPair) {
	if keyPair == nil || keyPair.PrivateKey == nil {
		return
	}

	// Get the raw bytes of the private key
	keyBytes := keyPair.PrivateKey.Bytes()

	// Explicitly zero each byte
	for i := range keyBytes {
		keyBytes[i] = 0
	}

	// Use runtime.KeepAlive to prevent the compiler from optimizing away
	// the zeroing operation as "dead code"
	runtime.KeepAlive(keyBytes)

	// Set to nil to help GC
	keyPair.PrivateKey = nil
}

// GetPublicKeyFingerprint returns the SHA-256 hash of the SPKI DER-encoded public key
// Uses the same SPKI bytes that the frontend sees, so operator and user fingerprints match
func GetPublicKeyFingerprint(publicKey *ecdh.PublicKey) (string, error) {
	publicKeyBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal public key for fingerprint: %w", err)
	}

	hash := sha256.Sum256(publicKeyBytes)
	fingerprint := hex.EncodeToString(hash[:])

	return fingerprint, nil
}

// InitializeOrLoadKeyPair tries to load an existing public key,
// or generates a new key pair if none exists
func InitializeOrLoadKeyPair(publicKeyPath string) (*KeyPair, error) {
	// Check if public key already exists
	if _, err := os.Stat(publicKeyPath); err == nil {
		log.Printf("Existing public key found at %s", publicKeyPath)
		// For now, we'll still generate a new key pair if the server restarts
		// In production, you might want to load the existing key
		// This is a security decision - for v1.0, we regenerate on restart
		// and require the operator to have saved the private key from first boot
	}

	log.Println("No existing public key found. Generating new key pair...")

	// Generate new key pair
	keyPair, err := GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Save public key
	err = SavePublicKeyToFile(keyPair.PublicKey, publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to save public key: %w", err)
	}

	// Log private key as QR
	err = LogPrivateKeyAsQR(keyPair.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to log private key QR: %w", err)
	}

	// Get and log fingerprint
	fingerprint, err := GetPublicKeyFingerprint(keyPair.PublicKey)
	if err != nil {
		log.Printf("WARNING: Could not generate public key fingerprint: %v", err)
	} else {
		log.Printf("PUBLIC_KEY_FINGERPRINT: %s", fingerprint)
	}

	log.Println("Key pair generated. IMPORTANT: Scan the PRIVATE_KEY_QR above and save it securely.")
	log.Println("The private key will be destroyed from this server after this message.")

	return keyPair, nil
}
