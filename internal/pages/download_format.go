package pages

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"createmod/internal/i18n"
	"createmod/internal/schematic"
	"createmod/internal/server"
	"createmod/internal/storage"
	"createmod/internal/store"
)

// convCacheVersion is baked into _conv cache keys; bump to invalidate all
// cached format conversions at once.
const convCacheVersion = "v1"

// convCacheKey builds the S3 key for a cached conversion. The schematic's
// Updated timestamp is part of the key, so edits naturally miss the cache
// (stale keys are removed by TTL cleanup).
func convCacheKey(schematicID string, updated time.Time, base, ext string) string {
	return fmt.Sprintf("_conv/%s/%s/%d/%s%s", convCacheVersion, schematicID, updated.Unix(), base, ext)
}

// serveConvertedSchematic streams the schematic's primary file converted to
// the requested format, using the S3 conversion cache. formatSlug must be a
// validated convertFormats slug other than "nbt".
func serveConvertedSchematic(e *server.RequestEvent, storageSvc *storage.Service, s *store.Schematic, formatSlug string) error {
	target, ext, ok := convertFormatBySlug(formatSlug)
	if !ok || formatSlug == "nbt" {
		return e.String(http.StatusBadRequest, "unsupported format")
	}
	if storageSvc == nil {
		return e.String(http.StatusServiceUnavailable, "file storage is not available")
	}
	primary := strings.TrimSpace(s.SchematicFile)
	if primary == "" {
		return e.String(http.StatusNotFound, "schematic has no file")
	}
	base := strings.TrimSuffix(filepath.Base(primary), filepath.Ext(primary))
	outName := sanitizeContentDispositionFilename(base + ext)
	ctx := e.Request.Context()

	serve := func(data []byte, warnings []schematic.Warning) error {
		if len(warnings) > 0 {
			if wj, err := json.Marshal(warnings); err == nil {
				e.Response.Header().Set("X-Conversion-Warnings", string(wj))
			}
		}
		e.Response.Header().Set("Content-Disposition", "attachment; filename=\""+outName+"\"")
		return e.Blob(http.StatusOK, "application/octet-stream", data)
	}

	key := convCacheKey(s.ID, s.Updated, base, ext)
	if exists, _ := storageSvc.ExistsRaw(ctx, key); exists {
		reader, err := storageSvc.DownloadRaw(ctx, key)
		if err == nil {
			defer reader.Close()
			data, err := io.ReadAll(reader)
			if err == nil {
				// Warnings are deterministic per (file, format); recompute
				// them cheaply only for the lossy legacy target.
				return serve(data, cachedFormatWarnings(target))
			}
		}
		// fall through to regenerate on any cache read error
	}

	src, err := storageSvc.Download(ctx, storage.CollectionPrefix("schematics"), s.ID, primary)
	if err != nil {
		return e.String(http.StatusNotFound, "schematic file not found")
	}
	defer src.Close()
	data, err := io.ReadAll(io.LimitReader(src, maxUploadSize+1))
	if err != nil || int64(len(data)) > maxUploadSize {
		return e.String(http.StatusInternalServerError, "failed to read schematic file")
	}

	res, err := schematic.Convert(data, target)
	if err != nil {
		return e.String(http.StatusUnprocessableEntity, "this schematic cannot be converted: "+convertUserError(err))
	}

	// Cache in the background with a fresh context (the request context is
	// cancelled once the response is written).
	out := res.Data
	go func() {
		bgCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		_ = storageSvc.UploadRawBytes(bgCtx, key, out, "application/octet-stream")
	}()

	return serve(res.Data, res.Warnings)
}

// cachedFormatWarnings mirrors the deterministic warnings Convert would
// produce for a cache hit, where the source is a known-good Create .nbt.
func cachedFormatWarnings(target schematic.Format) []schematic.Warning {
	switch target {
	case schematic.FormatLegacy:
		return []schematic.Warning{{Message: "legacy .schematic export collapses modern blockstates to pre-1.13 id:meta pairs"}}
	case schematic.FormatBG:
		return []schematic.Warning{{Message: "Building Gadgets templates cannot carry block entity data (chest contents, Create kinetics)"}}
	}
	return nil
}

// DownloadSplitItem is one entry in the shared download split-button menu.
type DownloadSplitItem struct {
	Label          string
	Href           string
	Note           string
	Lossy          bool
	LossyTitle     string
	Disabled       bool
	DisabledReason string
	Download       bool
}

// DownloadSplitData feeds template/include/download_split.html.
type DownloadSplitData struct {
	Primary  DownloadSplitItem
	Items    []DownloadSplitItem
	Language string
}

// schematicDownloadSplit builds the split-button model for a library
// schematic: primary = the existing interstitial .nbt flow, menu = the other
// formats through the same interstitial with a format parameter.
func schematicDownloadSplit(name, lang string, dims [3]int) DownloadSplitData {
	get := func(format string) string {
		if format == "" {
			return "/get/" + name
		}
		return "/get/" + name + "?format=" + format
	}
	return DownloadSplitData{
		Language: lang,
		Primary: DownloadSplitItem{
			Label: i18n.T(lang, "Download"),
			Href:  get(""),
		},
		Items: []DownloadSplitItem{
			{Label: "Create .nbt", Href: get(""), Note: i18n.T(lang, "original")},
			{Label: "WorldEdit .schem", Href: get("schem")},
			{Label: "Litematica .litematic", Href: get("litematic")},
			{
				Label:      "Legacy .schematic",
				Href:       get("schematic"),
				Lossy:      true,
				LossyTitle: i18n.T(lang, "Lossy: modern blocks without a pre-1.13 equivalent become air"),
			},
			{Label: "MineColonies .blueprint", Href: get("blueprint")},
			{
				Label:      "Building Gadgets .json",
				Href:       get("bg"),
				Lossy:      true,
				LossyTitle: i18n.T(lang, "Lossy: Building Gadgets templates cannot carry block entity data (chest contents, Create kinetics)"),
			},
			worldDownloadItem(name, lang, dims),
		},
	}
}

// worldDownloadItem builds the ready-to-play-world menu entry, disabled with
// the reason when the build exceeds the export guards.
func worldDownloadItem(name, lang string, dims [3]int) DownloadSplitItem {
	item := DownloadSplitItem{
		Label: i18n.T(lang, "Ready-to-play world (.zip)"),
		Href:  "/get/" + name + "?format=world",
		Note:  i18n.T(lang, "superflat"),
	}
	if ok, reason := schematic.CanExportWorld(dims); !ok {
		item.Disabled = true
		item.DisabledReason = i18n.T(lang, "too large: ") + reason
		item.Href = ""
	}
	return item
}
