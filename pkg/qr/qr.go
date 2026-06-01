// Package qr provides QR code generation and ESC/POS rasterization for thermal printers.
package qr

import (
	"encoding/base64"
	"fmt"

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

	// bytesPerRow = number of bytes needed to represent 'size' horizontal dots
	bytesPerRow := (size + 7) / 8

	// Build ESC/POS command sequence
	var escpos []byte

	// 1. Initialize printer
	escpos = append(escpos, 0x1B, 0x40)

	// 2. Set alignment to center
	escpos = append(escpos, 0x1B, 0x61, 0x01)

	// 3. Print header text
	header := "\nZERO DROP ENCRYPTED PAYLOAD\n\n"
	escpos = append(escpos, []byte(header)...)

	// 4. GS v 0 raster bit-image command
	// GS v 0 m=0 xL xH yL yH [data]
	gsCmd := []byte{0x1D, 0x76, 0x30, 0x00} // GS v 0, m=0 (normal density)
	// xL, xH = width in bytes (little-endian)
	gsCmd = append(gsCmd, byte(bytesPerRow&0xFF), byte((bytesPerRow>>8)&0xFF))
	// yL, yH = height in dots (little-endian)
	gsCmd = append(gsCmd, byte(size&0xFF), byte((size>>8)&0xFF))

	// Build image data: row-by-row, each byte encodes 8 horizontal pixels
	// MSB (bit 7) = leftmost pixel in that group of 8
	for y := 0; y < size; y++ {
		for xb := 0; xb < bytesPerRow; xb++ {
			var imgByte byte
			for bit := 0; bit < 8; bit++ {
				xPos := xb*8 + bit
				if xPos < size && bitmap[y][xPos] {
					imgByte |= 1 << (7 - bit) // MSB = leftmost pixel
				}
			}
			gsCmd = append(gsCmd, imgByte)
		}
	}
	escpos = append(escpos, gsCmd...)

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
