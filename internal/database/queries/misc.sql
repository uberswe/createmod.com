-- name: CreateReport :one
INSERT INTO reports (id, target_type, target_id, reason, reporter)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListReports :many
SELECT * FROM reports ORDER BY created DESC LIMIT $1 OFFSET $2;

-- name: DeleteReport :exec
DELETE FROM reports WHERE id = $1;

-- name: DeleteReportsByTarget :execrows
DELETE FROM reports WHERE target_type = $1 AND target_id = $2;

-- name: CreateSearch :exec
INSERT INTO searches (id, query, results_count, user_id, ip_address)
VALUES ($1, $2, $3, $4, $5);

-- name: RecordOutgoingClick :exec
INSERT INTO outgoing_clicks (id, url, source, source_id, user_id)
VALUES ($1, $2, $3, $4, $5);

-- name: CreateContactFormSubmission :one
INSERT INTO contact_form_submissions (id, author_id, title, content, name, postdate, status, type)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
RETURNING *;

-- name: GetNBTHash :one
SELECT * FROM nbt_hashes WHERE hash = $1;

-- name: CreateNBTHash :exec
INSERT INTO nbt_hashes (id, hash, schematic_id, uploaded_by) VALUES ($1, $2, $3, $4);

-- name: ListNBTHashesByUser :many
SELECT * FROM nbt_hashes
WHERE uploaded_by = $1 AND schematic_id IS NULL
ORDER BY created DESC;

-- name: DeleteNBTHash :exec
DELETE FROM nbt_hashes WHERE id = $1 AND uploaded_by = $2;

-- name: CheckHashIsBlacklisted :one
SELECT EXISTS(SELECT 1 FROM nbt_hashes WHERE hash = $1 AND schematic_id IS NULL) AS is_blacklisted;

-- name: CreateSchematicVersion :one
INSERT INTO schematic_versions (id, schematic_id, version, snapshot, note)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListSchematicVersions :many
SELECT * FROM schematic_versions
WHERE schematic_id = $1
ORDER BY version DESC;

-- name: GetLatestSchematicVersion :one
SELECT COALESCE(MAX(version), 0)::INTEGER AS latest_version
FROM schematic_versions
WHERE schematic_id = $1;

-- name: GetModMetadataByNamespace :one
SELECT * FROM mod_metadata WHERE namespace = $1;

-- name: UpsertModMetadata :one
INSERT INTO mod_metadata (id, namespace, display_name, description, icon_url,
    modrinth_slug, modrinth_url, curseforge_id, curseforge_url, source_url,
    last_fetched, manually_set, blocksitems_matched)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13)
ON CONFLICT (namespace) DO UPDATE SET
    display_name = EXCLUDED.display_name,
    description = EXCLUDED.description,
    icon_url = EXCLUDED.icon_url,
    modrinth_slug = EXCLUDED.modrinth_slug,
    modrinth_url = EXCLUDED.modrinth_url,
    curseforge_id = EXCLUDED.curseforge_id,
    curseforge_url = EXCLUDED.curseforge_url,
    source_url = EXCLUDED.source_url,
    last_fetched = EXCLUDED.last_fetched,
    manually_set = EXCLUDED.manually_set,
    blocksitems_matched = EXCLUDED.blocksitems_matched
RETURNING *;

-- name: ListModMetadataAll :many
SELECT * FROM mod_metadata
ORDER BY namespace;

-- name: ListModMetadataStale :many
SELECT * FROM mod_metadata
WHERE manually_set = false
  AND (last_fetched IS NULL OR last_fetched < NOW() - INTERVAL '7 days')
ORDER BY last_fetched NULLS FIRST
LIMIT $1;

-- name: GetExternalAuth :one
SELECT * FROM external_auths
WHERE provider = $1 AND provider_id = $2;

-- name: CreateExternalAuth :one
INSERT INTO external_auths (id, user_id, provider, provider_id)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListExternalAuthsByUser :many
SELECT * FROM external_auths WHERE user_id = $1;

-- name: CreateUserMeta :exec
INSERT INTO user_meta (id, user_id, key, value)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, key) DO UPDATE SET value = EXCLUDED.value;

-- name: GetUserMeta :one
SELECT * FROM user_meta WHERE user_id = $1 AND key = $2;

-- name: ListNews :many
SELECT * FROM news
WHERE status = 'publish'
ORDER BY postdate DESC
LIMIT $1 OFFSET $2;

-- name: ListVersions :many
SELECT * FROM createmod_versions ORDER BY version DESC;

-- name: ListMinecraftVersions :many
SELECT * FROM minecraft_versions ORDER BY version DESC;

-- name: GetMinecraftVersionByID :one
SELECT id, version, created FROM minecraft_versions WHERE id = $1;

-- name: GetCreatemodVersionByID :one
SELECT id, version, created FROM createmod_versions WHERE id = $1;

-- name: ListTopSearches :many
SELECT query, search_count
FROM search_query_counts
ORDER BY search_count DESC
LIMIT $1;

-- name: RefreshSearchQueryCounts :exec
-- Concurrent refresh: allows reads during the refresh. Requires a unique index
-- on the matview (idx_search_query_counts_query), which already exists.
REFRESH MATERIALIZED VIEW CONCURRENTLY search_query_counts;

-- name: CountUnresolvedCommentReportsByAuthor :one
SELECT COUNT(*) FROM reports r
JOIN comments c ON c.id = r.target_id
WHERE r.target_type = 'comment' AND c.author_id = $1;

-- name: PruneOldSearches :execrows
WITH single_use AS (
  SELECT LEFT(query, 500) AS q
  FROM searches
  GROUP BY LEFT(query, 500)
  HAVING COUNT(*) = 1
)
DELETE FROM searches s
USING single_use su
WHERE LEFT(s.query, 500) = su.q
  AND s.created < NOW() - INTERVAL '90 days';
