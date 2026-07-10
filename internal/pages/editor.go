package pages

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
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
					blocks = append(blocks, previewBlock{X: x, Y: y, Z: z, Type: previewTypeFor(st), Props: previewPropsFor(st)})
				}
			}
		}
		return writeJSON(e, http.StatusOK, map[string]interface{}{
			"blocks":    blocks,
			"materials": map[string]string{"woodType": "oak"},
			"truncated": truncated,
			"size":      model.Size,
		})
	}
}

// previewTypeFor maps blocks onto the generator renderer's coarse enum:
// 1=cube, 2=stair, 3=slab (shape fidelity for the common partial blocks,
// generic cubes for everything else).
func previewTypeFor(st schematic.BlockState) int {
	switch schematic.BlockFamily(st.Name) {
	case "stairs":
		return 2
	case "slabs":
		return 3
	default:
		return 1
	}
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
