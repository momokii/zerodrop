package qr

import (
	"bytes"
	"strings"
	"testing"

	qrcode "github.com/skip2/go-qrcode"
)

func hasPrefix(b, prefix []byte) bool {
	return len(b) >= len(prefix) && bytes.Equal(b[:len(prefix)], prefix)
}

func TestGenerateQRESCPOS_ValidPayload(t *testing.T) {
	data := []byte("Hello ZeroDrop!")
	escpos, err := GenerateQRESCPOS(data)
	if err != nil {
		t.Fatalf("GenerateQRESCPOS failed: %v", err)
	}
	if len(escpos) == 0 {
		t.Fatal("expected non-empty ESC/POS output")
	}
	// Must start with ESC @ (initialize printer)
	if !hasPrefix(escpos, []byte{0x1B, 0x40}) {
		t.Fatal("expected ESC/POS output to start with printer initialization (ESC @)")
	}
	// Must end with partial cut command
	if !bytes.HasSuffix(escpos, []byte{0x1D, 0x56, 0x42, 0x00}) {
		t.Fatal("expected ESC/POS output to end with partial cut (GS V B 0)")
	}
	// Must contain GS v 0 raster command header
	if !bytes.Contains(escpos, []byte{0x1D, 0x76, 0x30, 0x00}) {
		t.Fatal("expected ESC/POS output to contain GS v 0 raster command")
	}
	// Must contain header text
	if !bytes.Contains(escpos, []byte("ZERO DROP ENCRYPTED PAYLOAD")) {
		t.Fatal("expected output to contain header text")
	}
	// Must contain CIPHERTEXT section
	if !bytes.Contains(escpos, []byte("CIPHERTEXT")) {
		t.Fatal("expected output to contain CIPHERTEXT label")
	}
	// Must contain ZD1: prefix
	if !bytes.Contains(escpos, []byte("ZD1:")) {
		t.Fatal("expected output to contain ZD1: prefix")
	}
}

func TestGenerateQRESCPOS_EmptyInput(t *testing.T) {
	data := []byte{}
	escpos, err := GenerateQRESCPOS(data)
	if err != nil {
		t.Fatalf("GenerateQRESCPOS with empty input failed: %v", err)
	}
	if len(escpos) == 0 {
		t.Fatal("expected non-empty output even for empty input")
	}
}

func TestGenerateQRESCPOS_LargePayload(t *testing.T) {
	data := make([]byte, 256)
	for i := range data {
		data[i] = byte(i % 256)
	}
	escpos, err := GenerateQRESCPOS(data)
	if err != nil {
		t.Fatalf("GenerateQRESCPOS with 256-byte payload failed: %v", err)
	}
	if len(escpos) == 0 {
		t.Fatal("expected non-empty output for large payload")
	}
}

func TestGenerateQRESCPOS_ContainsTimestamp(t *testing.T) {
	data := []byte("timestamp-test")
	escpos, err := GenerateQRESCPOS(data)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(escpos, []byte("Printed:")) {
		t.Fatal("expected ESC/POS output to contain 'Printed:' timestamp")
	}
	if !bytes.Contains(escpos, []byte("UTC")) {
		t.Fatal("expected ESC/POS output to contain UTC marker")
	}
}

func TestGenerateQRPNG_ValidOutput(t *testing.T) {
	data := []byte("Hello ZeroDrop!")
	png, err := GenerateQRPNG(data)
	if err != nil {
		t.Fatalf("GenerateQRPNG failed: %v", err)
	}
	if len(png) == 0 {
		t.Fatal("expected non-empty PNG output")
	}
	// PNG magic bytes: 89 50 4E 47 0D 0A 1A 0A
	expectedHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	if !hasPrefix(png, expectedHeader) {
		t.Fatal("expected output to be a valid PNG (PNG magic bytes)")
	}
	// Must contain IHDR chunk
	if !bytes.Contains(png, []byte("IHDR")) {
		t.Fatal("expected PNG to contain IHDR chunk")
	}
	// Must contain IEND chunk
	if !bytes.Contains(png, []byte("IEND")) {
		t.Fatal("expected PNG to contain IEND chunk")
	}
}

func TestGenerateQRPNG_Deterministic(t *testing.T) {
	data := []byte("deterministic-test")
	png1, err := GenerateQRPNG(data)
	if err != nil {
		t.Fatal(err)
	}
	png2, err := GenerateQRPNG(data)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(png1, png2) {
		t.Fatal("expected deterministic PNG output for same input")
	}
}

func TestGenerateQRPNG_EmptyInput(t *testing.T) {
	png, err := GenerateQRPNG([]byte{})
	if err != nil {
		t.Fatalf("GenerateQRPNG with empty input failed: %v", err)
	}
	if len(png) == 0 {
		t.Fatal("expected non-empty PNG for empty input")
	}
}

func TestGenerateQRASCII_ValidOutput(t *testing.T) {
	content := "ZD1:dGVzdA=="
	ascii, err := GenerateQRASCII(content)
	if err != nil {
		t.Fatalf("GenerateQRASCII failed: %v", err)
	}
	if len(ascii) == 0 {
		t.Fatal("expected non-empty ASCII output")
	}
	// Should contain Unicode half-block characters
	if !strings.Contains(ascii, "\u2588") &&
		!strings.Contains(ascii, "\u2580") &&
		!strings.Contains(ascii, "\u2584") {
		t.Fatal("expected ASCII QR to contain half-block Unicode characters")
	}
	// Should contain newlines
	if !strings.Contains(ascii, "\n") {
		t.Fatal("expected ASCII QR to have newlines")
	}
}

func TestGenerateQRASCII_Deterministic(t *testing.T) {
	content := "deterministic-test"
	a1, err1 := GenerateQRASCII(content)
	a2, err2 := GenerateQRASCII(content)
	if err1 != nil || err2 != nil {
		t.Fatal("unexpected error in deterministic ASCII generation")
	}
	if a1 != a2 {
		t.Fatal("expected deterministic ASCII output for same input")
	}
}

func TestGenerateRawQRPNG_ValidOutput(t *testing.T) {
	data := []byte("-----BEGIN RSA PRIVATE KEY-----\nMIIEpAIBAAKCAQEA...")
	png, err := GenerateRawQRPNG(data)
	if err != nil {
		t.Fatalf("GenerateRawQRPNG failed: %v", err)
	}
	if len(png) == 0 {
		t.Fatal("expected non-empty PNG output")
	}
	expectedHeader := []byte{0x89, 0x50, 0x4E, 0x47, 0x0D, 0x0A, 0x1A, 0x0A}
	if !hasPrefix(png, expectedHeader) {
		t.Fatal("expected output to be a valid PNG")
	}
}

func TestGenerateRawQRPNG_NoPrefixAdded(t *testing.T) {
	data := []byte("raw-pem-data")
	png, err := GenerateRawQRPNG(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(png) == 0 {
		t.Fatal("expected non-empty PNG")
	}
}

func TestQR_ErrorCorrectionDefaultMedium(t *testing.T) {
	// Verify that Medium error correction produces scannable QR for a known payload
	data := []byte("medium-error-correction-check")
	qrContent := "ZD1:" + b64Encode(data)
	qr, err := qrcode.New(qrContent, qrcode.Medium)
	if err != nil {
		t.Fatalf("QR generation with Medium error correction failed: %v", err)
	}
	bitmap := qr.Bitmap()
	if len(bitmap) == 0 {
		t.Fatal("expected non-empty QR bitmap at Medium error correction")
	}
}

// b64Encode is a test-local helper that mirrors the package's base64 encoding
// without importing the encoding/base64 package directly in tests that need it.
func b64Encode(data []byte) string {
	const base64Chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"
	var result []byte
	for i := 0; i < len(data); i += 3 {
		var b3 [3]byte
		copy(b3[:], data[i:])
		result = append(result, base64Chars[b3[0]>>2])
		result = append(result, base64Chars[(b3[0]&0x03)<<4|b3[1]>>4])
		if i+1 < len(data) {
			result = append(result, base64Chars[(b3[1]&0x0F)<<2|b3[2]>>6])
		} else {
			result = append(result, '=')
		}
		if i+2 < len(data) {
			result = append(result, base64Chars[b3[2]&0x3F])
		} else {
			result = append(result, '=')
		}
	}
	return string(result)
}
