package pages

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/schematic"
	"createmod/internal/server"
	"createmod/internal/store"
)

// SafetyBadgeData drives the per-schematic safety badge. Two separate,
// accurate claims: FileSafe (valid NBT, passed hardening) and the manifest
// (what the content inspection found). "Contains command blocks" is a
// surfaced fact, not a failure state.
type SafetyBadgeData struct {
	Scanned    bool
	FileSafe   bool
	Notable    bool
	ParseError string
	Summary    string // e.g. "2 command blocks, 1 spawner"
	Findings   []schematic.Finding
	Truncated  bool
	ScannedAt  time.Time
}

var findingLabels = map[schematic.FindingType]string{
	schematic.FindingCommandBlock:  "command block(s)",
	schematic.FindingSpawner:       "mob spawner(s)",
	schematic.FindingSignCommand:   "sign click command(s)",
	schematic.FindingBookOrItemCmd: "command(s) in items",
	schematic.FindingStructureBlk:  "structure block(s)",
	schematic.FindingJigsaw:        "jigsaw block(s)",
}

// findingOrder keeps summaries deterministic.
var findingOrder = []schematic.FindingType{
	schematic.FindingCommandBlock,
	schematic.FindingSpawner,
	schematic.FindingSignCommand,
	schematic.FindingBookOrItemCmd,
	schematic.FindingStructureBlk,
	schematic.FindingJigsaw,
}

// safetyBadgeFor loads and shapes the badge model for a schematic page.
// Returns zero-value (Scanned=false) when the pipeline has not run yet.
func safetyBadgeFor(ctx context.Context, appStore *store.Store, schematicID string) SafetyBadgeData {
	var d SafetyBadgeData
	if appStore == nil {
		return d
	}
	row, err := appStore.SchematicSafety.GetBySchematicID(ctx, schematicID)
	if err != nil || row == nil {
		return d
	}
	d.Scanned = true
	d.FileSafe = row.FileSafe
	d.ScannedAt = row.ScannedAt

	var m schematic.Manifest
	if len(row.Manifest) > 0 {
		if err := json.Unmarshal(row.Manifest, &m); err == nil {
			d.Notable = m.Notable()
			d.Findings = m.Findings
			d.Truncated = m.FindingsTruncated
			d.Summary = manifestSummary(&m)
		}
		if !row.FileSafe {
			var failure struct {
				ParseError string `json:"parseError"`
			}
			if err := json.Unmarshal(row.Manifest, &failure); err == nil {
				d.ParseError = failure.ParseError
			}
		}
	}
	return d
}

func manifestSummary(m *schematic.Manifest) string {
	if m.Counts == nil {
		return ""
	}
	out := ""
	for _, t := range findingOrder {
		n := m.Counts[t]
		if n == 0 {
			continue
		}
		if out != "" {
			out += ", "
		}
		label, ok := findingLabels[t]
		if !ok {
			label = string(t)
		}
		out += itoa(n) + " " + label
	}
	return out
}

func itoa(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var b [20]byte
	i := len(b)
	for n > 0 {
		i--
		b[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		b[i] = '-'
	}
	return string(b[i:])
}

var safetyTemplates = append([]string{
	"./template/safety.html",
}, commonTemplates...)

type safetyPageData struct {
	DefaultData
}

// SafetyExplainerHandler renders /safety — what the validation badge means,
// what schematics can and cannot do.
func SafetyExplainerHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		setPublicCacheControl(e, 600)
		d := safetyPageData{}
		d.Populate(e)
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		d.Title = i18n.T(d.Language, "Are Minecraft Schematics Safe? How CreateMod.com Validates Files")
		d.Description = i18n.T(d.Language, "Schematics are data, not programs. Learn how CreateMod.com hardens uploads, inspects content for command blocks and spawners, and what the Validated badge means.")
		d.Slug = "/safety"
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Safety"))
		html, err := registry.LoadFiles(safetyTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

var safetyCheckTemplates = append([]string{
	"./template/safety_check.html",
}, commonTemplates...)

// SafetyCheckToolHandler renders the stateless check-a-file page.
func SafetyCheckToolHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		setPublicCacheControl(e, 600)
		d := safetyPageData{}
		d.Populate(e)
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		d.Title = i18n.T(d.Language, "Minecraft Schematic Safety Checker - Scan for Command Blocks")
		d.Description = i18n.T(d.Language, "Free online schematic safety check: upload a .nbt, .schem or .litematic file and see exactly what is inside - command blocks, spawners, sign commands - before you use it.")
		d.Slug = "/tools/safety-check"
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Tools"), "/generators", i18n.T(d.Language, "Safety Check"))
		html, err := registry.LoadFiles(safetyCheckTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}

// SafetyCheckAPIHandler inspects an uploaded file and returns the manifest.
// Stateless: nothing is stored.
// POST /api/safety-check (multipart: file)
func SafetyCheckAPIHandler() func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		if err := e.Request.ParseMultipartForm(maxUploadSize + 1<<20); err != nil {
			return writeJSON(e, http.StatusBadRequest, map[string]string{"error": "invalid form"})
		}
		file, header, err := e.Request.FormFile("file")
		if err != nil {
			return writeJSON(e, http.StatusBadRequest, map[string]string{"error": "missing file"})
		}
		defer file.Close()
		if header.Size > maxUploadSize {
			return writeJSON(e, http.StatusRequestEntityTooLarge, map[string]string{"error": "file exceeds 10 MB"})
		}
		data, err := io.ReadAll(io.LimitReader(file, maxUploadSize+1))
		if err != nil || int64(len(data)) > maxUploadSize {
			return writeJSON(e, http.StatusRequestEntityTooLarge, map[string]string{"error": "file exceeds 10 MB"})
		}

		resp := map[string]interface{}{"fileSafe": false}
		format, err := schematic.Detect(data)
		if err != nil {
			resp["parseError"] = convertUserError(err)
			return writeJSON(e, http.StatusOK, resp)
		}
		resp["format"] = string(format)
		resp["label"] = convertLabel(format)
		s, err := schematic.Read(data, format)
		if err != nil {
			resp["parseError"] = convertUserError(err)
			return writeJSON(e, http.StatusOK, resp)
		}
		resp["fileSafe"] = true
		caps := s.Capabilities()
		resp["size"] = caps.Size
		resp["blockCount"] = caps.BlockCount
		m := schematic.Inspect(s)
		resp["manifest"] = m
		resp["notable"] = m.Notable()
		resp["summary"] = manifestSummary(m)
		return writeJSON(e, http.StatusOK, resp)
	}
}
