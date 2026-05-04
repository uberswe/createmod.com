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
}

var schemaPropeller = []schemaField{
	{"blades", ftInt, nil},
	{"length", ftInt, nil},
	{"rootChord", ftInt, nil},
	{"tipChord", ftInt, nil},
	{"sweepDegrees", ftFloat, nil},
	{"swept", ftBool, nil},
	{"airfoilShape", ftEnum, codeToAirfoilShape},
	{"bladeMaterial", ftEnum, codeToBladeMaterial},
	{"bladeColor", ftEnum, codeToColor},
	{"rotation", ftFloat, nil},
	{"orientation", ftEnum, codeToOrientation},
}

var schemaBalloon = []schemaField{
	{"lengthX", ftInt, nil},
	{"widthZ", ftInt, nil},
	{"heightY", ftInt, nil},
	{"cylinderMid", ftFloat, nil},
	{"frontTaper", ftFloat, nil},
	{"rearTaper", ftFloat, nil},
	{"topFlatten", ftFloat, nil},
	{"bottomFlatten", ftFloat, nil},
	{"hollow", ftBool, nil},
	{"shell", ftInt, nil},
	{"ribEnabled", ftBool, nil},
	{"ribSpacing", ftInt, nil},
	{"keelEnabled", ftBool, nil},
	{"keelDepth", ftInt, nil},
	{"finEnabled", ftBool, nil},
	{"sideFinEnabled", ftBool, nil},
	{"finHeight", ftInt, nil},
	{"finLength", ftInt, nil},
	{"envelopeMaterial", ftEnum, codeToEnvelopeMaterial},
	{"envelopeColor", ftEnum, codeToColor},
	{"frameMaterial", ftEnum, codeToFrameMaterial},
	{"frameWoodType", ftEnum, codeToWood},
}

var schemaHull = []schemaField{
	{"woodType", ftEnum, codeToWood},
	{"length", ftInt, nil},
	{"beam", ftInt, nil},
	{"depth", ftInt, nil},
	{"bottomPinch", ftFloat, nil},
	{"hullFlare", ftFloat, nil},
	{"flareCurve", ftFloat, nil},
	{"tumblehome", ftFloat, nil},
	{"tumbleCurve", ftFloat, nil},
	{"sheerCurve", ftFloat, nil},
	{"sheerCurveExp", ftFloat, nil},
	{"bowLength", ftInt, nil},
	{"bowSharpness", ftFloat, nil},
	{"bowKeelRise", ftFloat, nil},
	{"bowKeelLength", ftInt, nil},
	{"sternStyle", ftEnum, codeToSternStyle},
	{"sternLength", ftInt, nil},
	{"sternSharpness", ftFloat, nil},
	{"sternKeelRise", ftFloat, nil},
	{"sternKeelLength", ftInt, nil},
	{"keelCurve", ftFloat, nil},
	{"castleBlend", ftInt, nil},
	{"hasRailings", ftBool, nil},
	{"hasTrim", ftBool, nil},
	{"hasWindows", ftBool, nil},
	{"castleHeight", ftInt, nil},
	{"castleLength", ftInt, nil},
	{"forecastleHeight", ftInt, nil},
	{"forecastleLength", ftInt, nil},
	{"hasGunPorts", ftBool, nil},
	{"gunPortRow", ftInt, nil},
	{"gunPortSpacing", ftInt, nil},
	{"bowCurve", ftFloat, nil},
	{"sternOverhang", ftFloat, nil},
	{"midWidthBias", ftFloat, nil},
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
		return nil
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
		}
	}
	return p
}
