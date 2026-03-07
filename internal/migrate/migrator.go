package migrate

import (
	"context"
	"database/sql"
	"fmt"
	"log"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Migrator reads from PocketBase SQLite and writes to PostgreSQL.
type Migrator struct {
	sqlite *sql.DB
	pg     *pgxpool.Pool
	dryRun bool
}

// New creates a new Migrator.
func New(sqlite *sql.DB, pg *pgxpool.Pool, dryRun bool) *Migrator {
	return &Migrator{
		sqlite: sqlite,
		pg:     pg,
		dryRun: dryRun,
	}
}

// Run executes the full migration in dependency order.
func (m *Migrator) Run(ctx context.Context) error {
	// Phase 1: No FK dependencies
	steps := []struct {
		name string
		fn   func(context.Context) error
	}{
		{"schematic_categories", m.migrateSchematicCategories},
		{"schematic_tags", m.migrateSchematicTags},
		{"createmod_versions", m.migrateCreatemodVersions},
		{"minecraft_versions", m.migrateMinecraftVersions},
		{"achievements", m.migrateAchievements},
		{"users", m.migrateUsers},
		// Phase 2: Depends on users
		{"external_auths", m.migrateExternalAuths},
		{"user_meta", m.migrateUserMeta},
		{"schematics", m.migrateSchematics},
		// Phase 3: Depends on schematics
		{"schematics_categories", m.migrateSchematiccategories},
		{"schematics_tags", m.migrateSchematictags},
		{"schematic_views", m.migrateSchematicViews},
		{"schematic_ratings", m.migrateSchematicRatings},
		{"schematic_downloads", m.migrateSchematicDownloads},
		{"schematic_versions", m.migrateSchematicVersions},
		{"schematic_files", m.migrateSchematicFiles},
		{"schematic_translations", m.migrateSchematicTranslations},
		{"nbt_hashes", m.migrateNBTHashes},
		{"comments", m.migrateComments},
		// Phase 4: Depends on users + schematics
		{"guides", m.migrateGuides},
		{"guide_translations", m.migrateGuideTranslations},
		{"collections", m.migrateCollections},
		{"collections_schematics", m.migrateCollectionsSchematics},
		{"collection_translations", m.migrateCollectionTranslations},
		// Phase 5: Remaining
		{"user_achievements", m.migrateUserAchievements},
		{"point_log", m.migratePointLog},
		{"api_keys", m.migrateAPIKeys},
		{"api_key_usage", m.migrateAPIKeyUsage},
		{"news", m.migrateNews},
		{"pages", m.migratePages},
		{"searches", m.migrateSearches},
		{"contact_form_submissions", m.migrateContactFormSubmissions},
		{"outgoing_clicks", m.migrateOutgoingClicks},
		{"reports", m.migrateReports},
		{"temp_uploads", m.migrateTempUploads},
		{"temp_upload_files", m.migrateTempUploadFiles},
		{"mod_metadata", m.migrateModMetadata},
	}

	for _, step := range steps {
		log.Printf("Migrating %s...", step.name)
		if err := step.fn(ctx); err != nil {
			return fmt.Errorf("migrating %s: %w", step.name, err)
		}
	}

	// Update comment parent references (deferred to avoid FK issues)
	log.Println("Updating comment parent references...")
	if err := m.updateCommentParents(ctx); err != nil {
		return fmt.Errorf("updating comment parents: %w", err)
	}

	return nil
}

// tableExists checks if a PocketBase collection table exists in SQLite.
func (m *Migrator) tableExists(table string) bool {
	var name string
	err := m.sqlite.QueryRow("SELECT name FROM sqlite_master WHERE type='table' AND name=?", table).Scan(&name)
	return err == nil && name != ""
}

// countSQLite returns the row count in a SQLite table.
func (m *Migrator) countSQLite(table string) int64 {
	if !m.tableExists(table) {
		return 0
	}
	var count int64
	_ = m.sqlite.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count)
	return count
}

// countPG returns the row count in a PostgreSQL table.
func (m *Migrator) countPG(ctx context.Context, table string) int64 {
	var count int64
	_ = m.pg.QueryRow(ctx, "SELECT COUNT(*) FROM "+table).Scan(&count)
	return count
}

// usersTable returns the correct SQLite table name for users.
func (m *Migrator) usersTable() string {
	if m.tableExists("users") {
		return "users"
	}
	return "_pb_users_auth_"
}

// PrintCounts prints row counts for all tables (dry-run mode).
func (m *Migrator) PrintCounts(ctx context.Context) {
	tables := []struct {
		pgTable     string
		sqliteTable string
	}{
		{"users", m.usersTable()},
		{"external_auths", "_externalAuths"},
		{"schematic_categories", "schematic_categories"},
		{"schematic_tags", "schematic_tags"},
		{"createmod_versions", "createmod_versions"},
		{"minecraft_versions", "minecraft_versions"},
		{"schematics", "schematics"},
		{"schematic_views", "schematic_views"},
		{"schematic_ratings", "schematic_ratings"},
		{"schematic_downloads", "schematic_downloads"},
		{"schematic_versions", "schematic_versions"},
		{"comments", "comments"},
		{"guides", "guides"},
		{"collections", "collections"},
		{"achievements", "achievements"},
		{"user_achievements", "user_achievements"},
		{"point_log", "point_log"},
		{"api_keys", "api_keys"},
		{"news", "news"},
		{"pages", "pages"},
		{"searches", "searches"},
		{"contact_form_submissions", "contact_form_submissions"},
		{"outgoing_clicks", "outgoing_clicks"},
		{"reports", "reports"},
		{"mod_metadata", "mod_metadata"},
	}

	fmt.Printf("\n%-35s %10s %10s\n", "Table", "SQLite", "PostgreSQL")
	fmt.Println(strings.Repeat("-", 57))
	for _, t := range tables {
		sc := m.countSQLite(t.sqliteTable)
		pc := m.countPG(ctx, t.pgTable)
		fmt.Printf("%-35s %10d %10d\n", t.pgTable, sc, pc)
	}
}

// Validate prints a comparison of row counts after migration.
func (m *Migrator) Validate(ctx context.Context) {
	tables := []struct {
		pgTable     string
		sqliteTable string
	}{
		{"users", m.usersTable()},
		{"external_auths", "_externalAuths"},
		{"schematic_categories", "schematic_categories"},
		{"schematic_tags", "schematic_tags"},
		{"createmod_versions", "createmod_versions"},
		{"minecraft_versions", "minecraft_versions"},
		{"schematics", "schematics"},
		{"schematic_views", "schematic_views"},
		{"schematic_ratings", "schematic_ratings"},
		{"schematic_downloads", "schematic_downloads"},
		{"schematic_versions", "schematic_versions"},
		{"comments", "comments"},
		{"guides", "guides"},
		{"collections", "collections"},
		{"achievements", "achievements"},
		{"user_achievements", "user_achievements"},
		{"point_log", "point_log"},
		{"api_keys", "api_keys"},
		{"news", "news"},
		{"pages", "pages"},
		{"searches", "searches"},
		{"contact_form_submissions", "contact_form_submissions"},
		{"outgoing_clicks", "outgoing_clicks"},
		{"reports", "reports"},
		{"mod_metadata", "mod_metadata"},
	}

	fmt.Printf("\n%-35s %10s %10s %6s\n", "Table", "SQLite", "PostgreSQL", "Match")
	fmt.Println(strings.Repeat("-", 65))
	for _, t := range tables {
		sc := m.countSQLite(t.sqliteTable)
		pc := m.countPG(ctx, t.pgTable)
		match := " "
		if sc == pc {
			match = "Y"
		} else if pc >= sc {
			match = "Y" // PG may have seed data (achievements)
		}
		fmt.Printf("%-35s %10d %10d %6s\n", t.pgTable, sc, pc, match)
	}
}

// ────────────────────────────────────────────────────────────
// Phase 1: Tables with no FK dependencies
// ────────────────────────────────────────────────────────────

func (m *Migrator) migrateSchematicCategories(ctx context.Context) error {
	if !m.tableExists("schematic_categories") {
		return nil
	}
	rows, err := m.sqlite.Query("SELECT id, key, name, created, updated FROM schematic_categories")
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, key, name, created, updated string
		if err := rows.Scan(&id, &key, &name, &created, &updated); err != nil {
			return err
		}
		_, err := m.pg.Exec(ctx,
			`INSERT INTO schematic_categories (id, key, name, created, updated)
			 VALUES ($1, $2, $3, $4, $5) ON CONFLICT DO NOTHING`,
			id, key, name, parsePBTimestampRequired(created), parsePBTimestampRequired(updated))
		if err != nil {
			return err
		}
		count++
	}
	log.Printf("  -> %d rows", count)
	return nil
}

func (m *Migrator) migrateSchematicTags(ctx context.Context) error {
	if !m.tableExists("schematic_tags") {
		return nil
	}
	rows, err := m.sqlite.Query("SELECT id, key, name, created, updated FROM schematic_tags")
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, key, name, created, updated string
		if err := rows.Scan(&id, &key, &name, &created, &updated); err != nil {
			return err
		}
		_, err := m.pg.Exec(ctx,
			`INSERT INTO schematic_tags (id, key, name, created, updated)
			 VALUES ($1, $2, $3, $4, $5) ON CONFLICT DO NOTHING`,
			id, key, name, parsePBTimestampRequired(created), parsePBTimestampRequired(updated))
		if err != nil {
			return err
		}
		count++
	}
	log.Printf("  -> %d rows", count)
	return nil
}

func (m *Migrator) migrateCreatemodVersions(ctx context.Context) error {
	if !m.tableExists("createmod_versions") {
		return nil
	}
	rows, err := m.sqlite.Query("SELECT id, version, created, updated FROM createmod_versions")
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, version, created, updated string
		if err := rows.Scan(&id, &version, &created, &updated); err != nil {
			return err
		}
		_, err := m.pg.Exec(ctx,
			`INSERT INTO createmod_versions (id, version, created, updated)
			 VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING`,
			id, version, parsePBTimestampRequired(created), parsePBTimestampRequired(updated))
		if err != nil {
			return err
		}
		count++
	}
	log.Printf("  -> %d rows", count)
	return nil
}

func (m *Migrator) migrateMinecraftVersions(ctx context.Context) error {
	if !m.tableExists("minecraft_versions") {
		return nil
	}
	rows, err := m.sqlite.Query("SELECT id, version, created, updated FROM minecraft_versions")
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, version, created, updated string
		if err := rows.Scan(&id, &version, &created, &updated); err != nil {
			return err
		}
		_, err := m.pg.Exec(ctx,
			`INSERT INTO minecraft_versions (id, version, created, updated)
			 VALUES ($1, $2, $3, $4) ON CONFLICT DO NOTHING`,
			id, version, parsePBTimestampRequired(created), parsePBTimestampRequired(updated))
		if err != nil {
			return err
		}
		count++
	}
	log.Printf("  -> %d rows", count)
	return nil
}

func (m *Migrator) migrateAchievements(ctx context.Context) error {
	if !m.tableExists("achievements") {
		return nil
	}
	rows, err := m.sqlite.Query("SELECT id, key, title, description, icon, created, updated FROM achievements")
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, key, title, description, icon, created, updated string
		if err := rows.Scan(&id, &key, &title, &description, &icon, &created, &updated); err != nil {
			return err
		}
		_, err := m.pg.Exec(ctx,
			`INSERT INTO achievements (id, key, title, description, icon, created, updated)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)
			 ON CONFLICT (key) DO NOTHING`,
			id, key, title, description, icon,
			parsePBTimestampRequired(created), parsePBTimestampRequired(updated))
		if err != nil {
			return err
		}
		count++
	}
	log.Printf("  -> %d rows", count)
	return nil
}

func (m *Migrator) migrateUsers(ctx context.Context) error {
	// PocketBase v0.29+ uses "users", older versions used "_pb_users_auth_"
	table := "users"
	if !m.tableExists(table) {
		table = "_pb_users_auth_"
		if !m.tableExists(table) {
			return nil
		}
	}
	rows, err := m.sqlite.Query(fmt.Sprintf(
		`SELECT id, COALESCE(email,''), COALESCE(username,''), COALESCE(password,''), COALESCE(old_password,''),
		        COALESCE(avatar,''), COALESCE(points,0), COALESCE(verified,0),
		        COALESCE(deleted,''), created, updated
		 FROM %s`, table))
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, email, username, passwordHash, oldPassword, avatar, deleted, created, updated string
		var points int
		var verified interface{}
		if err := rows.Scan(&id, &email, &username, &passwordHash, &oldPassword,
			&avatar, &points, &verified, &deleted, &created, &updated); err != nil {
			return err
		}
		_, err := m.pg.Exec(ctx,
			`INSERT INTO users (id, email, username, password_hash, old_password, avatar,
			                    points, verified, is_admin, deleted, created, updated)
			 VALUES ($1, $2, $3, $4, $5, $6, $7, $8, false, $9, $10, $11)
			 ON CONFLICT DO NOTHING`,
			id, email, username, passwordHash, oldPassword, avatar,
			points, sqliteBool(verified), parsePBTimestamp(deleted),
			parsePBTimestampRequired(created), parsePBTimestampRequired(updated))
		if err != nil {
			return err
		}
		count++
	}
	log.Printf("  -> %d rows", count)
	return nil
}

// ────────────────────────────────────────────────────────────
// Phase 2: Depends on users
// ────────────────────────────────────────────────────────────

func (m *Migrator) migrateExternalAuths(ctx context.Context) error {
	table := "_externalAuths"
	if !m.tableExists(table) {
		return nil
	}
	rows, err := m.sqlite.Query(fmt.Sprintf(
		`SELECT id, collectionRef, recordRef, provider, providerId, created, updated
		 FROM %s`, table))
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, collectionRef, userID, provider, providerID, created, updated string
		if err := rows.Scan(&id, &collectionRef, &userID, &provider, &providerID, &created, &updated); err != nil {
			return err
		}
		_, err := m.pg.Exec(ctx,
			`INSERT INTO external_auths (id, user_id, provider, provider_id, created, updated)
			 VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT DO NOTHING`,
			id, userID, provider, providerID,
			parsePBTimestampRequired(created), parsePBTimestampRequired(updated))
		if err != nil {
			log.Printf("  warning: skipping external_auth %s (FK): %v", id, err)
		} else {
			count++
		}
	}
	log.Printf("  -> %d rows", count)
	return nil
}

func (m *Migrator) migrateUserMeta(ctx context.Context) error {
	if !m.tableExists("user_meta") {
		return nil
	}
	rows, err := m.sqlite.Query("SELECT id, user, key, value, created, updated FROM user_meta")
	if err != nil {
		return nil // table may not exist in older PB schemas
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, userID, key, value, created, updated string
		if err := rows.Scan(&id, &userID, &key, &value, &created, &updated); err != nil {
			return err
		}
		_, err := m.pg.Exec(ctx,
			`INSERT INTO user_meta (id, user_id, key, value, created, updated)
			 VALUES ($1, $2, $3, $4, $5, $6) ON CONFLICT DO NOTHING`,
			id, userID, key, value,
			parsePBTimestampRequired(created), parsePBTimestampRequired(updated))
		if err != nil {
			log.Printf("  warning: skipping user_meta %s: %v", id, err)
		} else {
			count++
		}
	}
	log.Printf("  -> %d rows", count)
	return nil
}

func (m *Migrator) migrateSchematics(ctx context.Context) error {
	if !m.tableExists("schematics") {
		return nil
	}
	rows, err := m.sqlite.Query(
		`SELECT id, COALESCE(author,''), COALESCE(name,''), COALESCE(title,''),
		        COALESCE(description,''), COALESCE(excerpt,''), COALESCE(content,''),
		        COALESCE(postdate,''), COALESCE(modified,''), COALESCE(detected_language,''),
		        COALESCE(featured_image,''), COALESCE(gallery,''),
		        COALESCE(schematic_file,''), COALESCE(video,''),
		        COALESCE(has_dependencies,0), COALESCE(dependencies,''),
		        COALESCE(createmod_version,''), COALESCE(minecraft_version,''),
		        COALESCE(views,0),
		        COALESCE(block_count,0), COALESCE(dim_x,0), COALESCE(dim_y,0), COALESCE(dim_z,0),
		        COALESCE(materials,'[]'), COALESCE(mods,'[]'),
		        COALESCE(featured,0),
		        COALESCE(ai_description,''), COALESCE(moderated,0),
		        COALESCE(moderation_reason,''), COALESCE(blacklisted,0),
		        COALESCE(scheduled_at,''), COALESCE(deleted,''),
		        COALESCE(old_id,0), COALESCE(status,''), COALESCE(type,''),
		        created, updated
		 FROM schematics`)
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var (
			id, author, name, title, description, excerpt, content          string
			postdate, modified, detectedLanguage, featuredImage, galleryJSON string
			schematicFile, video, dependencies, createmodVersion            string
			minecraftVersion, materialsJSON, modsJSON, aiDescription       string
			moderationReason, scheduledAt, deleted                         string
			status, typ, created, updated                                  string
			hasDependencies, featured, moderated, blacklisted              interface{}
			views, blockCount, dimX, dimY, dimZ, oldID                     int
		)
		if err := rows.Scan(
			&id, &author, &name, &title, &description, &excerpt, &content,
			&postdate, &modified, &detectedLanguage, &featuredImage, &galleryJSON,
			&schematicFile, &video, &hasDependencies, &dependencies,
			&createmodVersion, &minecraftVersion,
			&views, &blockCount, &dimX, &dimY, &dimZ,
			&materialsJSON, &modsJSON,
			&featured, &aiDescription, &moderated,
			&moderationReason, &blacklisted, &scheduledAt, &deleted,
			&oldID, &status, &typ, &created, &updated,
		); err != nil {
			return fmt.Errorf("scanning schematic: %w", err)
		}

		gallery := parseJSONStringArray(galleryJSON)
		if gallery == nil {
			gallery = []string{}
		}

		if materialsJSON == "" {
			materialsJSON = "[]"
		}
		if modsJSON == "" {
			modsJSON = "[]"
		}

		var cmVerID, mcVerID *string
		if createmodVersion != "" {
			cmVerID = &createmodVersion
		}
		if minecraftVersion != "" {
			mcVerID = &minecraftVersion
		}

		var authorID *string
		if author != "" {
			authorID = &author
		}

		_, err := m.pg.Exec(ctx,
			`INSERT INTO schematics (
				id, author_id, name, title, description, excerpt, content,
				postdate, modified, detected_language, featured_image, gallery,
				schematic_file, video, has_dependencies, dependencies,
				createmod_version_id, minecraft_version_id,
				views, downloads, block_count, dim_x, dim_y, dim_z,
				materials, mods, paid, featured, ai_description, moderated,
				moderation_reason, blacklisted, scheduled_at, deleted,
				old_id, status, type, created, updated
			) VALUES (
				$1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16,$17,$18,
				$19,0,$20,$21,$22,$23,$24,$25,false,$26,$27,$28,
				$29,$30,$31,$32,
				$33,$34,$35,$36,$37
			) ON CONFLICT DO NOTHING`,
			id, authorID, name, title, description, excerpt, content,
			parsePBTimestamp(postdate), parsePBTimestamp(modified), detectedLanguage,
			featuredImage, gallery, schematicFile, video,
			sqliteBool(hasDependencies), dependencies,
			cmVerID, mcVerID,
			views, blockCount, dimX, dimY, dimZ,
			materialsJSON, modsJSON,
			sqliteBool(featured), aiDescription, sqliteBool(moderated),
			moderationReason, sqliteBool(blacklisted),
			parsePBTimestamp(scheduledAt), parsePBTimestamp(deleted),
			nilIfZero(oldID), status, typ,
			parsePBTimestampRequired(created), parsePBTimestampRequired(updated))
		if err != nil {
			log.Printf("  warning: skipping schematic %s: %v", id, err)
		} else {
			count++
		}
	}
	log.Printf("  -> %d rows", count)
	return nil
}

// ────────────────────────────────────────────────────────────
// Helpers for FK pre-filtering
// ────────────────────────────────────────────────────────────

// loadPGSchematicIDs returns a set of schematic IDs that exist in PostgreSQL.
func (m *Migrator) loadPGSchematicIDs(ctx context.Context) (map[string]bool, error) {
	rows, err := m.pg.Query(ctx, "SELECT id FROM schematics")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make(map[string]bool)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids[id] = true
	}
	return ids, nil
}

// loadPGUserIDs returns a set of user IDs that exist in PostgreSQL.
func (m *Migrator) loadPGUserIDs(ctx context.Context) (map[string]bool, error) {
	rows, err := m.pg.Query(ctx, "SELECT id FROM users")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := make(map[string]bool)
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids[id] = true
	}
	return ids, nil
}

// ────────────────────────────────────────────────────────────
// Phase 3: Depends on schematics
// ────────────────────────────────────────────────────────────

func (m *Migrator) migrateSchematiccategories(ctx context.Context) error {
	if !m.tableExists("schematics") {
		return nil
	}
	rows, err := m.sqlite.Query("SELECT id, COALESCE(categories,'') FROM schematics")
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, categoriesJSON string
		if err := rows.Scan(&id, &categoriesJSON); err != nil {
			return err
		}
		catIDs := parseJSONStringArray(categoriesJSON)
		for _, catID := range catIDs {
			_, err := m.pg.Exec(ctx,
				`INSERT INTO schematics_categories (schematic_id, category_id)
				 VALUES ($1, $2) ON CONFLICT DO NOTHING`,
				id, catID)
			if err != nil {
				log.Printf("  warning: skipping schematics_categories %s/%s: %v", id, catID, err)
			} else {
				count++
			}
		}
	}
	log.Printf("  -> %d rows", count)
	return nil
}

func (m *Migrator) migrateSchematictags(ctx context.Context) error {
	if !m.tableExists("schematics") {
		return nil
	}
	rows, err := m.sqlite.Query("SELECT id, COALESCE(tags,'') FROM schematics")
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, tagsJSON string
		if err := rows.Scan(&id, &tagsJSON); err != nil {
			return err
		}
		tagIDs := parseJSONStringArray(tagsJSON)
		for _, tagID := range tagIDs {
			_, err := m.pg.Exec(ctx,
				`INSERT INTO schematics_tags (schematic_id, tag_id)
				 VALUES ($1, $2) ON CONFLICT DO NOTHING`,
				id, tagID)
			if err != nil {
				log.Printf("  warning: skipping schematics_tags %s/%s: %v", id, tagID, err)
			} else {
				count++
			}
		}
	}
	log.Printf("  -> %d rows", count)
	return nil
}

func (m *Migrator) migrateSchematicViews(ctx context.Context) error {
	if !m.tableExists("schematic_views") {
		return nil
	}

	// Pre-load the set of migrated schematic IDs to skip orphaned views in-memory
	// instead of generating millions of FK-violation round-trips.
	validSchematics, err := m.loadPGSchematicIDs(ctx)
	if err != nil {
		return fmt.Errorf("loading schematic IDs: %w", err)
	}

	rows, err := m.sqlite.Query(
		`SELECT id, schematic, COALESCE(period,''), COALESCE(type,''),
		        COALESCE(count,0), created, updated
		 FROM schematic_views`)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Batch inserts for performance (2.4M+ rows)
	const batchSize = 1000
	batch := make([][]interface{}, 0, batchSize)
	count, skipped := 0, 0

	flushBatch := func() error {
		if len(batch) == 0 {
			return nil
		}
		_, err := m.pg.CopyFrom(ctx,
			pgx.Identifier{"schematic_views"},
			[]string{"id", "schematic_id", "period", "type", "count", "created", "updated"},
			pgx.CopyFromRows(batch))
		if err != nil {
			// If batch COPY fails (e.g., duplicate key), fall back to individual inserts
			for _, row := range batch {
				_, e := m.pg.Exec(ctx,
					`INSERT INTO schematic_views (id, schematic_id, period, type, count, created, updated)
					 VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT DO NOTHING`,
					row...)
				if e != nil {
					log.Printf("  warning: skipping schematic_views %s: %v", row[0], e)
				} else {
					count++
				}
			}
		} else {
			count += len(batch)
		}
		batch = batch[:0]
		return nil
	}

	for rows.Next() {
		var id, schematicID, period, typ, created, updated string
		var cnt int
		if err := rows.Scan(&id, &schematicID, &period, &typ, &cnt, &created, &updated); err != nil {
			return err
		}
		if !validSchematics[schematicID] {
			skipped++
			continue
		}
		batch = append(batch, []interface{}{
			id, schematicID, period, typ, cnt,
			parsePBTimestampRequired(created), parsePBTimestampRequired(updated),
		})
		if len(batch) >= batchSize {
			if err := flushBatch(); err != nil {
				return err
			}
		}
	}
	if err := flushBatch(); err != nil {
		return err
	}

	if skipped > 0 {
		log.Printf("  skipped %d orphaned rows", skipped)
	}
	log.Printf("  -> %d rows", count)
	return nil
}

func (m *Migrator) migrateSchematicRatings(ctx context.Context) error {
	if !m.tableExists("schematic_ratings") {
		return nil
	}

	validSchematics, err := m.loadPGSchematicIDs(ctx)
	if err != nil {
		return fmt.Errorf("loading schematic IDs: %w", err)
	}
	validUsers, err := m.loadPGUserIDs(ctx)
	if err != nil {
		return fmt.Errorf("loading user IDs: %w", err)
	}

	rows, err := m.sqlite.Query(
		`SELECT id, user, schematic, COALESCE(rating,0), COALESCE(rated_at,''), created, updated
		 FROM schematic_ratings`)
	if err != nil {
		return err
	}
	defer rows.Close()

	count, skipped := 0, 0
	for rows.Next() {
		var id, userID, schematicID, ratedAt, created, updated string
		var rating float64
		if err := rows.Scan(&id, &userID, &schematicID, &rating, &ratedAt, &created, &updated); err != nil {
			return err
		}
		if !validSchematics[schematicID] || !validUsers[userID] {
			skipped++
			continue
		}
		ratedAtTime := parsePBTimestampRequired(ratedAt)
		_, err := m.pg.Exec(ctx,
			`INSERT INTO schematic_ratings (id, user_id, schematic_id, rating, rated_at, created, updated)
			 VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT DO NOTHING`,
			id, userID, schematicID, rating, ratedAtTime,
			parsePBTimestampRequired(created), parsePBTimestampRequired(updated))
		if err != nil {
			log.Printf("  warning: skipping schematic_ratings %s: %v", id, err)
		} else {
			count++
		}
	}
	if skipped > 0 {
		log.Printf("  skipped %d orphaned rows", skipped)
	}
	log.Printf("  -> %d rows", count)
	return nil
}

func (m *Migrator) migrateSchematicDownloads(ctx context.Context) error {
	if !m.tableExists("schematic_downloads") {
		return nil
	}

	// The SQLite schematic_downloads table has two possible schemas:
	// 1. Aggregate counts: (id, schematic, period, type, count, created, updated) — like schematic_views
	// 2. Individual events: (id, schematic, user, created)
	// Detect which schema we have by checking for the "period" column.
	var hasUser bool
	testRows, err := m.sqlite.Query("SELECT sql FROM sqlite_master WHERE type='table' AND name='schematic_downloads'")
	if err == nil {
		defer testRows.Close()
		if testRows.Next() {
			var createSQL string
			_ = testRows.Scan(&createSQL)
			hasUser = strings.Contains(strings.ToLower(createSQL), "\"user\"")
		}
	}

	if !hasUser {
		// Aggregate format — sum the counts per schematic and update the schematics.downloads column
		log.Println("  (aggregate format detected — updating schematics.downloads)")
		rows, err := m.sqlite.Query(
			`SELECT schematic, SUM(count) FROM schematic_downloads GROUP BY schematic`)
		if err != nil {
			return err
		}
		defer rows.Close()
		count := 0
		for rows.Next() {
			var schematicID string
			var total int
			if err := rows.Scan(&schematicID, &total); err != nil {
				return err
			}
			_, err := m.pg.Exec(ctx,
				`UPDATE schematics SET downloads = $1 WHERE id = $2`,
				total, schematicID)
			if err != nil {
				log.Printf("  warning: skipping download count for %s: %v", schematicID, err)
			} else {
				count++
			}
		}
		log.Printf("  -> %d schematics updated with download counts", count)
		return nil
	}

	// Individual event format
	validSchematics, err := m.loadPGSchematicIDs(ctx)
	if err != nil {
		return fmt.Errorf("loading schematic IDs: %w", err)
	}

	rows, err := m.sqlite.Query(
		`SELECT id, schematic, COALESCE(user,''), created
		 FROM schematic_downloads`)
	if err != nil {
		return err
	}
	defer rows.Close()

	count, skipped := 0, 0
	for rows.Next() {
		var id, schematicID, userID, created string
		if err := rows.Scan(&id, &schematicID, &userID, &created); err != nil {
			return err
		}
		if !validSchematics[schematicID] {
			skipped++
			continue
		}
		_, err := m.pg.Exec(ctx,
			`INSERT INTO schematic_downloads (id, schematic_id, user_id, created)
			 VALUES ($1,$2,$3,$4) ON CONFLICT DO NOTHING`,
			id, schematicID, nullStr(userID), parsePBTimestampRequired(created))
		if err != nil {
			log.Printf("  warning: skipping schematic_downloads %s: %v", id, err)
		} else {
			count++
		}
	}
	if skipped > 0 {
		log.Printf("  skipped %d orphaned rows", skipped)
	}
	log.Printf("  -> %d rows", count)
	return nil
}

func (m *Migrator) migrateSchematicVersions(ctx context.Context) error {
	if !m.tableExists("schematic_versions") {
		return nil
	}
	rows, err := m.sqlite.Query(
		`SELECT id, schematic, version, COALESCE(snapshot,''), COALESCE(note,''), created, updated
		 FROM schematic_versions`)
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, schematicID, snapshot, note, created, updated string
		var version int
		if err := rows.Scan(&id, &schematicID, &version, &snapshot, &note, &created, &updated); err != nil {
			return err
		}
		_, err := m.pg.Exec(ctx,
			`INSERT INTO schematic_versions (id, schematic_id, version, snapshot, note, created, updated)
			 VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT DO NOTHING`,
			id, schematicID, version, snapshot, note,
			parsePBTimestampRequired(created), parsePBTimestampRequired(updated))
		if err != nil {
			log.Printf("  warning: skipping schematic_versions %s: %v", id, err)
		} else {
			count++
		}
	}
	log.Printf("  -> %d rows", count)
	return nil
}

func (m *Migrator) migrateSchematicFiles(ctx context.Context) error {
	if !m.tableExists("schematic_files") {
		return nil
	}
	rows, err := m.sqlite.Query(
		`SELECT id, COALESCE(schematic,''), COALESCE(file,''), COALESCE(description,''),
		        created
		 FROM schematic_files`)
	if err != nil {
		return nil // table may not exist
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, schematicID, filename, description, created string
		if err := rows.Scan(&id, &schematicID, &filename, &description, &created); err != nil {
			return err
		}
		ts := parsePBTimestampRequired(created)
		_, err := m.pg.Exec(ctx,
			`INSERT INTO schematic_files (id, schematic_id, filename, original_name, size, mime_type, created, updated)
			 VALUES ($1,$2,$3,$4,0,'',$5,$6) ON CONFLICT DO NOTHING`,
			id, schematicID, filename, description, ts, ts)
		if err != nil {
			log.Printf("  warning: skipping schematic_files %s: %v", id, err)
		} else {
			count++
		}
	}
	log.Printf("  -> %d rows", count)
	return nil
}

func (m *Migrator) migrateSchematicTranslations(ctx context.Context) error {
	if !m.tableExists("schematic_translations") {
		return nil
	}
	rows, err := m.sqlite.Query(
		`SELECT id, schematic, COALESCE(language,''), COALESCE(title,''),
		        COALESCE(description,''), COALESCE(content,''), created, updated
		 FROM schematic_translations`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, schematicID, lang, title, description, content, created, updated string
		if err := rows.Scan(&id, &schematicID, &lang, &title, &description, &content, &created, &updated); err != nil {
			return err
		}
		_, err := m.pg.Exec(ctx,
			`INSERT INTO schematic_translations (id, schematic_id, language, title, description, content, created, updated)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT DO NOTHING`,
			id, schematicID, lang, title, description, content,
			parsePBTimestampRequired(created), parsePBTimestampRequired(updated))
		if err != nil {
			log.Printf("  warning: skipping schematic_translations %s: %v", id, err)
		} else {
			count++
		}
	}
	log.Printf("  -> %d rows", count)
	return nil
}

func (m *Migrator) migrateNBTHashes(ctx context.Context) error {
	if !m.tableExists("nbt_hashes") {
		return nil
	}
	rows, err := m.sqlite.Query(
		`SELECT id, COALESCE(checksum,''), COALESCE(schematic,''), created FROM nbt_hashes`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, hash, schematicID, created string
		if err := rows.Scan(&id, &hash, &schematicID, &created); err != nil {
			return err
		}
		_, err := m.pg.Exec(ctx,
			`INSERT INTO nbt_hashes (id, hash, schematic_id, created)
			 VALUES ($1,$2,$3,$4) ON CONFLICT DO NOTHING`,
			id, hash, nullStr(schematicID), parsePBTimestampRequired(created))
		if err != nil {
			log.Printf("  warning: skipping nbt_hashes %s: %v", id, err)
		} else {
			count++
		}
	}
	log.Printf("  -> %d rows", count)
	return nil
}

func (m *Migrator) migrateComments(ctx context.Context) error {
	if !m.tableExists("comments") {
		return nil
	}
	rows, err := m.sqlite.Query(
		`SELECT id, COALESCE(author,''), COALESCE(schematic,''), COALESCE(parent,''),
		        COALESCE(content,''), COALESCE(published,''), COALESCE(approved,0),
		        COALESCE(type,'comment'), COALESCE(karma,0),
		        created, updated
		 FROM comments`)
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var (
			id, authorID, schematicID, parentID, content, published string
			typ, created, updated                                   string
			approved                                                interface{}
			karma                                                   int
		)
		if err := rows.Scan(&id, &authorID, &schematicID, &parentID, &content, &published,
			&approved, &typ, &karma, &created, &updated); err != nil {
			return err
		}
		_, err := m.pg.Exec(ctx,
			`INSERT INTO comments (id, author_id, schematic_id, parent_id, content, published,
			                       approved, type, karma, postdate, status, name, created, updated)
			 VALUES ($1,$2,$3,NULL,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) ON CONFLICT DO NOTHING`,
			id, nullStr(authorID), nullStr(schematicID), content,
			parsePBTimestamp(published), sqliteBool(approved), typ, karma,
			parsePBTimestamp(published), "", "",
			parsePBTimestampRequired(created), parsePBTimestampRequired(updated))
		if err != nil {
			log.Printf("  warning: skipping comment %s: %v", id, err)
		} else {
			count++
		}
	}
	log.Printf("  -> %d rows", count)
	return nil
}

func (m *Migrator) updateCommentParents(ctx context.Context) error {
	if !m.tableExists("comments") {
		return nil
	}
	rows, err := m.sqlite.Query("SELECT id, parent FROM comments WHERE parent != ''")
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, parentID string
		if err := rows.Scan(&id, &parentID); err != nil {
			return err
		}
		_, err := m.pg.Exec(ctx,
			"UPDATE comments SET parent_id = $1 WHERE id = $2 AND parent_id IS NULL",
			parentID, id)
		if err != nil {
			log.Printf("  warning: skipping comment parent %s -> %s: %v", id, parentID, err)
		} else {
			count++
		}
	}
	log.Printf("  -> %d parent refs updated", count)
	return nil
}

// ────────────────────────────────────────────────────────────
// Phase 4: Depends on users + schematics
// ────────────────────────────────────────────────────────────

func (m *Migrator) migrateGuides(ctx context.Context) error {
	if !m.tableExists("guides") {
		return nil
	}
	rows, err := m.sqlite.Query(
		`SELECT id, COALESCE(author,''), COALESCE(title,''), COALESCE(excerpt,''),
		        COALESCE(content,''), COALESCE(name,''),
		        created, updated
		 FROM guides`)
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, authorID, title, description, content, slug, created, updated string
		if err := rows.Scan(&id, &authorID, &title, &description, &content, &slug, &created, &updated); err != nil {
			return err
		}
		_, err := m.pg.Exec(ctx,
			`INSERT INTO guides (id, author_id, title, description, content, slug, upload_link, created, updated)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT DO NOTHING`,
			id, nullStr(authorID), title, description, content, slug, "",
			parsePBTimestampRequired(created), parsePBTimestampRequired(updated))
		if err != nil {
			log.Printf("  warning: skipping guide %s: %v", id, err)
		} else {
			count++
		}
	}
	log.Printf("  -> %d rows", count)
	return nil
}

func (m *Migrator) migrateGuideTranslations(ctx context.Context) error {
	if !m.tableExists("guide_translations") {
		return nil
	}
	rows, err := m.sqlite.Query(
		`SELECT id, guide, COALESCE(language,''), COALESCE(title,''),
		        COALESCE(description,''), COALESCE(content,''), created, updated
		 FROM guide_translations`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, guideID, lang, title, description, content, created, updated string
		if err := rows.Scan(&id, &guideID, &lang, &title, &description, &content, &created, &updated); err != nil {
			return err
		}
		_, err := m.pg.Exec(ctx,
			`INSERT INTO guide_translations (id, guide_id, language, title, description, content, created, updated)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT DO NOTHING`,
			id, guideID, lang, title, description, content,
			parsePBTimestampRequired(created), parsePBTimestampRequired(updated))
		if err != nil {
			log.Printf("  warning: skipping guide_translation %s: %v", id, err)
		} else {
			count++
		}
	}
	log.Printf("  -> %d rows", count)
	return nil
}

func (m *Migrator) migrateCollections(ctx context.Context) error {
	if !m.tableExists("collections") {
		return nil
	}
	rows, err := m.sqlite.Query(
		`SELECT id, COALESCE(author,''), COALESCE(title,''), COALESCE(name,''),
		        COALESCE(slug,''), COALESCE(description,''), COALESCE(banner_url,''),
		        COALESCE(featured,0), COALESCE(views,0), COALESCE(published,0),
		        COALESCE(deleted,''), created, updated
		 FROM collections`)
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, authorID, title, name, slug, description, bannerURL, deleted, created, updated string
		var featured, published interface{}
		var views int
		if err := rows.Scan(&id, &authorID, &title, &name, &slug, &description, &bannerURL,
			&featured, &views, &published, &deleted, &created, &updated); err != nil {
			return err
		}
		_, err := m.pg.Exec(ctx,
			`INSERT INTO collections (id, author_id, title, name, slug, description, banner_url,
			                          featured, views, published, deleted, created, updated)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13) ON CONFLICT DO NOTHING`,
			id, nullStr(authorID), title, name, slug, description, bannerURL,
			sqliteBool(featured), views, sqliteBool(published), deleted,
			parsePBTimestampRequired(created), parsePBTimestampRequired(updated))
		if err != nil {
			log.Printf("  warning: skipping collection %s: %v", id, err)
		} else {
			count++
		}
	}
	log.Printf("  -> %d rows", count)
	return nil
}

func (m *Migrator) migrateCollectionsSchematics(ctx context.Context) error {
	if !m.tableExists("collections") {
		return nil
	}
	rows, err := m.sqlite.Query("SELECT id, COALESCE(schematics,'') FROM collections")
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, schematicsJSON string
		if err := rows.Scan(&id, &schematicsJSON); err != nil {
			return err
		}
		schematicIDs := parseJSONStringArray(schematicsJSON)
		for pos, schematicID := range schematicIDs {
			_, err := m.pg.Exec(ctx,
				`INSERT INTO collections_schematics (collection_id, schematic_id, position)
				 VALUES ($1, $2, $3) ON CONFLICT DO NOTHING`,
				id, schematicID, pos)
			if err != nil {
				log.Printf("  warning: skipping collections_schematics %s/%s: %v", id, schematicID, err)
			} else {
				count++
			}
		}
	}
	log.Printf("  -> %d rows", count)
	return nil
}

func (m *Migrator) migrateCollectionTranslations(ctx context.Context) error {
	if !m.tableExists("collection_translations") {
		return nil
	}
	rows, err := m.sqlite.Query(
		`SELECT id, collection, COALESCE(language,''), COALESCE(title,''),
		        COALESCE(description,''), created, updated
		 FROM collection_translations`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, collectionID, lang, title, description, created, updated string
		if err := rows.Scan(&id, &collectionID, &lang, &title, &description, &created, &updated); err != nil {
			return err
		}
		_, err := m.pg.Exec(ctx,
			`INSERT INTO collection_translations (id, collection_id, language, title, description, created, updated)
			 VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT DO NOTHING`,
			id, collectionID, lang, title, description,
			parsePBTimestampRequired(created), parsePBTimestampRequired(updated))
		if err != nil {
			log.Printf("  warning: skipping collection_translation %s: %v", id, err)
		} else {
			count++
		}
	}
	log.Printf("  -> %d rows", count)
	return nil
}

// ────────────────────────────────────────────────────────────
// Phase 5: Remaining tables
// ────────────────────────────────────────────────────────────

func (m *Migrator) migrateUserAchievements(ctx context.Context) error {
	if !m.tableExists("user_achievements") {
		return nil
	}

	achMap, err := m.buildAchievementIDMap(ctx)
	if err != nil {
		return fmt.Errorf("building achievement ID map: %w", err)
	}

	rows, err := m.sqlite.Query(
		`SELECT id, user, achievement, created, updated FROM user_achievements`)
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, userID, achID, created, updated string
		if err := rows.Scan(&id, &userID, &achID, &created, &updated); err != nil {
			return err
		}
		pgAchID := achID
		if mapped, ok := achMap[achID]; ok {
			pgAchID = mapped
		}
		_, err := m.pg.Exec(ctx,
			`INSERT INTO user_achievements (id, user_id, achievement_id, created, updated)
			 VALUES ($1,$2,$3,$4,$5) ON CONFLICT DO NOTHING`,
			id, userID, pgAchID,
			parsePBTimestampRequired(created), parsePBTimestampRequired(updated))
		if err != nil {
			log.Printf("  warning: skipping user_achievement %s: %v", id, err)
		} else {
			count++
		}
	}
	log.Printf("  -> %d rows", count)
	return nil
}

func (m *Migrator) buildAchievementIDMap(ctx context.Context) (map[string]string, error) {
	result := make(map[string]string)

	pbAchs := make(map[string]string)
	if m.tableExists("achievements") {
		rows, err := m.sqlite.Query("SELECT id, key FROM achievements")
		if err != nil {
			return nil, err
		}
		defer rows.Close()
		for rows.Next() {
			var id, key string
			if err := rows.Scan(&id, &key); err != nil {
				return nil, err
			}
			pbAchs[key] = id
		}
	}

	pgRows, err := m.pg.Query(ctx, "SELECT id, key FROM achievements")
	if err != nil {
		return nil, err
	}
	defer pgRows.Close()

	pgAchs := make(map[string]string)
	for pgRows.Next() {
		var id, key string
		if err := pgRows.Scan(&id, &key); err != nil {
			return nil, err
		}
		pgAchs[key] = id
	}

	for key, pbID := range pbAchs {
		if pgID, ok := pgAchs[key]; ok && pbID != pgID {
			result[pbID] = pgID
		}
	}

	return result, nil
}

func (m *Migrator) migratePointLog(ctx context.Context) error {
	if !m.tableExists("point_log") {
		return nil
	}
	rows, err := m.sqlite.Query(
		`SELECT id, user, COALESCE(points,0), COALESCE(reason,''),
		        COALESCE(description,''), COALESCE(earned_at,''), created, updated
		 FROM point_log`)
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, userID, reason, description, earnedAt, created, updated string
		var points int
		if err := rows.Scan(&id, &userID, &points, &reason, &description, &earnedAt, &created, &updated); err != nil {
			return err
		}
		_, err := m.pg.Exec(ctx,
			`INSERT INTO point_log (id, user_id, points, reason, description, earned_at, created, updated)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8) ON CONFLICT DO NOTHING`,
			id, userID, points, reason, description,
			parsePBTimestampRequired(earnedAt),
			parsePBTimestampRequired(created), parsePBTimestampRequired(updated))
		if err != nil {
			log.Printf("  warning: skipping point_log %s: %v", id, err)
		} else {
			count++
		}
	}
	log.Printf("  -> %d rows", count)
	return nil
}

func (m *Migrator) migrateAPIKeys(ctx context.Context) error {
	if !m.tableExists("api_keys") {
		return nil
	}
	rows, err := m.sqlite.Query(
		`SELECT id, user, COALESCE(key_hash,''), COALESCE(label,''),
		        COALESCE(last8,''), created, updated
		 FROM api_keys`)
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, userID, keyHash, label, last8, created, updated string
		if err := rows.Scan(&id, &userID, &keyHash, &label, &last8, &created, &updated); err != nil {
			return err
		}
		_, err := m.pg.Exec(ctx,
			`INSERT INTO api_keys (id, user_id, key_hash, label, last8, created, updated)
			 VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT DO NOTHING`,
			id, userID, keyHash, label, last8,
			parsePBTimestampRequired(created), parsePBTimestampRequired(updated))
		if err != nil {
			log.Printf("  warning: skipping api_key %s: %v", id, err)
		} else {
			count++
		}
	}
	log.Printf("  -> %d rows", count)
	return nil
}

func (m *Migrator) migrateAPIKeyUsage(ctx context.Context) error {
	if !m.tableExists("api_key_usage") {
		return nil
	}
	rows, err := m.sqlite.Query(
		`SELECT id, key, COALESCE(endpoint,''), last_request FROM api_key_usage`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, apiKeyID, endpoint, lastRequest string
		if err := rows.Scan(&id, &apiKeyID, &endpoint, &lastRequest); err != nil {
			return err
		}
		_, err := m.pg.Exec(ctx,
			`INSERT INTO api_key_usage (id, api_key_id, endpoint, created)
			 VALUES ($1,$2,$3,$4) ON CONFLICT DO NOTHING`,
			id, apiKeyID, endpoint, parsePBTimestampRequired(lastRequest))
		if err != nil {
			log.Printf("  warning: skipping api_key_usage %s: %v", id, err)
		} else {
			count++
		}
	}
	log.Printf("  -> %d rows", count)
	return nil
}

func (m *Migrator) migrateNews(ctx context.Context) error {
	if !m.tableExists("news") {
		return nil
	}
	rows, err := m.sqlite.Query(
		`SELECT id, COALESCE(author,''), COALESCE(title,''), COALESCE(content,''),
		        COALESCE(excerpt,''), COALESCE(postdate,''), COALESCE(status,''),
		        COALESCE(name,''), COALESCE(type,''), created, updated
		 FROM news`)
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, authorID, title, content, excerpt, postdate, status, name, typ, created, updated string
		if err := rows.Scan(&id, &authorID, &title, &content, &excerpt, &postdate, &status, &name, &typ, &created, &updated); err != nil {
			return err
		}
		_, err := m.pg.Exec(ctx,
			`INSERT INTO news (id, author_id, title, content, excerpt, postdate, status, name, type, created, updated)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) ON CONFLICT DO NOTHING`,
			id, nullStr(authorID), title, content, excerpt, parsePBTimestamp(postdate),
			status, name, typ,
			parsePBTimestampRequired(created), parsePBTimestampRequired(updated))
		if err != nil {
			log.Printf("  warning: skipping news %s: %v", id, err)
		} else {
			count++
		}
	}
	log.Printf("  -> %d rows", count)
	return nil
}

func (m *Migrator) migratePages(ctx context.Context) error {
	if !m.tableExists("pages") {
		return nil
	}
	rows, err := m.sqlite.Query(
		`SELECT id, COALESCE(author,''), COALESCE(title,''), COALESCE(content,''),
		        COALESCE(excerpt,''), COALESCE(name,''), COALESCE(status,''),
		        COALESCE(type,''), COALESCE(postdate,''), COALESCE(modified,''),
		        COALESCE(menu_order,0), COALESCE(comment_count,0), created, updated
		 FROM pages`)
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, authorID, title, content, excerpt, name, status, typ string
		var postdate, modified, created, updated string
		var menuOrder, commentCount int
		if err := rows.Scan(&id, &authorID, &title, &content, &excerpt, &name, &status, &typ,
			&postdate, &modified, &menuOrder, &commentCount, &created, &updated); err != nil {
			return err
		}
		_, err := m.pg.Exec(ctx,
			`INSERT INTO pages (id, author_id, title, content, excerpt, name, status, type,
			                    postdate, modified, menu_order, comment_count, created, updated)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14) ON CONFLICT DO NOTHING`,
			id, nullStr(authorID), title, content, excerpt, name, status, typ,
			parsePBTimestamp(postdate), parsePBTimestamp(modified), menuOrder, commentCount,
			parsePBTimestampRequired(created), parsePBTimestampRequired(updated))
		if err != nil {
			log.Printf("  warning: skipping page %s: %v", id, err)
		} else {
			count++
		}
	}
	log.Printf("  -> %d rows", count)
	return nil
}

func (m *Migrator) migrateSearches(ctx context.Context) error {
	if !m.tableExists("searches") {
		return nil
	}
	rows, err := m.sqlite.Query(
		`SELECT id, COALESCE(term,''), COALESCE(results,0), created
		 FROM searches`)
	if err != nil {
		return err
	}
	defer rows.Close()

	// Batch inserts for performance (1M+ rows)
	const batchSize = 1000
	batch := make([][]interface{}, 0, batchSize)
	count := 0

	flushBatch := func() error {
		if len(batch) == 0 {
			return nil
		}
		_, err := m.pg.CopyFrom(ctx,
			pgx.Identifier{"searches"},
			[]string{"id", "query", "results_count", "user_id", "ip_address", "created"},
			pgx.CopyFromRows(batch))
		if err != nil {
			// Fall back to individual inserts on duplicate key
			for _, row := range batch {
				_, e := m.pg.Exec(ctx,
					`INSERT INTO searches (id, query, results_count, user_id, ip_address, created)
					 VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT DO NOTHING`,
					row...)
				if e == nil {
					count++
				}
			}
		} else {
			count += len(batch)
		}
		batch = batch[:0]
		return nil
	}

	for rows.Next() {
		var id, query, created string
		var resultsCount int
		if err := rows.Scan(&id, &query, &resultsCount, &created); err != nil {
			return err
		}
		batch = append(batch, []interface{}{
			id, query, resultsCount, nil, "",
			parsePBTimestampRequired(created),
		})
		if len(batch) >= batchSize {
			if err := flushBatch(); err != nil {
				return err
			}
		}
	}
	if err := flushBatch(); err != nil {
		return err
	}

	log.Printf("  -> %d rows", count)
	return nil
}

func (m *Migrator) migrateContactFormSubmissions(ctx context.Context) error {
	if !m.tableExists("contact_form_submissions") {
		return nil
	}
	rows, err := m.sqlite.Query(
		`SELECT id, COALESCE(email,''), COALESCE(content,''), created, updated
		 FROM contact_form_submissions`)
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, email, content, created, updated string
		if err := rows.Scan(&id, &email, &content, &created, &updated); err != nil {
			return err
		}
		_, err := m.pg.Exec(ctx,
			`INSERT INTO contact_form_submissions (id, author_id, title, content, name, postdate, status, type, created, updated)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10) ON CONFLICT DO NOTHING`,
			id, nil, "", content, email, nil,
			"", "",
			parsePBTimestampRequired(created), parsePBTimestampRequired(updated))
		if err != nil {
			log.Printf("  warning: skipping contact_form_submission %s: %v", id, err)
		} else {
			count++
		}
	}
	log.Printf("  -> %d rows", count)
	return nil
}

func (m *Migrator) migrateOutgoingClicks(ctx context.Context) error {
	if !m.tableExists("outgoing_clicks") {
		return nil
	}
	rows, err := m.sqlite.Query(
		`SELECT id, COALESCE(url,''), COALESCE(source_type,''), COALESCE(source_id,''),
		        created
		 FROM outgoing_clicks`)
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, url, source, sourceID, created string
		if err := rows.Scan(&id, &url, &source, &sourceID, &created); err != nil {
			return err
		}
		_, err := m.pg.Exec(ctx,
			`INSERT INTO outgoing_clicks (id, url, source, source_id, user_id, created)
			 VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT DO NOTHING`,
			id, url, source, sourceID, nil,
			parsePBTimestampRequired(created))
		if err != nil {
			log.Printf("  warning: skipping outgoing_click %s: %v", id, err)
		} else {
			count++
		}
	}
	log.Printf("  -> %d rows", count)
	return nil
}

func (m *Migrator) migrateReports(ctx context.Context) error {
	if !m.tableExists("reports") {
		return nil
	}
	rows, err := m.sqlite.Query(
		`SELECT id, COALESCE(target_type,''), COALESCE(target_id,''),
		        COALESCE(reason,''), COALESCE(reporter,''), created, updated
		 FROM reports`)
	if err != nil {
		return err
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, targetType, targetID, reason, reporter, created, updated string
		if err := rows.Scan(&id, &targetType, &targetID, &reason, &reporter, &created, &updated); err != nil {
			return err
		}
		_, err := m.pg.Exec(ctx,
			`INSERT INTO reports (id, target_type, target_id, reason, reporter, created, updated)
			 VALUES ($1,$2,$3,$4,$5,$6,$7) ON CONFLICT DO NOTHING`,
			id, targetType, targetID, reason, reporter,
			parsePBTimestampRequired(created), parsePBTimestampRequired(updated))
		if err != nil {
			log.Printf("  warning: skipping report %s: %v", id, err)
		} else {
			count++
		}
	}
	log.Printf("  -> %d rows", count)
	return nil
}

func (m *Migrator) migrateTempUploads(ctx context.Context) error {
	if !m.tableExists("temp_uploads") {
		return nil
	}
	rows, err := m.sqlite.Query(
		`SELECT id, COALESCE(token,''), COALESCE(filename,''), COALESCE(size,0),
		        COALESCE(checksum,''), COALESCE(parsed_summary,''), COALESCE(nbt_file,''),
		        COALESCE(block_count,0), COALESCE(dim_x,0), COALESCE(dim_y,0), COALESCE(dim_z,0),
		        COALESCE(materials,'[]'), COALESCE(mods,'[]'),
		        COALESCE(uploaded_by,''), created, updated
		 FROM temp_uploads`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, token, filename, checksum, parsedSummary, nbtFile string
		var materialsJSON, modsJSON, uploadedBy, created, updated string
		var size int64
		var blockCount, dimX, dimY, dimZ int
		if err := rows.Scan(&id, &token, &filename, &size, &checksum, &parsedSummary, &nbtFile,
			&blockCount, &dimX, &dimY, &dimZ, &materialsJSON, &modsJSON,
			&uploadedBy, &created, &updated); err != nil {
			return err
		}
		if materialsJSON == "" {
			materialsJSON = "[]"
		}
		if modsJSON == "" {
			modsJSON = "[]"
		}
		_, err := m.pg.Exec(ctx,
			`INSERT INTO temp_uploads (id, token, filename, size, checksum, parsed_summary, nbt_file,
			                           block_count, dim_x, dim_y, dim_z, materials, mods,
			                           uploaded_by, created, updated)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$15,$16) ON CONFLICT DO NOTHING`,
			id, token, filename, size, checksum, parsedSummary, nbtFile,
			blockCount, dimX, dimY, dimZ, materialsJSON, modsJSON,
			uploadedBy, parsePBTimestampRequired(created), parsePBTimestampRequired(updated))
		if err != nil {
			log.Printf("  warning: skipping temp_upload %s: %v", id, err)
		} else {
			count++
		}
	}
	log.Printf("  -> %d rows", count)
	return nil
}

func (m *Migrator) migrateTempUploadFiles(ctx context.Context) error {
	if !m.tableExists("temp_upload_files") {
		return nil
	}
	rows, err := m.sqlite.Query(
		`SELECT id, COALESCE(token,''), COALESCE(filename,''), COALESCE(description,''),
		        COALESCE(size,0), COALESCE(checksum,''),
		        COALESCE(block_count,0), COALESCE(dim_x,0), COALESCE(dim_y,0), COALESCE(dim_z,0),
		        COALESCE(nbt_file,''), created
		 FROM temp_upload_files`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, token, filename, description, checksum, nbtFile, created string
		var size int64
		var blockCount, dimX, dimY, dimZ int
		if err := rows.Scan(&id, &token, &filename, &description, &size, &checksum,
			&blockCount, &dimX, &dimY, &dimZ, &nbtFile, &created); err != nil {
			return err
		}
		_, err := m.pg.Exec(ctx,
			`INSERT INTO temp_upload_files (id, token, filename, description, size, checksum,
			                                block_count, dim_x, dim_y, dim_z, nbt_s3_key, created)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12) ON CONFLICT DO NOTHING`,
			id, token, filename, description, size, checksum,
			blockCount, dimX, dimY, dimZ, nbtFile,
			parsePBTimestampRequired(created))
		if err != nil {
			log.Printf("  warning: skipping temp_upload_file %s: %v", id, err)
		} else {
			count++
		}
	}
	log.Printf("  -> %d rows", count)
	return nil
}

func (m *Migrator) migrateModMetadata(ctx context.Context) error {
	if !m.tableExists("mod_metadata") {
		return nil
	}
	rows, err := m.sqlite.Query(
		`SELECT id, COALESCE(namespace,''), COALESCE(display_name,''), COALESCE(description,''),
		        COALESCE(icon_url,''), COALESCE(modrinth_slug,''), COALESCE(modrinth_url,''),
		        COALESCE(curseforge_id,''), COALESCE(curseforge_url,''), COALESCE(source_url,''),
		        COALESCE(last_fetched,''), COALESCE(manually_set,0), created, updated
		 FROM mod_metadata`)
	if err != nil {
		return nil
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id, namespace, displayName, description, iconURL string
		var modrinthSlug, modrinthURL, curseforgeID, curseforgeURL, sourceURL string
		var lastFetched, created, updated string
		var manuallySet interface{}
		if err := rows.Scan(&id, &namespace, &displayName, &description, &iconURL,
			&modrinthSlug, &modrinthURL, &curseforgeID, &curseforgeURL, &sourceURL,
			&lastFetched, &manuallySet, &created, &updated); err != nil {
			return err
		}
		_, err := m.pg.Exec(ctx,
			`INSERT INTO mod_metadata (id, namespace, display_name, description, icon_url,
			                           modrinth_slug, modrinth_url, curseforge_id, curseforge_url,
			                           source_url, last_fetched, manually_set, created, updated)
			 VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14) ON CONFLICT DO NOTHING`,
			id, namespace, displayName, description, iconURL,
			modrinthSlug, modrinthURL, curseforgeID, curseforgeURL, sourceURL,
			parsePBTimestamp(lastFetched), sqliteBool(manuallySet),
			parsePBTimestampRequired(created), parsePBTimestampRequired(updated))
		if err != nil {
			log.Printf("  warning: skipping mod_metadata %s: %v", id, err)
		} else {
			count++
		}
	}
	log.Printf("  -> %d rows", count)
	return nil
}
