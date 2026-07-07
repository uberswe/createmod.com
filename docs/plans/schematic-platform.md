# Schematic Platform — Implementation Plan

Feature scope: normalized schematic library, format conversion, Sable blueprint
support, editor (transforms), NBT viewer, world export, shared download
component, MC version modal, and the tiered content-safety pipeline with
per-build "Validated" badge.

This plan is grounded in the current codebase (file references verified
2026-07). Each phase lists what exists to reuse, what is new, and how it ships.

---

## 0. Current-state facts the plan builds on

| Area | What exists today | Anchor |
|---|---|---|
| NBT parsing | Create/vanilla structure NBT only, via fork `uberswe/mcnbt` + `Tnze/go-mc/nbt`; 100 MB decompression cap, dimension caps | `internal/nbtparser/parser.go` |
| Replace blocks | **Live feature**: `ReplacePalette()` + modify pages + saved variations (max 10/user, 1000 replacements) | `internal/nbtparser/replacer.go:51`, `internal/pages/modify.go` |
| TAG_List quirk | Tnze encoder writes `TAG_Int_Array`; Create requires `TAG_List(Int)` for `size`/`pos`. Byte-patch workaround exists and MUST be reused by any writer | `replacer.go:182,211`; generator uses `nbtIntList` (`internal/generator/nbt.go:16`) |
| DataVersion | **Not tracked anywhere** on parse; generator hardcodes 3955 on write | `internal/generator/nbt.go:296` |
| Upload pipeline | `.nbt`-only, 10 MB cap, temp-upload token flow → publish handler (~550 lines, needs extraction to be reusable) | `internal/pages/upload.go:1057` (intake), `:279` (publish) |
| Anvil/world | `Tnze/go-mc v1.20.2` ships `save/` (chunk, level.dat), `save/region` (.mca), `level/` (palette bit-packing) — **unused, ready** | go.mod |
| 3D viewers | External links only: Bloxelizer + Shulkr viewer URLs; modify preview already serves CORS-open temp NBT for them | `internal/pages/schematic.go:323-333`, `modify.go:262-276` |
| 3D renderer | The generator preview (`GeneratorApp.renderBlocks`) renders `{blocks[],materials}` with instanced meshes incl. stairs/slabs — reusable for an in-house block view | `template/static/generator.js` |
| Storage | S3/Minio client, B2-compatible (CopyRaw has B2 fallback); raw-key namespaces `_thumbs/*`, `temp/*` precedents for caches | `internal/storage/s3.go` |
| Rate limits | Route token-buckets (`downloadRateLimit` 10/min) + global daily download cap (100/IP/day) | `internal/router/main.go:726,1261`, `internal/pages/download.go:19` |
| Content safety | Tier-1 partially exists (decompression bomb caps). **No** NBT content inspection, no AV. Text/image moderation pipeline exists as the structural model | `parser.go:24-51`, `internal/jobs/moderation.go` |
| Tools landing | `/generators` card grid is the de-facto tools page | `template/generators.html` |

---

## 1. Phase F — The normalized schematic library (`internal/schematic`)

Everything else depends on this. New package, no HTTP concerns.

### Model

```go
package schematic

type BlockState struct {
    Name       string            // "minecraft:oak_stairs"
    Properties map[string]string // sorted-key canonical form for dedup
}

type BlockEntity struct {
    Pos [3]int
    Raw nbt.RawMessage // opaque round-trip by default; typed views later
}

type Schematic struct {
    Size          [3]int
    Palette       []BlockState
    Blocks        []int32        // palette indices, X-major → Z → Y (document!)
    BlockEntities []BlockEntity
    Entities      []nbt.RawMessage
    DataVersion   int            // 0 = unknown (legacy sources)
    Meta          Meta           // name, author, source format, lossy flags
}
```

Design decisions:
- **Paletted representation** matches all four formats and Anvil sections 1:1.
- **Opaque `RawMessage` for block entities/entities** — round-trips Create
  kinetic data byte-faithfully without modeling it. Typed accessors (sign
  text, chest contents) come later in the NBT-viewer phase.
- **Single NBT library**: `Tnze/go-mc/nbt` only. The `uberswe/mcnbt` fork
  stays confined to the legacy `nbtparser` package; new code never imports it.
- The structure-NBT **writer embeds the TAG_List(Int) fix** natively (use the
  generator's `nbtIntList` type rather than byte-patching).
- **Capabilities API**: `func (s *Schematic) Capabilities() Caps` reporting
  size, block-entity presence, entity presence, DataVersion — drives the
  download component's per-build menu and the world-export size guard.

### Readers/writers (this phase: structure NBT only)

- `ReadStructureNBT` / `WriteStructureNBT` (Create `.nbt`): full fidelity
  including DataVersion (finally tracked), block entities via `blocks[].nbt`.
- Shims so existing features can migrate incrementally:
  `ExtractMaterials/Stats/Dimensions` equivalents over the model, and a
  `ReplacePalette`-compatible transform. `internal/pages` keeps calling
  `nbtparser` until each call site is migrated; no big-bang rewrite.

### Tests

- Golden fixtures: real `.nbt` files (small hand-made + one Create schematic
  with kinetic block entities) committed under `internal/schematic/testdata/`.
- Round-trip property: read → write → read == deep-equal; byte-stability where
  the format allows.
- Fuzz the reader (Go native fuzzing) — this doubles as Tier-1 safety work.

**Ships as:** one PR, no user-visible change. ~3-4 days.

---

## 2. Phase C — Format conversion

### Readers/writers per format

| Format | Read | Write | Notes |
|---|---|---|---|
| `.nbt` structure | Phase F | Phase F | native |
| `.schem` (Sponge v2/v3) | yes | yes | closest cousin: palette + varint block array + `BlockEntities`; v3 adds nested `Blocks` container. DataVersion present |
| `.litematic` | yes | yes | regions with bit-packed `BlockStates` (reuse `go-mc/level` BitStorage math), metadata compound; multi-region files flatten to bounding box with a lossy flag |
| `.schematic` (legacy MCEdit) | yes | yes (lossy) | numeric IDs + data values; needs embedded 1.12 flattening map (id:meta ↔ modern name). Import marks `DataVersion: 1343`; export is **lossy-labeled** (modern blocks downgrade or map to fallback) |
| Sable Blueprint v1 | detect + read | no (deliberate) | sniff root tags to distinguish from vanilla structure NBT; offer labeled blocks-only flatten to other formats. Revisit writing when the format stabilizes |

The flattening map is the only large data artifact — embed as a generated Go
table from a JSON source file, with tests asserting spot mappings.

### Service + UI

- `internal/schematic/convert.go`: `Convert(in []byte, from, to Format, opts)
  → ([]byte, []Warning)`. Warnings carry the lossy details the UI shows.
- **Converter page** (`/tools/convert`): upload → detect format (sniff, don't
  trust extension) → pick target → download. Reuses upload hardening caps.
  Card on `/generators` (rename nav item to "Tools", route alias `/tools` →
  same landing).
- **Per-pair SEO landing pages**: one handler + template parameterized by
  (from, to) pair, e.g. `/tools/convert/litematic-to-schem`. Static copy per
  pair (what each format is, which mods use it, lossiness notes), the shared
  converter widget prefilled. Add to sitemap + hreflang; index only the
  supported pair permutations (12 pages + Sable flatten pages).

**Ships as:** PR 1 readers/writers + tests; PR 2 converter page; PR 3 SEO
pages. ~5-6 days total. Conversion is also where **multi-`DataVersion`
remapping** will land later — out of scope for v1 beyond passthrough +
warnings on mismatch.

---

## 3. Phase D — Shared download component + MC version modal

Build these before the editor/world export so every later feature plugs in.

### Download split button (shared template include + JS)

- `template/include/download_split.html` + `template/static/download-split.js`
  following the adrail/webmcp pattern (no framework).
- Server passes a **capability-derived menu model**: list of `{format, label,
  href, lossy?, disabledReason?, needsVersionPrompt?}` computed from
  `schematic.Capabilities()`:
  - default/primary: Create `.nbt` (exactly today's download URL — the daily
    cap and interstitial flow stay untouched)
  - `.schem`, `.litematic`, `.schematic` (lossy ⇒ warning icon + hover detail)
  - download-as-world (disabled + annotated when over the size guard)
- Keyboard navigation, click-outside close, ARIA menu semantics.
- **Generators adopt the same component**: `GenerateResult` → structure NBT →
  normalized model → any format. Generators never learn about formats.

### Conversion/download endpoints + caching

- Library: `GET /schematics/{name}/download/{format}?v={mcversion}` →
  interstitial-equivalent flow (respect `downloadRateLimitAllow`).
- Generators: `POST /api/generators/{type}/download/{format}`.
- **Cache** converted bytes in S3 under `_conv/{v}/…`:
  - library key: `(schematicID, fileChecksum, format, mcVersion)`
  - generator key: `(generator, paramsHash, format, mcVersion)`
  - invalidation: keys include the source checksum, so edits naturally miss;
    TTL cleanup via a River job (mirror `_thumbs` handling).

### MC version modal (shared, conditional)

- One template include + JS, invoked by the download component only when the
  chosen entry has `needsVersionPrompt`:
  - **always**: world export
  - **sometimes**: cross-version conversion (v1: only when source DataVersion
    is unknown/legacy)
  - **never**: native-version schematic downloads
- Pre-selected to the build's native DataVersion → common case is one tap.
- Version list: a small static table (MC release → DataVersion) in
  `internal/schematic/versions.go`; v1 world export supports **one** target.

**Ships as:** PR 1 component + library wiring; PR 2 generator adoption; PR 3
version modal (with world export stubbed disabled). ~4-5 days.

---

## 4. Phase V — NBT viewer (read-only)

- **Backend**: `GET /api/schematics/{name}/nbt-tree?path=&depth=` — serves
  the parsed tree in pages (children-of-path), so the client never holds a
  100 MB document. SNBT rendering server-side per node. Create block entities
  flagged (`create:` namespace) for the special badge/rendering.
- **Frontend**: virtualized tree (plain JS, windowed list — same no-framework
  discipline), SNBT toggle per node, key search (server-side index of paths,
  capped), copy-path button.
- **Placement**: a tab/section on every schematic page + **standalone
  upload-and-view page** `/tools/nbt-viewer` (SEO: "nbt viewer online",
  "view minecraft nbt file") reusing upload hardening. Standalone page never
  persists the file (memory/temp only).
- **Structured edits** (sign text, chest contents, DataVersion bump, rename
  block) come after the editor phase exists, as editor operations — the
  viewer stays read-only.

**Ships as:** one PR backend + one PR frontend. ~4 days.

---

## 5. Phase E — Editor (transforms first)

### Model: server-authoritative sessions, command-pattern ops

- **Session** = a temp upload (existing token infra) + an op log:
  `editor_sessions(id, user_id, temp_token, source_schematic_id?, ops JSONB,
  created, updated)` — one migration.
- Operations (all pure functions over the normalized model in
  `internal/schematic/ops.go`): `crop`, `resize(grow)`, `rotate90`, `mirror`,
  `fill`, `hollow`, `replaceBlocks` (delegates to the palette-replace logic),
  `deleteRegion`. Undo/redo = replay op log prefix (ops are cheap at ≤10 MB
  schematics; no inverse-op complexity).
- API: `POST /api/editor/{session}/op`, `POST /api/editor/{session}/undo`,
  `GET /api/editor/{session}/preview.nbt` (CORS-open like modify previews),
  `GET /api/editor/{session}/materials`.

### UI

- `/tools/editor` page: 3D block view (**reuse `GeneratorApp` renderer** —
  map the normalized palette to its block-type enum with a coarse
  name→type/color table; unknown blocks render as generic cubes), op sidebar,
  undo/redo, material list live-updated.
- External viewer buttons (Bloxelizer/Shulkr) pointing at the session preview
  URL — same mechanism modify.go uses today.
- **Entry points**: header button (≥lg viewports; collapses into the
  existing menu below that) opening an empty session; "Edit this schematic"
  button on schematic pages opening a prefilled session; card on the tools
  landing. Same editor, two ways in.
- **Publish-from-editor**: extract the core of `UploadMakePublicHandler`
  (form args → published schematic) into a reusable function first —
  prerequisite refactor PR. Editor "publish" feeds its output NBT through
  `UploadNBTHandler`'s validation path then the extracted publish core.
- Deferred explicitly: free 3D block placement, multi-select gizmos.

**Ships as:** PR 1 ops + tests; PR 2 sessions + API; PR 3 UI; PR 4 publish
refactor + wiring. ~8-10 days. (The existing modify/variations feature stays
as-is; long-term it becomes an editor preset but don't touch it now.)

---

## 6. Phase W — Download as ready-to-play world

- **Writer** (`internal/schematic/world.go`): normalized model → Anvil chunks
  using `go-mc/save` + `save/region` + `level` bit-packing (already in
  go.mod — do NOT hand-roll). Deterministic placement: build centered at a
  fixed origin, spawn point offset looking at it, fixed gamerules
  (`doDaylightCycle` off, `keepInventory` on, peaceful), superflat preset
  (void or grass — the `preset` cache-key dimension).
- **level.dat**: templated via `save.Level` with target `DataVersion`;
  version modal always prompts (pre-set to native).
- **Streaming**: `archive/zip` straight to the response writer; region files
  generated per-region and written sequentially — flat memory.
- **Guards**: size validation up front (block volume + region count cap →
  clear "too large for world export — download the schematic instead"
  message and a disabled menu entry via `Capabilities()`); route rate limit
  (stricter than downloads, e.g. 3/min/IP) + the global daily cap; generation
  timeout.
- **Cache**: first click generates, then bytes stored at
  `_worlds/{schematicID}/{checksum}/{mcVersion}-{preset}.zip` (or generator
  params-hash key); serve from cache after. River TTL cleanup. Cache key
  matches the download-component scheme from Phase D, so this is purely
  additive.
- **Version targeting v1**: single MC target (whatever DataVersion the model
  carries, validated against the supported table); multi-version later via
  the converter's remapping.
- Block entities pass through untouched (Create kinetic config carries over).
  Entities: include if trivially placeable, else drop with a manifest note.

**Ships as:** PR 1 writer + golden world fixture (load-tested manually in a
real client — acceptance gate); PR 2 endpoint + cache + guards + menu
enablement. ~6-8 days. Highest-risk phase; the fixture world loading cleanly
in vanilla is the go/no-go checkpoint.

---

## 7. Phase S — Content-safety pipeline + badge

### Tier 1 — hardening (mostly consolidation)

- Move/dedupe the existing caps (100 MB decompression, dimension, block-ID
  length) into `internal/schematic/limits.go`; add NBT depth cap, total-tag
  cap, palette-size cap, string-length cap. Apply to every reader.
- Zip-slip validation: only relevant once zipped inputs exist (none today;
  world export is output-only). Implement in the shared zip helper when the
  first zipped input ships.
- Fuzz corpus from Phase F carries the regression suite.

### Tier 2 — Minecraft content inspection

- `internal/schematic/inspect.go`: walk block entities + palette for command
  blocks (incl. chain/repeating), spawners (+ what they spawn), sign/book
  click commands (`run_command` actions), structure blocks, and datapack
  function references. Output = **manifest** JSON: counts + locations +
  extracted command text.
- Runs in the existing moderation River job on upload AND on editor publish;
  re-runs on edit. Stored in a new table:
  `schematic_safety(schematic_id, checksum, file_safe bool, manifest JSONB,
  pipeline_version int, scanned_at)` — one migration. `pipeline_version`
  lets a future inspector improvement re-queue old builds.
- Backfill job for the existing library (batched, low priority).
- **User-facing "check this file" page** `/tools/schematic-safety-check`:
  upload → hardening + inspection → manifest report. Never persists. SEO
  page in its own right.

### Badge + explainer

- Download pages/schematic page: badge component with **two separate,
  honest claims** driven by `schematic_safety`:
  - `file_safe` → "Validated file" (valid NBT, hardening passed; ClamAV
    listed only when a scan actually ran)
  - manifest empty → "Contents transparent"; manifest non-empty → same badge
    slot shows "Contains command blocks — view details" linking the manifest.
    This state is a feature, not a warning-red failure.
- No badge until the pipeline has actually run for that checksum.
- **Explainer page** `/safety` (SEO: "are minecraft schematics safe", "can a
  schematic have a virus"): schematics are data, not programs; what we harden
  against; what we inspect for; what the manifest means. Accurate, not
  absolute. Badge links here.
- **ClamAV**: deferred until zipped/world *inputs* exist — today's `.nbt`
  uploads gain nothing from AV that hardening doesn't already cover. When
  added: clamd sidecar container, scan gate in upload for zip types only.

**Ships as:** PR 1 tier-1 consolidation + fuzz; PR 2 inspector + table +
job + backfill; PR 3 badge + explainer + check-a-file page. ~5-6 days.

---

## 8. Cross-cutting

- **API/MCP surface**: as each phase lands, expose it: conversion endpoint in
  the public API (`POST /api/convert`, API-key gated, documented in
  `/api` + OpenAPI), an MCP `convert_schematic` tool later; update the
  `find-schematics` agent skill and `auth.md` when endpoints appear.
- **i18n**: all new UI copy through `T()`; en keys only (existing pattern).
- **SEO integration**: every new tools page gets title/description/h1/JSON-LD
  (`SoftwareApplication` for tools pages is a supported rich-result type),
  sitemap entries, hreflang.
- **Testing discipline**: golden fixtures per format; round-trip properties;
  the world fixture loaded in a real client once per format-affecting change;
  template render tests per new page (existing pattern).
- **Metrics**: counter per conversion pair, world exports, cache hit rates —
  existing Prometheus metrics package.

## 9. Sequencing, dependencies, effort

```
F (foundation, 3-4d)
├─→ C (conversion, 5-6d)  ──→ D (download component + modal, 4-5d)
│                              ├─→ W (world export, 6-8d)
│                              └─→ generators adoption
├─→ V (NBT viewer, 4d)
├─→ E (editor, 8-10d)  [needs publish-refactor PR first]
└─→ S (safety, 5-6d)   [independent after F; badge benefits from landing
                        before editor publish so edits re-scan from day one]
```

Recommended order: **F → C → D → S → V → W → E** — SEO/traffic value early
(conversion pages, safety explainer), the riskiest engineering (world export)
after the download plumbing exists, and the editor last because it composes
everything (ops over F, publish pipeline, download component, safety re-scan).

Total: roughly 35-43 focused days. Each phase is independently shippable and
each PR within a phase keeps CI green.

## 10. Risks & open questions

1. **Legacy `.schematic` flattening map** — largest data artifact; mitigate
   by importing a community-maintained mapping (verify license) rather than
   hand-building.
2. **Sable format drift** — sniffing is committed; writing deliberately
   deferred. Revisit when Sable publishes a stable spec.
3. **World export client compatibility** — DataVersion/chunk-format details
   vary per MC version; v1 pins one version and the golden-world manual test
   gates releases. go-mc tracks 1.20.x; targeting newer MC may need the
   version table + light chunk-format shims.
4. **`uberswe/mcnbt` fork retirement** — goal state is Tnze-only; migrate
   `nbtparser` call sites opportunistically, never as a blocking rewrite.
5. **Publish-handler extraction** — the 550-line `UploadMakePublicHandler`
   refactor is the editor's real prerequisite; do it as its own reviewed PR
   with the existing upload flow as regression coverage.
