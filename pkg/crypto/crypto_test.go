package crypto

import (
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateKeyPair(t *testing.T) {
	keyPair, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	if keyPair.PrivateKey == nil {
		t.Error("PrivateKey is nil")
	}

	if keyPair.PublicKey == nil {
		t.Error("PublicKey is nil")
	}
}

func TestSavePublicKeyToFile(t *testing.T) {
	keyPair, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	// Create temp directory
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test_public_key.pem")

	// Save public key
	err = SavePublicKeyToFile(keyPair.PublicKey, testFile)
	if err != nil {
		t.Fatalf("SavePublicKeyToFile failed: %v", err)
	}

	// Verify file exists
	if _, err := os.Stat(testFile); os.IsNotExist(err) {
		t.Error("Public key file was not created")
	}
}

func TestLogPrivateKeyAsQR(t *testing.T) {
	keyPair, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	// This should not panic, just log (pass false for logEnabled in tests)
	err = LogPrivateKeyAsQR(keyPair.PrivateKey, t.TempDir(), false)
	if err != nil {
		t.Fatalf("LogPrivateKeyAsQR failed: %v", err)
	}
}

func TestBurnProtocol(t *testing.T) {
	keyPair, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	// Execute burn protocol
	BurnProtocol(keyPair)

	// Verify private key is nil
	if keyPair.PrivateKey != nil {
		t.Error("PrivateKey was not set to nil after BurnProtocol")
	}
}

func TestGetPublicKeyFingerprint(t *testing.T) {
	keyPair, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	fingerprint, err := GetPublicKeyFingerprint(keyPair.PublicKey)
	if err != nil {
		t.Fatalf("GetPublicKeyFingerprint failed: %v", err)
	}

	// SHA-256 hash should be 64 hex characters
	if len(fingerprint) != 64 {
		t.Errorf("Fingerprint has wrong length: got %d, want 64", len(fingerprint))
	}
}

func TestInitializeOrLoadKeyPair(t *testing.T) {
	tmpDir := t.TempDir()
	publicKeyPath := filepath.Join(tmpDir, "public_key.pem")
	privateKeyPath := filepath.Join(tmpDir, "private_key.pem")

	// First call should generate new key
	keyPair1, err := InitializeOrLoadKeyPair(publicKeyPath, privateKeyPath, false, false)
	if err != nil {
		t.Fatalf("InitializeOrLoadKeyPair failed: %v", err)
	}

	if keyPair1.PrivateKey == nil {
		t.Error("PrivateKey is nil after first initialization")
	}

	// Verify both key files were created
	if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
		t.Error("Public key file was not created")
	}
	if _, err := os.Stat(privateKeyPath); os.IsNotExist(err) {
		t.Error("Private key file was not created")
	}

	// Second call should reuse existing key pair
	keyPair2, err := InitializeOrLoadKeyPair(publicKeyPath, privateKeyPath, false, false)
	if err != nil {
		t.Fatalf("InitializeOrLoadKeyPair reuse failed: %v", err)
	}

	if keyPair2.PrivateKey != nil {
		t.Error("PrivateKey should be nil when reusing existing key pair")
	}

	if keyPair2.PublicKey == nil {
		t.Error("PublicKey is nil when reusing existing key pair")
	}

	// Fingerprints should match
	fp1, _ := GetPublicKeyFingerprint(keyPair1.PublicKey)
	fp2, _ := GetPublicKeyFingerprint(keyPair2.PublicKey)
	if fp1 != fp2 {
		t.Errorf("Fingerprints don't match: %s vs %s", fp1, fp2)
	}
}

func TestInitializeOrLoadKeyPair_KeyRotate(t *testing.T) {
	tmpDir := t.TempDir()
	publicKeyPath := filepath.Join(tmpDir, "public_key.pem")
	privateKeyPath := filepath.Join(tmpDir, "private_key.pem")

	// Generate initial key
	keyPair1, _ := InitializeOrLoadKeyPair(publicKeyPath, privateKeyPath, false, false)
	fp1, _ := GetPublicKeyFingerprint(keyPair1.PublicKey)

	// Rotate — should generate new key pair
	keyPair2, err := InitializeOrLoadKeyPair(publicKeyPath, privateKeyPath, true, false)
	if err != nil {
		t.Fatalf("InitializeOrLoadKeyPair with KEY_ROTATE failed: %v", err)
	}

	if keyPair2.PrivateKey == nil {
		t.Error("PrivateKey should not be nil after key rotation (new generation)")
	}

	fp2, _ := GetPublicKeyFingerprint(keyPair2.PublicKey)
	if fp1 == fp2 {
		t.Error("Fingerprints should differ after key rotation")
	}
}

func TestLoadPublicKeyFromFile(t *testing.T) {
	keyPair, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test_pub.pem")

	err = SavePublicKeyToFile(keyPair.PublicKey, path)
	if err != nil {
		t.Fatalf("SavePublicKeyToFile failed: %v", err)
	}

	loaded, err := LoadPublicKeyFromFile(path)
	if err != nil {
		t.Fatalf("LoadPublicKeyFromFile failed: %v", err)
	}

	fp1, _ := GetPublicKeyFingerprint(keyPair.PublicKey)
	fp2, _ := GetPublicKeyFingerprint(loaded)
	if fp1 != fp2 {
		t.Error("Loaded key fingerprint doesn't match original")
	}
}

func TestSavePrivateKeyToFile(t *testing.T) {
	keyPair, err := GenerateKeyPair()
	if err != nil {
		t.Fatalf("GenerateKeyPair failed: %v", err)
	}

	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "test_priv.pem")

	err = SavePrivateKeyToFile(keyPair.PrivateKey, path)
	if err != nil {
		t.Fatalf("SavePrivateKeyToFile failed: %v", err)
	}

	if _, err := os.Stat(path); os.IsNotExist(err) {
		t.Error("Private key file was not created")
	}

	// Verify file has 0600 permissions
	info, _ := os.Stat(path)
	if info.Mode().Perm() != 0600 {
		t.Errorf("Private key file has wrong permissions: %o", info.Mode().Perm())
	}
}
