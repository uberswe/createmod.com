package pages

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math"
	"net/http"
	"strings"

	"createmod/internal/cache"
	"createmod/internal/i18n"
	"createmod/internal/schematic"
	"createmod/internal/server"
	"createmod/internal/storage"
	"createmod/internal/store"
)

// The editor is server-authoritative: a session references its source
// (published schematic, temp upload, or a blank canvas) plus an op log with
// an undo cursor. Every view is a replay of ops[0:cursor] over the source —
// undo/redo just move the cursor. Publishing hands the current NBT to the
// existing upload pipeline, so validation, dedup, moderation and safety
// scanning all apply unchanged.

const editorBlankMaxDim = 128

// editorSourceModel loads and parses a session's source model.
func editorSourceModel(ctx context.Context, appStore *store.Store, storageSvc *storage.Service, sess *store.EditorSession) (*schematic.Schematic, error) {
	switch sess.SourceKind {
	case "blank":
		var dims [3]int
		if err := json.Unmarshal([]byte(sess.SourceRef), &dims); err != nil {
			return nil, fmt.Errorf("bad blank canvas size")
		}
		for a := 0; a < 3; a++ {
			if dims[a] < 1 || dims[a] > editorBlankMaxDim {
				return nil, fmt.Errorf("blank canvas must be 1-%d per axis", editorBlankMaxDim)
			}
		}
		s := schematic.New(dims[0], dims[1], dims[2])
		s.DataVersion = 3955
		return s, nil
	case "schematic":
		s, err := appStore.Schematics.GetByName(ctx, sess.SourceRef)
		if err != nil || s == nil || !store.IsPublicState(s.ModerationState) || (s.Deleted != nil && !s.Deleted.IsZero()) {
			return nil, fmt.Errorf("source schematic not found")
		}
		primary := strings.TrimSpace(s.SchematicFile)
		if primary == "" || storageSvc == nil {
			return nil, fmt.Errorf("source schematic has no file")
		}
		reader, err := storageSvc.Download(ctx, storage.CollectionPrefix("schematics"), s.ID, primary)
		if err != nil {
			return nil, fmt.Errorf("source file unavailable")
		}
		defer reader.Close()
		data, err := io.ReadAll(io.LimitReader(reader, maxUploadSize+1))
		if err != nil {
			return nil, err
		}
		return schematic.ReadStructureNBT(data)
	case "upload":
		tu, err := appStore.TempUploads.GetByToken(ctx, sess.SourceRef)
		if err != nil || tu == nil || storageSvc == nil {
			return nil, fmt.Errorf("source upload not found")
		}
		reader, err := storageSvc.DownloadRaw(ctx, tu.NbtS3Key)
		if err != nil {
			return nil, fmt.Errorf("source file unavailable")
		}
		defer reader.Close()
		data, err := io.ReadAll(io.LimitReader(reader, maxUploadSize+1))
		if err != nil {
			return nil, err
		}
		return schematic.ReadStructureNBT(data)
	default:
		return nil, fmt.Errorf("unknown source kind")
	}
}

// editorCurrentModel replays the active op prefix.
func editorCurrentModel(ctx context.Context, appStore *store.Store, storageSvc *storage.Service, sess *store.EditorSession) (*schematic.Schematic, []schematic.Op, error) {
	src, err := editorSourceModel(ctx, appStore, storageSvc, sess)
	if err != nil {
		return nil, nil, err
	}
	var ops []schematic.Op
	if len(sess.Ops) > 0 {
		if err := json.Unmarshal(sess.Ops, &ops); err != nil {
			return nil, nil, fmt.Errorf("corrupt op log")
		}
	}
	if sess.Cursor < 0 || sess.Cursor > len(ops) {
		return nil, nil, fmt.Errorf("corrupt op cursor")
	}
	cur, err := schematic.ApplyOps(src, ops[:sess.Cursor])
	if err != nil {
		return nil, nil, err
	}
	return cur, ops, nil
}

type editorStateView struct {
	ID         string                    `json:"id"`
	Size       [3]int                    `json:"size"`
	BlockCount int                       `json:"blockCount"`
	Ops        []schematic.Op            `json:"ops"`
	Cursor     int                       `json:"cursor"`
	Materials  []schematic.MaterialCount `json:"materials"`
	Palette    []string                  `json:"palette"`
}

func editorState(sess *store.EditorSession, model *schematic.Schematic, ops []schematic.Op) editorStateView {
	palette := make([]string, 0, len(model.Palette))
	seen := map[string]bool{}
	for _, st := range model.Palette {
		if st.IsAir() || seen[st.Name] {
			continue
		}
		seen[st.Name] = true
		palette = append(palette, st.Name)
	}
	if ops == nil {
		ops = []schematic.Op{}
	}
	return editorStateView{
		ID:         sess.ID,
		Size:       model.Size,
		BlockCount: model.BlockCount(),
		Ops:        ops,
		Cursor:     sess.Cursor,
		Materials:  model.Materials(),
		Palette:    palette,
	}
}

// EditorCreateSessionHandler creates a session.
// POST /api/editor/sessions {"source":"schematic","ref":"name"} |
// {"source":"upload","ref":"token"} | {"source":"blank","dims":[x,y,z]}
func EditorCreateSessionHandler(appStore *store.Store, storageSvc *storage.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		var req struct {
			Source string `json:"source"`
			Ref    string `json:"ref"`
			Dims   [3]int `json:"dims"`
		}
		if err := json.NewDecoder(io.LimitReader(e.Request.Body, 4096)).Decode(&req); err != nil {
			return writeJSON(e, http.StatusBadRequest, map[string]string{"error": "invalid request"})
		}
		ref := strings.TrimSpace(req.Ref)
		switch req.Source {
		case "blank":
			refBytes, _ := json.Marshal(req.Dims)
			ref = string(refBytes)
		case "schematic", "upload":
			if ref == "" {
				return writeJSON(e, http.StatusBadRequest, map[string]string{"error": "missing source ref"})
			}
		default:
			return writeJSON(e, http.StatusBadRequest, map[string]string{"error": "unknown source"})
		}
		sess := &store.EditorSession{SourceKind: req.Source, SourceRef: ref}
		// Validate the source parses before creating the session.
		if _, err := editorSourceModel(e.Request.Context(), appStore, storageSvc, sess); err != nil {
			return writeJSON(e, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		}
		id, err := appStore.EditorSessions.Create(e.Request.Context(), authenticatedUserID(e), req.Source, ref)
		if err != nil {
			return writeJSON(e, http.StatusInternalServerError, map[string]string{"error": "failed to create session"})
		}
		return writeJSON(e, http.StatusOK, map[string]string{"id": id})
	}
}

func loadEditorSession(e *server.RequestEvent, appStore *store.Store) (*store.EditorSession, error) {
	id := e.Request.PathValue("id")
	if id == "" || len(id) > 64 {
		return nil, fmt.Errorf("missing session id")
	}
	sess, err := appStore.EditorSessions.GetByID(e.Request.Context(), id)
	if err != nil || sess == nil {
		return nil, fmt.Errorf("session not found")
	}
	// Sessions created by a logged-in user belong to that user. Anonymous
	// sessions are capability-by-UUID (122-bit random id), the same model
	// as temp upload tokens and modify previews.
	if sess.UserID != "" && sess.UserID != authenticatedUserID(e) {
		return nil, fmt.Errorf("session not found")
	}
	return sess, nil
}

// EditorStateHandler returns the session state (ops, cursor, materials).
// GET /api/editor/{id}
func EditorStateHandler(appStore *store.Store, storageSvc *storage.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		sess, err := loadEditorSession(e, appStore)
		if err != nil {
			return writeJSON(e, http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		model, ops, err := editorCurrentModel(e.Request.Context(), appStore, storageSvc, sess)
		if err != nil {
			return writeJSON(e, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		}
		return writeJSON(e, http.StatusOK, editorState(sess, model, ops))
	}
}

// EditorOpHandler appends an operation (truncating any redo tail).
// POST /api/editor/{id}/op {op}
func EditorOpHandler(appStore *store.Store, storageSvc *storage.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		sess, err := loadEditorSession(e, appStore)
		if err != nil {
			return writeJSON(e, http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		var op schematic.Op
		if err := json.NewDecoder(io.LimitReader(e.Request.Body, 64*1024)).Decode(&op); err != nil {
			return writeJSON(e, http.StatusBadRequest, map[string]string{"error": "invalid operation"})
		}
		model, ops, err := editorCurrentModel(e.Request.Context(), appStore, storageSvc, sess)
		if err != nil {
			return writeJSON(e, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		}
		next, err := schematic.ApplyOp(model, op)
		if err != nil {
			return writeJSON(e, http.StatusUnprocessableEntity, map[string]string{"error": convertUserError(err)})
		}
		newOps := append(append([]schematic.Op{}, ops[:sess.Cursor]...), op)
		if len(newOps) > schematic.MaxOpsPerSession {
			return writeJSON(e, http.StatusUnprocessableEntity, map[string]string{"error": "operation limit reached for this session"})
		}
		opsJSON, err := json.Marshal(newOps)
		if err != nil {
			return writeJSON(e, http.StatusInternalServerError, map[string]string{"error": "failed to save"})
		}
		sess.Cursor = len(newOps)
		if err := appStore.EditorSessions.UpdateOps(e.Request.Context(), sess.ID, opsJSON, sess.Cursor); err != nil {
			return writeJSON(e, http.StatusInternalServerError, map[string]string{"error": "failed to save"})
		}
		return writeJSON(e, http.StatusOK, editorState(sess, next, newOps))
	}
}

// EditorUndoRedoHandler moves the cursor.
// POST /api/editor/{id}/undo | /api/editor/{id}/redo
func EditorUndoRedoHandler(appStore *store.Store, storageSvc *storage.Service, redo bool) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		sess, err := loadEditorSession(e, appStore)
		if err != nil {
			return writeJSON(e, http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		var ops []schematic.Op
		if len(sess.Ops) > 0 {
			_ = json.Unmarshal(sess.Ops, &ops)
		}
		if redo {
			if sess.Cursor < len(ops) {
				sess.Cursor++
			}
		} else if sess.Cursor > 0 {
			sess.Cursor--
		}
		if err := appStore.EditorSessions.UpdateOps(e.Request.Context(), sess.ID, sess.Ops, sess.Cursor); err != nil {
			return writeJSON(e, http.StatusInternalServerError, map[string]string{"error": "failed to save"})
		}
		model, _, err := editorCurrentModel(e.Request.Context(), appStore, storageSvc, sess)
		if err != nil {
			return writeJSON(e, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		}
		return writeJSON(e, http.StatusOK, editorState(sess, model, ops))
	}
}

// EditorPreviewNBTHandler serves the current NBT (CORS-open so external
// viewers like Bloxelizer can fetch it, same mechanism as modify previews).
// GET /api/editor/{id}/preview.nbt
func EditorPreviewNBTHandler(appStore *store.Store, storageSvc *storage.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		sess, err := loadEditorSession(e, appStore)
		if err != nil {
			return e.String(http.StatusNotFound, err.Error())
		}
		model, _, err := editorCurrentModel(e.Request.Context(), appStore, storageSvc, sess)
		if err != nil {
			return e.String(http.StatusUnprocessableEntity, err.Error())
		}
		data, err := schematic.WriteStructureNBT(model)
		if err != nil {
			return e.String(http.StatusInternalServerError, "failed to serialize")
		}
		// CORS-open like modify previews: external viewers (Bloxelizer)
		// fetch this URL. The unguessable session id is the access control;
		// Referrer-Policy strict-origin-when-cross-origin keeps it out of
		// cross-origin referers.
		e.Response.Header().Set("Access-Control-Allow-Origin", "*")
		e.Response.Header().Set("Content-Disposition", "attachment; filename=\"edited.nbt\"")
		return e.Blob(http.StatusOK, "application/octet-stream", data)
	}
}

// editorPreviewBlockCap bounds the 3D preview payload.
const editorPreviewBlockCap = 250000

// EditorPreviewJSONHandler serves blocks for the in-page 3D view (the
// generator renderer's format: coarse type mapping, one tone).
// GET /api/editor/{id}/preview.json
func EditorPreviewJSONHandler(appStore *store.Store, storageSvc *storage.Service) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		sess, err := loadEditorSession(e, appStore)
		if err != nil {
			return writeJSON(e, http.StatusNotFound, map[string]string{"error": err.Error()})
		}
		model, _, err := editorCurrentModel(e.Request.Context(), appStore, storageSvc, sess)
		if err != nil {
			return writeJSON(e, http.StatusUnprocessableEntity, map[string]string{"error": err.Error()})
		}
		type previewBlock struct {
			X     int               `json:"x"`
			Y     int               `json:"y"`
			Z     int               `json:"z"`
			Type  int               `json:"type"`
			Color int               `json:"color"`
			Props map[string]string `json:"props,omitempty"`
		}
		blocks := make([]previewBlock, 0, 4096)
		truncated := false
		for y := 0; y < model.Size[1] && !truncated; y++ {
			for z := 0; z < model.Size[2] && !truncated; z++ {
				for x := 0; x < model.Size[0]; x++ {
					st := model.Palette[model.Blocks[model.Index(x, y, z)]]
					if st.IsAir() {
						continue
					}
					if len(blocks) >= editorPreviewBlockCap {
						truncated = true
						break
					}
					blocks = append(blocks, previewBlock{X: x, Y: y, Z: z, Type: previewTypeFor(st), Color: previewColorFor(st), Props: previewPropsFor(st)})
				}
			}
		}
		return writeJSON(e, http.StatusOK, map[string]interface{}{
			"blocks":    blocks,
			"materials": map[string]string{"woodType": "oak"},
			"truncated": truncated,
			"size":      model.Size,
			// The generator renderer reads sizeX/Y/Z, not the size array.
			"sizeX": model.Size[0],
			"sizeY": model.Size[1],
			"sizeZ": model.Size[2],
		})
	}
}

// previewTypeFor maps blocks onto the generator renderer's shape enum:
// 4=stair (with facing/half/shape props), 2=bottom slab, 3=top slab,
// generic cubes for everything else.
func previewTypeFor(st schematic.BlockState) int {
	switch schematic.BlockFamily(st.Name) {
	case "stairs":
		return 4
	case "slabs":
		if st.Properties["type"] == "top" {
			return 3
		}
		return 2
	default:
		return 1
	}
}

// previewDyeColors are Minecraft's sixteen dye tones, used for wool,
// concrete, terracotta, carpet and stained glass in the preview.
var previewDyeColors = map[string]int{
	"white": 0xe8e8e8, "orange": 0xf07613, "magenta": 0xbd44b3, "light_blue": 0x3aafd9,
	"yellow": 0xf8c527, "lime": 0x70b919, "pink": 0xed8dac, "gray": 0x3e4447,
	"light_gray": 0x8e8e86, "cyan": 0x158991, "purple": 0x792aac, "blue": 0x35399d,
	"brown": 0x724728, "green": 0x546d1b, "red": 0xa12722, "black": 0x141519,
}

// previewBlockColors gives common blocks a recognizable tone.
var previewBlockColors = map[string]int{
	"stone": 0x7d7d7d, "cobblestone": 0x797979, "stone_bricks": 0x777777,
	"deepslate": 0x4c4c4c, "andesite": 0x888888, "granite": 0x9a6b57, "diorite": 0xc9c9c6,
	"dirt": 0x8a5f3b, "grass_block": 0x5d923a, "sand": 0xdbcf9c, "gravel": 0x84807d,
	"oak_planks": 0xb8945f, "spruce_planks": 0x6b4226, "birch_planks": 0xd5c98c,
	"dark_oak_planks": 0x3e2912, "jungle_planks": 0xb88764, "acacia_planks": 0xa85632,
	"cherry_planks": 0xe8c4b8, "crimson_planks": 0x6b3344, "warped_planks": 0x2b6b5e,
	"oak_log": 0x6b5839, "spruce_log": 0x3a2718, "birch_log": 0xd5cda1,
	"glass": 0xc9e8ea, "water": 0x3a56d9, "lava": 0xd96514, "obsidian": 0x15121e,
	"iron_block": 0xd8d8d8, "gold_block": 0xf5cd30, "copper_block": 0xc06843,
	"brass_block": 0xd1a866, "brass_casing": 0xb08d4e, "andesite_casing": 0x9a9484,
}

// previewColorFor returns an RGB tone for the 3D preview so different block
// types are visually distinct: dye-colored blocks use their dye, common
// blocks use hand-picked tones, and everything else gets a stable color
// derived from a hash of the block name.
func previewColorFor(st schematic.BlockState) int {
	name := st.Name
	if i := strings.IndexByte(name, ':'); i >= 0 {
		name = name[i+1:]
	}
	if c, ok := previewBlockColors[name]; ok {
		return c
	}
	for dye, c := range previewDyeColors {
		if strings.HasPrefix(name, dye+"_") {
			return c
		}
	}
	// FNV-1a hash of the name → hue, at fixed saturation/lightness.
	var h uint32 = 2166136261
	for i := 0; i < len(name); i++ {
		h ^= uint32(name[i])
		h *= 16777619
	}
	return hslToRGB(float64(h%360), 0.45, 0.55)
}

// hslToRGB converts HSL (h in degrees, s/l in 0..1) to a packed RGB int.
func hslToRGB(h, s, l float64) int {
	c := (1 - math.Abs(2*l-1)) * s
	x := c * (1 - math.Abs(math.Mod(h/60, 2)-1))
	m := l - c/2
	var r, g, b float64
	switch {
	case h < 60:
		r, g, b = c, x, 0
	case h < 120:
		r, g, b = x, c, 0
	case h < 180:
		r, g, b = 0, c, x
	case h < 240:
		r, g, b = 0, x, c
	case h < 300:
		r, g, b = x, 0, c
	default:
		r, g, b = c, 0, x
	}
	ri := int((r + m) * 255)
	gi := int((g + m) * 255)
	bi := int((b + m) * 255)
	return ri<<16 | gi<<8 | bi
}

func previewPropsFor(st schematic.BlockState) map[string]string {
	switch schematic.BlockFamily(st.Name) {
	case "stairs":
		out := map[string]string{"facing": "south", "half": "bottom"}
		if v := st.Properties["facing"]; v != "" {
			out["facing"] = v
		}
		if v := st.Properties["half"]; v != "" {
			out["half"] = v
		}
		if v := st.Properties["shape"]; v != "" {
			out["shape"] = v
		}
		return out
	case "slabs":
		if st.Properties["type"] == "top" {
			return map[string]string{"type": "top"}
		}
		return nil
	default:
		return nil
	}
}

var editorTemplates = append([]string{
	"./template/editor.html",
}, commonTemplates...)

type editorPageData struct {
	DefaultData
	SourceName string // prefill: schematic name from ?source=
}

// EditorPageHandler renders /tools/editor.
func EditorPageHandler(registry *server.Registry, cacheService *cache.Service, appStore *store.Store) func(e *server.RequestEvent) error {
	return func(e *server.RequestEvent) error {
		d := editorPageData{}
		d.Populate(e)
		d.Categories = allCategoriesFromStoreOnly(appStore, cacheService)
		d.Title = i18n.T(d.Language, "Schematic Editor - Crop, Rotate, Mirror and Replace Blocks Online")
		d.Description = i18n.T(d.Language, "Edit Minecraft schematics in your browser: crop, expand, rotate, mirror, fill, hollow, replace blocks and delete regions - with undo, live 3D preview and one-click publishing to CreateMod.com.")
		d.Slug = "/tools/editor"
		d.Breadcrumbs = NewBreadcrumbs(d.Language, i18n.T(d.Language, "Tools"), "/generators", i18n.T(d.Language, "Editor"))
		d.SourceName = e.Request.URL.Query().Get("source")
		html, err := registry.LoadFiles(editorTemplates...).Render(d)
		if err != nil {
			return err
		}
		return e.HTML(http.StatusOK, html)
	}
}
