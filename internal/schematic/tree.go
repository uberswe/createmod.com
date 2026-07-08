package schematic

import (
	"encoding/binary"
	"fmt"
	"math"
	"strconv"
	"strings"

	"github.com/Tnze/go-mc/nbt"
)

// Read-only NBT tree navigation for the viewer. Nodes are addressed by
// index paths (child ordinals in document order), which need no escaping;
// each node also carries a human-readable display path for copy-path.
//
// All functions operate on raw (decompressed) NBT and slice into the
// original buffer — no full-document materialization, so paging into a
// 100 MB document stays cheap.

// TreeNode is one NBT node shaped for the viewer.
type TreeNode struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Value       string `json:"value,omitempty"` // scalar value or preview
	Path        string `json:"path"`            // index path, e.g. "2/0/5"
	DisplayPath string `json:"displayPath"`     // e.g. blocks[5].nbt.Items
	ChildCount  int    `json:"childCount"`
	HasChildren bool   `json:"hasChildren"`
	Create      bool   `json:"create,omitempty"` // Create-mod payload flag
}

// TreePage is a paged listing of one node's children.
type TreePage struct {
	Node     TreeNode   `json:"node"`
	Children []TreeNode `json:"children"`
	Total    int        `json:"total"`
	Offset   int        `json:"offset"`
	// Nested holds child pages when depth > 1, keyed by child index.
	Nested map[int]*TreePage `json:"nested,omitempty"`
}

var tagTypeNames = map[byte]string{
	1: "byte", 2: "short", 3: "int", 4: "long", 5: "float", 6: "double",
	7: "byte[]", 8: "string", 9: "list", 10: "compound",
	11: "int[]", 12: "long[]",
}

// rawNode is a cursor into the document.
type rawNode struct {
	typ     byte
	name    string
	payload []byte // exactly this node's payload bytes
}

// payloadEnd returns the length of the payload for tagType starting data.
func payloadEnd(data []byte, tagType byte) (int, error) {
	w := &nbtWalker{data: data}
	if err := w.skipPayload(tagType, 0); err != nil {
		return 0, err
	}
	return w.pos, nil
}

// rootNode parses the document header.
func rootNode(raw []byte) (rawNode, error) {
	if len(raw) < 3 || raw[0] != 10 {
		return rawNode{}, fmt.Errorf("schematic: not a compound-rooted NBT document")
	}
	n := int(binary.BigEndian.Uint16(raw[1:3]))
	if len(raw) < 3+n {
		return rawNode{}, fmt.Errorf("schematic: truncated NBT root")
	}
	return rawNode{typ: 10, name: string(raw[3 : 3+n]), payload: raw[3+n:]}, nil
}

// childAt returns the i-th child of a compound or list node, plus the total
// child count when countOnly scanning completes. For arrays, elements are
// virtualized as scalar children.
func nbtChildren(n rawNode) ([]rawNode, error) {
	switch n.typ {
	case 10: // compound: entries in document order
		var out []rawNode
		pos := 0
		data := n.payload
		for {
			if pos >= len(data) {
				return nil, fmt.Errorf("schematic: unterminated compound")
			}
			t := data[pos]
			pos++
			if t == 0 {
				return out, nil
			}
			if pos+2 > len(data) {
				return nil, fmt.Errorf("schematic: truncated compound entry")
			}
			nameLen := int(binary.BigEndian.Uint16(data[pos:]))
			pos += 2
			if pos+nameLen > len(data) {
				return nil, fmt.Errorf("schematic: truncated entry name")
			}
			name := string(data[pos : pos+nameLen])
			pos += nameLen
			end, err := payloadEnd(data[pos:], t)
			if err != nil {
				return nil, err
			}
			out = append(out, rawNode{typ: t, name: name, payload: data[pos : pos+end]})
			pos += end
		}
	case 9: // list
		data := n.payload
		if len(data) < 5 {
			return nil, fmt.Errorf("schematic: truncated list")
		}
		elem := data[0]
		count := int(int32(binary.BigEndian.Uint32(data[1:5])))
		pos := 5
		var out []rawNode
		for i := 0; i < count; i++ {
			end, err := payloadEnd(data[pos:], elem)
			if err != nil {
				return nil, err
			}
			out = append(out, rawNode{typ: elem, name: "[" + strconv.Itoa(i) + "]", payload: data[pos : pos+end]})
			pos += end
		}
		return out, nil
	case 7, 11, 12: // arrays: virtualize as scalars
		width := map[byte]int{7: 1, 11: 4, 12: 8}[n.typ]
		scalarType := map[byte]byte{7: 1, 11: 3, 12: 4}[n.typ]
		if len(n.payload) < 4 {
			return nil, fmt.Errorf("schematic: truncated array")
		}
		count := int(int32(binary.BigEndian.Uint32(n.payload[0:4])))
		var out []rawNode
		for i := 0; i < count; i++ {
			off := 4 + i*width
			if off+width > len(n.payload) {
				break
			}
			out = append(out, rawNode{typ: scalarType, name: "[" + strconv.Itoa(i) + "]", payload: n.payload[off : off+width]})
		}
		return out, nil
	default:
		return nil, nil
	}
}

func scalarValue(n rawNode) string {
	p := n.payload
	switch n.typ {
	case 1:
		if len(p) >= 1 {
			return strconv.Itoa(int(int8(p[0])))
		}
	case 2:
		if len(p) >= 2 {
			return strconv.Itoa(int(int16(binary.BigEndian.Uint16(p))))
		}
	case 3:
		if len(p) >= 4 {
			return strconv.Itoa(int(int32(binary.BigEndian.Uint32(p))))
		}
	case 4:
		if len(p) >= 8 {
			return strconv.FormatInt(int64(binary.BigEndian.Uint64(p)), 10)
		}
	case 5:
		if len(p) >= 4 {
			return strconv.FormatFloat(float64(math.Float32frombits(binary.BigEndian.Uint32(p))), 'g', -1, 32)
		}
	case 6:
		if len(p) >= 8 {
			return strconv.FormatFloat(math.Float64frombits(binary.BigEndian.Uint64(p)), 'g', -1, 64)
		}
	case 8:
		if len(p) >= 2 {
			ln := int(binary.BigEndian.Uint16(p))
			if len(p) >= 2+ln {
				s := string(p[2 : 2+ln])
				if len(s) > 200 {
					s = s[:200] + "…"
				}
				return s
			}
		}
	}
	return ""
}

func childCountOf(n rawNode) int {
	switch n.typ {
	case 10:
		kids, err := nbtChildren(n)
		if err != nil {
			return 0
		}
		return len(kids)
	case 9:
		if len(n.payload) >= 5 {
			return int(int32(binary.BigEndian.Uint32(n.payload[1:5])))
		}
	case 7, 11, 12:
		if len(n.payload) >= 4 {
			return int(int32(binary.BigEndian.Uint32(n.payload[0:4])))
		}
	}
	return 0
}

func toTreeNode(n rawNode, path, displayPath string) TreeNode {
	tn := TreeNode{
		Name:        n.name,
		Type:        tagTypeNames[n.typ],
		Path:        path,
		DisplayPath: displayPath,
	}
	switch n.typ {
	case 9:
		cc := childCountOf(n)
		tn.ChildCount = cc
		tn.HasChildren = cc > 0
		if len(n.payload) >= 1 {
			tn.Value = fmt.Sprintf("%d × %s", cc, tagTypeNames[n.payload[0]])
		}
	case 10:
		cc := childCountOf(n)
		tn.ChildCount = cc
		tn.HasChildren = cc > 0
	case 7, 11, 12:
		cc := childCountOf(n)
		tn.ChildCount = cc
		tn.HasChildren = cc > 0
		tn.Value = fmt.Sprintf("%d entries", cc)
	default:
		tn.Value = scalarValue(n)
	}
	if strings.Contains(tn.Value, "create:") || strings.HasPrefix(tn.Value, "create:") {
		tn.Create = true
	}
	return tn
}

// descend walks an index path from the root.
func descend(raw []byte, path string) (rawNode, string, error) {
	node, err := rootNode(raw)
	if err != nil {
		return rawNode{}, "", err
	}
	display := ""
	if path == "" {
		return node, display, nil
	}
	for _, seg := range strings.Split(path, "/") {
		idx, err := strconv.Atoi(seg)
		if err != nil || idx < 0 {
			return rawNode{}, "", fmt.Errorf("schematic: bad tree path")
		}
		kids, err := nbtChildren(node)
		if err != nil {
			return rawNode{}, "", err
		}
		if idx >= len(kids) {
			return rawNode{}, "", fmt.Errorf("schematic: tree path out of range")
		}
		child := kids[idx]
		if strings.HasPrefix(child.name, "[") {
			display += child.name
		} else if display == "" {
			display = child.name
		} else {
			display += "." + child.name
		}
		node = child
	}
	return node, display, nil
}

const (
	treeMaxLimit    = 1000
	treeMaxDepth    = 4
	treeMaxNodes    = 20000
)

// NBTTreePage lists the children of the node at path (index path), paged.
// depth > 1 nests child pages (bounded by treeMaxDepth / treeMaxNodes).
func NBTTreePage(raw []byte, path string, depth, offset, limit int) (*TreePage, error) {
	if err := validateNBTLimits(raw); err != nil {
		return nil, err
	}
	if limit <= 0 || limit > treeMaxLimit {
		limit = 200
	}
	if depth <= 0 {
		depth = 1
	}
	if depth > treeMaxDepth {
		depth = treeMaxDepth
	}
	if offset < 0 {
		offset = 0
	}
	node, display, err := descend(raw, path)
	if err != nil {
		return nil, err
	}
	budget := treeMaxNodes
	return buildPage(node, path, display, depth, offset, limit, &budget)
}

func buildPage(node rawNode, path, display string, depth, offset, limit int, budget *int) (*TreePage, error) {
	page := &TreePage{Node: toTreeNode(node, path, display), Offset: offset}
	kids, err := nbtChildren(node)
	if err != nil {
		return nil, err
	}
	page.Total = len(kids)
	if offset > len(kids) {
		offset = len(kids)
	}
	end := offset + limit
	if end > len(kids) {
		end = len(kids)
	}
	for i := offset; i < end; i++ {
		if *budget <= 0 {
			break
		}
		*budget--
		child := kids[i]
		childPath := strconv.Itoa(i)
		if path != "" {
			childPath = path + "/" + childPath
		}
		childDisplay := display
		if strings.HasPrefix(child.name, "[") {
			childDisplay += child.name
		} else if childDisplay == "" {
			childDisplay = child.name
		} else {
			childDisplay += "." + child.name
		}
		tn := toTreeNode(child, childPath, childDisplay)
		page.Children = append(page.Children, tn)
		if depth > 1 && tn.HasChildren && *budget > 0 {
			sub, err := buildPage(child, childPath, childDisplay, depth-1, 0, limit, budget)
			if err == nil {
				if page.Nested == nil {
					page.Nested = map[int]*TreePage{}
				}
				page.Nested[i] = sub
			}
		}
	}
	return page, nil
}

// NBTNodeSNBT renders one node (by index path) as SNBT text, capped.
func NBTNodeSNBT(raw []byte, path string, maxBytes int) (string, error) {
	if err := validateNBTLimits(raw); err != nil {
		return "", err
	}
	node, _, err := descend(raw, path)
	if err != nil {
		return "", err
	}
	if maxBytes <= 0 {
		maxBytes = 256 * 1024
	}
	if len(node.payload) > maxBytes {
		return "", fmt.Errorf("schematic: node too large for SNBT view (%d bytes)", len(node.payload))
	}
	doc := make([]byte, 0, len(node.payload)+3)
	doc = append(doc, node.typ, 0, 0)
	doc = append(doc, node.payload...)
	var s nbt.StringifiedMessage
	if err := nbt.Unmarshal(doc, &s); err != nil {
		return "", fmt.Errorf("schematic: SNBT render: %w", err)
	}
	return string(s), nil
}

// TreeSearchResult is one key-search hit.
type TreeSearchResult struct {
	Path        string `json:"path"`
	DisplayPath string `json:"displayPath"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	Value       string `json:"value,omitempty"`
}

// NBTTreeSearch finds nodes whose name or string value contains q
// (case-insensitive), capped at maxResults.
func NBTTreeSearch(raw []byte, q string, maxResults int) ([]TreeSearchResult, error) {
	if err := validateNBTLimits(raw); err != nil {
		return nil, err
	}
	if maxResults <= 0 || maxResults > 500 {
		maxResults = 200
	}
	q = strings.ToLower(q)
	if q == "" {
		return nil, nil
	}
	root, err := rootNode(raw)
	if err != nil {
		return nil, err
	}
	var out []TreeSearchResult
	visited := 0
	var walk func(n rawNode, path, display string)
	walk = func(n rawNode, path, display string) {
		if len(out) >= maxResults || visited > treeMaxNodes*10 {
			return
		}
		kids, err := nbtChildren(n)
		if err != nil {
			return
		}
		for i, child := range kids {
			if len(out) >= maxResults {
				return
			}
			visited++
			childPath := strconv.Itoa(i)
			if path != "" {
				childPath = path + "/" + childPath
			}
			childDisplay := display
			if strings.HasPrefix(child.name, "[") {
				childDisplay += child.name
			} else if childDisplay == "" {
				childDisplay = child.name
			} else {
				childDisplay += "." + child.name
			}
			val := scalarValue(child)
			if strings.Contains(strings.ToLower(child.name), q) || (val != "" && strings.Contains(strings.ToLower(val), q)) {
				out = append(out, TreeSearchResult{
					Path: childPath, DisplayPath: childDisplay,
					Name: child.name, Type: tagTypeNames[child.typ], Value: val,
				})
			}
			// Skip descending into large arrays; their virtual children are
			// numbers and rarely useful for key search.
			if child.typ == 9 || child.typ == 10 {
				walk(child, childPath, childDisplay)
			}
		}
	}
	walk(root, "", "")
	return out, nil
}

// DecompressForTree exposes hardened decompression for viewer handlers.
func DecompressForTree(data []byte) ([]byte, error) {
	return decompress(data)
}
