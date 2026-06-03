// Package qr provides QR code generation and ESC/POS rasterization for thermal printers.
package qr

import (
	"encoding/base64"
	"fmt"
	"time"

	qrcode "github.com/skip2/go-qrcode"
)

// GenerateQRESCPOS generates a QR code from ciphertext bytes and rasterizes it
// to ESC/POS GS v 0 bit-image commands suitable for 58mm thermal printers.
//
// Unlike native QR commands (GS ( k), GS v 0 raster is universally supported
// by all ESC/POS thermal printers including Zjiang, POS-5890, and Epson.
// The QR is generated server-side using go-qrcode and sent as a pixel bitmap.
//
// The QR content is formatted as "ZD1:" + base64(ciphertext) for forward compatibility.
//
// GS v 0 command format: GS v 0 m xL xH yL yH d1...dk
//   m = 0 (normal density)
//   xL, xH = width in bytes (little-endian) — each byte encodes 8 horizontal dots
//   yL, yH = height in dots (little-endian)
//   Data: row-by-row, each byte = 8 horizontal pixels, MSB = leftmost
func GenerateQRESCPOS(data []byte) ([]byte, error) {
	// Build QR content: ZD1: + base64(ciphertext)
	qrContent := "ZD1:" + base64.StdEncoding.EncodeToString(data)

	// Generate QR code bitmap using go-qrcode
	qr, err := qrcode.New(qrContent, qrcode.Medium)
	if err != nil {
		return nil, fmt.Errorf("failed to generate QR code: %w", err)
	}

	// bitmap[y][x] is true for black pixel, false for white
	bitmap := qr.Bitmap()
	size := len(bitmap)
	if size == 0 {
		return nil, fmt.Errorf("QR bitmap is empty")
	}

	// Scale factor: 4x increases QR from ~4.6mm to ~19mm for easy scanning
	const scale = 4
	scaledSize := size * scale

	// bytesPerRow for the scaled bitmap
	bytesPerRow := (scaledSize + 7) / 8

	// Build ESC/POS command sequence
	var escpos []byte

	// 1. Initialize printer (twice to handle residual buffer on cheap printers)
	escpos = append(escpos, 0x1B, 0x40)
	escpos = append(escpos, 0x1B, 0x40)

	// 2. Set alignment to center
	escpos = append(escpos, 0x1B, 0x61, 0x01)

	// 3. Print header text with UTC timestamp
	header := fmt.Sprintf("\nZERO DROP ENCRYPTED PAYLOAD\nPrinted: %s UTC\n\n",
		time.Now().UTC().Format("2006-01-02 15:04:05"))
	escpos = append(escpos, []byte(header)...)

	// 4. NUL padding to absorb buffer overflow from large GS v 0 raster
	for i := 0; i < 512; i++ {
		escpos = append(escpos, 0x00)
	}

	// 5. Flush text buffer by re-sending alignment before raster data
	escpos = append(escpos, 0x1B, 0x61, 0x01) // ESC a 1 (center)

	// 6. GS v 0 raster bit-image command (normal density)
	gsCmd := []byte{0x1D, 0x76, 0x30, 0x00} // GS v 0, m=0 (normal density)
	gsCmd = append(gsCmd, byte(bytesPerRow&0xFF), byte((bytesPerRow>>8)&0xFF))
	gsCmd = append(gsCmd, byte(scaledSize&0xFF), byte((scaledSize>>8)&0xFF))

	// Build image data: each source pixel becomes a 3x3 block
	for sy := 0; sy < size; sy++ {
		for ry := 0; ry < scale; ry++ {
			for xb := 0; xb < bytesPerRow; xb++ {
				var imgByte byte
				for bit := 0; bit < 8; bit++ {
					xPos := xb*8 + bit
					if xPos < scaledSize {
						sx := xPos / scale
						if sx < size && bitmap[sy][sx] {
							imgByte |= 1 << (7 - bit)
						}
					}
				}
				gsCmd = append(gsCmd, imgByte)
			}
		}
	}
	escpos = append(escpos, gsCmd...)

	// 7. Print ciphertext below QR
	escpos = append(escpos, []byte("\n\n--- CIPHERTEXT ---\n")...)
	payload := qrContent
	for i := 0; i < len(payload); i += 48 {
		end := i + 48
		if end > len(payload) {
			end = len(payload)
		}
		escpos = append(escpos, []byte(payload[i:end]+"\n")...)
	}

	// 8. Feed 5 lines and partial cut
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
