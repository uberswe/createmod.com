package schematic

import "strings"

// Block family classification for similarity fingerprints. Families make
// material comparison invariant to cosmetic choices: an oak build and its
// spruce copy have identical family histograms. The list and matching rules
// are versioned via FingerprintVersion — changing them requires a bump.

// BlockFamilies is the fixed family list; vector indices are stable.
var BlockFamilies = []string{
	"planks", "logs", "stairs", "slabs", "fences_walls", "doors_trapdoors",
	"glass", "wool_carpet", "concrete_terracotta", "stone", "bricks",
	"dirt_organic", "sand_gravel", "metal_blocks", "ores", "leaves_plants",
	"storage", "crafting_utility", "redstone", "rails", "lighting",
	"furniture_deco", "liquids",
	// Create-mod functional families (also feed the function profile)
	"create_rotation", "create_transmission", "create_movement",
	"create_processing", "create_logistics", "create_power",
	"create_casing_deco", "modded_other",
}

var familyIndex = func() map[string]int {
	m := make(map[string]int, len(BlockFamilies))
	for i, f := range BlockFamilies {
		m[f] = i
	}
	return m
}()

// functionFamilies are the families that form the Create functional profile.
var functionFamilies = []string{
	"create_rotation", "create_transmission", "create_movement",
	"create_processing", "create_logistics", "create_power",
}

// keyword rule: first match wins, checked in order.
type familyRule struct {
	keywords []string
	family   string
}

// createRules match within the create: namespace (and other kinetic mods).
var createRules = []familyRule{
	{[]string{"water_wheel", "windmill", "steam_engine", "motor", "hand_crank", "flywheel"}, "create_power"},
	{[]string{"gearbox", "clutch", "gearshift", "sequenced", "rotation_speed", "adjustable_chain"}, "create_transmission"},
	{[]string{"bearing", "pulley", "gantry", "super_glue", "sticker", "cart_assembler", "contraption"}, "create_movement"},
	{[]string{"press", "mixer", "millstone", "crushing", "drill", "saw", "deployer", "spout", "fan", "basin", "blaze_burner", "crafter"}, "create_processing"},
	{[]string{"funnel", "chute", "tunnel", "belt", "depot", "arm", "pipe", "pump", "valve", "hose", "tank", "vault", "smart_observer", "threshold"}, "create_logistics"},
	{[]string{"shaft", "cogwheel", "gear"}, "create_rotation"},
	{[]string{"casing", "girder", "scaffolding"}, "create_casing_deco"},
}

// vanillaRules match any namespace by block-name keywords, first hit wins.
var vanillaRules = []familyRule{
	{[]string{"command_block", "structure_block", "jigsaw"}, "crafting_utility"},
	{[]string{"stairs"}, "stairs"},
	{[]string{"slab"}, "slabs"},
	{[]string{"fence", "wall"}, "fences_walls"},
	{[]string{"door", "trapdoor"}, "doors_trapdoors"},
	{[]string{"glass", "pane"}, "glass"},
	{[]string{"wool", "carpet"}, "wool_carpet"},
	{[]string{"concrete", "terracotta", "glazed"}, "concrete_terracotta"},
	{[]string{"planks"}, "planks"},
	{[]string{"log", "stem", "hyphae", "bamboo_block", "stripped"}, "logs"},
	{[]string{"brick"}, "bricks"},
	{[]string{"rail"}, "rails"},
	{[]string{"redstone", "repeater", "comparator", "observer", "piston", "lever", "button", "pressure_plate", "hopper", "dropper", "dispenser", "tripwire", "daylight", "target"}, "redstone"},
	{[]string{"chest", "barrel", "shulker"}, "storage"},
	{[]string{"torch", "lantern", "lamp", "glowstone", "sea_lantern", "shroomlight", "froglight", "campfire", "candle", "end_rod"}, "lighting"},
	{[]string{"crafting", "furnace", "smoker", "anvil", "grindstone", "stonecutter", "loom", "smithing", "cartography", "fletching", "lectern", "enchanting", "brewing", "beacon", "bell", "composter", "cauldron"}, "crafting_utility"},
	{[]string{"leaves", "sapling", "flower", "grass", "fern", "vine", "moss", "lichen", "mushroom", "fungus", "roots", "sprouts", "bush", "azalea", "lily", "kelp", "pickle", "coral", "wheat", "carrot", "potato", "beetroot", "melon", "pumpkin", "cactus", "sugar_cane", "hay"}, "leaves_plants"},
	{[]string{"dirt", "podzol", "mycelium", "mud", "clay", "farmland", "grass_block", "rooted", "soul_soil"}, "dirt_organic"},
	{[]string{"sand", "gravel", "dripstone", "snow", "ice", "powder"}, "sand_gravel"},
	{[]string{"iron_block", "gold_block", "copper", "netherite_block", "emerald_block", "diamond_block", "lapis_block", "coal_block", "raw_"}, "metal_blocks"},
	{[]string{"_ore", "ancient_debris", "amethyst"}, "ores"},
	{[]string{"water", "lava"}, "liquids"},
	{[]string{"bed", "banner", "sign", "flower_pot", "skull", "head", "item_frame", "painting", "chain", "bars", "ladder", "scaffolding"}, "furniture_deco"},
	{[]string{"stone", "andesite", "granite", "diorite", "deepslate", "tuff", "basalt", "blackstone", "cobble", "obsidian", "netherrack", "end_stone", "purpur", "quartz", "calcite", "sandstone", "prismarine"}, "stone"},
}

// BlockFamily classifies a block id into its family.
func BlockFamily(name string) string {
	ns, path, _ := cutNamespace(name)
	if ns == "create" {
		for _, r := range createRules {
			for _, kw := range r.keywords {
				if strings.Contains(path, kw) {
					return r.family
				}
			}
		}
		// unmatched create blocks are still deco-adjacent
		return "create_casing_deco"
	}
	for _, r := range vanillaRules {
		for _, kw := range r.keywords {
			if strings.Contains(path, kw) {
				return r.family
			}
		}
	}
	if ns != "minecraft" && ns != "" {
		return "modded_other"
	}
	return "stone" // conservative default bucket for unmatched vanilla
}
