package schematic

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"fmt"
	"io"
)

// decompress sniffs gzip/zlib magic bytes and returns the decompressed
// payload, capped at MaxDecompressedSize. Uncompressed data passes through
// (also capped).
func decompress(data []byte) ([]byte, error) {
	var r io.Reader = bytes.NewReader(data)
	switch {
	case len(data) >= 2 && data[0] == 0x1f && data[1] == 0x8b:
		gz, err := gzip.NewReader(r)
		if err != nil {
			return nil, fmt.Errorf("schematic: bad gzip stream: %w", err)
		}
		defer gz.Close()
		r = gz
	case len(data) >= 2 && data[0] == 0x78 && (data[1] == 0x01 || data[1] == 0x9c || data[1] == 0xda):
		zr, err := zlib.NewReader(r)
		if err != nil {
			return nil, fmt.Errorf("schematic: bad zlib stream: %w", err)
		}
		defer zr.Close()
		r = zr
	}
	out, err := io.ReadAll(io.LimitReader(r, MaxDecompressedSize+1))
	if err != nil {
		return nil, fmt.Errorf("schematic: decompress: %w", err)
	}
	if len(out) > MaxDecompressedSize {
		return nil, fmt.Errorf("schematic: decompressed payload exceeds %d bytes", MaxDecompressedSize)
	}
	if err := validateNBTLimits(out); err != nil {
		return nil, err
	}
	return out, nil
}

// gzipBytes compresses data with gzip (the framing every Minecraft tool
// expects for NBT files on disk).
func gzipBytes(data []byte) ([]byte, error) {
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	if _, err := gz.Write(data); err != nil {
		return nil, err
	}
	if err := gz.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
