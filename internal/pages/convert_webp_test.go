package pages

import (
	"bytes"
	"encoding/binary"
	"errors"
	"hash/crc32"
	"image"
	"image/color"
	"image/png"
	"strings"
	"testing"
)

// minimalPNGHeader builds a valid PNG signature + IHDR chunk declaring the given
// dimensions. image.DecodeConfig only reads through IHDR, so this is enough to
// exercise the dimension guard without allocating a real bitmap.
func minimalPNGHeader(width, height int) []byte {
	var buf bytes.Buffer
	buf.Write([]byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a})

	ihdr := make([]byte, 13)
	binary.BigEndian.PutUint32(ihdr[0:4], uint32(width))
	binary.BigEndian.PutUint32(ihdr[4:8], uint32(height))
	ihdr[8] = 8 // bit depth
	ihdr[9] = 6 // color type: RGBA
	// remaining bytes (compression, filter, interlace) are 0

	chunkType := []byte("IHDR")
	length := make([]byte, 4)
	binary.BigEndian.PutUint32(length, uint32(len(ihdr)))
	buf.Write(length)
	buf.Write(chunkType)
	buf.Write(ihdr)

	crc := crc32.NewIEEE()
	crc.Write(chunkType)
	crc.Write(ihdr)
	crcBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(crcBytes, crc.Sum32())
	buf.Write(crcBytes)

	return buf.Bytes()
}

func TestConvertToWebP_RejectsDecompressionBomb(t *testing.T) {
	// 10000x10000 = 100 MP, above maxDecodePixels (60 MP).
	data := minimalPNGHeader(10000, 10000)

	out, _, _, err := convertToWebP(data, "bomb.png")
	if !errors.Is(err, errImageTooLarge) {
		t.Fatalf("expected errImageTooLarge for oversized image, got err=%v", err)
	}
	if out != nil {
		t.Fatalf("expected nil data when rejecting oversized image, got %d bytes", len(out))
	}
}

func TestConvertToWebP_ConvertsNormalImage(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 64, 64))
	img.Set(0, 0, color.RGBA{R: 255, A: 255})
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, img); err != nil {
		t.Fatalf("failed to encode test png: %v", err)
	}

	out, filename, contentType, err := convertToWebP(encoded.Bytes(), "shot.png")
	if err != nil {
		t.Fatalf("unexpected error converting normal image: %v", err)
	}
	if len(out) == 0 {
		t.Fatal("expected converted data, got empty")
	}
	if contentType != "image/webp" {
		t.Fatalf("expected image/webp content type, got %q", contentType)
	}
	if !strings.HasSuffix(filename, ".webp") {
		t.Fatalf("expected .webp filename, got %q", filename)
	}
}

func TestConvertToWebP_PassesThroughWebP(t *testing.T) {
	// A .webp file should be returned unchanged (we don't re-encode it).
	data := []byte("RIFF....WEBPVP8 fake")
	out, filename, _, err := convertToWebP(data, "already.webp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !bytes.Equal(out, data) {
		t.Fatal("expected webp data to pass through unchanged")
	}
	if filename != "already.webp" {
		t.Fatalf("expected filename unchanged, got %q", filename)
	}
}
