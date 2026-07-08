package schematic

import (
	"sort"
	"strings"
)

// Tier-2 content inspection: surface everything in a schematic that can act
// on a world or player beyond placing blocks — command blocks, spawners,
// sign/book click commands, structure blocks and jigsaws. Findings are
// facts, not verdicts: "contains command blocks" is information the download
// page shows, not a rejection.

// FindingType classifies one inspection finding.
type FindingType string

const (
	FindingCommandBlock  FindingType = "command_block"
	FindingSpawner       FindingType = "spawner"
	FindingSignCommand   FindingType = "sign_command"
	FindingBookOrItemCmd FindingType = "item_command"
	FindingStructureBlk  FindingType = "structure_block"
	FindingJigsaw        FindingType = "jigsaw"
)

// Finding is one notable thing found at one position.
type Finding struct {
	Type   FindingType `json:"type"`
	Pos    [3]int      `json:"pos"`
	Detail string      `json:"detail,omitempty"` // command text, spawned entity, ...
}

// MaxFindings caps the detailed findings list; counts stay exact.
const MaxFindings = 200

// InspectorVersion is stored with results so improved inspectors can
// re-queue previously scanned builds.
const InspectorVersion = 1

// Manifest is the transparency report for one schematic.
type Manifest struct {
	InspectorVersion int                 `json:"inspectorVersion"`
	Counts           map[FindingType]int `json:"counts,omitempty"`
	Findings         []Finding           `json:"findings,omitempty"`
	FindingsTruncated bool               `json:"findingsTruncated,omitempty"`
	// ModNamespaces lists non-vanilla block namespaces (informational).
	ModNamespaces []string `json:"modNamespaces,omitempty"`
}

// Notable reports whether the manifest contains anything worth surfacing.
func (m *Manifest) Notable() bool { return len(m.Counts) > 0 }

var commandBlockNames = map[string]bool{
	"minecraft:command_block":           true,
	"minecraft:chain_command_block":     true,
	"minecraft:repeating_command_block": true,
}

var spawnerNames = map[string]bool{
	"minecraft:spawner":       true,
	"minecraft:mob_spawner":   true, // pre-flattening id kept by some tools
	"minecraft:trial_spawner": true,
}

// Inspect walks the model and produces its transparency manifest.
func Inspect(s *Schematic) *Manifest {
	m := &Manifest{InspectorVersion: InspectorVersion, Counts: map[FindingType]int{}}

	add := func(t FindingType, pos [3]int, detail string) {
		m.Counts[t]++
		if len(m.Findings) < MaxFindings {
			if len(detail) > 256 {
				detail = detail[:256] + "…"
			}
			m.Findings = append(m.Findings, Finding{Type: t, Pos: pos, Detail: detail})
		} else {
			m.FindingsTruncated = true
		}
	}

	// Block entities by position for detail extraction.
	beAt := map[[3]int]map[string]rawField{}
	for _, be := range s.BlockEntities {
		if fields, err := compoundFields(be.Raw); err == nil {
			rf := map[string]rawField{}
			for k, v := range fields {
				rf[k] = rawField(v)
			}
			beAt[be.Pos] = rf
		}
	}

	// Palette-driven scan: find positions of notable block types.
	notablePalette := map[int32]FindingType{}
	namespaces := map[string]bool{}
	for i, st := range s.Palette {
		if ns, _, ok := cutNamespace(st.Name); ok && ns != "minecraft" {
			namespaces[ns] = true
		}
		switch {
		case commandBlockNames[st.Name]:
			notablePalette[int32(i)] = FindingCommandBlock
		case spawnerNames[st.Name]:
			notablePalette[int32(i)] = FindingSpawner
		case st.Name == "minecraft:structure_block":
			notablePalette[int32(i)] = FindingStructureBlk
		case st.Name == "minecraft:jigsaw":
			notablePalette[int32(i)] = FindingJigsaw
		}
	}
	if len(notablePalette) > 0 {
		for y := 0; y < s.Size[1]; y++ {
			for z := 0; z < s.Size[2]; z++ {
				for x := 0; x < s.Size[0]; x++ {
					t, ok := notablePalette[s.Blocks[s.Index(x, y, z)]]
					if !ok {
						continue
					}
					pos := [3]int{x, y, z}
					detail := ""
					if fields, ok := beAt[pos]; ok {
						switch t {
						case FindingCommandBlock:
							detail, _ = strField(fields, "Command")
						case FindingSpawner:
							detail = spawnerEntity(fields)
						}
					}
					add(t, pos, detail)
				}
			}
		}
	}

	// Block-entity payload scan: sign click commands and command text inside
	// items (books, containers). This is a conservative textual sweep of the
	// raw payload — cheap, format-agnostic, and immune to nesting tricks.
	for _, be := range s.BlockEntities {
		payload := be.Raw.Data
		if cmds := extractRunCommands(payload); len(cmds) > 0 {
			t := FindingSignCommand
			if fields, ok := beAt[be.Pos]; ok {
				if id, ok2 := strField(fields, "id"); ok2 && !strings.Contains(id, "sign") {
					t = FindingBookOrItemCmd
				}
			}
			for _, c := range cmds {
				add(t, be.Pos, c)
			}
		}
	}

	for ns := range namespaces {
		m.ModNamespaces = append(m.ModNamespaces, ns)
	}
	sort.Strings(m.ModNamespaces)
	if len(m.Counts) == 0 {
		m.Counts = nil
	}
	return m
}

type rawField struct {
	Type byte
	Data []byte
}

type rawFieldsMap = map[string]rawField

// strField reads a TAG_String field from decoded compound fields.
func strField(fields rawFieldsMap, key string) (string, bool) {
	rf, ok := fields[key]
	if !ok || rf.Type != 8 /* TagString */ || len(rf.Data) < 2 {
		return "", false
	}
	n := int(uint16(rf.Data[0])<<8 | uint16(rf.Data[1]))
	if len(rf.Data) < 2+n {
		return "", false
	}
	return string(rf.Data[2 : 2+n]), true
}

// spawnerEntity extracts the spawned entity id from spawner NBT (textual
// sweep of SpawnData/SpawnPotentials for an entity id).
func spawnerEntity(fields rawFieldsMap) string {
	for _, key := range []string{"SpawnData", "SpawnPotentials"} {
		rf, ok := fields[key]
		if !ok {
			continue
		}
		if id := firstEntityID(rf.Data); id != "" {
			return id
		}
	}
	return ""
}

// firstEntityID scans raw NBT bytes for an `id` string field that looks like
// an entity identifier.
func firstEntityID(data []byte) string {
	for i := 0; i+5 < len(data); i++ {
		// TAG_String(8) + name length 2 + "id"
		if data[i] == 8 && data[i+1] == 0 && data[i+2] == 2 && data[i+3] == 'i' && data[i+4] == 'd' {
			rest := data[i+5:]
			if len(rest) < 2 {
				continue
			}
			n := int(uint16(rest[0])<<8 | uint16(rest[1]))
			if n > 0 && len(rest) >= 2+n {
				v := string(rest[2 : 2+n])
				if strings.Contains(v, ":") {
					return v
				}
			}
		}
	}
	return ""
}

// extractRunCommands finds run_command click events in raw NBT payloads.
// Modern signs/books embed JSON text components; the pattern below matches
// both `"action":"run_command","value":"..."` (pre-1.21.5) and
// `"action":"run_command","command":"..."` orderings, conservatively.
func extractRunCommands(data []byte) []string {
	s := string(data)
	var out []string
	idx := 0
	for {
		i := strings.Index(s[idx:], "run_command")
		if i < 0 {
			break
		}
		i += idx
		// Grab the nearest quoted value after the marker (the command text),
		// looking within a bounded window.
		window := s[i:min(i+512, len(s))]
		if cmd := firstCommandInWindow(window); cmd != "" {
			out = append(out, cmd)
		} else {
			out = append(out, "(command present, text not extracted)")
		}
		idx = i + len("run_command")
		if len(out) >= 32 {
			break
		}
	}
	return out
}

func firstCommandInWindow(w string) string {
	// Text components nest: a sign message may itself be a JSON string, so
	// quotes can appear escaped one or more levels deep. Peel escaping until
	// a marker matches.
	for tries := 0; tries < 3; tries++ {
		for _, marker := range []string{`"value":"`, `"value": "`, `"command":"`, `"command": "`} {
			if j := strings.Index(w, marker); j >= 0 {
				rest := w[j+len(marker):]
				if k := findUnescapedQuote(rest); k >= 0 {
					return rest[:k]
				}
			}
		}
		if !strings.Contains(w, `\"`) {
			break
		}
		w = strings.ReplaceAll(w, `\"`, `"`)
	}
	return ""
}

func findUnescapedQuote(s string) int {
	for i := 0; i < len(s); i++ {
		if s[i] == '"' && (i == 0 || s[i-1] != '\\') {
			return i
		}
	}
	return -1
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
