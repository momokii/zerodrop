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
	"path/filepath"
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

// LogPrivateKeyAsQR logs the private key as QR codes (PEM + JWK) to stdout
// and saves PNG files for reliable phone scanning. The key is exported in
// PKCS#8 DER format so it can be imported by reader.html via
// crypto.subtle.importKey("pkcs8", ...).
//
// All output goes to stdout as a single write to prevent Docker from
// interleaving it with stderr log lines.
func LogPrivateKeyAsQR(privateKey *ecdh.PrivateKey, saveDir string, logEnabled bool) error {
	pkcs8Bytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("failed to marshal private key to PKCS#8: %w", err)
	}
	privateKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: pkcs8Bytes,
	})

	// Build JWK (RFC 8037) — recommended format for reader.html
	privRaw := privateKey.Bytes()
	pubRaw := privateKey.PublicKey().Bytes()
	jwk := map[string]string{
		"kty": "OKP",
		"crv": "X25519",
		"x":   base64.RawURLEncoding.EncodeToString(pubRaw),
		"d":   base64.RawURLEncoding.EncodeToString(privRaw),
	}
	jwkJSON, _ := json.Marshal(jwk)

	// Save scannable PNG QRs to file
	var pngPaths []string
	for _, item := range []struct {
		name    string
		content []byte
	}{
		{"private_key_qr.png", privateKeyPEM},
		{"private_key_jwk_qr.png", jwkJSON},
	} {
		p := filepath.Join(saveDir, item.name)
		pngData, pngErr := qr.GenerateRawQRPNG(item.content)
		if pngErr == nil {
			if writeErr := os.WriteFile(p, pngData, 0600); writeErr != nil {
				log.Printf("WARNING: Could not save %s: %v", item.name, writeErr)
				continue
			}
			pngPaths = append(pngPaths, item.name)
		} else {
			log.Printf("WARNING: Could not generate %s: %v", item.name, pngErr)
		}
	}

	// Build the ENTIRE output in one buffer so it hits stdout as a single write.
	var buf strings.Builder

	buf.WriteString("\n")
	buf.WriteString("╔═══════════════════════════════════════════════════════╗\n")
	buf.WriteString("║          PRIVATE KEY — SAVE SECURELY                 ║\n")
	buf.WriteString("║  Scan PNG files with phone for best results.         ║\n")
	buf.WriteString("║  DELETE files after scanning — key burned from RAM.  ║\n")
	buf.WriteString("╚═══════════════════════════════════════════════════════╝\n")
	buf.WriteString("\n")

	if len(pngPaths) > 0 {
		buf.WriteString("  Scannable PNG QR files in data/:\n")
		for _, p := range pngPaths {
			buf.WriteString(fmt.Sprintf("    - %s\n", p))
		}
		buf.WriteString("\n")
	}

	// PEM QR (for reader.html PKCS#8 import)
	buf.WriteString("  ── PEM format ──\n")
	pemQR, pemErr := qr.GenerateQRASCII(string(privateKeyPEM))
	if pemErr == nil {
		buf.WriteString(pemQR)
	}
	buf.WriteString("\n")

	// JWK QR (recommended for reader.html)
	buf.WriteString("  ── JWK format (recommended) ──\n")
	jwkQR, jwkErr := qr.GenerateQRASCII(string(jwkJSON))
	if jwkErr == nil {
		buf.WriteString(jwkQR)
	}
	buf.WriteString("\n")

	// Plaintext copies when logging is enabled
	if logEnabled {
		buf.WriteString("=== PRIVATE KEY (JWK) ===\n")
		buf.Write(jwkJSON)
		buf.WriteString("\n=== END JWK ===\n\n")
		buf.WriteString("=== PRIVATE KEY (PEM) ===\n")
		buf.Write(privateKeyPEM)
		buf.WriteString("=== END PEM ===\n\n")
	}

	buf.WriteString("╔═══════════════════════════════════════════════════════╗\n")
	buf.WriteString("║              END OF PRIVATE KEY                      ║\n")
	buf.WriteString("╚═══════════════════════════════════════════════════════╝\n")
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

func fileExists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

// LoadPublicKeyFromFile loads a SPKI PEM-encoded public key from disk.
func LoadPublicKeyFromFile(path string) (*ecdh.PublicKey, error) {
	pemData, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read public key file: %w", err)
	}
	block, _ := pem.Decode(pemData)
	if block == nil || block.Type != "PUBLIC KEY" {
		return nil, fmt.Errorf("failed to decode PEM block from public key file")
	}
	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse SPKI public key: %w", err)
	}
	ecdhPub, ok := pub.(*ecdh.PublicKey)
	if !ok {
		return nil, fmt.Errorf("public key is not an ECDH key")
	}
	return ecdhPub, nil
}

// SavePrivateKeyToFile saves the private key as PKCS#8 PEM with 0600 permissions.
func SavePrivateKeyToFile(privateKey *ecdh.PrivateKey, path string) error {
	pkcs8Bytes, err := x509.MarshalPKCS8PrivateKey(privateKey)
	if err != nil {
		return fmt.Errorf("failed to marshal private key: %w", err)
	}
	block := &pem.Block{
		Type:  "PRIVATE KEY",
		Bytes: pkcs8Bytes,
	}
	pemData := pem.EncodeToMemory(block)
	return os.WriteFile(path, pemData, 0600)
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

// InitializeOrLoadKeyPair tries to load an existing key pair from disk,
// or generates a new key pair if none exists. When both key files exist
// and rotation is not requested, the public key is loaded and the private
// key stays on disk (never loaded into RAM).
// All operator-facing output (QR, fingerprint, instructions) is written to
// stdout to prevent Docker from interleaving stderr log lines.
func InitializeOrLoadKeyPair(publicKeyPath, privateKeyPath string, keyRotate, logEnabled bool) (*KeyPair, error) {
	pubExists := fileExists(publicKeyPath)
	privExists := fileExists(privateKeyPath)

	if pubExists && privExists && !keyRotate {
		log.Println("Existing key pair found. Reusing (set KEY_ROTATE=true to regenerate).")

		publicKey, err := LoadPublicKeyFromFile(publicKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to load existing public key: %w", err)
		}

		fingerprint, _ := GetPublicKeyFingerprint(publicKey)
		fmt.Fprintf(os.Stdout, "Reusing existing key pair (fingerprint: %s)\n", fingerprint)
		fmt.Fprintln(os.Stdout, "Private key is on disk. Download via admin dashboard if needed.")

		return &KeyPair{
			PublicKey:  publicKey,
			PrivateKey: nil,
		}, nil
	}

	// First run or rotation requested
	if keyRotate {
		log.Println("KEY_ROTATE=true: forcing new key pair generation...")
	} else {
		log.Println("No existing key pair found. Generating new key pair...")
	}

	keyPair, err := GenerateKeyPair()
	if err != nil {
		return nil, fmt.Errorf("failed to generate key pair: %w", err)
	}

	// Save public key
	err = SavePublicKeyToFile(keyPair.PublicKey, publicKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to save public key: %w", err)
	}

	// Save private key to disk (0600)
	err = SavePrivateKeyToFile(keyPair.PrivateKey, privateKeyPath)
	if err != nil {
		return nil, fmt.Errorf("failed to save private key: %w", err)
	}

	// Determine directory for saving private key QR PNG (same dir as public key)
	dataDir := filepath.Dir(publicKeyPath)

	// Log private key as QR + save PNG (writes entire block to stdout atomically)
	err = LogPrivateKeyAsQR(keyPair.PrivateKey, dataDir, logEnabled)
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
	fmt.Fprintln(os.Stdout, "Key pair generated. IMPORTANT: Save the private key QR above.")
	fmt.Fprintln(os.Stdout, "The private key will be burned from server memory after this message.")

	return keyPair, nil
}
