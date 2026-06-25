package pages

import (
	"bytes"
	"encoding/binary"
	"errors"
	"image"
	"image/color"
	"image/color/palette"
	"image/gif"
	"image/jpeg"
	"image/png"
	"testing"
)

// --- WebP chunk surgery ---

func webpChunk(fourcc string, payload []byte) []byte {
	b := append([]byte(nil), fourcc...)
	var sz [4]byte
	binary.LittleEndian.PutUint32(sz[:], uint32(len(payload)))
	b = append(b, sz[:]...)
	b = append(b, payload...)
	if len(payload)%2 == 1 {
		b = append(b, 0) // even padding
	}
	return b
}

func buildWebP(chunks ...[]byte) []byte {
	var body bytes.Buffer
	for _, c := range chunks {
		body.Write(c)
	}
	out := append([]byte(nil), "RIFF"...)
	var sz [4]byte
	binary.LittleEndian.PutUint32(sz[:], uint32(4+body.Len()))
	out = append(out, sz[:]...)
	out = append(out, "WEBP"...)
	out = append(out, body.Bytes()...)
	return out
}

func webpChunkFourCCs(data []byte) []string {
	var ccs []string
	body := data[12:]
	for i := 0; i+8 <= len(body); {
		cc := string(body[i : i+4])
		size := int(binary.LittleEndian.Uint32(body[i+4 : i+8]))
		ccs = append(ccs, cc)
		i += 8 + size + (size & 1)
	}
	return ccs
}

func TestStripWebPMetadata(t *testing.T) {
	vp8x := make([]byte, 10)
	vp8x[0] = 0x0C // EXIF (bit3) + XMP (bit2) flags set
	in := buildWebP(
		webpChunk("VP8X", vp8x),
		webpChunk("VP8 ", []byte("IMAGEDATA")),
		webpChunk("EXIF", []byte("EXIF-GPS-SECRET")),
		webpChunk("XMP ", []byte("<xmp>SECRET</xmp>")),
	)

	out := stripWebPMetadata(in)

	if bytes.Contains(out, []byte("EXIF-GPS-SECRET")) || bytes.Contains(out, []byte("SECRET</xmp>")) {
		t.Fatal("metadata payload survived stripping")
	}
	if !bytes.Contains(out, []byte("IMAGEDATA")) {
		t.Fatal("image data chunk was lost")
	}
	ccs := webpChunkFourCCs(out)
	for _, cc := range ccs {
		if cc == "EXIF" || cc == "XMP " {
			t.Fatalf("metadata chunk %q not removed", cc)
		}
	}
	// VP8X flags byte (body offset 8 of first chunk) must have EXIF/XMP bits cleared.
	if out[12+8]&0x0C != 0 {
		t.Fatalf("VP8X EXIF/XMP flags not cleared: %02x", out[12+8])
	}
	// RIFF size header must match the actual remaining bytes.
	if got := binary.LittleEndian.Uint32(out[4:8]); int(got) != len(out)-8 {
		t.Fatalf("RIFF size %d != %d", got, len(out)-8)
	}
}

func TestStripWebPMetadata_NoMetadataUnchanged(t *testing.T) {
	in := buildWebP(webpChunk("VP8 ", []byte("IMAGEDATA")))
	if out := stripWebPMetadata(in); !bytes.Equal(out, in) {
		t.Fatal("webp without metadata should be byte-identical after strip")
	}
}

// --- JPEG ---

func TestStripJPEGMetadata(t *testing.T) {
	app1 := append([]byte{0xFF, 0xE1}, segLen(2+len("Exif\x00\x00EXIFSECRET"))...)
	app1 = append(app1, []byte("Exif\x00\x00EXIFSECRET")...)
	com := append([]byte{0xFF, 0xFE}, segLen(2+len("COMSECRET"))...)
	com = append(com, []byte("COMSECRET")...)
	dqt := append([]byte{0xFF, 0xDB}, segLen(2+len("DQTKEEP"))...)
	dqt = append(dqt, []byte("DQTKEEP")...)

	var in []byte
	in = append(in, 0xFF, 0xD8) // SOI
	in = append(in, app1...)
	in = append(in, com...)
	in = append(in, dqt...)
	in = append(in, 0xFF, 0xDA) // SOS
	in = append(in, []byte("SCANDATA")...)
	in = append(in, 0xFF, 0xD9) // EOI

	out := stripJPEGMetadata(in)
	if bytes.Contains(out, []byte("EXIFSECRET")) {
		t.Fatal("EXIF APP1 not stripped")
	}
	if bytes.Contains(out, []byte("COMSECRET")) {
		t.Fatal("COM comment not stripped")
	}
	if !bytes.Contains(out, []byte("DQTKEEP")) || !bytes.Contains(out, []byte("SCANDATA")) {
		t.Fatal("essential JPEG segments were dropped")
	}
}

func segLen(n int) []byte {
	b := make([]byte, 2)
	binary.BigEndian.PutUint16(b, uint16(n))
	return b
}

func TestStripJPEGMetadata_StillDecodes(t *testing.T) {
	// Real JPEG with an injected EXIF APP1 segment must still decode after strip.
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	img.Set(1, 1, color.RGBA{R: 200, G: 100, B: 50, A: 255})
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, nil); err != nil {
		t.Fatal(err)
	}
	enc := buf.Bytes()
	exif := append([]byte{0xFF, 0xE1}, segLen(2+len("Exif\x00\x00GPSDATA"))...)
	exif = append(exif, []byte("Exif\x00\x00GPSDATA")...)
	withExif := append([]byte{}, enc[:2]...) // SOI
	withExif = append(withExif, exif...)
	withExif = append(withExif, enc[2:]...)

	out := stripJPEGMetadata(withExif)
	if bytes.Contains(out, []byte("GPSDATA")) {
		t.Fatal("EXIF survived")
	}
	if _, err := jpeg.Decode(bytes.NewReader(out)); err != nil {
		t.Fatalf("stripped JPEG no longer decodes: %v", err)
	}
}

// --- PNG ---

func TestStripPNGMetadata_StillDecodes(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 8, 8))
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		t.Fatal(err)
	}
	enc := buf.Bytes()
	// Inject a tEXt chunk just before the trailing IEND chunk (last 12 bytes).
	text := pngChunk("tEXt", []byte("Comment\x00PNGSECRET"))
	withText := append([]byte{}, enc[:len(enc)-12]...)
	withText = append(withText, text...)
	withText = append(withText, enc[len(enc)-12:]...)

	out := stripPNGMetadata(withText)
	if bytes.Contains(out, []byte("PNGSECRET")) {
		t.Fatal("tEXt metadata survived strip")
	}
	if _, err := png.Decode(bytes.NewReader(out)); err != nil {
		t.Fatalf("stripped PNG no longer decodes: %v", err)
	}
}

func pngChunk(ctype string, payload []byte) []byte {
	b := make([]byte, 4)
	binary.BigEndian.PutUint32(b, uint32(len(payload)))
	b = append(b, ctype...)
	b = append(b, payload...)
	// CRC over type+payload
	crc := pngCRC(append([]byte(ctype), payload...))
	var c [4]byte
	binary.BigEndian.PutUint32(c[:], crc)
	return append(b, c[:]...)
}

func pngCRC(b []byte) uint32 {
	// minimal IEEE CRC32 (same as hash/crc32 IEEE)
	var table [256]uint32
	for i := range table {
		c := uint32(i)
		for k := 0; k < 8; k++ {
			if c&1 != 0 {
				c = 0xedb88320 ^ (c >> 1)
			} else {
				c >>= 1
			}
		}
		table[i] = c
	}
	crc := uint32(0xffffffff)
	for _, x := range b {
		crc = table[(crc^uint32(x))&0xff] ^ (crc >> 8)
	}
	return crc ^ 0xffffffff
}

// --- GIF animation ---

func buildGIF(frames int) []byte {
	g := &gif.GIF{}
	for i := 0; i < frames; i++ {
		pi := image.NewPaletted(image.Rect(0, 0, 4, 4), palette.Plan9)
		g.Image = append(g.Image, pi)
		g.Delay = append(g.Delay, 0)
	}
	var buf bytes.Buffer
	if err := gif.EncodeAll(&buf, g); err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func TestGIFFrameCount(t *testing.T) {
	if n, ok := gifFrameCount(buildGIF(1)); !ok || n != 1 {
		t.Fatalf("static gif: got (%d,%v), want (1,true)", n, ok)
	}
	if n, ok := gifFrameCount(buildGIF(3)); !ok || n != 3 {
		t.Fatalf("animated gif: got (%d,%v), want (3,true)", n, ok)
	}
	if isAnimatedGIF(buildGIF(1)) {
		t.Fatal("static gif reported as animated")
	}
	if !isAnimatedGIF(buildGIF(2)) {
		t.Fatal("animated gif not detected")
	}
}

func TestConvertToWebP_RejectsAnimatedGIF(t *testing.T) {
	out, _, _, err := convertToWebP(buildGIF(3), "loop.gif")
	if !errors.Is(err, errAnimatedGIF) {
		t.Fatalf("expected errAnimatedGIF, got %v", err)
	}
	if out != nil {
		t.Fatalf("expected nil data for rejected animated gif, got %d bytes", len(out))
	}
}

func TestConvertToWebP_StripsWebPPassthroughMetadata(t *testing.T) {
	// The key regression: a WebP upload is passthrough (not re-encoded), so it
	// must be stripped inline or its EXIF/GPS leaks to the public.
	vp8x := make([]byte, 10)
	vp8x[0] = 0x08 // EXIF flag
	in := buildWebP(
		webpChunk("VP8X", vp8x),
		webpChunk("VP8 ", []byte("PIXELS")),
		webpChunk("EXIF", []byte("LAT-LONG-LEAK")),
	)
	out, filename, ct, err := convertToWebP(in, "photo.webp")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if bytes.Contains(out, []byte("LAT-LONG-LEAK")) {
		t.Fatal("EXIF metadata leaked through the WebP passthrough")
	}
	if filename != "photo.webp" || ct != "image/webp" {
		t.Fatalf("unexpected filename/ct: %q %q", filename, ct)
	}
}
