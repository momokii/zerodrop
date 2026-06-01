// Package qr provides QR code generation and ESC/POS rasterization for thermal printers.
package qr

import (
	"encoding/base64"
	"fmt"

	qrcode "github.com/skip2/go-qrcode"
)

// GenerateQRESCPOS generates a QR code from ciphertext bytes and produces
// ESC/POS commands suitable for 58mm thermal printers.
//
// It uses native ESC/POS QR code commands (GS ( k) which are supported by
// most thermal printers including Zjiang, POS-5890, Epson, and XPrinter.
// These commands are more reliable than GS v 0 raster bit-image because
// the printer handles QR encoding internally — no manual pixel rasterization.
//
// The QR content is formatted as "ZD1:" + base64(ciphertext) for forward compatibility.
func GenerateQRESCPOS(data []byte) ([]byte, error) {
	// Build QR content: ZD1: + base64(ciphertext)
	qrContent := "ZD1:" + base64.StdEncoding.EncodeToString(data)
	qrData := []byte(qrContent)

	// Validate content is non-empty
	if len(qrData) == 0 {
		return nil, fmt.Errorf("QR content is empty")
	}

	// Build ESC/POS command sequence
	var escpos []byte

	// 1. Initialize printer
	escpos = append(escpos, 0x1B, 0x40)

	// 2. Set alignment to center
	escpos = append(escpos, 0x1B, 0x61, 0x01)

	// 3. Print header text
	header := "\nZERO DROP ENCRYPTED PAYLOAD\n\n"
	escpos = append(escpos, []byte(header)...)

	// 4. Generate native ESC/POS QR code commands (GS ( k)
	// This is the standard ESC/POS QR code approach — the printer renders the
	// QR internally, avoiding pixel-level rasterization issues.
	//
	// Command format: GS ( k pL pH cn fn [params...]
	//   cn = 49 (0x31) = QR Code
	//   pL, pH = parameter length (little-endian)

	// 4a. Select QR model 2
	//   fn = 65 (0x41) = select model
	//   params: n1=50 (model 2), n2=0
	escpos = append(escpos, 0x1D, 0x28, 0x6B, 0x04, 0x00, 0x31, 0x41, 0x32, 0x00)

	// 4b. Set module (dot) size
	//   fn = 67 (0x43) = set module size
	//   params: n = module size in dots (1-16)
	//
	// Size 6 produces a ~160-220 dot wide QR (fits 58mm paper at 384 dot width).
	// For longer content (more ciphertext bytes), a smaller module size may be
	// needed so the QR fits within page width. Content up to ~535 chars (ZD1: +
	// base64 of 400 ciphertext bytes) with module 6 yields QR version ~10-12,
	// which fits 58mm paper.
	moduleSize := byte(6)
	escpos = append(escpos, 0x1D, 0x28, 0x6B, 0x03, 0x00, 0x31, 0x43, moduleSize)

	// 4c. Store QR data in printer buffer
	//   fn = 80 (0x50) = store data
	//   params: m=48 (store in symbol storage), then raw data bytes
	//   pL = len(data) + 3 (3 = 1 byte m + 2 bytes for the length header itself)
	dataLen := len(qrData)
	pl := dataLen + 3
	escpos = append(escpos, 0x1D, 0x28, 0x6B, byte(pl&0xFF), byte((pl>>8)&0xFF), 0x31, 0x50, 0x30)
	escpos = append(escpos, qrData...)

	// 4d. Print the stored QR code
	//   fn = 81 (0x51) = print QR
	//   params: m=48
	escpos = append(escpos, 0x1D, 0x28, 0x6B, 0x03, 0x00, 0x31, 0x51, 0x30)

	// 5. Print ciphertext as readable text below QR (fallback)
	escpos = append(escpos, []byte("\n\n--- CIPHERTEXT ---\n")...)
	payload := string(data)
	for i := 0; i < len(payload); i += 48 {
		end := i + 48
		if end > len(payload) {
			end = len(payload)
		}
		escpos = append(escpos, []byte(payload[i:end]+"\n")...)
	}

	// 6. Feed 5 lines and partial cut
	escpos = append(escpos, 0x1B, 0x64, 0x05)       // Feed 5 lines
	escpos = append(escpos, 0x1D, 0x56, 0x42, 0x00) // Partial cut

	return escpos, nil
}

// GenerateQRPNG generates a QR code PNG image from the given data.
// The QR content is formatted as "ZD1:" + base64(data) — suitable for payload ciphertexts.
func GenerateQRPNG(data []byte) ([]byte, error) {
	qrContent := "ZD1:" + base64.StdEncoding.EncodeToString(data)

	png, err := qrcode.Encode(qrContent, qrcode.Medium, 256)
	if err != nil {
		return nil, fmt.Errorf("failed to generate QR PNG: %w", err)
	}

	return png, nil
}

// GenerateQRASCII generates an ASCII art representation of a QR code suitable
// for terminal display. Uses the go-qrcode ToString() which renders with
// unicode block characters (██ and spaces) for clear visual scanning.
func GenerateQRASCII(content string) (string, error) {
	qr, err := qrcode.New(content, qrcode.Medium)
	if err != nil {
		return "", fmt.Errorf("failed to generate QR code for ASCII: %w", err)
	}

	return qr.ToString(false), nil
}

// GenerateRawQRPNG generates a QR code PNG image from raw data.
// Unlike GenerateQRPNG, the data is encoded as-is without any prefix or base64 wrapping.
// This is suitable for PEM private keys and other self-describing content.
func GenerateRawQRPNG(data []byte) ([]byte, error) {
	png, err := qrcode.Encode(string(data), qrcode.Medium, 256)
	if err != nil {
		return nil, fmt.Errorf("failed to generate raw QR PNG: %w", err)
	}

	return png, nil
}
