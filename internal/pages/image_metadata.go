package pages

import (
	"bytes"
	"encoding/binary"
	"errors"
)

// errAnimatedGIF is returned by convertToWebP when an uploaded GIF has more than
// one frame. Animated GIFs are not allowed.
var errAnimatedGIF = errors.New("animated GIFs are not allowed")

// stripImageMetadata removes embedded metadata (EXIF, XMP, GPS coordinates,
// textual comments) from an encoded image before it is stored, so a user's
// original camera/phone metadata is never persisted or served. It is format
// aware and lenient: it parses the container, drops the metadata segments, and
// copies all image-bearing data through verbatim. Inputs it does not recognize
// or cannot safely parse are returned unchanged.
//
// JPEG and PNG are also re-encoded downstream (which strips metadata on its
// own), but stripping here keeps the WebP passthrough — the one path that is
// NOT re-encoded — safe as well.
func stripImageMetadata(data []byte) []byte {
	switch {
	case len(data) >= 3 && data[0] == 0xFF && data[1] == 0xD8 && data[2] == 0xFF:
		return stripJPEGMetadata(data)
	case len(data) >= 8 && bytes.Equal(data[:8], pngSignature):
		return stripPNGMetadata(data)
	case len(data) >= 12 && bytes.Equal(data[:4], []byte("RIFF")) && bytes.Equal(data[8:12], []byte("WEBP")):
		return stripWebPMetadata(data)
	default:
		return data
	}
}

var pngSignature = []byte{0x89, 'P', 'N', 'G', 0x0d, 0x0a, 0x1a, 0x0a}

// stripJPEGMetadata drops APPn (0xFFE0–0xFFEF, which hold EXIF/JFIF/XMP/ICC) and
// COM (0xFFFE) comment segments. All other segments and the entropy-coded scan
// data are preserved.
func stripJPEGMetadata(data []byte) []byte {
	if len(data) < 2 || data[0] != 0xFF || data[1] != 0xD8 {
		return data
	}
	out := make([]byte, 0, len(data))
	out = append(out, 0xFF, 0xD8) // SOI
	i := 2
	for i+1 < len(data) {
		if data[i] != 0xFF {
			out = append(out, data[i:]...)
			return out
		}
		// Skip any 0xFF fill bytes; the marker is the first non-0xFF byte.
		j := i + 1
		for j < len(data) && data[j] == 0xFF {
			j++
		}
		if j >= len(data) {
			out = append(out, data[i:]...)
			return out
		}
		marker := data[j]
		switch {
		case marker == 0x01 || (marker >= 0xD0 && marker <= 0xD7):
			// Standalone markers (TEM, RSTn) carry no length.
			out = append(out, 0xFF, marker)
			i = j + 1
			continue
		case marker == 0xD9: // EOI
			out = append(out, 0xFF, 0xD9)
			return out
		case marker == 0xDA: // SOS — copy the scan header and all entropy data verbatim.
			out = append(out, data[i:]...)
			return out
		}
		if j+3 > len(data) {
			out = append(out, data[i:]...)
			return out
		}
		segLen := int(binary.BigEndian.Uint16(data[j+1 : j+3]))
		segEnd := j + 1 + segLen
		if segLen < 2 || segEnd > len(data) {
			out = append(out, data[i:]...)
			return out
		}
		if (marker >= 0xE0 && marker <= 0xEF) || marker == 0xFE {
			i = segEnd // drop APPn / COM
			continue
		}
		out = append(out, 0xFF, marker)
		out = append(out, data[j+1:segEnd]...)
		i = segEnd
	}
	out = append(out, data[i:]...)
	return out
}

// pngMetadataChunks are the ancillary chunks that carry text/EXIF/timestamps.
var pngMetadataChunks = map[string]bool{
	"tEXt": true, "zTXt": true, "iTXt": true, "eXIf": true, "tIME": true,
}

// stripPNGMetadata drops textual, EXIF, and timestamp chunks while keeping all
// critical and rendering chunks (IHDR, PLTE, IDAT, IEND, tRNS, iCCP, gAMA, …).
func stripPNGMetadata(data []byte) []byte {
	if len(data) < 8 || !bytes.Equal(data[:8], pngSignature) {
		return data
	}
	out := make([]byte, 0, len(data))
	out = append(out, pngSignature...)
	i := 8
	for i+8 <= len(data) {
		length := int(binary.BigEndian.Uint32(data[i : i+4]))
		ctype := string(data[i+4 : i+8])
		end := i + 12 + length // length(4) + type(4) + data + crc(4)
		if length < 0 || end < i || end > len(data) {
			out = append(out, data[i:]...)
			return out
		}
		if !pngMetadataChunks[ctype] {
			out = append(out, data[i:end]...)
		}
		i = end
		if ctype == "IEND" {
			return out
		}
	}
	if i < len(data) {
		out = append(out, data[i:]...)
	}
	return out
}

// stripWebPMetadata drops the EXIF and "XMP " RIFF chunks and clears the
// corresponding flags in the VP8X header, preserving image and animation data
// (VP8/VP8L/VP8X/ANIM/ANMF/ALPH/ICCP). On any structural anomaly it returns the
// input unchanged.
func stripWebPMetadata(data []byte) []byte {
	if len(data) < 12 || !bytes.Equal(data[:4], []byte("RIFF")) || !bytes.Equal(data[8:12], []byte("WEBP")) {
		return data
	}
	body := data[12:]
	var kept [][]byte
	vp8xIdx := -1
	i := 0
	for i+8 <= len(body) {
		fourcc := string(body[i : i+4])
		size := int(binary.LittleEndian.Uint32(body[i+4 : i+8]))
		if size < 0 {
			return data
		}
		end := i + 8 + size
		padded := end + (size & 1) // chunks are padded to an even size
		if end < i || end > len(body) || padded > len(body) {
			return data
		}
		seg := body[i:padded]
		if fourcc == "EXIF" || fourcc == "XMP " {
			// drop
		} else {
			if fourcc == "VP8X" {
				vp8xIdx = len(kept)
			}
			kept = append(kept, seg)
		}
		i = padded
	}
	if i != len(body) {
		return data // trailing bytes we don't understand — leave as-is
	}
	// Clear the EXIF (bit 3) and XMP (bit 2) flags in the VP8X flags byte
	// (payload byte 0 == chunk byte 8).
	if vp8xIdx >= 0 && len(kept[vp8xIdx]) >= 9 {
		c := append([]byte(nil), kept[vp8xIdx]...)
		c[8] &^= 0x0C
		kept[vp8xIdx] = c
	}
	var bodyOut bytes.Buffer
	for _, c := range kept {
		bodyOut.Write(c)
	}
	out := make([]byte, 0, 12+bodyOut.Len())
	out = append(out, 'R', 'I', 'F', 'F')
	var sz [4]byte
	binary.LittleEndian.PutUint32(sz[:], uint32(4+bodyOut.Len()))
	out = append(out, sz[:]...)
	out = append(out, 'W', 'E', 'B', 'P')
	out = append(out, bodyOut.Bytes()...)
	return out
}

// isAnimatedGIF reports whether the GIF contains more than one frame. It walks
// the block structure without decoding pixels, so it is safe against
// decompression bombs.
func isAnimatedGIF(data []byte) bool {
	n, ok := gifFrameCount(data)
	return ok && n > 1
}

// gifFrameCount counts image descriptors (frames) by walking GIF blocks. The
// bool is false when the structure can't be parsed.
func gifFrameCount(data []byte) (int, bool) {
	if len(data) < 13 || (!bytes.HasPrefix(data, []byte("GIF87a")) && !bytes.HasPrefix(data, []byte("GIF89a"))) {
		return 0, false
	}
	i := 13
	if packed := data[10]; packed&0x80 != 0 { // global color table present
		i += 3 * (1 << ((packed & 0x07) + 1))
	}
	frames := 0
	for i < len(data) {
		switch data[i] {
		case 0x3B: // trailer
			return frames, true
		case 0x21: // extension introducer: skip label + sub-blocks
			i += 2
			var ok bool
			if i, ok = skipGIFSubBlocks(data, i); !ok {
				return frames, false
			}
		case 0x2C: // image descriptor
			frames++
			if i+10 > len(data) {
				return frames, false
			}
			lp := data[i+9]
			i += 10
			if lp&0x80 != 0 { // local color table
				i += 3 * (1 << ((lp & 0x07) + 1))
			}
			if i >= len(data) {
				return frames, false
			}
			i++ // LZW minimum code size
			var ok bool
			if i, ok = skipGIFSubBlocks(data, i); !ok {
				return frames, false
			}
		default:
			return frames, false
		}
	}
	return frames, false
}

// skipGIFSubBlocks advances past a chain of GIF sub-blocks (each a length byte
// followed by that many bytes, terminated by a zero-length block).
func skipGIFSubBlocks(data []byte, i int) (int, bool) {
	for i < len(data) {
		sz := int(data[i])
		i++
		if sz == 0 {
			return i, true
		}
		i += sz
		if i > len(data) {
			return i, false
		}
	}
	return i, false
}
