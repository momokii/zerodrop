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

	pubBytes := kp.PublicKey.Bytes()
	if len(pubBytes) != 32 {
		t.Errorf("[SECURITY] X25519 public key: expected 32 bytes, got %d", len(pubBytes))
	}

	privBytes := kp.PrivateKey.Bytes()
	if len(privBytes) != 32 {
		t.Errorf("[SECURITY] X25519 private key: expected 32 bytes, got %d", len(privBytes))
	}

	if kp.PublicKey.Curve() != ecdh.X25519() {
		t.Errorf("[SECURITY] Expected X25519 curve")
	}
	t.Logf("[SECURITY] Key generation uses X25519 (Curve25519) with correct 32-byte keys: PASS")
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

	if len(fp) != 64 {
		t.Errorf("[SECURITY] Fingerprint length: expected 64 (SHA-256 hex), got %d", len(fp))
	}

	for _, c := range fp {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f')) {
			t.Errorf("[SECURITY] Fingerprint contains non-hex char: %c", c)
			break
		}
	}
	t.Logf("[SECURITY] Public key fingerprint is SHA-256 (64 hex chars) for operator verification: PASS")
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

	fp1, err := GetPublicKeyFingerprint(kp.PublicKey)
	if err != nil {
		t.Fatal(err)
	}

	loaded, err := LoadPublicKeyFromFile(pubPath)
	if err != nil {
		t.Fatal(err)
	}
	fp2, err := GetPublicKeyFingerprint(loaded)
	if err != nil {
		t.Fatal(err)
	}

	if fp1 != fp2 {
		t.Errorf("[SECURITY] Fingerprints don't match after file save/reload:\n  in-memory: %s\n  from-file:  %s", fp1, fp2)
	}
	t.Logf("[SECURITY] Fingerprint identical after saving to disk and reloading (no corruption): PASS")
}

func TestSecurity_Crypto_ZD1Format_CorrectStructure(t *testing.T) {
	ephKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ephPubRaw := ephKey.PublicKey().Bytes()

	iv := make([]byte, 12)
	rand.Read(iv)

	fakeCiphertext := make([]byte, 48+16)
	rand.Read(fakeCiphertext)

	combined := make([]byte, 0, 32+12+len(fakeCiphertext))
	combined = append(combined, ephPubRaw...)
	combined = append(combined, iv...)
	combined = append(combined, fakeCiphertext...)

	zd1 := "ZD1:" + base64.StdEncoding.EncodeToString(combined)

	if !parseZD1(t, zd1) {
		t.Error("[SECURITY] ZD1 payload structure validation failed")
	}
	t.Logf("[SECURITY] ZD1 payload format (ephemeral key + IV + AES ciphertext) has correct structure: PASS")
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
	serverKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}
	ephKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	browserShared, err := ephKey.ECDH(serverKey.PublicKey())
	if err != nil {
		t.Fatal(err)
	}

	serverShared, err := serverKey.ECDH(ephKey.PublicKey())
	if err != nil {
		t.Fatal(err)
	}

	if len(browserShared) != 32 {
		t.Errorf("[SECURITY] Browser shared secret: expected 32 bytes, got %d", len(browserShared))
	}
	if fmt.Sprintf("%x", browserShared) != fmt.Sprintf("%x", serverShared) {
		t.Error("[SECURITY] ECDH shared secrets don't match — browser and server would be unable to communicate!")
	}
	t.Logf("[SECURITY] Browser and server derive the same encryption key (ECDH shared secret matches): PASS")
}

func TestSecurity_Crypto_ECDH_GoBrowserCompatible(t *testing.T) {
	// Full end-to-end encryption roundtrip: encrypt in browser, decrypt on server.
	// Proves the entire crypto chain (X25519 + AES-256-GCM) works correctly.
	// In production the server never has the private key, so it CANNOT do this.

	serverKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	ephKey, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		t.Fatal(err)
	}

	// Browser side: derive shared secret, encrypt plaintext
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

	iv := make([]byte, gcm.NonceSize())
	rand.Read(iv)

	plaintext := []byte("The server can never read this secret message!")
	ciphertext := gcm.Seal(nil, iv, plaintext, nil)

	// Construct ZD1 payload: ephPub(32) + iv(12) + ciphertext+tag
	ephPubRaw := ephKey.PublicKey().Bytes()
	combined := make([]byte, 0, len(ephPubRaw)+len(iv)+len(ciphertext))
	combined = append(combined, ephPubRaw...)
	combined = append(combined, iv...)
	combined = append(combined, ciphertext...)

	zd1Payload := "ZD1:" + base64.StdEncoding.EncodeToString(combined)

	// Server side: decode ZD1 payload, derive same shared secret, decrypt
	decoded, err := base64.StdEncoding.DecodeString(zd1Payload[4:])
	if err != nil {
		t.Fatal(err)
	}
	decEphPub := decoded[:32]
	decIV := decoded[32:44]
	decCiphertext := decoded[44:]

	decEphPubKey, err := ecdh.X25519().NewPublicKey(decEphPub)
	if err != nil {
		t.Fatal(err)
	}

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
		t.Fatalf("[SECURITY] Decryption failed — crypto chain is broken: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("[SECURITY] Decrypted text doesn't match original:\n  got:      %q\n  expected: %q", decrypted, plaintext)
	}
	t.Logf("[SECURITY] Full encryption roundtrip works: browser encrypts -> ZD1 payload -> server decrypts: PASS")
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
		t.Errorf("[SECURITY] Private key file permissions: expected 0600 (owner-only), got %04o (other users can read!)", perm)
	}
	t.Logf("[SECURITY] Private key file is owner-only (0600) — other users cannot read it: PASS")
}
