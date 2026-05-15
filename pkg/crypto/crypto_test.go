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

	// This should not panic, just log
	err = LogPrivateKeyAsQR(keyPair.PrivateKey)
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

	// First call should generate new key
	keyPair1, err := InitializeOrLoadKeyPair(publicKeyPath)
	if err != nil {
		t.Fatalf("InitializeOrLoadKeyPair failed: %v", err)
	}

	if keyPair1.PrivateKey == nil {
		t.Error("PrivateKey is nil after initialization")
	}

	// Verify public key file was created
	if _, err := os.Stat(publicKeyPath); os.IsNotExist(err) {
		t.Error("Public key file was not created")
	}
}
