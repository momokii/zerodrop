// Package qr provides QR code generation and ESC/POS rasterization for thermal printers.
package qr

import (
	"encoding/base64"
	"fmt"

	qrcode "github.com/skip2/go-qrcode"
)

// GenerateQRESCPOS generates a QR code from ciphertext bytes and rasterizes it
// to ESC/POS GS v 0 bit-image commands suitable for 58mm thermal printers.
// The QR content is formatted as "ZD1:" + base64(ciphertext) for forward compatibility.
func GenerateQRESCPOS(data []byte) ([]byte, error) {
	// Build QR content: ZD1: + base64(ciphertext)
	qrContent := "ZD1:" + base64.StdEncoding.EncodeToString(data)

	// Generate QR code with Medium error correction
	qr, err := qrcode.New(qrContent, qrcode.Medium)
	if err != nil {
		return nil, fmt.Errorf("failed to generate QR code: %w", err)
	}

	// Get bitmap from QR code (true = black, false = white)
	bitmap := qr.Bitmap()
	size := len(bitmap) // bitmap is square

	if size == 0 {
		return nil, fmt.Errorf("QR bitmap is empty")
	}

	// Build ESC/POS command sequence
	var escpos []byte

	// 1. Initialize printer
	escpos = append(escpos, []byte{0x1B, 0x40}...)

	// 2. Set alignment to center
	escpos = append(escpos, []byte{0x1B, 0x61, 0x01}...)

	// 3. Print header text
	header := "\nZERO DROP ENCRYPTED PAYLOAD\n\n"
	escpos = append(escpos, []byte(header)...)

	// 4. Generate GS v 0 raster bit-image commands
	// GS v 0 m xL xH yL yH d1...dk
	// m = 48 (normal density)
	// xL, xH = width in bytes (little-endian), each byte = 8 vertical pixels
	// yL, yH = height in dots (little-endian)
	// d1...dk = image data organized in vertical columns

	// Calculate width in bytes (each byte encodes 8 vertical pixels)
	widthBytes := (size + 7) / 8

	// For each horizontal slice of 8 rows (or less for last slice)
	for rowStart := 0; rowStart < size; rowStart += 8 {
		rowsInSlice := 8
		if rowStart+rowsInSlice > size {
			rowsInSlice = size - rowStart
		}

		// Build GS v 0 command for this slice
		sliceCmd := []byte{0x1D, 0x76, 0x30, 0x00} // GS v 0, m=0 (normal density)
		// xL, xH
		sliceCmd = append(sliceCmd, byte(widthBytes&0xFF), byte((widthBytes>>8)&0xFF))
		// yL, yH
		sliceCmd = append(sliceCmd, byte(size&0xFF), byte((size>>8)&0xFF))

		// Build image data: for each column, encode rowsInSlice vertical bits into widthBytes bytes
		// Each byte = 8 horizontal pixels (MSB = leftmost pixel in that column's group of 8 rows)
		for col := 0; col < size; col++ {
			// For each byte in this column (covering rowsInSlice vertical rows)
			for byteIdx := 0; byteIdx < widthBytes; byteIdx++ {
				var imgByte byte
				for bit := 0; bit < 8; bit++ {
					row := byteIdx*8 + bit
					if row < size && bitmap[row][col] {
						imgByte |= (1 << (7 - bit)) // MSB = top pixel
					}
				}
				sliceCmd = append(sliceCmd, imgByte)
			}
		}

		escpos = append(escpos, sliceCmd...)
	}

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
	escpos = append(escpos, []byte{0x1B, 0x64, 0x05}...) // Feed 5 lines
	escpos = append(escpos, []byte{0x1D, 0x56, 0x42, 0x00}...) // Partial cut

	return escpos, nil
}

// GenerateQRPNG generates a QR code PNG image from the given data.
// The QR content is formatted as "ZD1:" + base64(data) for consistency.
func GenerateQRPNG(data []byte) ([]byte, error) {
	qrContent := "ZD1:" + base64.StdEncoding.EncodeToString(data)

	png, err := qrcode.Encode(qrContent, qrcode.Medium, 256)
	if err != nil {
		return nil, fmt.Errorf("failed to generate QR PNG: %w", err)
	}

	return png, nil
}
