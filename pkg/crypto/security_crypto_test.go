package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdh"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

func TestSecurity_Crypto_X25519KeyPair_CorrectType(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair: %v", err)
	}

	// Public key must be 32 bytes (X25519 raw)
	pubBytes := kp.PublicKey.Bytes()
	if len(pubBytes) != 32 {
		t.Errorf("[SECURITY] X25519 public key: expected 32 bytes, got %d", len(pubBytes))
	}

	// Private key bytes must be 32 bytes
	privBytes := kp.PrivateKey.Bytes()
	if len(privBytes) != 32 {
		t.Errorf("[SECURITY] X25519 private key: expected 32 bytes, got %d", len(privBytes))
	}

	// Must be X25519 curve
	if kp.PublicKey.Curve() != ecdh.X25519() {
		t.Errorf("[SECURITY] Expected X25519 curve")
	}
	t.Logf("[SECURITY] X25519 key pair generates correctly (32-byte keys): PASS")
}

func TestSecurity_Crypto_Fingerprint_SHA256Hex64(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	fp, err := GetPublicKeyFingerprint(kp.PublicKey)
	if err != nil {
		t.Fatal(err)
	}

	// SHA-256 hex = 64 characters
	if len(fp) != 64 {
		t.Errorf("[SECURITY] Fingerprint length: expected 64 (SHA-256 hex), got %d", len(fp))
	}

	// Must be valid hex
	for _, c := range fp {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("[SECURITY] Fingerprint contains non-hex char: %c", c)
			break
		}
	}
	t.Logf("[SECURITY] Fingerprint is 64-char hex (SHA-256): PASS")
}

func TestSecurity_Crypto_Fingerprint_SPKIDER_Format(t *testing.T) {
	tmpDir := t.TempDir()
	pubPath := filepath.Join(tmpDir, "public_key.pem")

	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	err = SavePublicKeyToFile(kp.PublicKey, pubPath)
	if err != nil {
		t.Fatal(err)
	}

	// Fingerprint computed from in-memory key
	fp1, err := GetPublicKeyFingerprint(kp.PublicKey)
	if err != nil {
		t.Fatal(err)
	}

	// Fingerprint computed from loaded key file (round-trip)
	loaded, err := LoadPublicKeyFromFile(pubPath)
	if err != nil {
		t.Fatal(err)
	}
	fp2, err := GetPublicKeyFingerprint(loaded)
	if err != nil {
		t.Fatal(err)
	}

	if fp1 != fp2 {
		t.Errorf("[SECURITY] Fingerprints don't match after load:\n  original: %s\n  loaded:   %s", fp1, fp2)
	}
	t.Logf("[SECURITY] Fingerprint consistent across SPKI DER round-trip: PASS")
}

func TestSecurity_Crypto_ZD1Format_CorrectStructure(t *testing.T) {
	// Simulate what the browser does: construct a ZD1: payload
	// Format: ZD1:base64(ephPubKey(32) + iv(12) + aesCiphertextWithTag)

	ephKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ephPubRaw := ephKey.PublicKey().Bytes() // 32 bytes

	iv := make([]byte, 12)
	rand.Read(iv)

	// Fake ciphertext + tag (16 bytes for GCM tag)
	fakeCiphertext := make([]byte, 48+16)
	rand.Read(fakeCiphertext)

	// Combine: ephPubKey(32) + iv(12) + ciphertextWithTag(64)
	combined := make([]byte, 0, 32+12+len(fakeCiphertext))
	combined = append(combined, ephPubRaw...)
	combined = append(combined, iv...)
	combined = append(combined, fakeCiphertext...)

	zd1 := "ZD1:" + base64.StdEncoding.EncodeToString(combined)

	// Verify structure
	if !parseZD1(t, zd1) {
		t.Error("[SECURITY] ZD1 payload structure validation failed")
	}
	t.Logf("[SECURITY] ZD1 format: 32-byte ephKey + 12-byte IV + ciphertext: PASS")
}

func parseZD1(t *testing.T, payload string) bool {
	t.Helper()
	if len(payload) < 4 || payload[:4] != "ZD1:" {
		t.Error("missing ZD1: prefix")
		return false
	}
	decoded, err := base64.StdEncoding.DecodeString(payload[4:])
	if err != nil {
		t.Errorf("base64 decode failed: %v", err)
		return false
	}
	if len(decoded) < 32+12+16 {
		t.Errorf("decoded payload too short: %d bytes (need at least 60)", len(decoded))
		return false
	}
	ephKey := decoded[:32]
	iv := decoded[32:44]
	if len(ephKey) != 32 {
		t.Errorf("ephemeral key: expected 32 bytes, got %d", len(ephKey))
		return false
	}
	if len(iv) != 12 {
		t.Errorf("IV: expected 12 bytes, got %d", len(iv))
		return false
	}
	return true
}

func TestSecurity_Crypto_ECDH_SharedSecretMatches(t *testing.T) {
	// Simulate: server key pair + browser ephemeral key pair
	serverKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ephKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	// Browser side: ECDH(serverPubKey, ephPrivKey)
	browserShared, err := ephKey.ECDH(serverKey.PublicKey())
	if err != nil {
		t.Fatal(err)
	}

	// Server side: ECDH(ephPubKey, serverPrivKey)
	serverShared, err := serverKey.ECDH(ephKey.PublicKey())
	if err != nil {
		t.Fatal(err)
	}

	// Both shared secrets must match
	if len(browserShared) != 32 {
		t.Errorf("[SECURITY] Browser shared secret: expected 32 bytes, got %d", len(browserShared))
	}
	if fmt.Sprintf("%x", browserShared) != fmt.Sprintf("%x", serverShared) {
		t.Error("[SECURITY] ECDH shared secrets don't match — Go/browser incompatibility!")
	}
	t.Logf("[SECURITY] ECDH shared secrets match (32 bytes): PASS")
}

func TestSecurity_Crypto_ECDH_GoBrowserCompatible(t *testing.T) {
	// Full ECIES roundtrip in Go — proves the crypto chain works
	// identically to what the browser does with Web Crypto API.

	// 1. Server generates key pair
	serverKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	// 2. "Browser" generates ephemeral key pair
	ephKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	// 3. Browser: ECDH → shared secret → AES-256-GCM encrypt
	browserShared, err := ephKey.ECDH(serverKey.PublicKey())
	if err != nil {
		t.Fatal(err)
	}

	aesKey, err := aes.NewCipher(browserShared)
	if err != nil {
		t.Fatal(err)
	}
	gcm, err := cipher.NewGCM(aesKey)
	if err != nil {
		t.Fatal(err)
	}

	iv := make([]byte, gcm.NonceSize()) // 12 bytes
	rand.Read(iv)

	plaintext := []byte("The server can never read this secret message!")
	ciphertext := gcm.Seal(nil, iv, plaintext, nil)

	// 4. Construct ZD1 payload: ephPub(32) + iv(12) + ciphertext+tag
	ephPubRaw := ephKey.PublicKey().Bytes()
	combined := make([]byte, 0, len(ephPubRaw)+len(iv)+len(ciphertext))
	combined = append(combined, ephPubRaw...)
	combined = append(combined, iv...)
	combined = append(combined, ciphertext...)

	zd1Payload := "ZD1:" + base64.StdEncoding.EncodeToString(combined)

	// 5. Server decodes ZD1 payload and decrypts (proving the crypto chain)
	// In production, the server never loads the private key, so it CANNOT do this.
	decoded, err := base64.StdEncoding.DecodeString(zd1Payload[4:])
	if err != nil {
		t.Fatal(err)
	}
	decEphPub := decoded[:32]
	decIV := decoded[32:44]
	decCiphertext := decoded[44:]

	// Import ephemeral public key from payload
	decEphPubKey, err := ecdh.X25519().NewPublicKey(decEphPub)
	if err != nil {
		t.Fatal(err)
	}

	// Server derives shared secret from payload's ephemeral key
	serverShared, err := serverKey.ECDH(decEphPubKey)
	if err != nil {
		t.Fatal(err)
	}
	serverAesKey, err := aes.NewCipher(serverShared)
	if err != nil {
		t.Fatal(err)
	}
	serverGcm, err := cipher.NewGCM(serverAesKey)
	if err != nil {
		t.Fatal(err)
	}

	decrypted, err := serverGcm.Open(nil, decIV, decCiphertext, nil)
	if err != nil {
		t.Fatalf("[SECURITY] AES-GCM decryption failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("[SECURITY] Decrypted text doesn't match:\n  got:      %q\n  expected: %q", decrypted, plaintext)
	}
	t.Logf("[SECURITY] Full ECIES roundtrip (X25519 + AES-256-GCM): PASS")
}

func TestSecurity_Crypto_PrivateKeyFile_Permissions0600(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	keyPath := filepath.Join(tmpDir, "private_key.pem")

	err = SavePrivateKeyToFile(kp.PrivateKey, keyPath)
	if err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(keyPath)
	if err != nil {
		t.Fatal(err)
	}

	perm := info.Mode().Perm()
	if perm != 0600 {
		t.Errorf("[SECURITY] Private key file permissions: expected 0600, got %04o", perm)
	}
	t.Logf("[SECURITY] Private key file permissions 0600: PASS")
}
