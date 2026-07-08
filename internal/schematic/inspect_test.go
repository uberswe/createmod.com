package schematic

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Tnze/go-mc/nbt"
)

// suspiciousFixture builds a schematic containing a command block with a
// command, a spawner, and a sign with a run_command click event.
func suspiciousFixture(t *testing.T) *Schematic {
	t.Helper()
	s := New(4, 1, 1)
	s.DataVersion = 3955
	cb := s.PaletteIndex(BlockState{Name: "minecraft:command_block"})
	sp := s.PaletteIndex(BlockState{Name: "minecraft:spawner"})
	sign := s.PaletteIndex(BlockState{Name: "minecraft:oak_sign"})
	stone := s.PaletteIndex(BlockState{Name: "create:cogwheel"})
	s.Blocks[s.Index(0, 0, 0)] = cb
	s.Blocks[s.Index(1, 0, 0)] = sp
	s.Blocks[s.Index(2, 0, 0)] = sign
	s.Blocks[s.Index(3, 0, 0)] = stone

	// command block BE
	s.BlockEntities = append(s.BlockEntities, BlockEntity{
		Pos: [3]int{0, 0, 0},
		Raw: compoundFromFields(map[string]nbt.RawMessage{
			"id":      rawString("minecraft:command_block"),
			"Command": rawString("/give @p diamond 64"),
		}),
	})
	// spawner BE with SpawnData containing an entity id
	var spawnData bytes.Buffer
	type entityRef struct {
		Entity struct {
			ID string `nbt:"id"`
		} `nbt:"entity"`
	}
	var er entityRef
	er.Entity.ID = "minecraft:zombie"
	if err := nbt.NewEncoder(&spawnData).Encode(er, ""); err != nil {
		t.Fatal(err)
	}
	s.BlockEntities = append(s.BlockEntities, BlockEntity{
		Pos: [3]int{1, 0, 0},
		Raw: compoundFromFields(map[string]nbt.RawMessage{
			"id":        rawString("minecraft:mob_spawner"),
			"SpawnData": {Type: nbt.TagCompound, Data: spawnData.Bytes()[3:]},
		}),
	})
	// sign BE with a run_command click event in its JSON text
	signText := `{"messages":["{\"text\":\"click me\",\"clickEvent\":{\"action\":\"run_command\",\"value\":\"/op @p\"}}"]}`
	s.BlockEntities = append(s.BlockEntities, BlockEntity{
		Pos: [3]int{2, 0, 0},
		Raw: compoundFromFields(map[string]nbt.RawMessage{
			"id":         rawString("minecraft:sign"),
			"front_text": rawString(signText),
		}),
	})
	return s
}

func Test_Inspect_FindsEverything(t *testing.T) {
	m := Inspect(suspiciousFixture(t))
	if !m.Notable() {
		t.Fatal("manifest not notable")
	}
	if m.Counts[FindingCommandBlock] != 1 {
		t.Errorf("command blocks = %d", m.Counts[FindingCommandBlock])
	}
	if m.Counts[FindingSpawner] != 1 {
		t.Errorf("spawners = %d", m.Counts[FindingSpawner])
	}
	if m.Counts[FindingSignCommand] < 1 {
		t.Errorf("sign commands = %d", m.Counts[FindingSignCommand])
	}
	var gotCmd, gotSpawn, gotSign bool
	for _, f := range m.Findings {
		switch f.Type {
		case FindingCommandBlock:
			gotCmd = f.Detail == "/give @p diamond 64"
		case FindingSpawner:
			gotSpawn = f.Detail == "minecraft:zombie"
		case FindingSignCommand:
			gotSign = strings.Contains(f.Detail, "/op @p")
		}
	}
	if !gotCmd || !gotSpawn || !gotSign {
		t.Errorf("details missing: cmd=%v spawn=%v sign=%v findings=%+v", gotCmd, gotSpawn, gotSign, m.Findings)
	}
	if len(m.ModNamespaces) != 1 || m.ModNamespaces[0] != "create" {
		t.Errorf("mod namespaces = %v", m.ModNamespaces)
	}
}

func Test_Inspect_CleanBuild(t *testing.T) {
	s, err := ReadStructureNBT(generatorFixture(t))
	if err != nil {
		t.Fatal(err)
	}
	m := Inspect(s)
	if m.Notable() {
		t.Errorf("clean generator hull flagged: %+v", m.Counts)
	}
	if m.FindingsTruncated {
		t.Errorf("truncated on clean build")
	}
}

func Test_Inspect_SurvivesConversion(t *testing.T) {
	// The inspection must find the same content regardless of which format
	// the schematic arrived in.
	src := suspiciousFixture(t)
	data, err := WriteSponge(src)
	if err != nil {
		t.Fatal(err)
	}
	back, err := ReadSponge(data)
	if err != nil {
		t.Fatal(err)
	}
	m := Inspect(back)
	if m.Counts[FindingCommandBlock] != 1 || m.Counts[FindingSpawner] != 1 || m.Counts[FindingSignCommand] < 1 {
		t.Errorf("post-conversion inspection lost findings: %+v", m.Counts)
	}
}

func Test_NBTLimits(t *testing.T) {
	// Deep nesting bomb: 200 nested compounds
	var deep bytes.Buffer
	deep.WriteByte(10)
	deep.Write([]byte{0, 0})
	for i := 0; i < 200; i++ {
		deep.WriteByte(10)             // child compound
		deep.Write([]byte{0, 1, 'a'}) // name "a"
	}
	if err := validateNBTLimits(deep.Bytes()); err == nil {
		t.Errorf("depth bomb accepted")
	}

	// List claiming a billion entries in a tiny document
	var bomb bytes.Buffer
	bomb.WriteByte(10)
	bomb.Write([]byte{0, 0})
	bomb.WriteByte(9)                          // list
	bomb.Write([]byte{0, 1, 'l'})              // name
	bomb.WriteByte(3)                          // int elements
	bomb.Write([]byte{0x3B, 0x9A, 0xCA, 0x00}) // 1,000,000,000
	if err := validateNBTLimits(bomb.Bytes()); err == nil {
		t.Errorf("list-length bomb accepted")
	}

	// Valid documents pass
	if err := validateNBTLimits([]byte{}); err != nil {
		t.Errorf("empty: %v", err)
	}
	raw, err := decompress(generatorFixture(t))
	if err != nil {
		t.Fatalf("real fixture rejected: %v", err)
	}
	if err := validateNBTLimits(raw); err != nil {
		t.Errorf("real fixture rejected by limits: %v", err)
	}
}
