package pages

import (
	"createmod/internal/nbtparser"
	"createmod/internal/store"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"strings"

	"github.com/pocketbase/dbx"
	"github.com/pocketbase/pocketbase"
	"github.com/pocketbase/pocketbase/core"
	"github.com/pocketbase/pocketbase/tools/filesystem"
)

// tempUploadFile represents an additional NBT file attached to a temp upload.
type tempUploadFile struct {
	ID           string
	Token        string
	Filename     string
	Description  string
	Size         int64
	Checksum     string
	BlockCount   int
	DimX         int
	DimY         int
	DimZ         int
	Mods         []string
	Materials    []nbtparser.Material
	PBRecordID   string
	NBTStoredName string
}

// loadTempUploadFiles queries the temp_upload_files collection for all files
// belonging to the given token, ordered by creation time.
func loadTempUploadFiles(app *pocketbase.PocketBase, token string) []tempUploadFile {
	if app == nil || token == "" {
		return nil
	}
	coll, err := app.FindCollectionByNameOrId("temp_upload_files")
	if err != nil || coll == nil {
		return nil
	}
	recs, err := app.FindRecordsByFilter(coll.Id, "token = {:t}", "created", -1, 0, dbx.Params{"t": token})
	if err != nil || len(recs) == 0 {
		return nil
	}
	files := make([]tempUploadFile, 0, len(recs))
	for _, r := range recs {
		var materials []nbtparser.Material
		rawMat := r.Get("materials")
		if rawMat != nil {
			if b, err := json.Marshal(rawMat); err == nil {
				_ = json.Unmarshal(b, &materials)
			}
		}
		var mods []string
		rawMods := r.Get("mods")
		if rawMods != nil {
			if b, err := json.Marshal(rawMods); err == nil {
				_ = json.Unmarshal(b, &mods)
			}
		}
		files = append(files, tempUploadFile{
			ID:            r.Id,
			Token:         r.GetString("token"),
			Filename:      r.GetString("filename"),
			Description:   r.GetString("description"),
			Size:          int64(r.GetInt("size")),
			Checksum:      r.GetString("checksum"),
			BlockCount:    r.GetInt("block_count"),
			DimX:          r.GetInt("dim_x"),
			DimY:          r.GetInt("dim_y"),
			DimZ:          r.GetInt("dim_z"),
			Mods:          mods,
			Materials:     materials,
			PBRecordID:    r.Id,
			NBTStoredName: r.GetString("nbt_file"),
		})
	}
	return files
}

// addFileResponse is the JSON response for a successful additional file upload.
type addFileResponse struct {
	ID         string `json:"id"`
	Filename   string `json:"filename"`
	Description string `json:"description"`
	Size       int64  `json:"size"`
	BlockCount int    `json:"block_count"`
	DimX       int    `json:"dim_x"`
	DimY       int    `json:"dim_y"`
	DimZ       int    `json:"dim_z"`
	FileURL    string `json:"file_url,omitempty"`
}

// UploadAddFileHandler accepts a POST with an additional .nbt file and description
// for a given temp upload token. Requires auth + ownership.
func UploadAddFileHandler(app *pocketbase.PocketBase, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if !isAuthenticated(e) {
			return e.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		}

		token := e.Request.PathValue("token")
		if token == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "missing token"})
		}

		// Verify ownership: the parent temp upload must belong to this user
		parentEntry, ok := loadTempUploadPB(app, token)
		if !ok {
			return e.JSON(http.StatusNotFound, map[string]string{"error": "invalid or expired token"})
		}
		if parentEntry.UploadedBy == "" || parentEntry.UploadedBy != authenticatedUserID(e) {
			return e.JSON(http.StatusForbidden, map[string]string{"error": "you do not own this upload"})
		}

		_ = e.Request.ParseMultipartForm(maxUploadSize + 1<<20)

		file, header, err := e.Request.FormFile("nbt")
		if err != nil {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "missing NBT file in form (expected field 'nbt')"})
		}
		if file != nil {
			defer file.Close()
		}

		if header == nil || header.Filename == "" || !strings.HasSuffix(strings.ToLower(header.Filename), ".nbt") {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "invalid file type: expected .nbt"})
		}
		if header.Size > maxUploadSize {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "file too large: maximum size is 10MB"})
		}

		data, err := io.ReadAll(io.LimitReader(file, maxUploadSize+1))
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to read uploaded file"})
		}
		if int64(len(data)) > maxUploadSize {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "file too large: maximum size is 10MB"})
		}

		if ok, reason := nbtparser.Validate(data); !ok {
			msg := "invalid NBT file"
			if reason != "" {
				msg += ": " + reason
			}
			return e.JSON(http.StatusBadRequest, map[string]string{"error": msg})
		}

		n := int64(len(data))
		sum := sha256.Sum256(data)
		checksum := hex.EncodeToString(sum[:])

		// Duplicate detection within the same token
		existing := loadTempUploadFiles(app, token)
		for _, ef := range existing {
			if ef.Checksum == checksum {
				return e.JSON(http.StatusConflict, map[string]string{"error": "this file has already been added to this upload"})
			}
		}

		// Parse NBT stats
		blockCount, _, _ := nbtparser.ExtractStats(data)
		parsedMaterials, _ := nbtparser.ExtractMaterials(data)
		dimX, dimY, dimZ, _ := nbtparser.ExtractDimensions(data)

		// Extract mod namespaces
		modSet := make(map[string]struct{})
		for _, m := range parsedMaterials {
			parts := strings.SplitN(m.BlockID, ":", 2)
			if len(parts) == 2 && parts[0] != "minecraft" && parts[0] != "" {
				modSet[parts[0]] = struct{}{}
			}
		}
		mods := make([]string, 0, len(modSet))
		for mod := range modSet {
			mods = append(mods, mod)
		}

		description := strings.TrimSpace(e.Request.FormValue("description"))

		// Persist to PocketBase
		coll, err := app.FindCollectionByNameOrId("temp_upload_files")
		if err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": "temp_upload_files collection not found"})
		}

		rec := core.NewRecord(coll)
		rec.Set("token", token)
		rec.Set("filename", header.Filename)
		rec.Set("description", description)
		rec.Set("size", n)
		rec.Set("checksum", checksum)
		rec.Set("block_count", blockCount)
		rec.Set("dim_x", dimX)
		rec.Set("dim_y", dimY)
		rec.Set("dim_z", dimZ)
		if mods != nil {
			rec.Set("mods", mods)
		}
		if parsedMaterials != nil {
			rec.Set("materials", parsedMaterials)
		}

		f, fErr := filesystem.NewFileFromBytes(data, header.Filename)
		if fErr == nil {
			rec.Set("nbt_file", f)
		}

		if err := app.Save(rec); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to save file record"})
		}

		storedName := rec.GetString("nbt_file")
		var fileURL string
		if storedName != "" {
			fileURL = "/api/files/temp_upload_files/" + rec.Id + "/" + storedName
		}

		return e.JSON(http.StatusOK, addFileResponse{
			ID:          rec.Id,
			Filename:    header.Filename,
			Description: description,
			Size:        n,
			BlockCount:  blockCount,
			DimX:        dimX,
			DimY:        dimY,
			DimZ:        dimZ,
			FileURL:     fileURL,
		})
	}
}

// UploadDeleteFileHandler deletes an additional file from a temp upload.
// Requires auth + ownership.
func UploadDeleteFileHandler(app *pocketbase.PocketBase, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		if !isAuthenticated(e) {
			return e.JSON(http.StatusUnauthorized, map[string]string{"error": "authentication required"})
		}

		token := e.Request.PathValue("token")
		fileId := e.Request.PathValue("fileId")
		if token == "" || fileId == "" {
			return e.JSON(http.StatusBadRequest, map[string]string{"error": "missing token or file ID"})
		}

		// Verify ownership
		parentEntry, ok := loadTempUploadPB(app, token)
		if !ok {
			return e.JSON(http.StatusNotFound, map[string]string{"error": "invalid or expired token"})
		}
		if parentEntry.UploadedBy == "" || parentEntry.UploadedBy != authenticatedUserID(e) {
			return e.JSON(http.StatusForbidden, map[string]string{"error": "you do not own this upload"})
		}

		// Find the file record
		rec, err := app.FindRecordById("temp_upload_files", fileId)
		if err != nil {
			return e.JSON(http.StatusNotFound, map[string]string{"error": "file not found"})
		}

		// Verify the file belongs to this token
		if rec.GetString("token") != token {
			return e.JSON(http.StatusForbidden, map[string]string{"error": "file does not belong to this upload"})
		}

		if err := app.Delete(rec); err != nil {
			return e.JSON(http.StatusInternalServerError, map[string]string{"error": "failed to delete file"})
		}

		return e.NoContent(http.StatusNoContent)
	}
}

// UploadFileDownloadHandler serves an additional NBT file for download.
// Public access (no auth required), matching primary file behavior.
func UploadFileDownloadHandler(app *pocketbase.PocketBase, appStore *store.Store) func(e *core.RequestEvent) error {
	return func(e *core.RequestEvent) error {
		token := e.Request.PathValue("token")
		fileId := e.Request.PathValue("fileId")
		if token == "" || fileId == "" {
			return e.String(http.StatusBadRequest, "missing token or file ID")
		}

		rec, err := app.FindRecordById("temp_upload_files", fileId)
		if err != nil {
			return e.String(http.StatusNotFound, "file not found")
		}

		if rec.GetString("token") != token {
			return e.String(http.StatusNotFound, "file not found")
		}

		storedName := rec.GetString("nbt_file")
		if storedName == "" {
			return e.String(http.StatusNotFound, "file not available")
		}

		coll, err := app.FindCollectionByNameOrId("temp_upload_files")
		if err != nil {
			return e.String(http.StatusInternalServerError, "collection not found")
		}

		fileKey := coll.Id + "/" + rec.Id + "/" + storedName
		fsys, err := app.NewFilesystem()
		if err != nil {
			return e.String(http.StatusInternalServerError, "storage error")
		}
		defer fsys.Close()

		blob, err := fsys.GetReader(fileKey)
		if err != nil {
			return e.String(http.StatusNotFound, "file not found in storage")
		}
		defer blob.Close()

		filename := rec.GetString("filename")
		e.Response.Header().Set("Content-Disposition", "attachment; filename=\""+filename+"\"")
		e.Response.Header().Set("Content-Type", "application/octet-stream")
		return e.Stream(http.StatusOK, "application/octet-stream", blob)
	}
}
