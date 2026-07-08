package schematic

import (
	"encoding/binary"
	"fmt"
)

// Tier-1 structural limits on raw NBT documents, enforced before any real
// decoding. The walker recurses but depth is checked before each descent
// (bounded at MaxNBTDepth, so hostile nesting cannot overflow the stack) and
// declared lengths are validated against the actual payload size, so a
// 20-byte file claiming a billion-entry list is rejected before any
// allocation happens.
const (
	// MaxNBTDepth caps compound/list nesting.
	MaxNBTDepth = 128
	// MaxNBTTags caps the total number of tags in a document.
	MaxNBTTags = 16 * 1024 * 1024
)

type nbtWalker struct {
	data []byte
	pos  int
	tags int
}

func (w *nbtWalker) need(n int) error {
	if n < 0 || w.pos+n > len(w.data) {
		return fmt.Errorf("schematic: NBT truncated or declares out-of-bounds length")
	}
	return nil
}

func (w *nbtWalker) u16() (int, error) {
	if err := w.need(2); err != nil {
		return 0, err
	}
	v := int(binary.BigEndian.Uint16(w.data[w.pos:]))
	w.pos += 2
	return v, nil
}

func (w *nbtWalker) i32() (int, error) {
	if err := w.need(4); err != nil {
		return 0, err
	}
	v := int(int32(binary.BigEndian.Uint32(w.data[w.pos:])))
	w.pos += 4
	return v, nil
}

func (w *nbtWalker) countTag() error {
	w.tags++
	if w.tags > MaxNBTTags {
		return fmt.Errorf("schematic: NBT exceeds %d tags", MaxNBTTags)
	}
	return nil
}

// skipPayload consumes the payload of the given tag type.
func (w *nbtWalker) skipPayload(tagType byte, depth int) error {
	if depth > MaxNBTDepth {
		return fmt.Errorf("schematic: NBT nesting exceeds depth %d", MaxNBTDepth)
	}
	switch tagType {
	case 1: // Byte
		return w.skip(1)
	case 2: // Short
		return w.skip(2)
	case 3, 5: // Int, Float
		return w.skip(4)
	case 4, 6: // Long, Double
		return w.skip(8)
	case 7: // ByteArray
		n, err := w.i32()
		if err != nil {
			return err
		}
		return w.skip(n)
	case 8: // String
		n, err := w.u16()
		if err != nil {
			return err
		}
		return w.skip(n)
	case 9: // List
		if err := w.need(1); err != nil {
			return err
		}
		elem := w.data[w.pos]
		w.pos++
		n, err := w.i32()
		if err != nil {
			return err
		}
		if n < 0 {
			n = 0
		}
		if elem == 0 && n > 0 {
			return fmt.Errorf("schematic: NBT list of TAG_End with nonzero length")
		}
		// Cheap lower-bound check before iterating: every element needs at
		// least one byte of payload except empty compounds (1 byte TagEnd).
		if n > len(w.data)-w.pos {
			return fmt.Errorf("schematic: NBT list length exceeds document size")
		}
		for i := 0; i < n; i++ {
			if err := w.countTag(); err != nil {
				return err
			}
			if err := w.skipPayload(elem, depth+1); err != nil {
				return err
			}
		}
		return nil
	case 10: // Compound
		for {
			if err := w.need(1); err != nil {
				return err
			}
			t := w.data[w.pos]
			w.pos++
			if t == 0 {
				return nil
			}
			if err := w.countTag(); err != nil {
				return err
			}
			n, err := w.u16()
			if err != nil {
				return err
			}
			if err := w.skip(n); err != nil {
				return err
			}
			if err := w.skipPayload(t, depth+1); err != nil {
				return err
			}
		}
	case 11: // IntArray
		n, err := w.i32()
		if err != nil {
			return err
		}
		if n < 0 {
			return fmt.Errorf("schematic: negative NBT array length")
		}
		return w.skip(4 * n)
	case 12: // LongArray
		n, err := w.i32()
		if err != nil {
			return err
		}
		if n < 0 {
			return fmt.Errorf("schematic: negative NBT array length")
		}
		return w.skip(8 * n)
	default:
		return fmt.Errorf("schematic: unknown NBT tag type %d", tagType)
	}
}

func (w *nbtWalker) skip(n int) error {
	if err := w.need(n); err != nil {
		return err
	}
	w.pos += n
	return nil
}

// validateNBTLimits walks a raw (uncompressed) NBT document and enforces
// depth, tag-count and length-sanity limits. Non-NBT data (e.g. Building
// Gadgets JSON) passes through untouched.
func validateNBTLimits(data []byte) error {
	if len(data) == 0 || data[0] != 10 {
		return nil // not a compound-rooted NBT document
	}
	w := &nbtWalker{data: data, pos: 1}
	n, err := w.u16() // root name
	if err != nil {
		return err
	}
	if err := w.skip(n); err != nil {
		return err
	}
	if err := w.countTag(); err != nil {
		return err
	}
	return w.skipPayload(10, 0)
}
