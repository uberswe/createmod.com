package generator

import (
	"encoding/base64"
	"errors"
	"strconv"
	"strings"
)

var codeToColor = map[string]string{
	"wh": "white", "or": "orange", "ma": "magenta", "lb": "light_blue",
	"ye": "yellow", "li": "lime", "pk": "pink", "gy": "gray",
	"lg": "light_gray", "cy": "cyan", "pu": "purple", "bl": "blue",
	"br": "brown", "gn": "green", "re": "red", "bk": "black",
}

var codeToWood = map[string]string{
	"o": "oak", "s": "spruce", "b": "birch", "d": "dark_oak",
	"j": "jungle", "a": "acacia", "ch": "cherry", "cr": "crimson", "wa": "warped",
}

var codeToAirfoilShape = map[string]string{"l": "linear", "c": "curved"}
var codeToBladeMaterial = map[string]string{"w": "wool", "s": "sail"}
var codeToEnvelopeMaterial = map[string]string{"w": "wool", "e": "envelope"}
var codeToFrameMaterial = map[string]string{"w": "wood", "a": "andesite_casing"}
var codeToSternStyle = map[string]string{"r": "round", "s": "square", "p": "pointed"}
var codeToOrientation = map[string]string{"h": "horizontal", "v": "vertical"}
var codeToBowStyle = map[string]string{"d": "default", "pt": "pointed", "cl": "clipper", "rk": "raked", "pl": "plumb"}

type fieldType int

const (
	ftBool fieldType = iota
	ftInt
	ftFloat
	ftEnum
)

type schemaField struct {
	key     string
	ft      fieldType
	enumMap map[string]string
	// def is the value applied when the hash omits this field. The frontend
	// (generator.js encodeCompact) drops any param equal to its encode-default
	// — the page's initial slider/checkbox/select values captured by
	// setEncodeDefaults(getParams()) — so an absent field means "use the
	// default", not "zero". These defaults MUST mirror the HTML initial values
	// in template/generator-*.html; if they drift, share-link previews (the OG
	// images) reconstruct a different build than the frontend shows.
	def interface{}
}

var schemaPropeller = []schemaField{
	{"blades", ftInt, nil, 4},
	{"length", ftInt, nil, 10},
	{"rootChord", ftInt, nil, 3},
	{"tipChord", ftInt, nil, 1},
	{"sweepDegrees", ftFloat, nil, 25.0},
	{"swept", ftBool, nil, true},
	{"airfoilShape", ftEnum, codeToAirfoilShape, "curved"},
	{"bladeMaterial", ftEnum, codeToBladeMaterial, "wool"},
	{"bladeColor", ftEnum, codeToColor, "white"},
	{"rotation", ftFloat, nil, 0.0},
	{"orientation", ftEnum, codeToOrientation, "horizontal"},
}

var schemaBalloon = []schemaField{
	{"lengthX", ftInt, nil, 36},
	{"widthZ", ftInt, nil, 18},
	{"heightY", ftInt, nil, 16},
	{"cylinderMid", ftFloat, nil, 0.0},
	{"frontTaper", ftFloat, nil, 0.0},
	{"rearTaper", ftFloat, nil, 0.0},
	{"topFlatten", ftFloat, nil, 0.0},
	{"bottomFlatten", ftFloat, nil, 0.0},
	{"hollow", ftBool, nil, true},
	{"shell", ftInt, nil, 1},
	{"ribEnabled", ftBool, nil, false},
	{"ribSpacing", ftInt, nil, 4},
	{"keelEnabled", ftBool, nil, false},
	{"keelDepth", ftInt, nil, 1},
	{"finEnabled", ftBool, nil, false},
	{"sideFinEnabled", ftBool, nil, false},
	{"finHeight", ftInt, nil, 4},
	{"finLength", ftInt, nil, 5},
	{"envelopeMaterial", ftEnum, codeToEnvelopeMaterial, "wool"},
	{"envelopeColor", ftEnum, codeToColor, "white"},
	{"frameMaterial", ftEnum, codeToFrameMaterial, "wood"},
	{"frameWoodType", ftEnum, codeToWood, "spruce"},
	{"ribOffset", ftInt, nil, 0},
}

var schemaHull = []schemaField{
	{"woodType", ftEnum, codeToWood, "spruce"},
	{"length", ftInt, nil, 40},
	{"beam", ftInt, nil, 10},
	{"depth", ftInt, nil, 6},
	{"bottomPinch", ftFloat, nil, 0.25},
	{"hullFlare", ftFloat, nil, 0.2},
	{"flareCurve", ftFloat, nil, 2.6},
	{"tumblehome", ftFloat, nil, 0.15},
	{"tumbleCurve", ftFloat, nil, 3.0},
	{"sheerCurve", ftFloat, nil, 0.12},
	{"sheerCurveExp", ftFloat, nil, 2.0},
	{"bowLength", ftInt, nil, 10},
	{"bowSharpness", ftFloat, nil, 1.2},
	{"bowKeelRise", ftFloat, nil, 0.5},
	{"bowKeelLength", ftInt, nil, 10},
	{"sternStyle", ftEnum, codeToSternStyle, "round"},
	{"sternLength", ftInt, nil, 6},
	{"sternSharpness", ftFloat, nil, 0.8},
	{"sternKeelRise", ftFloat, nil, 0.2},
	{"sternKeelLength", ftInt, nil, 6},
	{"keelCurve", ftFloat, nil, 1.7},
	{"castleBlend", ftInt, nil, 4},
	{"hasRailings", ftBool, nil, true},
	{"hasTrim", ftBool, nil, true},
	{"hasWindows", ftBool, nil, true},
	{"castleHeight", ftInt, nil, 2},
	{"castleLength", ftInt, nil, 8},
	{"forecastleHeight", ftInt, nil, 1},
	{"forecastleLength", ftInt, nil, 5},
	{"hasGunPorts", ftBool, nil, false},
	{"gunPortRow", ftInt, nil, 2},
	{"gunPortSpacing", ftInt, nil, 4},
	{"bowCurve", ftFloat, nil, 0.0},
	{"sternOverhang", ftFloat, nil, 0.0},
	{"midWidthBias", ftFloat, nil, 0.0},
	{"bowStyle", ftEnum, codeToBowStyle, "default"},
	// v2 (version >= 3) fields, appended so pre-v3 hashes (shorter value
	// lists) keep decoding; field order below is frozen once released.
	{"deadrise", ftFloat, nil, 0.0},
	{"midFullness", ftFloat, nil, 0.65},
	{"bowSectionV", ftFloat, nil, 0.55},
	{"sternFullness", ftFloat, nil, 0.5},
	{"stemRake", ftFloat, nil, 0.35},
	{"stemCurve", ftFloat, nil, 0.15},
	{"sternRake", ftFloat, nil, 0.35},
	{"rocker", ftFloat, nil, 0.0},
	{"parallelMidbody", ftFloat, nil, 0.0},
	{"stemPostHeight", ftInt, nil, 0},
	{"sternPostHeight", ftInt, nil, 0},
	{"doubleEnder", ftBool, nil, false},
	{"closedHull", ftBool, nil, false},
}

// DecodeHash decodes a base64url generator hash into a GenerateResult.
// Returns the result and generator type ("propeller", "balloon", "hull").
func DecodeHash(hash string) (*GenerateResult, string, error) {
	compact, err := decodeBase64URL(hash)
	if err != nil {
		return nil, "", err
	}

	parts := strings.Split(compact, ".")
	if len(parts) < 2 {
		return nil, "", errors.New("invalid hash: too few parts")
	}

	header := parts[0]
	if len(header) < 2 {
		return nil, "", errors.New("invalid hash header")
	}

	prefix := string(header[0])
	version, _ := strconv.Atoi(header[1:])
	if version == 0 {
		version = 2
	}

	values := parts[1:]

	switch prefix {
	case "p":
		params := decodePropellerParams(values, version)
		result, genErr := GeneratePropeller(params)
		if genErr != nil {
			return nil, "", genErr
		}
		return result, "propeller", nil
	case "b":
		params := decodeBalloonParams(values, version)
		result, genErr := GenerateBalloon(params)
		if genErr != nil {
			return nil, "", genErr
		}
		return result, "balloon", nil
	case "h":
		params := decodeHullParams(values, version)
		result, genErr := GenerateHull(params)
		if genErr != nil {
			return nil, "", genErr
		}
		return result, "hull", nil
	default:
		return nil, "", errors.New("unknown generator type: " + prefix)
	}
}

func decodeBase64URL(s string) (string, error) {
	s = strings.ReplaceAll(s, "-", "+")
	s = strings.ReplaceAll(s, "_", "/")
	switch len(s) % 4 {
	case 2:
		s += "=="
	case 3:
		s += "="
	}
	data, err := base64.StdEncoding.DecodeString(s)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func decodeFieldValue(raw string, field schemaField, version int) interface{} {
	if raw == "" {
		// Absent field: the frontend dropped it because it equals the encode
		// default, so restore that default rather than leaving the param at its
		// zero value (which Validate would then clamp to a tiny minimum,
		// producing a fragmented preview instead of the intended build).
		return field.def
	}
	switch field.ft {
	case ftBool:
		return raw == "1"
	case ftInt:
		v, _ := strconv.Atoi(raw)
		return v
	case ftFloat:
		n, _ := strconv.ParseFloat(raw, 64)
		if version >= 2 {
			n = n / 100.0
		}
		return n
	case ftEnum:
		if field.enumMap != nil {
			if val, ok := field.enumMap[raw]; ok {
				return val
			}
		}
		return ""
	}
	return nil
}

func decodePropellerParams(values []string, version int) PropellerParams {
	p := PropellerParams{Version: version}
	for i, field := range schemaPropeller {
		raw := ""
		if i < len(values) {
			raw = values[i]
		}
		v := decodeFieldValue(raw, field, version)
		if v == nil {
			continue
		}
		switch field.key {
		case "blades":
			p.Blades = v.(int)
		case "length":
			p.Length = v.(int)
		case "rootChord":
			p.RootChord = v.(int)
		case "tipChord":
			p.TipChord = v.(int)
		case "sweepDegrees":
			p.SweepDegrees = v.(float64)
		case "swept":
			p.Swept = v.(bool)
		case "airfoilShape":
			p.AirfoilShape = v.(string)
		case "bladeMaterial":
			p.BladeMaterial = v.(string)
		case "bladeColor":
			p.BladeColor = v.(string)
		case "rotation":
			p.Rotation = v.(float64)
		case "orientation":
			p.Orientation = v.(string)
		}
	}
	return p
}

func decodeBalloonParams(values []string, version int) BalloonParams {
	p := BalloonParams{Version: version}
	for i, field := range schemaBalloon {
		raw := ""
		if i < len(values) {
			raw = values[i]
		}
		v := decodeFieldValue(raw, field, version)
		if v == nil {
			continue
		}
		switch field.key {
		case "lengthX":
			p.LengthX = v.(int)
		case "widthZ":
			p.WidthZ = v.(int)
		case "heightY":
			p.HeightY = v.(int)
		case "cylinderMid":
			p.CylinderMid = v.(float64)
		case "frontTaper":
			p.FrontTaper = v.(float64)
		case "rearTaper":
			p.RearTaper = v.(float64)
		case "topFlatten":
			p.TopFlatten = v.(float64)
		case "bottomFlatten":
			p.BottomFlatten = v.(float64)
		case "hollow":
			p.Hollow = v.(bool)
		case "shell":
			p.Shell = v.(int)
		case "ribEnabled":
			p.RibEnabled = v.(bool)
		case "ribSpacing":
			p.RibSpacing = v.(int)
		case "keelEnabled":
			p.KeelEnabled = v.(bool)
		case "keelDepth":
			p.KeelDepth = v.(int)
		case "finEnabled":
			p.FinEnabled = v.(bool)
		case "sideFinEnabled":
			p.SideFinEnabled = v.(bool)
		case "finHeight":
			p.FinHeight = v.(int)
		case "finLength":
			p.FinLength = v.(int)
		case "envelopeMaterial":
			p.EnvelopeMaterial = v.(string)
		case "envelopeColor":
			p.EnvelopeColor = v.(string)
		case "frameMaterial":
			p.FrameMaterial = v.(string)
		case "frameWoodType":
			p.FrameWoodType = v.(string)
		case "ribOffset":
			p.RibOffset = v.(int)
		}
	}
	return p
}

func decodeHullParams(values []string, version int) HullParams {
	p := HullParams{Version: version}
	for i, field := range schemaHull {
		raw := ""
		if i < len(values) {
			raw = values[i]
		}
		v := decodeFieldValue(raw, field, version)
		if v == nil {
			continue
		}
		switch field.key {
		case "woodType":
			p.WoodType = v.(string)
		case "length":
			p.Length = v.(int)
		case "beam":
			p.Beam = v.(int)
		case "depth":
			p.Depth = v.(int)
		case "bottomPinch":
			p.BottomPinch = v.(float64)
		case "hullFlare":
			p.HullFlare = v.(float64)
		case "flareCurve":
			p.FlareCurve = v.(float64)
		case "tumblehome":
			p.Tumblehome = v.(float64)
		case "tumbleCurve":
			p.TumbleCurve = v.(float64)
		case "sheerCurve":
			p.SheerCurve = v.(float64)
		case "sheerCurveExp":
			p.SheerCurveExp = v.(float64)
		case "bowLength":
			p.BowLength = v.(int)
		case "bowSharpness":
			p.BowSharpness = v.(float64)
		case "bowKeelRise":
			p.BowKeelRise = v.(float64)
		case "bowKeelLength":
			p.BowKeelLength = v.(int)
		case "sternStyle":
			p.SternStyle = v.(string)
		case "sternLength":
			p.SternLength = v.(int)
		case "sternSharpness":
			p.SternSharpness = v.(float64)
		case "sternKeelRise":
			p.SternKeelRise = v.(float64)
		case "sternKeelLength":
			p.SternKeelLength = v.(int)
		case "keelCurve":
			p.KeelCurve = v.(float64)
		case "castleBlend":
			p.CastleBlend = v.(int)
		case "hasRailings":
			p.HasRailings = v.(bool)
		case "hasTrim":
			p.HasTrim = v.(bool)
		case "hasWindows":
			p.HasWindows = v.(bool)
		case "castleHeight":
			p.CastleHeight = v.(int)
		case "castleLength":
			p.CastleLength = v.(int)
		case "forecastleHeight":
			p.ForecastleHeight = v.(int)
		case "forecastleLength":
			p.ForecastleLength = v.(int)
		case "hasGunPorts":
			p.HasGunPorts = v.(bool)
		case "gunPortRow":
			p.GunPortRow = v.(int)
		case "gunPortSpacing":
			p.GunPortSpacing = v.(int)
		case "bowCurve":
			p.BowCurve = v.(float64)
		case "sternOverhang":
			p.SternOverhang = v.(float64)
		case "midWidthBias":
			p.MidWidthBias = v.(float64)
		case "bowStyle":
			p.BowStyle = v.(string)
		case "deadrise":
			p.Deadrise = v.(float64)
		case "midFullness":
			p.MidFullness = v.(float64)
		case "bowSectionV":
			p.BowSectionV = v.(float64)
		case "sternFullness":
			p.SternFullness = v.(float64)
		case "stemRake":
			p.StemRake = v.(float64)
		case "stemCurve":
			p.StemCurve = v.(float64)
		case "sternRake":
			p.SternRake = v.(float64)
		case "rocker":
			p.Rocker = v.(float64)
		case "parallelMidbody":
			p.ParallelMidbody = v.(float64)
		case "stemPostHeight":
			p.StemPostHeight = v.(int)
		case "sternPostHeight":
			p.SternPostHeight = v.(int)
		case "doubleEnder":
			p.DoubleEnder = v.(bool)
		case "closedHull":
			p.ClosedHull = v.(bool)
		}
	}
	return p
}
