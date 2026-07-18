package schematic

import (
	"bytes"
	"encoding/base64"
	"encoding/binary"
	"fmt"
	"io"
	"sort"
	"strings"

	"github.com/Tnze/go-mc/nbt"
)

// ParseStateString parses the canonical blockstate string form used by
// Sponge .schem palettes: "minecraft:oak_stairs[facing=east,half=bottom]".
func ParseStateString(s string) (BlockState, error) {
	if len(s) == 0 || len(s) > MaxBlockIDLength+1024 {
		return BlockState{}, fmt.Errorf("schematic: invalid state string length")
	}
	open := strings.IndexByte(s, '[')
	if open < 0 {
		return BlockState{Name: s}, nil
	}
	if !strings.HasSuffix(s, "]") {
		return BlockState{}, fmt.Errorf("schematic: unterminated state string %q", s)
	}
	name := s[:open]
	if name == "" {
		return BlockState{}, fmt.Errorf("schematic: state string with empty block name")
	}
	body := s[open+1 : len(s)-1]
	props := map[string]string{}
	if body != "" {
		for _, kv := range strings.Split(body, ",") {
			eq := strings.IndexByte(kv, '=')
			if eq <= 0 || eq == len(kv)-1 {
				return BlockState{}, fmt.Errorf("schematic: bad property %q in state string", kv)
			}
			props[kv[:eq]] = kv[eq+1:]
		}
	}
	if len(props) == 0 {
		props = nil
	}
	return BlockState{Name: name, Properties: props}, nil
}

// compoundFields decodes a raw compound payload into its named fields.
func compoundFields(raw nbt.RawMessage) (map[string]nbt.RawMessage, error) {
	if raw.Type != nbt.TagCompound {
		return nil, fmt.Errorf("schematic: expected compound, got tag %d", raw.Type)
	}
	// RawMessage.Data is a bare compound payload; wrap it in a root header
	// so the decoder accepts it as a document.
	doc := make([]byte, 0, len(raw.Data)+3)
	doc = append(doc, byte(nbt.TagCompound), 0, 0)
	doc = append(doc, raw.Data...)
	fields := map[string]nbt.RawMessage{}
	if err := nbt.Unmarshal(doc, &fields); err != nil {
		return nil, fmt.Errorf("schematic: decode compound fields: %w", err)
	}
	return fields, nil
}

// compoundFromFields encodes named fields back into a raw compound payload,
// with sorted field order for byte-stable output.
func compoundFromFields(fields map[string]nbt.RawMessage) nbt.RawMessage {
	keys := make([]string, 0, len(fields))
	for k := range fields {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var buf bytes.Buffer
	for _, k := range keys {
		f := fields[k]
		buf.WriteByte(f.Type)
		var l [2]byte
		binary.BigEndian.PutUint16(l[:], uint16(len(k)))
		buf.Write(l[:])
		buf.WriteString(k)
		buf.Write(f.Data)
	}
	buf.WriteByte(byte(nbt.TagEnd))
	return nbt.RawMessage{Type: nbt.TagCompound, Data: buf.Bytes()}
}

// rawString encodes a string as a raw TAG_String message.
func rawString(s string) nbt.RawMessage {
	var buf bytes.Buffer
	var l [2]byte
	binary.BigEndian.PutUint16(l[:], uint16(len(s)))
	buf.Write(l[:])
	buf.WriteString(s)
	return nbt.RawMessage{Type: nbt.TagString, Data: buf.Bytes()}
}

// stringFromRaw decodes a raw TAG_String message.
func stringFromRaw(r nbt.RawMessage) (string, bool) {
	if r.Type != nbt.TagString || len(r.Data) < 2 {
		return "", false
	}
	n := int(binary.BigEndian.Uint16(r.Data[:2]))
	if len(r.Data) < 2+n {
		return "", false
	}
	return string(r.Data[2 : 2+n]), true
}

// intArray is a []int32 that relies on the encoder's default TAG_Int_Array
// framing (correct for Sponge Pos/Offset fields, unlike structure NBT).
type intArray []int32

// orderedPalette encodes a Sponge palette compound {state string: index}
// in slice order, keeping writer output byte-stable.
type orderedPalette []string

func (p orderedPalette) TagType() byte { return nbt.TagCompound }

func (p orderedPalette) MarshalNBT(w io.Writer) error {
	for i, state := range p {
		if _, err := w.Write([]byte{nbt.TagInt}); err != nil {
			return err
		}
		var l [2]byte
		binary.BigEndian.PutUint16(l[:], uint16(len(state)))
		if _, err := w.Write(l[:]); err != nil {
			return err
		}
		if _, err := io.WriteString(w, state); err != nil {
			return err
		}
		if err := binary.Write(w, binary.BigEndian, int32(i)); err != nil {
			return err
		}
	}
	_, err := w.Write([]byte{nbt.TagEnd})
	return err
}

// unmarshalRaw decodes a RawMessage payload into a Go value.
func unmarshalRaw(r nbt.RawMessage, v interface{}) error {
	doc := make([]byte, 0, len(r.Data)+3)
	doc = append(doc, r.Type, 0, 0)
	doc = append(doc, r.Data...)
	return nbt.Unmarshal(doc, v)
}

// uvarint reads an unsigned LEB128 varint (the Sponge .schem block-data
// encoding); returns the value and bytes consumed (0 on truncation).
func uvarint(b []byte) (uint64, int) {
	var v uint64
	for i := 0; i < len(b) && i < 10; i++ {
		v |= uint64(b[i]&0x7f) << (7 * i)
		if b[i]&0x80 == 0 {
			return v, i + 1
		}
	}
	return 0, 0
}

// putUvarint writes an unsigned LEB128 varint.
func putUvarint(buf *bytes.Buffer, v uint64) {
	for v >= 0x80 {
		buf.WriteByte(byte(v) | 0x80)
		v >>= 7
	}
	buf.WriteByte(byte(v))
}

// encodeNBT encodes v as an uncompressed NBT document with an empty root name.
func encodeNBT(buf *bytes.Buffer, v interface{}) error {
	return nbt.NewEncoder(buf).Encode(v, "")
}

// rawInt encodes an int32 as a raw TAG_Int message.
func rawInt(v int32) nbt.RawMessage {
	var b [4]byte
	binary.BigEndian.PutUint32(b[:], uint32(v))
	return nbt.RawMessage{Type: nbt.TagInt, Data: b[:]}
}

// encodeAndGzip encodes v as gzip-compressed NBT (test/serialization helper).
func encodeAndGzip(v interface{}) ([]byte, error) {
	var buf bytes.Buffer
	if err := nbt.NewEncoder(&buf).Encode(v, ""); err != nil {
		return nil, err
	}
	return gzipBytes(buf.Bytes())
}

// intFromRaw decodes a raw TAG_Int message.
func intFromRaw(r nbt.RawMessage) (int32, bool) {
	if r.Type != nbt.TagInt || len(r.Data) != 4 {
		return 0, false
	}
	return int32(binary.BigEndian.Uint32(r.Data)), true
}

// rawList marshals a list of raw compound payloads. The Tnze encoder honors
// the Marshaler interface on struct fields but NOT on slice elements — a
// []rawNBT silently encodes each element's {Type, Data} struct fields — so
// raw lists must implement Marshaler at the list level.
type rawList []nbt.RawMessage

func (l rawList) TagType() byte { return nbt.TagList }

func (l rawList) MarshalNBT(w io.Writer) error {
	elem := byte(nbt.TagEnd)
	if len(l) > 0 {
		elem = l[0].Type
	}
	if _, err := w.Write([]byte{elem}); err != nil {
		return err
	}
	if err := binary.Write(w, binary.BigEndian, int32(len(l))); err != nil {
		return err
	}
	for _, e := range l {
		if e.Type != elem {
			return fmt.Errorf("schematic: mixed tag types in raw list")
		}
		if _, err := w.Write(e.Data); err != nil {
			return err
		}
	}
	return nil
}

// base64Encode is a small helper for Building Gadgets v1 bodies.
func base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}
