package crypto

import (
	"crypto/ecdh"
	"crypto/rand"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"

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
// and as plain PEM text. The key is exported in PKCS#8 DER format so it can be
// imported by reader.html via crypto.subtle.importKey("pkcs8", ...).
//
// All output goes to stdout as a single write to prevent Docker from
// interleaving it with stderr log lines (which would break QR scannability).
func LogPrivateKeyAsQR(privateKey *ecdh.PrivateKey, logEnabled bool) error {
	pkcs8Bytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("failed to marshal private key to PKCS#8: %w", err)
	}
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: pkcs8Bytes,
	})

	// Build the ENTIRE output in one buffer so it hits stdout as a single write.
	// This prevents Docker from interleaving stderr log lines into the QR art.
	var buf strings.Builder

	buf.WriteString("\n")
	buf.WriteString("╔══════════════════════════════════════════════════╗\n")
	buf.WriteString("║           PRIVATE KEY — SCAN THIS QR            ║\n")
	buf.WriteString("║     Save securely. Key is destroyed after       ║\n")
	buf.WriteString("║     this message. This is your ONLY chance.     ║\n")
	buf.WriteString("╚══════════════════════════════════════════════════╝\n")
	buf.WriteString("\n")

	asciiArt, asciiErr := qr.GenerateQRASCII(string(privateKeyPEM))
	if asciiErr == nil {
		buf.WriteString(asciiArt)
		buf.WriteString("\n")
	}

	// JWK and PEM plaintext only when structured logging is enabled
	if logEnabled {
		// JWK format (RFC 8037) - more portable across browsers
		privRaw := privateKey.Bytes()
		pubRaw := privateKey.PublicKey().Bytes()
		jwk := map[string]string{
			"kty": "OKP",
			"crv": "X25519",
			"x":   base64.RawURLEncoding.EncodeToString(pubRaw),
			"d":   base64.RawURLEncoding.EncodeToString(privRaw),
		}
		jwkJSON, _ := json.Marshal(jwk)

		buf.WriteString("\n=== PRIVATE KEY (JWK) - Recommended for reader.html ===\n")
		buf.Write(jwkJSON)
		buf.WriteString("\n=== END PRIVATE KEY JWK ===\n\n")
		buf.WriteString("=== PRIVATE KEY (PEM) - Alternative format ===\n")
		buf.Write(privateKeyPEM)
		buf.WriteString("\n=== END PRIVATE KEY PEM ===\n\n")
	}

	buf.WriteString("╔══════════════════════════════════════════════════╗\n")
	buf.WriteString("║              END OF PRIVATE KEY                 ║\n")
	buf.WriteString("╚══════════════════════════════════════════════════╝\n")
	buf.WriteString("\n")

	os.Stdout.WriteString(buf.String())

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
// or generates a new key pair if none exists.
// All operator-facing output (QR, fingerprint, instructions) is written to
// stdout to prevent Docker from interleaving stderr log lines.
func InitializeOrLoadKeyPair(publicKeyPath string, logEnabled bool) (*KeyPair, error) {
	// Check if public key already exists
	if _, err := os.Stat(publicKeyPath); err == nil {
		log.Printf("Existing public key found at %s (will be overwritten)", publicKeyPath)
		log.Println("Regenerating new key pair (restart always generates fresh keys) ...")
	} else {
		log.Println("No existing public key found. Generating new key pair ...")
	}

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

	// Log private key as QR (writes entire block to stdout atomically)
	err = LogPrivateKeyAsQR(keyPair.PrivateKey, logEnabled)
	if err != nil {
		return nil, fmt.Errorf("failed to log private key QR: %w", err)
	}

	// Get fingerprint
	fingerprint, err := GetPublicKeyFingerprint(keyPair.PublicKey)
	fingerprintStr := "unavailable"
	if err != nil {
		log.Printf("WARNING: Could not generate public key fingerprint: %v", err)
	} else {
		fingerprintStr = fingerprint
	}

	// Post-QR instructions — also go to stdout so they stay grouped with the QR
	fmt.Fprintf(os.Stdout, "PUBLIC_KEY_FINGERPRINT: %s\n", fingerprintStr)
	fmt.Fprintln(os.Stdout, "Key pair generated. IMPORTANT: Scan the QR above and save it securely.")
	fmt.Fprintln(os.Stdout, "The private key will be destroyed from this server after this message.")

	return keyPair, nil
}
