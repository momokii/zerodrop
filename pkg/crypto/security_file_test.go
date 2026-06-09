package crypto

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/zerodrop/terminal/pkg/config"
)

func TestSecurity_File_PublicKeyFile_Permissions0644(t *testing.T) {
	kp, err := GenerateKeyPair()
	if err != nil {
		t.Fatal(err)
	}

	tmpDir := t.TempDir()
	pubPath := filepath.Join(tmpDir, "public_key.pem")

	err = SavePublicKeyToFile(kp.PublicKey, pubPath)
	if err != nil {
		t.Fatal(err)
	}

	info, err := os.Stat(pubPath)
	if err != nil {
		t.Fatal(err)
	}

	perm := info.Mode().Perm()
	if perm != 0644 {
		t.Errorf("[SECURITY] Public key file permissions: expected 0644 (world-readable, owner-writable), got %04o", perm)
	}
	t.Logf("[SECURITY] Public key file is world-readable (0644) — safe to share: PASS")
}

func TestSecurity_Config_DefaultLoggingOff(t *testing.T) {
	cfg := config.DefaultConfig()
	if cfg.LogEnabled {
		t.Error("[SECURITY] Default config should have logging disabled — sensitive data could be written to disk")
	}
	t.Logf("[SECURITY] Logging is OFF by default (no sensitive data written to log files): PASS")
}

func TestSecurity_Config_DefaultRateLimitPositive(t *testing.T) {
	cfg := config.DefaultConfig()
	if cfg.RateLimitRequestsPerHour < 1 {
		t.Errorf("[SECURITY] Default rate limit must be >= 1, got %d — server would be unprotected!", cfg.RateLimitRequestsPerHour)
	}
	if cfg.RateLimitBurst < 1 {
		t.Errorf("[SECURITY] Default rate limit burst must be >= 1, got %d", cfg.RateLimitBurst)
	}
	t.Logf("[SECURITY] Rate limiting active by default (%d req/hr, burst %d): PASS", cfg.RateLimitRequestsPerHour, cfg.RateLimitBurst)
}

func TestSecurity_Config_DefaultSecureValues(t *testing.T) {
	cfg := config.DefaultConfig()

	if cfg.PrinterType != "mock" {
		t.Errorf("[SECURITY] Default printer type should be 'mock' (safe for dev), got %q", cfg.PrinterType)
	}

	if cfg.TLSEnabled {
		t.Error("[SECURITY] TLS should be off by default (opt-in, not forced)")
	}

	if cfg.KeyRotate {
		t.Error("[SECURITY] Key rotation should be off by default — rotating keys makes old ciphertext undecryptable")
	}

	if cfg.AdminToken != "" {
		t.Error("[SECURITY] Admin token should be empty by default — admin disabled until explicitly configured")
	}

	t.Logf("[SECURITY] All default config values are production-safe (mock printer, no TLS, no rotation, no admin token): PASS")
}

func TestSecurity_KeyReuse_SameFingerprint_NilPrivateKey(t *testing.T) {
	tmpDir := t.TempDir()
	pubPath := filepath.Join(tmpDir, "public_key.pem")
	privPath := filepath.Join(tmpDir, "private_key.pem")

	// First run: generate keys (simulates first boot)
	kp1, err := InitializeOrLoadKeyPair(pubPath, privPath, false, false)
	if err != nil {
		t.Fatal(err)
	}
	fp1, _ := GetPublicKeyFingerprint(kp1.PublicKey)

	// Burn the private key from memory (simulates what main.go does after QR display)
	BurnProtocol(kp1)

	// Second run: reload keys (simulates server restart)
	kp2, err := InitializeOrLoadKeyPair(pubPath, privPath, false, false)
	if err != nil {
		t.Fatal(err)
	}
	fp2, _ := GetPublicKeyFingerprint(kp2.PublicKey)

	// Public key fingerprint must be identical (same key pair)
	if fp1 != fp2 {
		t.Errorf("[SECURITY] Fingerprints differ after restart:\n  first:  %s\n  reload: %s — previously encrypted data is now undecryptable!", fp1, fp2)
	}

	// Private key must NOT be loaded into memory on restart
	if kp2.PrivateKey != nil {
		t.Error("[SECURITY] Private key was loaded into RAM on restart — server could decrypt payloads!")
	}

	t.Logf("[SECURITY] Server restart: same key pair (fingerprint matches), private key stays on disk (nil in RAM): PASS")
}
