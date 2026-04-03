// Package database provides the PostgreSQL-backed implementation of all
// store interfaces defined in internal/store.
package database

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	db "createmod/internal/database/gen"
	"createmod/internal/store"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PostgresStore implements all store interfaces using sqlc-generated queries.
type PostgresStore struct {
	q    *db.Queries
	pool *pgxpool.Pool
}

// NewPostgresStore creates a new PostgresStore backed by the given pool.
func NewPostgresStore(pool *pgxpool.Pool) *PostgresStore {
	return &PostgresStore{
		q:    db.New(pool),
		pool: pool,
	}
}

// NewStore returns a store.Store wired to the PostgreSQL implementation.
// It delegates to NewStoreFromPool which uses separate impl types
// to avoid method name collisions between interfaces.
func NewStore(pool *pgxpool.Pool) *store.Store {
	return NewStoreFromPool(pool)
}

// generateID generates a random 15-character alphanumeric ID matching PocketBase format.
func generateID() string {
	b := make([]byte, 8)
	_, _ = rand.Read(b)
	return hex.EncodeToString(b)[:15]
}

// --------------------------------------------------------------------------
// Helpers: pgtype conversions
// --------------------------------------------------------------------------

func toPgTimestamptz(t *time.Time) pgtype.Timestamptz {
	if t == nil {
		return pgtype.Timestamptz{Valid: false}
	}
	return pgtype.Timestamptz{Time: *t, Valid: true}
}

func fromPgTimestamptz(ts pgtype.Timestamptz) *time.Time {
	if !ts.Valid {
		return nil
	}
	t := ts.Time
	return &t
}

// --------------------------------------------------------------------------
// Helpers: db model → store model conversions
// --------------------------------------------------------------------------

func userFromDB(u db.User) *store.User {
	return &store.User{
		ID:           u.ID,
		Email:        u.Email,
		Username:     u.Username,
		PasswordHash: u.PasswordHash,
		OldPassword:  u.OldPassword,
		Avatar:       u.Avatar,
		Points:       int(u.Points),
		Verified:     u.Verified,
		IsAdmin:      u.IsAdmin,
		Deleted:      fromPgTimestamptz(u.Deleted),
		Created:      u.Created,
		Updated:      u.Updated,
	}
}

func schematicFromDB(s db.Schematic) store.Schematic {
	return store.Schematic{
		ID:                 s.ID,
		AuthorID:           derefStr(s.AuthorID),
		Name:               s.Name,
		Title:              s.Title,
		Description:        s.Description,
		Excerpt:            s.Excerpt,
		Content:            s.Content,
		Postdate:           fromPgTimestamptz(s.Postdate),
		Modified:           fromPgTimestamptz(s.Modified),
		DetectedLanguage:   s.DetectedLanguage,
		FeaturedImage:      s.FeaturedImage,
		Gallery:            s.Gallery,
		SchematicFile:      s.SchematicFile,
		Video:              s.Video,
		HasDependencies:    s.HasDependencies,
		Dependencies:       s.Dependencies,
		CreatemodVersionID: s.CreatemodVersionID,
		MinecraftVersionID: s.MinecraftVersionID,
		Views:              int(s.Views),
		Downloads:          int(s.Downloads),
		BlockCount:         int(s.BlockCount),
		DimX:               int(s.DimX),
		DimY:               int(s.DimY),
		DimZ:               int(s.DimZ),
		Materials:          json.RawMessage(s.Materials),
		Mods:               json.RawMessage(s.Mods),
		Paid:               s.Paid,
		ExternalURL:        s.ExternalUrl,
		Featured:           s.Featured,
		AIDescription:      s.AiDescription,
		ModerationState:    s.ModerationState,
		ModerationReason:   s.ModerationReason,
		ScheduledAt:        fromPgTimestamptz(s.ScheduledAt),
		Deleted:            fromPgTimestamptz(s.Deleted),
		OldID:              ptrInt32ToInt(s.OldID),
		Status:             s.Status,
		Type:               s.Type,
		Created:            s.Created,
		Updated:            s.Updated,
	}
}

func schematicSliceFromDB(rows []db.Schematic) []store.Schematic {
	result := make([]store.Schematic, len(rows))
	for i, r := range rows {
		result[i] = schematicFromDB(r)
	}
	return result
}

func commentFromDB(c db.Comment, authorUsername, authorAvatar string) store.Comment {
	return store.Comment{
		ID:             c.ID,
		AuthorID:       c.AuthorID,
		SchematicID:    c.SchematicID,
		ParentID:       c.ParentID,
		Content:        c.Content,
		Published:      fromPgTimestamptz(c.Published),
		Approved:       c.Approved,
		Type:           c.Type,
		Karma:          int(c.Karma),
		AuthorUsername: authorUsername,
		AuthorAvatar:   authorAvatar,
		Created:        c.Created,
		Updated:        c.Updated,
	}
}

func guideFromDB(g db.Guide) store.Guide {
	return store.Guide{
		ID:          g.ID,
		AuthorID:    g.AuthorID,
		Title:       g.Title,
		Description: g.Description,
		Content:     g.Content,
		Slug:        g.Slug,
		UploadLink:  g.UploadLink,
		BannerURL:   g.BannerUrl,
		Views:       int(g.Views),
		Created:     g.Created,
		Updated:     g.Updated,
	}
}

func collectionFromDB(c db.Collection) store.Collection {
	return store.Collection{
		ID:          c.ID,
		AuthorID:    c.AuthorID,
		Title:       c.Title,
		Name:        c.Name,
		Slug:        c.Slug,
		Description: c.Description,
		BannerURL:   c.BannerUrl,
		CollageURL:  c.CollageUrl,
		Featured:    c.Featured,
		Views:       int(c.Views),
		Published:   c.Published,
		Deleted:     c.Deleted,
		Created:     c.Created,
		Updated:     c.Updated,
	}
}

func achievementFromDB(a db.Achievement) store.Achievement {
	return store.Achievement{
		ID:          a.ID,
		Key:         a.Key,
		Title:       a.Title,
		Description: a.Description,
		Icon:        a.Icon,
	}
}

func reportFromDB(r db.Report) store.Report {
	return store.Report{
		ID:         r.ID,
		TargetType: r.TargetType,
		TargetID:   r.TargetID,
		Reason:     r.Reason,
		Reporter:   r.Reporter,
		Created:    r.Created,
	}
}

func apiKeyFromDB(k db.ApiKey) store.APIKey {
	return store.APIKey{
		ID:      k.ID,
		UserID:  k.UserID,
		KeyHash: k.KeyHash,
		Label:   k.Label,
		Last8:   k.Last8,
		Created: k.Created,
	}
}

func externalAuthFromDB(ea db.ExternalAuth) store.ExternalAuth {
	return store.ExternalAuth{
		ID:         ea.ID,
		UserID:     ea.UserID,
		Provider:   ea.Provider,
		ProviderID: ea.ProviderID,
		Created:    ea.Created,
	}
}

func modMetadataFromDB(m db.ModMetadatum) store.ModMetadata {
	return store.ModMetadata{
		ID:                 m.ID,
		Namespace:          m.Namespace,
		DisplayName:        m.DisplayName,
		Description:        m.Description,
		IconURL:            m.IconUrl,
		ModrinthSlug:       m.ModrinthSlug,
		ModrinthURL:        m.ModrinthUrl,
		CurseforgeID:       m.CurseforgeID,
		CurseforgeURL:      m.CurseforgeUrl,
		SourceURL:          m.SourceUrl,
		LastFetched:        fromPgTimestamptz(m.LastFetched),
		ManuallySet:        m.ManuallySet,
		BlocksitemsMatched: m.BlocksitemsMatched,
	}
}

// --------------------------------------------------------------------------
// Small pointer helpers
// --------------------------------------------------------------------------

func derefStr(s *string) string {
	if s == nil {
		return ""
	}
	return *s
}

func ptrStr(s string) *string {
	return &s
}

func ptrInt32ToInt(p *int32) *int {
	if p == nil {
		return nil
	}
	v := int(*p)
	return &v
}

// ============================================================================
// UserStoreImpl implements store.UserStore.
// Separated from PostgresStore to avoid method name collision with
// SchematicStore.ListForSitemap.
// ============================================================================

// UserStoreImpl implements store.UserStore.
type UserStoreImpl struct{ q *db.Queries }

func (us *UserStoreImpl) GetUserByID(ctx context.Context, id string) (*store.User, error) {
	u, err := us.q.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return userFromDB(u), nil
}

func (us *UserStoreImpl) GetUserByEmail(ctx context.Context, email string) (*store.User, error) {
	u, err := us.q.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	return userFromDB(u), nil
}

func (us *UserStoreImpl) GetUserByUsername(ctx context.Context, username string) (*store.User, error) {
	u, err := us.q.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	return userFromDB(u), nil
}

func (us *UserStoreImpl) CreateUser(ctx context.Context, u *store.User) error {
	if u.ID == "" {
		u.ID = generateID()
	}
	created, err := us.q.CreateUser(ctx, db.CreateUserParams{
		ID:           u.ID,
		Email:        u.Email,
		Username:     u.Username,
		PasswordHash: u.PasswordHash,
		OldPassword:  u.OldPassword,
		Avatar:       u.Avatar,
		Verified:     u.Verified,
	})
	if err != nil {
		return err
	}
	u.Created = created.Created
	u.Updated = created.Updated
	return nil
}

func (us *UserStoreImpl) UpdateUser(ctx context.Context, u *store.User) error {
	_, err := us.q.UpdateUser(ctx, db.UpdateUserParams{
		ID:           u.ID,
		Email:        ptrStr(u.Email),
		Username:     ptrStr(u.Username),
		PasswordHash: ptrStr(u.PasswordHash),
		OldPassword:  ptrStr(u.OldPassword),
		Avatar:       ptrStr(u.Avatar),
		Points:       ptrInt32(int32(u.Points)),
		Verified:     ptrBool(u.Verified),
		IsAdmin:      ptrBool(u.IsAdmin),
	})
	return err
}

func (us *UserStoreImpl) UpdateUserPoints(ctx context.Context, id string, points int) error {
	return us.q.UpdateUserPoints(ctx, db.UpdateUserPointsParams{
		ID:     id,
		Points: int32(points),
	})
}

func (us *UserStoreImpl) UpdateUserPassword(ctx context.Context, id string, hash string) error {
	return us.q.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		ID:           id,
		PasswordHash: hash,
	})
}

func (us *UserStoreImpl) UpdateUserAvatar(ctx context.Context, id string, avatar string) error {
	return us.q.UpdateUserAvatar(ctx, db.UpdateUserAvatarParams{
		ID:     id,
		Avatar: avatar,
	})
}

func (us *UserStoreImpl) SoftDeleteUser(ctx context.Context, id string) error {
	return us.q.SoftDeleteUser(ctx, id)
}

func (us *UserStoreImpl) IsContributor(ctx context.Context, userID string) (bool, error) {
	return us.q.GetUserIsContributor(ctx, &userID)
}

func (us *UserStoreImpl) ListUsers(ctx context.Context, limit, offset int) ([]store.User, error) {
	rows, err := us.q.ListUsers(ctx, db.ListUsersParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}
	result := make([]store.User, len(rows))
	for i, r := range rows {
		result[i] = *userFromDB(r)
	}
	return result, nil
}

func (us *UserStoreImpl) CountUsers(ctx context.Context) (int64, error) {
	return us.q.CountUsers(ctx)
}

func (us *UserStoreImpl) ListForSitemap(ctx context.Context) ([]store.SitemapUser, error) {
	rows, err := us.q.ListUsersForSitemap(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]store.SitemapUser, len(rows))
	for i, r := range rows {
		result[i] = store.SitemapUser{
			ID:       r.ID,
			Username: r.Username,
			Updated:  r.Updated,
		}
	}
	return result, nil
}

func (us *UserStoreImpl) ListAdminEmails(ctx context.Context) ([]string, error) {
	return us.q.ListAdminEmails(ctx)
}

// ============================================================================
// SessionStore implementation
// ============================================================================

func (ps *PostgresStore) CreateSession(ctx context.Context, s *store.Session) error {
	if s.ID == "" {
		s.ID = generateID()
	}
	created, err := ps.q.CreateSession(ctx, db.CreateSessionParams{
		ID:        s.ID,
		UserID:    s.UserID,
		ExpiresAt: s.ExpiresAt,
	})
	if err != nil {
		return err
	}
	s.Created = created.Created
	return nil
}

func (ps *PostgresStore) GetSession(ctx context.Context, id string) (*store.Session, error) {
	row, err := ps.q.GetSession(ctx, id)
	if err != nil {
		return nil, err
	}
	return &store.Session{
		ID:        row.ID,
		UserID:    row.UserID,
		ExpiresAt: row.ExpiresAt,
		Created:   row.Created,
		User: &store.User{
			ID:       row.UserID_2,
			Email:    row.UserEmail,
			Username: row.UserUsername,
			Avatar:   row.UserAvatar,
			Points:   int(row.UserPoints),
			IsAdmin:  row.UserIsAdmin,
			Verified: row.UserVerified,
		},
	}, nil
}

func (ps *PostgresStore) DeleteSession(ctx context.Context, id string) error {
	return ps.q.DeleteSession(ctx, id)
}

func (ps *PostgresStore) DeleteUserSessions(ctx context.Context, userID string) error {
	return ps.q.DeleteUserSessions(ctx, userID)
}

func (ps *PostgresStore) CleanupExpired(ctx context.Context) error {
	return ps.q.CleanupExpiredSessions(ctx)
}

// ============================================================================
// SchematicStore implementation
// ============================================================================

func (ps *PostgresStore) GetByID(ctx context.Context, id string) (*store.Schematic, error) {
	row, err := ps.q.GetSchematicByID(ctx, id)
	if err != nil {
		return nil, err
	}
	s := schematicFromDB(row)
	return &s, nil
}

func (ps *PostgresStore) GetByName(ctx context.Context, name string) (*store.Schematic, error) {
	row, err := ps.q.GetSchematicByName(ctx, name)
	if err != nil {
		return nil, err
	}
	s := schematicFromDB(row)
	return &s, nil
}

func (ps *PostgresStore) ListApproved(ctx context.Context, limit, offset int) ([]store.Schematic, error) {
	rows, err := ps.q.ListApprovedSchematics(ctx, db.ListApprovedSchematicsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return schematicSliceFromDB(rows), nil
}

func (ps *PostgresStore) CountApproved(ctx context.Context) (int64, error) {
	return ps.q.CountApprovedSchematics(ctx)
}

func (ps *PostgresStore) ListByAuthor(ctx context.Context, authorID string, limit, offset int) ([]store.Schematic, error) {
	rows, err := ps.q.ListSchematicsByAuthor(ctx, db.ListSchematicsByAuthorParams{
		AuthorID: &authorID,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return schematicSliceFromDB(rows), nil
}

func (ps *PostgresStore) ListByAuthorExcluding(ctx context.Context, authorID, excludeID string, limit int) ([]store.Schematic, error) {
	rows, err := ps.q.ListSchematicsByAuthorExcluding(ctx, db.ListSchematicsByAuthorExcludingParams{
		AuthorID: &authorID,
		ID:       excludeID,
		Limit:    int32(limit),
	})
	if err != nil {
		return nil, err
	}
	return schematicSliceFromDB(rows), nil
}

func (ps *PostgresStore) ListByIDs(ctx context.Context, ids []string) ([]store.Schematic, error) {
	rows, err := ps.q.ListSchematicsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	return schematicSliceFromDB(rows), nil
}

func (ps *PostgresStore) ListFeatured(ctx context.Context, limit int) ([]store.Schematic, error) {
	rows, err := ps.q.ListFeaturedSchematics(ctx, int32(limit))
	if err != nil {
		return nil, err
	}
	return schematicSliceFromDB(rows), nil
}

func (ps *PostgresStore) ListAllForIndex(ctx context.Context) ([]store.Schematic, error) {
	rows, err := ps.q.ListAllApprovedSchematicsForIndex(ctx)
	if err != nil {
		return nil, err
	}
	return schematicSliceFromDB(rows), nil
}

func (ps *PostgresStore) Create(ctx context.Context, s *store.Schematic) error {
	if s.ID == "" {
		s.ID = generateID()
	}
	created, err := ps.q.CreateSchematic(ctx, db.CreateSchematicParams{
		ID:                 s.ID,
		AuthorID:           ptrStrNonEmpty(s.AuthorID),
		Name:               s.Name,
		Title:              s.Title,
		Description:        s.Description,
		Excerpt:            s.Excerpt,
		Content:            s.Content,
		Postdate:           toPgTimestamptz(s.Postdate),
		DetectedLanguage:   s.DetectedLanguage,
		FeaturedImage:      s.FeaturedImage,
		Gallery:            s.Gallery,
		SchematicFile:      s.SchematicFile,
		Video:              s.Video,
		HasDependencies:    s.HasDependencies,
		Dependencies:       s.Dependencies,
		CreatemodVersionID: s.CreatemodVersionID,
		MinecraftVersionID: s.MinecraftVersionID,
		BlockCount:         int32(s.BlockCount),
		DimX:               int32(s.DimX),
		DimY:               int32(s.DimY),
		DimZ:               int32(s.DimZ),
		Materials:          json.RawMessage(s.Materials),
		Mods:               json.RawMessage(s.Mods),
		Paid:               s.Paid,
		ModerationState:    s.ModerationState,
		Type:               s.Type,
		Status:             s.Status,
	})
	if err != nil {
		return err
	}
	s.Created = created.Created
	s.Updated = created.Updated
	return nil
}

func (ps *PostgresStore) Update(ctx context.Context, s *store.Schematic) error {
	_, err := ps.q.UpdateSchematic(ctx, db.UpdateSchematicParams{
		ID:                 s.ID,
		Title:              ptrStr(s.Title),
		Description:        ptrStr(s.Description),
		Excerpt:            ptrStr(s.Excerpt),
		Content:            ptrStr(s.Content),
		FeaturedImage:      ptrStr(s.FeaturedImage),
		Gallery:            s.Gallery,
		Video:              ptrStr(s.Video),
		HasDependencies:    ptrBool(s.HasDependencies),
		Dependencies:       ptrStr(s.Dependencies),
		CreatemodVersionID: s.CreatemodVersionID,
		MinecraftVersionID: s.MinecraftVersionID,
		AiDescription:      ptrStr(s.AIDescription),
		ModerationState:    ptrStr(s.ModerationState),
		ModerationReason:   ptrStr(s.ModerationReason),
		Featured:           ptrBool(s.Featured),
		ScheduledAt:        toPgTimestamptz(s.ScheduledAt),
		BlockCount:         ptrInt32(int32(s.BlockCount)),
		DimX:               ptrInt32(int32(s.DimX)),
		DimY:               ptrInt32(int32(s.DimY)),
		DimZ:               ptrInt32(int32(s.DimZ)),
		Materials:          s.Materials,
		Mods:               s.Mods,
		Paid:               ptrBool(s.Paid),
		ExternalUrl:        ptrStr(s.ExternalURL),
	})
	return err
}

func (ps *PostgresStore) SoftDelete(ctx context.Context, id string) error {
	return ps.q.SoftDeleteSchematic(ctx, id)
}

func (ps *PostgresStore) SetModerationState(ctx context.Context, id, state, reason string) error {
	return ps.q.SetModerationState(ctx, db.SetModerationStateParams{
		ID:               id,
		ModerationState:  state,
		ModerationReason: reason,
	})
}

func (ps *PostgresStore) ListForAdmin(ctx context.Context, filter string, limit, offset int) ([]store.Schematic, error) {
	rows, err := ps.q.ListSchematicsForAdmin(ctx, db.ListSchematicsForAdminParams{
		Limit:  int32(limit),
		Offset: int32(offset),
		Filter: filter,
	})
	if err != nil {
		return nil, err
	}
	return schematicSliceFromDB(rows), nil
}

func (ps *PostgresStore) CountForAdmin(ctx context.Context, filter string) (int64, error) {
	return ps.q.CountSchematicsForAdmin(ctx, filter)
}

func (ps *PostgresStore) GetByIDAdmin(ctx context.Context, id string) (*store.Schematic, error) {
	row, err := ps.q.GetSchematicByIDAdmin(ctx, id)
	if err != nil {
		return nil, err
	}
	s := schematicFromDB(row)
	return &s, nil
}

func (ps *PostgresStore) GetCategoryIDs(ctx context.Context, schematicID string) ([]string, error) {
	return ps.q.GetSchematicCategoryIDs(ctx, schematicID)
}

func (ps *PostgresStore) GetTagIDs(ctx context.Context, schematicID string) ([]string, error) {
	return ps.q.GetSchematicTagIDs(ctx, schematicID)
}

func (ps *PostgresStore) SetCategories(ctx context.Context, schematicID string, categoryIDs []string) error {
	// Delete existing, then add new ones
	if err := ps.q.SetSchematicCategories(ctx, schematicID); err != nil {
		return err
	}
	for _, catID := range categoryIDs {
		if err := ps.q.AddSchematicCategory(ctx, db.AddSchematicCategoryParams{
			SchematicID: schematicID,
			CategoryID:  catID,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (ps *PostgresStore) SetTags(ctx context.Context, schematicID string, tagIDs []string) error {
	if err := ps.q.SetSchematicTags(ctx, schematicID); err != nil {
		return err
	}
	for _, tagID := range tagIDs {
		if err := ps.q.AddSchematicTag(ctx, db.AddSchematicTagParams{
			SchematicID: schematicID,
			TagID:       tagID,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (ps *PostgresStore) ListApprovedWithVideo(ctx context.Context, limit, offset int) ([]store.Schematic, error) {
	rows, err := ps.q.ListApprovedSchematicsWithVideo(ctx, db.ListApprovedSchematicsWithVideoParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return schematicSliceFromDB(rows), nil
}

func (ps *PostgresStore) ListRandomApproved(ctx context.Context, limit int) ([]store.Schematic, error) {
	rows, err := ps.q.ListRandomApprovedSchematics(ctx, int32(limit))
	if err != nil {
		return nil, err
	}
	return schematicSliceFromDB(rows), nil
}

func (ps *PostgresStore) ListByCategoryIDs(ctx context.Context, categoryIDs []string, excludeIDs []string, limit int) ([]store.Schematic, error) {
	rows, err := ps.q.ListSchematicsByCategoryIDs(ctx, db.ListSchematicsByCategoryIDsParams{
		Column1: categoryIDs,
		Column2: excludeIDs,
		Limit:   int32(limit),
	})
	if err != nil {
		return nil, err
	}
	return schematicSliceFromDB(rows), nil
}

func (ps *PostgresStore) ListHighestRated(ctx context.Context, limit, offset int) ([]store.Schematic, error) {
	rows, err := ps.q.ListHighestRatedSchematics(ctx, db.ListHighestRatedSchematicsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return schematicSliceFromDB(rows), nil
}

func (ps *PostgresStore) ListForSitemap(ctx context.Context) ([]store.SitemapSchematic, error) {
	rows, err := ps.q.ListSchematicsForSitemap(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]store.SitemapSchematic, len(rows))
	for i, r := range rows {
		result[i] = store.SitemapSchematic{
			ID:      r.ID,
			Name:    r.Name,
			Updated: r.Updated,
		}
	}
	return result, nil
}

func (ps *PostgresStore) CountByAuthor(ctx context.Context, authorID string) (int64, error) {
	return ps.q.CountSchematicsByAuthor(ctx, &authorID)
}

func (ps *PostgresStore) CountSoftDeletedByAuthor(ctx context.Context, authorID string) (int64, error) {
	return ps.q.CountSoftDeletedByAuthor(ctx, &authorID)
}

func (ps *PostgresStore) GetByChecksum(ctx context.Context, checksum string) (string, error) {
	id, err := ps.q.GetSchematicByChecksum(ctx, checksum)
	if err != nil {
		return "", err
	}
	if id == nil {
		return "", nil
	}
	return *id, nil
}

func (ps *PostgresStore) UpdateName(ctx context.Context, id, name string) error {
	return ps.q.UpdateSchematicName(ctx, db.UpdateSchematicNameParams{
		ID:   id,
		Name: name,
	})
}

func (ps *PostgresStore) UpdateDetectedLanguage(ctx context.Context, id, lang string) error {
	return ps.q.UpdateSchematicDetectedLanguage(ctx, db.UpdateSchematicDetectedLanguageParams{
		ID:               id,
		DetectedLanguage: lang,
	})
}

func (ps *PostgresStore) ListByAuthorAll(ctx context.Context, authorID string, limit, offset int) ([]store.Schematic, error) {
	rows, err := ps.q.ListSchematicsByAuthorAll(ctx, db.ListSchematicsByAuthorAllParams{
		AuthorID: &authorID,
		Limit:    int32(limit),
		Offset:   int32(offset),
	})
	if err != nil {
		return nil, err
	}
	return schematicSliceFromDB(rows), nil
}

func (ps *PostgresStore) CountByAuthorAll(ctx context.Context, authorID string) (int64, error) {
	return ps.q.CountSchematicsByAuthorAll(ctx, &authorID)
}

func (ps *PostgresStore) ListByNamePattern(ctx context.Context, pattern string, limit int) ([]store.Schematic, error) {
	rows, err := ps.q.ListSchematicsByNamePattern(ctx, db.ListSchematicsByNamePatternParams{
		Name:  pattern,
		Limit: int32(limit),
	})
	if err != nil {
		return nil, err
	}
	return schematicSliceFromDB(rows), nil
}

func (ps *PostgresStore) UpdateTrendingScore(ctx context.Context, id string, score float64) error {
	return ps.q.UpdateSchematicTrendingScore(ctx, db.UpdateSchematicTrendingScoreParams{
		ID:            id,
		TrendingScore: float32(score),
	})
}

func (ps *PostgresStore) UpdateRatingAggregates(ctx context.Context, id string, avgRating float64, ratingCount int) error {
	return ps.q.UpdateSchematicRatingAggregates(ctx, db.UpdateSchematicRatingAggregatesParams{
		ID:          id,
		AvgRating:   float32(avgRating),
		RatingCount: int32(ratingCount),
	})
}

func (ps *PostgresStore) RefreshRatingAggregates(ctx context.Context, id string) error {
	return ps.q.RefreshSchematicRatingAggregates(ctx, id)
}

func (ps *PostgresStore) BatchGetCategoriesForSchematics(ctx context.Context, ids []string) (map[string][]store.SchematicCategoryInfo, error) {
	rows, err := ps.q.BatchGetSchematicCategories(ctx, ids)
	if err != nil {
		return nil, err
	}
	result := make(map[string][]store.SchematicCategoryInfo, len(ids))
	for _, r := range rows {
		result[r.SchematicID] = append(result[r.SchematicID], store.SchematicCategoryInfo{
			ID:   r.ID,
			Key:  r.Key,
			Name: r.Name,
		})
	}
	return result, nil
}

func (ps *PostgresStore) BatchGetTagsForSchematics(ctx context.Context, ids []string) (map[string][]store.SchematicTagInfo, error) {
	rows, err := ps.q.BatchGetSchematicTags(ctx, ids)
	if err != nil {
		return nil, err
	}
	result := make(map[string][]store.SchematicTagInfo, len(ids))
	for _, r := range rows {
		result[r.SchematicID] = append(result[r.SchematicID], store.SchematicTagInfo{
			ID:   r.ID,
			Key:  r.Key,
			Name: r.Name,
		})
	}
	return result, nil
}

func (ps *PostgresStore) ListModCounts(ctx context.Context) ([]store.ModCount, error) {
	rows, err := ps.pool.Query(ctx, `
		SELECT j.mod_name, COUNT(DISTINCT s.id)::int AS count
		FROM schematics s,
		     LATERAL jsonb_array_elements_text(s.mods) AS j(mod_name)
		WHERE s.deleted IS NULL
		  AND s.moderation_state = 'published'
		  AND (s.scheduled_at IS NULL OR s.scheduled_at <= NOW())
		GROUP BY j.mod_name
		ORDER BY count DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("listing mod counts: %w", err)
	}
	defer rows.Close()

	var result []store.ModCount
	for rows.Next() {
		var mc store.ModCount
		if err := rows.Scan(&mc.ModName, &mc.Count); err != nil {
			return nil, fmt.Errorf("scanning mod count: %w", err)
		}
		result = append(result, mc)
	}
	return result, rows.Err()
}

func (ps *PostgresStore) CountVanilla(ctx context.Context) (int, error) {
	var count int
	err := ps.pool.QueryRow(ctx, `
		SELECT COUNT(*)::int FROM schematics
		WHERE deleted IS NULL
		  AND moderation_state = 'published'
		  AND (scheduled_at IS NULL OR scheduled_at <= NOW())
		  AND (mods IS NULL OR mods = '[]'::jsonb OR mods = 'null'::jsonb)
	`).Scan(&count)
	if err != nil {
		return 0, fmt.Errorf("counting vanilla schematics: %w", err)
	}
	return count, nil
}

func (ps *PostgresStore) ListByMod(ctx context.Context, mod string, limit, offset int) ([]store.Schematic, int, error) {
	// Get total count
	var totalCount int
	err := ps.pool.QueryRow(ctx, `
		SELECT COUNT(DISTINCT s.id)::int
		FROM schematics s,
		     LATERAL jsonb_array_elements_text(s.mods) AS j(mod_name)
		WHERE j.mod_name = $1
		  AND s.deleted IS NULL
		  AND s.moderation_state = 'published'
		  AND (s.scheduled_at IS NULL OR s.scheduled_at <= NOW())
	`, mod).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("counting mod schematics: %w", err)
	}

	// Get IDs for this page
	idRows, err := ps.pool.Query(ctx, `
		SELECT DISTINCT s.id
		FROM schematics s,
		     LATERAL jsonb_array_elements_text(s.mods) AS j(mod_name)
		WHERE j.mod_name = $1
		  AND s.deleted IS NULL
		  AND s.moderation_state = 'published'
		  AND (s.scheduled_at IS NULL OR s.scheduled_at <= NOW())
		ORDER BY s.id
		LIMIT $2 OFFSET $3
	`, mod, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("querying mod schematics: %w", err)
	}
	defer idRows.Close()

	var ids []string
	for idRows.Next() {
		var id string
		if err := idRows.Scan(&id); err != nil {
			return nil, 0, err
		}
		ids = append(ids, id)
	}
	if err := idRows.Err(); err != nil {
		return nil, 0, err
	}

	if len(ids) == 0 {
		return nil, totalCount, nil
	}

	schematics, err := ps.ListByIDs(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	return schematics, totalCount, nil
}

func (ps *PostgresStore) ListVanilla(ctx context.Context, limit, offset int) ([]store.Schematic, int, error) {
	// Get total count
	var totalCount int
	err := ps.pool.QueryRow(ctx, `
		SELECT COUNT(*)::int FROM schematics
		WHERE deleted IS NULL
		  AND moderation_state = 'published'
		  AND (scheduled_at IS NULL OR scheduled_at <= NOW())
		  AND (mods IS NULL OR mods = '[]'::jsonb OR mods = 'null'::jsonb)
	`).Scan(&totalCount)
	if err != nil {
		return nil, 0, fmt.Errorf("counting vanilla schematics: %w", err)
	}

	// Get IDs for this page
	idRows, err := ps.pool.Query(ctx, `
		SELECT id FROM schematics
		WHERE deleted IS NULL
		  AND moderation_state = 'published'
		  AND (scheduled_at IS NULL OR scheduled_at <= NOW())
		  AND (mods IS NULL OR mods = '[]'::jsonb OR mods = 'null'::jsonb)
		ORDER BY created DESC
		LIMIT $1 OFFSET $2
	`, limit, offset)
	if err != nil {
		return nil, 0, fmt.Errorf("querying vanilla schematics: %w", err)
	}
	defer idRows.Close()

	var ids []string
	for idRows.Next() {
		var id string
		if err := idRows.Scan(&id); err != nil {
			return nil, 0, err
		}
		ids = append(ids, id)
	}
	if err := idRows.Err(); err != nil {
		return nil, 0, err
	}

	if len(ids) == 0 {
		return nil, totalCount, nil
	}

	schematics, err := ps.ListByIDs(ctx, ids)
	if err != nil {
		return nil, 0, err
	}
	return schematics, totalCount, nil
}

// ============================================================================
// Separate store implementations to avoid method name collisions.
// CategoryStore, TagStore, etc. share method names (List, GetByID, Create)
// that conflict with SchematicStore, so each gets its own wrapper type.
// ============================================================================

// CategoryStoreImpl implements store.CategoryStore.
type CategoryStoreImpl struct{ q *db.Queries }

func (cs *CategoryStoreImpl) List(ctx context.Context) ([]store.Category, error) {
	rows, err := cs.q.ListCategories(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]store.Category, len(rows))
	for i, r := range rows {
		result[i] = store.Category{ID: r.ID, Key: r.Key, Name: r.Name, Public: r.Public}
	}
	return result, nil
}

func (cs *CategoryStoreImpl) GetByID(ctx context.Context, id string) (*store.Category, error) {
	row, err := cs.q.GetCategoryByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &store.Category{ID: row.ID, Key: row.Key, Name: row.Name, Public: row.Public}, nil
}

func (cs *CategoryStoreImpl) GetByIDs(ctx context.Context, ids []string) ([]store.Category, error) {
	rows, err := cs.q.GetCategoriesByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	result := make([]store.Category, len(rows))
	for i, r := range rows {
		result[i] = store.Category{ID: r.ID, Key: r.Key, Name: r.Name, Public: r.Public}
	}
	return result, nil
}

func (cs *CategoryStoreImpl) Create(ctx context.Context, c *store.Category) error {
	if c.ID == "" {
		c.ID = generateID()
	}
	_, err := cs.q.CreateCategory(ctx, db.CreateCategoryParams{
		ID:     c.ID,
		Key:    c.Key,
		Name:   c.Name,
		Public: c.Public,
	})
	return err
}

func (cs *CategoryStoreImpl) ListAll(ctx context.Context) ([]store.Category, error) {
	rows, err := cs.q.ListAllCategories(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]store.Category, len(rows))
	for i, r := range rows {
		result[i] = store.Category{ID: r.ID, Key: r.Key, Name: r.Name, Public: r.Public}
	}
	return result, nil
}

func (cs *CategoryStoreImpl) ListPending(ctx context.Context) ([]store.Category, error) {
	rows, err := cs.q.ListPendingCategories(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]store.Category, len(rows))
	for i, r := range rows {
		result[i] = store.Category{ID: r.ID, Key: r.Key, Name: r.Name, Public: r.Public}
	}
	return result, nil
}

func (cs *CategoryStoreImpl) Approve(ctx context.Context, id string) error {
	return cs.q.ApproveCategoryByID(ctx, id)
}

func (cs *CategoryStoreImpl) Delete(ctx context.Context, id string) error {
	return cs.q.DeleteCategoryByID(ctx, id)
}

func (cs *CategoryStoreImpl) GetByKey(ctx context.Context, key string) (*store.Category, error) {
	row, err := cs.q.GetCategoryByKeyIncludingPending(ctx, key)
	if err != nil {
		return nil, err
	}
	return &store.Category{ID: row.ID, Key: row.Key, Name: row.Name, Public: row.Public}, nil
}

// TagStoreImpl implements store.TagStore.
type TagStoreImpl struct{ q *db.Queries }

func (ts *TagStoreImpl) List(ctx context.Context) ([]store.Tag, error) {
	rows, err := ts.q.ListTags(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]store.Tag, len(rows))
	for i, r := range rows {
		result[i] = store.Tag{ID: r.ID, Key: r.Key, Name: r.Name, Public: r.Public}
	}
	return result, nil
}

func (ts *TagStoreImpl) GetByID(ctx context.Context, id string) (*store.Tag, error) {
	row, err := ts.q.GetTagByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &store.Tag{ID: row.ID, Key: row.Key, Name: row.Name, Public: row.Public}, nil
}

func (ts *TagStoreImpl) GetByIDs(ctx context.Context, ids []string) ([]store.Tag, error) {
	rows, err := ts.q.GetTagsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	result := make([]store.Tag, len(rows))
	for i, r := range rows {
		result[i] = store.Tag{ID: r.ID, Key: r.Key, Name: r.Name, Public: r.Public}
	}
	return result, nil
}

func (ts *TagStoreImpl) ListWithCount(ctx context.Context) ([]store.TagWithCount, error) {
	rows, err := ts.q.ListTagsWithCount(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]store.TagWithCount, len(rows))
	for i, r := range rows {
		result[i] = store.TagWithCount{
			ID:    r.ID,
			Key:   r.Key,
			Name:  r.Name,
			Count: r.Count,
		}
	}
	return result, nil
}

func (ts *TagStoreImpl) Create(ctx context.Context, t *store.Tag) error {
	if t.ID == "" {
		t.ID = generateID()
	}
	_, err := ts.q.CreateTag(ctx, db.CreateTagParams{
		ID:     t.ID,
		Key:    t.Key,
		Name:   t.Name,
		Public: t.Public,
	})
	return err
}

func (ts *TagStoreImpl) ListAll(ctx context.Context) ([]store.Tag, error) {
	rows, err := ts.q.ListAllTags(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]store.Tag, len(rows))
	for i, r := range rows {
		result[i] = store.Tag{ID: r.ID, Key: r.Key, Name: r.Name, Public: r.Public}
	}
	return result, nil
}

func (ts *TagStoreImpl) ListPending(ctx context.Context) ([]store.Tag, error) {
	rows, err := ts.q.ListPendingTags(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]store.Tag, len(rows))
	for i, r := range rows {
		result[i] = store.Tag{ID: r.ID, Key: r.Key, Name: r.Name, Public: r.Public}
	}
	return result, nil
}

func (ts *TagStoreImpl) Approve(ctx context.Context, id string) error {
	return ts.q.ApproveTagByID(ctx, id)
}

func (ts *TagStoreImpl) Delete(ctx context.Context, id string) error {
	return ts.q.DeleteTagByID(ctx, id)
}

func (ts *TagStoreImpl) GetByKey(ctx context.Context, key string) (*store.Tag, error) {
	row, err := ts.q.GetTagByKeyIncludingPending(ctx, key)
	if err != nil {
		return nil, err
	}
	return &store.Tag{ID: row.ID, Key: row.Key, Name: row.Name, Public: row.Public}, nil
}

// CommentStoreImpl implements store.CommentStore.
type CommentStoreImpl struct{ q *db.Queries }

func (cs *CommentStoreImpl) GetByID(ctx context.Context, id string) (*store.Comment, error) {
	row, err := cs.q.GetCommentByID(ctx, id)
	if err != nil {
		return nil, err
	}
	c := commentFromDB(row, "", "")
	return &c, nil
}

func (cs *CommentStoreImpl) ListBySchematic(ctx context.Context, schematicID string) ([]store.Comment, error) {
	rows, err := cs.q.ListCommentsBySchematic(ctx, &schematicID)
	if err != nil {
		return nil, err
	}
	result := make([]store.Comment, len(rows))
	for i, r := range rows {
		result[i] = store.Comment{
			ID:             r.ID,
			AuthorID:       r.AuthorID,
			SchematicID:    r.SchematicID,
			ParentID:       r.ParentID,
			Content:        r.Content,
			Published:      fromPgTimestamptz(r.Published),
			Approved:       r.Approved,
			Type:           r.Type,
			Karma:          int(r.Karma),
			AuthorUsername: derefStr(r.AuthorUsername),
			AuthorAvatar:   derefStr(r.AuthorAvatar),
			Created:        r.Created,
			Updated:        r.Updated,
		}
	}
	return result, nil
}

func (cs *CommentStoreImpl) CountBySchematic(ctx context.Context, schematicID string) (int64, error) {
	return cs.q.CountCommentsBySchematic(ctx, &schematicID)
}

func (cs *CommentStoreImpl) Create(ctx context.Context, c *store.Comment) error {
	if c.ID == "" {
		c.ID = generateID()
	}
	_, err := cs.q.CreateComment(ctx, db.CreateCommentParams{
		ID:          c.ID,
		AuthorID:    c.AuthorID,
		SchematicID: c.SchematicID,
		ParentID:    c.ParentID,
		Content:     c.Content,
		Published:   toPgTimestamptz(c.Published),
		Approved:    c.Approved,
		Type:        c.Type,
	})
	return err
}

func (cs *CommentStoreImpl) Approve(ctx context.Context, id string) error {
	return cs.q.ApproveComment(ctx, id)
}

func (cs *CommentStoreImpl) Delete(ctx context.Context, id string) error {
	return cs.q.DeleteComment(ctx, id)
}

func (cs *CommentStoreImpl) CountByUser(ctx context.Context, userID string) (int64, error) {
	return cs.q.CountUserComments(ctx, &userID)
}

// GuideStoreImpl implements store.GuideStore.
type GuideStoreImpl struct{ q *db.Queries }

func (gs *GuideStoreImpl) GetByID(ctx context.Context, id string) (*store.Guide, error) {
	row, err := gs.q.GetGuideByID(ctx, id)
	if err != nil {
		return nil, err
	}
	g := guideFromDB(row)
	return &g, nil
}

func (gs *GuideStoreImpl) GetBySlug(ctx context.Context, slug string) (*store.Guide, error) {
	row, err := gs.q.GetGuideBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	g := guideFromDB(row)
	return &g, nil
}

func (gs *GuideStoreImpl) List(ctx context.Context, limit, offset int) ([]store.Guide, error) {
	rows, err := gs.q.ListGuides(ctx, db.ListGuidesParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}
	result := make([]store.Guide, len(rows))
	for i, r := range rows {
		result[i] = guideFromDB(r)
	}
	return result, nil
}

func (gs *GuideStoreImpl) Create(ctx context.Context, g *store.Guide) error {
	if g.ID == "" {
		g.ID = generateID()
	}
	_, err := gs.q.CreateGuide(ctx, db.CreateGuideParams{
		ID:          g.ID,
		AuthorID:    g.AuthorID,
		Title:       g.Title,
		Description: g.Description,
		Content:     g.Content,
		Slug:        g.Slug,
		UploadLink:  g.UploadLink,
		BannerUrl:   g.BannerURL,
	})
	return err
}

func (gs *GuideStoreImpl) Update(ctx context.Context, g *store.Guide) error {
	_, err := gs.q.UpdateGuide(ctx, db.UpdateGuideParams{
		ID:          g.ID,
		Title:       ptrStr(g.Title),
		Description: ptrStr(g.Description),
		Content:     ptrStr(g.Content),
		UploadLink:  ptrStr(g.UploadLink),
		BannerUrl:   ptrStr(g.BannerURL),
	})
	return err
}

func (gs *GuideStoreImpl) Delete(ctx context.Context, id string) error {
	return gs.q.DeleteGuide(ctx, id)
}

func (gs *GuideStoreImpl) CountByUser(ctx context.Context, userID string) (int64, error) {
	return gs.q.CountUserGuides(ctx, &userID)
}

func (gs *GuideStoreImpl) IncrementViews(ctx context.Context, id string) error {
	return gs.q.IncrementGuideViews(ctx, id)
}

func (gs *GuideStoreImpl) ListForSitemap(ctx context.Context) ([]store.SitemapGuide, error) {
	rows, err := gs.q.ListGuidesForSitemap(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]store.SitemapGuide, len(rows))
	for i, r := range rows {
		result[i] = store.SitemapGuide{
			ID:      r.ID,
			Slug:    r.Slug,
			Updated: r.Updated,
		}
	}
	return result, nil
}

// CollectionStoreImpl implements store.CollectionStore.
type CollectionStoreImpl struct{ q *db.Queries }

func (cs *CollectionStoreImpl) GetByID(ctx context.Context, id string) (*store.Collection, error) {
	row, err := cs.q.GetCollectionByID(ctx, id)
	if err != nil {
		return nil, err
	}
	c := collectionFromDB(row)
	return &c, nil
}

func (cs *CollectionStoreImpl) GetBySlug(ctx context.Context, slug string) (*store.Collection, error) {
	row, err := cs.q.GetCollectionBySlug(ctx, slug)
	if err != nil {
		return nil, err
	}
	c := collectionFromDB(row)
	return &c, nil
}

func (cs *CollectionStoreImpl) List(ctx context.Context, limit, offset int) ([]store.Collection, error) {
	rows, err := cs.q.ListCollections(ctx, db.ListCollectionsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}
	result := make([]store.Collection, len(rows))
	for i, r := range rows {
		result[i] = collectionFromDB(r)
	}
	return result, nil
}

func (cs *CollectionStoreImpl) ListByAuthor(ctx context.Context, authorID string) ([]store.Collection, error) {
	rows, err := cs.q.ListCollectionsByAuthor(ctx, &authorID)
	if err != nil {
		return nil, err
	}
	result := make([]store.Collection, len(rows))
	for i, r := range rows {
		result[i] = collectionFromDB(r)
	}
	return result, nil
}

func (cs *CollectionStoreImpl) ListFeatured(ctx context.Context, limit int) ([]store.Collection, error) {
	rows, err := cs.q.ListFeaturedCollections(ctx, int32(limit))
	if err != nil {
		return nil, err
	}
	result := make([]store.Collection, len(rows))
	for i, r := range rows {
		result[i] = collectionFromDB(r)
	}
	return result, nil
}

func (cs *CollectionStoreImpl) ListPublished(ctx context.Context, limit, offset int) ([]store.Collection, error) {
	rows, err := cs.q.ListPublishedCollections(ctx, db.ListPublishedCollectionsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}
	result := make([]store.Collection, len(rows))
	for i, r := range rows {
		result[i] = collectionFromDB(r)
	}
	return result, nil
}

func (cs *CollectionStoreImpl) Create(ctx context.Context, c *store.Collection) error {
	if c.ID == "" {
		c.ID = generateID()
	}
	_, err := cs.q.CreateCollection(ctx, db.CreateCollectionParams{
		ID:          c.ID,
		AuthorID:    c.AuthorID,
		Title:       c.Title,
		Name:        c.Name,
		Slug:        c.Slug,
		Description: c.Description,
		BannerUrl:   c.BannerURL,
		Published:   c.Published,
	})
	return err
}

func (cs *CollectionStoreImpl) Update(ctx context.Context, c *store.Collection) error {
	_, err := cs.q.UpdateCollection(ctx, db.UpdateCollectionParams{
		ID:          c.ID,
		Title:       ptrStr(c.Title),
		Description: ptrStr(c.Description),
		BannerUrl:   ptrStr(c.BannerURL),
		CollageUrl:  ptrStr(c.CollageURL),
		Featured:    ptrBool(c.Featured),
		Published:   ptrBool(c.Published),
	})
	return err
}

func (cs *CollectionStoreImpl) UpdateCollageURL(ctx context.Context, id, collageURL string) error {
	return cs.q.UpdateCollectionCollageURL(ctx, db.UpdateCollectionCollageURLParams{
		ID:         id,
		CollageUrl: collageURL,
	})
}

func (cs *CollectionStoreImpl) SoftDelete(ctx context.Context, id string) error {
	return cs.q.SoftDeleteCollection(ctx, id)
}

func (cs *CollectionStoreImpl) GetSchematicIDs(ctx context.Context, collectionID string) ([]string, error) {
	return cs.q.GetCollectionSchematicIDs(ctx, collectionID)
}

func (cs *CollectionStoreImpl) AddSchematic(ctx context.Context, collectionID, schematicID string, position int) error {
	return cs.q.AddSchematicToCollection(ctx, db.AddSchematicToCollectionParams{
		CollectionID: collectionID,
		SchematicID:  schematicID,
		Position:     int32(position),
	})
}

func (cs *CollectionStoreImpl) RemoveSchematic(ctx context.Context, collectionID, schematicID string) error {
	return cs.q.RemoveSchematicFromCollection(ctx, db.RemoveSchematicFromCollectionParams{
		CollectionID: collectionID,
		SchematicID:  schematicID,
	})
}

func (cs *CollectionStoreImpl) ClearSchematics(ctx context.Context, collectionID string) error {
	return cs.q.ClearCollectionSchematics(ctx, collectionID)
}

func (cs *CollectionStoreImpl) IncrementViews(ctx context.Context, id string) error {
	return cs.q.IncrementCollectionViews(ctx, id)
}

func (cs *CollectionStoreImpl) CountByUser(ctx context.Context, userID string) (int64, error) {
	return cs.q.CountUserCollections(ctx, &userID)
}

func (cs *CollectionStoreImpl) ListForSitemap(ctx context.Context) ([]store.SitemapCollection, error) {
	rows, err := cs.q.ListCollectionsForSitemap(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]store.SitemapCollection, len(rows))
	for i, r := range rows {
		result[i] = store.SitemapCollection{
			ID:      r.ID,
			Slug:    r.Slug,
			Updated: r.Updated,
		}
	}
	return result, nil
}

// AchievementStoreImpl implements store.AchievementStore.
type AchievementStoreImpl struct{ q *db.Queries }

func (as *AchievementStoreImpl) GetByKey(ctx context.Context, key string) (*store.Achievement, error) {
	row, err := as.q.GetAchievementByKey(ctx, key)
	if err != nil {
		return nil, err
	}
	a := achievementFromDB(row)
	return &a, nil
}

func (as *AchievementStoreImpl) List(ctx context.Context) ([]store.Achievement, error) {
	rows, err := as.q.ListAchievements(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]store.Achievement, len(rows))
	for i, r := range rows {
		result[i] = achievementFromDB(r)
	}
	return result, nil
}

func (as *AchievementStoreImpl) ListUserAchievements(ctx context.Context, userID string) ([]store.Achievement, error) {
	rows, err := as.q.ListUserAchievements(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]store.Achievement, len(rows))
	for i, r := range rows {
		result[i] = achievementFromDB(r)
	}
	return result, nil
}

func (as *AchievementStoreImpl) Award(ctx context.Context, userID, achievementID string) error {
	_, err := as.q.AwardAchievement(ctx, db.AwardAchievementParams{
		ID:            generateID(),
		UserID:        userID,
		AchievementID: achievementID,
	})
	// ON CONFLICT DO NOTHING means no rows returned if already awarded;
	// pgx returns "no rows in result set" which we treat as success.
	if err != nil && err.Error() == "no rows in result set" {
		return nil
	}
	return err
}

func (as *AchievementStoreImpl) HasAchievement(ctx context.Context, userID, achievementID string) (bool, error) {
	return as.q.HasAchievement(ctx, db.HasAchievementParams{
		UserID:        userID,
		AchievementID: achievementID,
	})
}

func (as *AchievementStoreImpl) CreatePointLog(ctx context.Context, entry *store.PointLogEntry) error {
	if entry.ID == "" {
		entry.ID = generateID()
	}
	_, err := as.q.CreatePointLog(ctx, db.CreatePointLogParams{
		ID:          entry.ID,
		UserID:      entry.UserID,
		Points:      int32(entry.Points),
		Reason:      entry.Reason,
		Description: entry.Description,
		EarnedAt:    entry.EarnedAt,
	})
	// ON CONFLICT DO NOTHING
	if err != nil && err.Error() == "no rows in result set" {
		return nil
	}
	return err
}

func (as *AchievementStoreImpl) GetPointLog(ctx context.Context, userID string) ([]store.PointLogEntry, error) {
	rows, err := as.q.GetPointLog(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]store.PointLogEntry, len(rows))
	for i, r := range rows {
		result[i] = store.PointLogEntry{
			ID:          r.ID,
			UserID:      r.UserID,
			Points:      int(r.Points),
			Reason:      r.Reason,
			Description: r.Description,
			EarnedAt:    r.EarnedAt,
		}
	}
	return result, nil
}

func (as *AchievementStoreImpl) SumUserPoints(ctx context.Context, userID string) (int, error) {
	total, err := as.q.SumUserPoints(ctx, userID)
	return int(total), err
}

// TranslationStoreImpl implements store.TranslationStore.
type TranslationStoreImpl struct{ q *db.Queries }

func (ts *TranslationStoreImpl) GetSchematicTranslation(ctx context.Context, schematicID, lang string) (*store.Translation, error) {
	row, err := ts.q.GetSchematicTranslation(ctx, db.GetSchematicTranslationParams{
		SchematicID: schematicID,
		Language:    lang,
	})
	if err != nil {
		return nil, err
	}
	return &store.Translation{
		ID:          row.ID,
		Language:    row.Language,
		Title:       row.Title,
		Description: row.Description,
		Content:     row.Content,
	}, nil
}

func (ts *TranslationStoreImpl) ListSchematicTranslations(ctx context.Context, schematicID string) ([]store.Translation, error) {
	rows, err := ts.q.ListSchematicTranslations(ctx, schematicID)
	if err != nil {
		return nil, err
	}
	result := make([]store.Translation, len(rows))
	for i, r := range rows {
		result[i] = store.Translation{
			ID:          r.ID,
			Language:    r.Language,
			Title:       r.Title,
			Description: r.Description,
			Content:     r.Content,
		}
	}
	return result, nil
}

func (ts *TranslationStoreImpl) UpsertSchematicTranslation(ctx context.Context, schematicID string, t *store.Translation) error {
	if t.ID == "" {
		t.ID = generateID()
	}
	_, err := ts.q.UpsertSchematicTranslation(ctx, db.UpsertSchematicTranslationParams{
		ID:          t.ID,
		SchematicID: schematicID,
		Language:    t.Language,
		Title:       t.Title,
		Description: t.Description,
		Content:     t.Content,
	})
	return err
}

func (ts *TranslationStoreImpl) ListSchematicsWithoutTranslation(ctx context.Context, lang string, limit int) ([]store.Schematic, error) {
	rows, err := ts.q.ListSchematicsWithoutTranslation(ctx, db.ListSchematicsWithoutTranslationParams{
		Language: lang,
		Limit:    int32(limit),
	})
	if err != nil {
		return nil, err
	}
	result := make([]store.Schematic, len(rows))
	for i, r := range rows {
		result[i] = store.Schematic{
			ID:          r.ID,
			Title:       r.Title,
			Description: r.Description,
			Content:     r.Content,
		}
	}
	return result, nil
}

func (ts *TranslationStoreImpl) GetGuideTranslation(ctx context.Context, guideID, lang string) (*store.Translation, error) {
	row, err := ts.q.GetGuideTranslation(ctx, db.GetGuideTranslationParams{
		GuideID:  guideID,
		Language: lang,
	})
	if err != nil {
		return nil, err
	}
	return &store.Translation{
		ID:          row.ID,
		Language:    row.Language,
		Title:       row.Title,
		Description: row.Description,
		Content:     row.Content,
	}, nil
}

func (ts *TranslationStoreImpl) UpsertGuideTranslation(ctx context.Context, guideID string, t *store.Translation) error {
	if t.ID == "" {
		t.ID = generateID()
	}
	_, err := ts.q.UpsertGuideTranslation(ctx, db.UpsertGuideTranslationParams{
		ID:          t.ID,
		GuideID:     guideID,
		Language:    t.Language,
		Title:       t.Title,
		Description: t.Description,
		Content:     t.Content,
	})
	return err
}

func (ts *TranslationStoreImpl) GetCollectionTranslation(ctx context.Context, collectionID, lang string) (*store.Translation, error) {
	row, err := ts.q.GetCollectionTranslation(ctx, db.GetCollectionTranslationParams{
		CollectionID: collectionID,
		Language:     lang,
	})
	if err != nil {
		return nil, err
	}
	return &store.Translation{
		ID:          row.ID,
		Language:    row.Language,
		Title:       row.Title,
		Description: row.Description,
	}, nil
}

func (ts *TranslationStoreImpl) UpsertCollectionTranslation(ctx context.Context, collectionID string, t *store.Translation) error {
	if t.ID == "" {
		t.ID = generateID()
	}
	_, err := ts.q.UpsertCollectionTranslation(ctx, db.UpsertCollectionTranslationParams{
		ID:           t.ID,
		CollectionID: collectionID,
		Language:     t.Language,
		Title:        t.Title,
		Description:  t.Description,
	})
	return err
}

// ViewRatingStoreImpl implements store.ViewRatingStore.
type ViewRatingStoreImpl struct{ q *db.Queries }

func (vs *ViewRatingStoreImpl) GetViewCount(ctx context.Context, schematicID string) (int, error) {
	count, err := vs.q.GetSchematicViewCount(ctx, schematicID)
	return int(count), err
}

func (vs *ViewRatingStoreImpl) GetDownloadCount(ctx context.Context, schematicID string) (int, error) {
	count, err := vs.q.GetSchematicDownloadCount(ctx, schematicID)
	return int(count), err
}

func (vs *ViewRatingStoreImpl) RecordDownload(ctx context.Context, schematicID string, userID *string) error {
	return vs.q.RecordSchematicDownload(ctx, db.RecordSchematicDownloadParams{
		ID:          generateID(),
		SchematicID: schematicID,
		UserID:      userID,
	})
}

func (vs *ViewRatingStoreImpl) GetRating(ctx context.Context, schematicID string) (*store.SchematicRating, error) {
	row, err := vs.q.GetSchematicRating(ctx, schematicID)
	if err != nil {
		return nil, err
	}
	return &store.SchematicRating{
		AvgRating:   float64(row.AvgRating),
		RatingCount: int(row.RatingCount),
	}, nil
}

func (vs *ViewRatingStoreImpl) UpsertRating(ctx context.Context, userID, schematicID string, rating float64) error {
	return vs.q.UpsertSchematicRating(ctx, db.UpsertSchematicRatingParams{
		ID:          generateID(),
		UserID:      userID,
		SchematicID: schematicID,
		Rating:      float32(rating),
	})
}

func (vs *ViewRatingStoreImpl) RecordView(ctx context.Context, schematicID string) error {
	now := time.Now().UTC()
	year, week := now.ISOWeek()

	periods := []struct {
		typ    string
		period string
	}{
		{"0", now.Format("20060102")},                    // daily: YYYYMMDD
		{"1", fmt.Sprintf("%d%02d", year, week)},         // weekly: YYYYWW
		{"2", now.Format("200601")},                      // monthly: YYYYMM
		{"3", now.Format("2006")},                        // yearly: YYYY
		{"4", "total"},                                    // all-time
	}

	for _, p := range periods {
		if err := vs.q.UpsertSchematicViewCount(ctx, db.UpsertSchematicViewCountParams{
			ID:          generateID(),
			SchematicID: schematicID,
			Type:        p.typ,
			Period:      p.period,
		}); err != nil {
			return err
		}
	}
	return nil
}

func (vs *ViewRatingStoreImpl) GetTotalViewCount(ctx context.Context, schematicID string) (int, error) {
	count, err := vs.q.GetTotalViewCount(ctx, schematicID)
	return int(count), err
}

func (vs *ViewRatingStoreImpl) FetchTrendingData(ctx context.Context, recentDays int) (*store.TrendingData, error) {
	cutoff := time.Now().UTC().Add(-time.Duration(recentDays) * 24 * time.Hour)

	// Fetch only IDs and created timestamps (lightweight query for trending calc)
	schematicRows, err := vs.q.ListApprovedSchematicIDsAndCreated(ctx)
	if err != nil {
		return nil, fmt.Errorf("list schematics for trending: %w", err)
	}

	ids := make([]string, len(schematicRows))
	createdMap := make(map[string]time.Time, len(schematicRows))
	for i, s := range schematicRows {
		ids[i] = s.ID
		createdMap[s.ID] = s.Created
	}

	// Fetch recent views
	recentViewRows, err := vs.q.FetchRecentViewsBySchematic(ctx, cutoff)
	if err != nil {
		return nil, fmt.Errorf("fetch recent views: %w", err)
	}
	recentViews := make(map[string]float64, len(recentViewRows))
	for _, r := range recentViewRows {
		recentViews[r.ID] = float64(r.V)
	}

	// Fetch total views
	totalViewRows, err := vs.q.FetchTotalViewsBySchematic(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch total views: %w", err)
	}
	totalViews := make(map[string]float64, len(totalViewRows))
	for _, r := range totalViewRows {
		totalViews[r.ID] = float64(r.V)
	}

	// Fetch rating sums
	ratingSumRows, err := vs.q.FetchRatingSumBySchematic(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch rating sums: %w", err)
	}
	ratingSum := make(map[string]float64, len(ratingSumRows))
	for _, r := range ratingSumRows {
		ratingSum[r.ID] = float64(r.V)
	}

	// Fetch rating counts
	ratingCountRows, err := vs.q.FetchRatingCountBySchematic(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch rating counts: %w", err)
	}
	ratingCount := make(map[string]float64, len(ratingCountRows))
	for _, r := range ratingCountRows {
		ratingCount[r.ID] = float64(r.V)
	}

	// Fetch recent downloads
	recentDownloadRows, err := vs.q.FetchRecentDownloadsBySchematic(ctx, cutoff)
	if err != nil {
		return nil, fmt.Errorf("fetch recent downloads: %w", err)
	}
	recentDownloads := make(map[string]float64, len(recentDownloadRows))
	for _, r := range recentDownloadRows {
		recentDownloads[r.ID] = float64(r.V)
	}

	// Fetch total downloads
	totalDownloadRows, err := vs.q.FetchTotalDownloadsBySchematic(ctx)
	if err != nil {
		return nil, fmt.Errorf("fetch total downloads: %w", err)
	}
	totalDownloads := make(map[string]float64, len(totalDownloadRows))
	for _, r := range totalDownloadRows {
		totalDownloads[r.ID] = float64(r.V)
	}

	return &store.TrendingData{
		SchematicIDs:     ids,
		SchematicCreated: createdMap,
		RecentViews:      recentViews,
		TotalViews:       totalViews,
		RatingSum:        ratingSum,
		RatingCount:      ratingCount,
		RecentDownloads:  recentDownloads,
		TotalDownloads:   totalDownloads,
	}, nil
}

func (vs *ViewRatingStoreImpl) BatchGetViewCounts(ctx context.Context, ids []string) (map[string]int, error) {
	rows, err := vs.q.BatchGetViewCounts(ctx, ids)
	if err != nil {
		return nil, err
	}
	result := make(map[string]int, len(rows))
	for _, r := range rows {
		result[r.SchematicID] = int(r.ViewCount)
	}
	return result, nil
}

func (vs *ViewRatingStoreImpl) BatchGetDownloadCounts(ctx context.Context, ids []string) (map[string]int, error) {
	rows, err := vs.q.BatchGetDownloadCounts(ctx, ids)
	if err != nil {
		return nil, err
	}
	result := make(map[string]int, len(rows))
	for _, r := range rows {
		result[r.SchematicID] = int(r.DownloadCount)
	}
	return result, nil
}

func (vs *ViewRatingStoreImpl) BatchGetRatings(ctx context.Context, ids []string) (map[string]*store.SchematicRating, error) {
	rows, err := vs.q.BatchGetRatings(ctx, ids)
	if err != nil {
		return nil, err
	}
	result := make(map[string]*store.SchematicRating, len(rows))
	for _, r := range rows {
		result[r.SchematicID] = &store.SchematicRating{
			AvgRating:   float64(r.AvgRating),
			RatingCount: int(r.RatingCount),
		}
	}
	return result, nil
}

// VersionStoreImpl implements store.VersionStore.
type VersionStoreImpl struct{ q *db.Queries }

func (vs *VersionStoreImpl) Create(ctx context.Context, v *store.SchematicVersion) error {
	if v.ID == "" {
		v.ID = generateID()
	}
	_, err := vs.q.CreateSchematicVersion(ctx, db.CreateSchematicVersionParams{
		ID:          v.ID,
		SchematicID: v.SchematicID,
		Version:     int32(v.Version),
		Snapshot:    v.Snapshot,
		Note:        v.Note,
	})
	return err
}

func (vs *VersionStoreImpl) ListBySchematic(ctx context.Context, schematicID string) ([]store.SchematicVersion, error) {
	rows, err := vs.q.ListSchematicVersions(ctx, schematicID)
	if err != nil {
		return nil, err
	}
	result := make([]store.SchematicVersion, len(rows))
	for i, r := range rows {
		result[i] = store.SchematicVersion{
			ID:          r.ID,
			SchematicID: r.SchematicID,
			Version:     int(r.Version),
			Snapshot:    r.Snapshot,
			Note:        r.Note,
			Created:     r.Created,
		}
	}
	return result, nil
}

func (vs *VersionStoreImpl) GetLatestVersion(ctx context.Context, schematicID string) (int, error) {
	v, err := vs.q.GetLatestSchematicVersion(ctx, schematicID)
	return int(v), err
}

// APIKeyStoreImpl implements store.APIKeyStore.
type APIKeyStoreImpl struct{ q *db.Queries }

func (as *APIKeyStoreImpl) GetByLast8(ctx context.Context, last8 string) (*store.APIKey, error) {
	row, err := as.q.GetAPIKeyByLast8(ctx, last8)
	if err != nil {
		return nil, err
	}
	k := apiKeyFromDB(row)
	return &k, nil
}

func (as *APIKeyStoreImpl) ListByUser(ctx context.Context, userID string) ([]store.APIKey, error) {
	rows, err := as.q.ListAPIKeysByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]store.APIKey, len(rows))
	for i, r := range rows {
		result[i] = apiKeyFromDB(r)
	}
	return result, nil
}

func (as *APIKeyStoreImpl) Create(ctx context.Context, k *store.APIKey) error {
	if k.ID == "" {
		k.ID = generateID()
	}
	_, err := as.q.CreateAPIKey(ctx, db.CreateAPIKeyParams{
		ID:      k.ID,
		UserID:  k.UserID,
		KeyHash: k.KeyHash,
		Label:   k.Label,
		Last8:   k.Last8,
	})
	return err
}

func (as *APIKeyStoreImpl) Delete(ctx context.Context, id, userID string) error {
	return as.q.DeleteAPIKey(ctx, db.DeleteAPIKeyParams{
		ID:     id,
		UserID: userID,
	})
}

func (as *APIKeyStoreImpl) LogUsage(ctx context.Context, apiKeyID, endpoint string) error {
	return as.q.LogAPIKeyUsage(ctx, db.LogAPIKeyUsageParams{
		ID:       generateID(),
		ApiKeyID: apiKeyID,
		Endpoint: endpoint,
	})
}

// AuthStoreImpl implements store.AuthStore.
type AuthStoreImpl struct{ q *db.Queries }

func (as *AuthStoreImpl) GetByProvider(ctx context.Context, provider, providerID string) (*store.ExternalAuth, error) {
	row, err := as.q.GetExternalAuth(ctx, db.GetExternalAuthParams{
		Provider:   provider,
		ProviderID: providerID,
	})
	if err != nil {
		return nil, err
	}
	ea := externalAuthFromDB(row)
	return &ea, nil
}

func (as *AuthStoreImpl) Create(ctx context.Context, ea *store.ExternalAuth) error {
	if ea.ID == "" {
		ea.ID = generateID()
	}
	_, err := as.q.CreateExternalAuth(ctx, db.CreateExternalAuthParams{
		ID:         ea.ID,
		UserID:     ea.UserID,
		Provider:   ea.Provider,
		ProviderID: ea.ProviderID,
	})
	return err
}

func (as *AuthStoreImpl) ListByUser(ctx context.Context, userID string) ([]store.ExternalAuth, error) {
	rows, err := as.q.ListExternalAuthsByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	result := make([]store.ExternalAuth, len(rows))
	for i, r := range rows {
		result[i] = externalAuthFromDB(r)
	}
	return result, nil
}

// ReportStoreImpl implements store.ReportStore.
type ReportStoreImpl struct{ q *db.Queries }

func (rs *ReportStoreImpl) Create(ctx context.Context, r *store.Report) error {
	if r.ID == "" {
		r.ID = generateID()
	}
	_, err := rs.q.CreateReport(ctx, db.CreateReportParams{
		ID:         r.ID,
		TargetType: r.TargetType,
		TargetID:   r.TargetID,
		Reason:     r.Reason,
		Reporter:   r.Reporter,
	})
	return err
}

func (rs *ReportStoreImpl) List(ctx context.Context, limit, offset int) ([]store.Report, error) {
	rows, err := rs.q.ListReports(ctx, db.ListReportsParams{
		Limit:  int32(limit),
		Offset: int32(offset),
	})
	if err != nil {
		return nil, err
	}
	result := make([]store.Report, len(rows))
	for i, r := range rows {
		result[i] = reportFromDB(r)
	}
	return result, nil
}

func (rs *ReportStoreImpl) Delete(ctx context.Context, id string) error {
	return rs.q.DeleteReport(ctx, id)
}

func (rs *ReportStoreImpl) DeleteByTarget(ctx context.Context, targetType, targetID string) (int64, error) {
	return rs.q.DeleteReportsByTarget(ctx, db.DeleteReportsByTargetParams{
		TargetType: targetType,
		TargetID:   targetID,
	})
}

// ModMetadataStoreImpl implements store.ModMetadataStore.
type ModMetadataStoreImpl struct{ q *db.Queries }

func (ms *ModMetadataStoreImpl) GetByNamespace(ctx context.Context, namespace string) (*store.ModMetadata, error) {
	row, err := ms.q.GetModMetadataByNamespace(ctx, namespace)
	if err != nil {
		return nil, err
	}
	m := modMetadataFromDB(row)
	return &m, nil
}

func (ms *ModMetadataStoreImpl) Upsert(ctx context.Context, m *store.ModMetadata) error {
	if m.ID == "" {
		m.ID = generateID()
	}
	_, err := ms.q.UpsertModMetadata(ctx, db.UpsertModMetadataParams{
		ID:                 m.ID,
		Namespace:          m.Namespace,
		DisplayName:        m.DisplayName,
		Description:        m.Description,
		IconUrl:            m.IconURL,
		ModrinthSlug:       m.ModrinthSlug,
		ModrinthUrl:        m.ModrinthURL,
		CurseforgeID:       m.CurseforgeID,
		CurseforgeUrl:      m.CurseforgeURL,
		SourceUrl:          m.SourceURL,
		LastFetched:        toPgTimestamptz(m.LastFetched),
		ManuallySet:        m.ManuallySet,
		BlocksitemsMatched: m.BlocksitemsMatched,
	})
	return err
}

func (ms *ModMetadataStoreImpl) ListAll(ctx context.Context) ([]store.ModMetadata, error) {
	rows, err := ms.q.ListModMetadataAll(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]store.ModMetadata, len(rows))
	for i, r := range rows {
		result[i] = modMetadataFromDB(r)
	}
	return result, nil
}

func (ms *ModMetadataStoreImpl) ListStale(ctx context.Context, limit int) ([]store.ModMetadata, error) {
	rows, err := ms.q.ListModMetadataStale(ctx, int32(limit))
	if err != nil {
		return nil, err
	}
	result := make([]store.ModMetadata, len(rows))
	for i, r := range rows {
		result[i] = modMetadataFromDB(r)
	}
	return result, nil
}

// VersionLookupStoreImpl implements store.VersionLookupStore.
type VersionLookupStoreImpl struct{ q *db.Queries }

func (vl *VersionLookupStoreImpl) ListMinecraftVersions(ctx context.Context) ([]store.GameVersion, error) {
	rows, err := vl.q.ListMinecraftVersions(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]store.GameVersion, len(rows))
	for i, r := range rows {
		result[i] = store.GameVersion{
			ID:      r.ID,
			Version: r.Version,
			Created: r.Created,
		}
	}
	return result, nil
}

func (vl *VersionLookupStoreImpl) ListCreatemodVersions(ctx context.Context) ([]store.GameVersion, error) {
	rows, err := vl.q.ListVersions(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]store.GameVersion, len(rows))
	for i, r := range rows {
		result[i] = store.GameVersion{
			ID:      r.ID,
			Version: r.Version,
			Created: r.Created,
		}
	}
	return result, nil
}

func (vl *VersionLookupStoreImpl) GetMinecraftVersionByID(ctx context.Context, id string) (*store.GameVersion, error) {
	row, err := vl.q.GetMinecraftVersionByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &store.GameVersion{
		ID:      row.ID,
		Version: row.Version,
		Created: row.Created,
	}, nil
}

func (vl *VersionLookupStoreImpl) GetCreatemodVersionByID(ctx context.Context, id string) (*store.GameVersion, error) {
	row, err := vl.q.GetCreatemodVersionByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &store.GameVersion{
		ID:      row.ID,
		Version: row.Version,
		Created: row.Created,
	}, nil
}

// SearchTrackingStoreImpl implements store.SearchTrackingStore.
type SearchTrackingStoreImpl struct{ q *db.Queries }

// sanitizeSearchQuery strips control characters and truncates to 200 chars.
func sanitizeSearchQuery(q string) string {
	clean := make([]rune, 0, len(q))
	for _, r := range q {
		if r < 32 || r == 127 {
			continue
		}
		clean = append(clean, r)
	}
	if len(clean) > 200 {
		clean = clean[:200]
	}
	return string(clean)
}

func (st *SearchTrackingStoreImpl) RecordSearch(ctx context.Context, query string, resultsCount int, userID, ip string) error {
	query = sanitizeSearchQuery(query)
	if query == "" {
		return nil
	}
	var uid *string
	if userID != "" {
		uid = &userID
	}
	return st.q.CreateSearch(ctx, db.CreateSearchParams{
		ID:           generateID(),
		Query:        query,
		ResultsCount: int32(resultsCount),
		UserID:       uid,
		IpAddress:    ip,
	})
}

func (st *SearchTrackingStoreImpl) ListTopSearches(ctx context.Context, limit int) ([]store.SearchEntry, error) {
	rows, err := st.q.ListTopSearches(ctx, int32(limit))
	if err != nil {
		return nil, err
	}
	result := make([]store.SearchEntry, len(rows))
	for i, r := range rows {
		result[i] = store.SearchEntry{
			Query:        r.Query,
			ResultsCount: int(r.SearchCount),
		}
	}
	return result, nil
}

func (st *SearchTrackingStoreImpl) RefreshSearchQueryCounts(ctx context.Context) error {
	return st.q.RefreshSearchQueryCounts(ctx)
}

func (st *SearchTrackingStoreImpl) PruneOldSearches(ctx context.Context) (int64, error) {
	return st.q.PruneOldSearches(ctx)
}

// OutgoingClickStoreImpl implements store.OutgoingClickStore.
type OutgoingClickStoreImpl struct{ q *db.Queries }

func (oc *OutgoingClickStoreImpl) RecordClick(ctx context.Context, url, source, sourceID string, userID *string) error {
	return oc.q.RecordOutgoingClick(ctx, db.RecordOutgoingClickParams{
		ID:       generateID(),
		Url:      url,
		Source:   source,
		SourceID: sourceID,
		UserID:   userID,
	})
}

// ContactStoreImpl implements store.ContactStore.
type ContactStoreImpl struct{ q *db.Queries }

func (cs *ContactStoreImpl) CreateSubmission(ctx context.Context, authorID *string, title, content, name string) error {
	now := time.Now().UTC()
	_, err := cs.q.CreateContactFormSubmission(ctx, db.CreateContactFormSubmissionParams{
		ID:       generateID(),
		AuthorID: authorID,
		Title:    title,
		Content:  content,
		Name:     name,
		Postdate: pgtype.Timestamptz{Time: now, Valid: true},
		Status:   "new",
		Type:     "contact",
	})
	return err
}

// StatsStoreImpl implements store.StatsStore.
type StatsStoreImpl struct{ q *db.Queries }

func (ss *StatsStoreImpl) HourlyStats(ctx context.Context, table string, cutoff time.Time) ([]store.HourlyStat, error) {
	var result []store.HourlyStat

	switch table {
	case "schematics":
		rows, err := ss.q.HourlySchematicStats(ctx, cutoff)
		if err != nil {
			return nil, err
		}
		result = make([]store.HourlyStat, len(rows))
		for i, r := range rows {
			result[i] = store.HourlyStat{Hour: r.Hour, Count: r.Count}
		}
	case "comments":
		rows, err := ss.q.HourlyCommentStats(ctx, cutoff)
		if err != nil {
			return nil, err
		}
		result = make([]store.HourlyStat, len(rows))
		for i, r := range rows {
			result[i] = store.HourlyStat{Hour: r.Hour, Count: r.Count}
		}
	case "users":
		rows, err := ss.q.HourlyUserStats(ctx, cutoff)
		if err != nil {
			return nil, err
		}
		result = make([]store.HourlyStat, len(rows))
		for i, r := range rows {
			result[i] = store.HourlyStat{Hour: r.Hour, Count: r.Count}
		}
	case "schematic_views":
		rows, err := ss.q.HourlyViewStats(ctx, cutoff)
		if err != nil {
			return nil, err
		}
		result = make([]store.HourlyStat, len(rows))
		for i, r := range rows {
			result[i] = store.HourlyStat{Hour: r.Hour, Count: r.Count}
		}
	case "schematic_downloads":
		rows, err := ss.q.HourlyDownloadStats(ctx, cutoff)
		if err != nil {
			return nil, err
		}
		result = make([]store.HourlyStat, len(rows))
		for i, r := range rows {
			result[i] = store.HourlyStat{Hour: r.Hour, Count: r.Count}
		}
	default:
		return nil, fmt.Errorf("unknown table for hourly stats: %s", table)
	}

	return result, nil
}

func (ss *StatsStoreImpl) MonthlyUserStats(ctx context.Context, userID string, months int) ([]store.MonthlyDataPoint, error) {
	uid := &userID

	uploadsRows, err := ss.q.MonthlyUserUploads(ctx, db.MonthlyUserUploadsParams{
		AuthorID: uid,
		Column2:  int32(months),
	})
	if err != nil {
		return nil, fmt.Errorf("monthly uploads: %w", err)
	}

	downloadsRows, err := ss.q.MonthlyUserDownloads(ctx, db.MonthlyUserDownloadsParams{
		AuthorID: uid,
		Column2:  int32(months),
	})
	if err != nil {
		return nil, fmt.Errorf("monthly downloads: %w", err)
	}

	viewsRows, err := ss.q.MonthlyUserViews(ctx, db.MonthlyUserViewsParams{
		AuthorID: uid,
		Column2:  int32(months),
	})
	if err != nil {
		return nil, fmt.Errorf("monthly views: %w", err)
	}

	// Merge into a single slice keyed by month
	monthMap := make(map[string]*store.MonthlyDataPoint)
	for _, r := range uploadsRows {
		dp := monthMap[r.Month]
		if dp == nil {
			dp = &store.MonthlyDataPoint{Month: r.Month}
			monthMap[r.Month] = dp
		}
		dp.Uploads = r.Count
	}
	for _, r := range downloadsRows {
		dp := monthMap[r.Month]
		if dp == nil {
			dp = &store.MonthlyDataPoint{Month: r.Month}
			monthMap[r.Month] = dp
		}
		dp.Downloads = r.Count
	}
	for _, r := range viewsRows {
		dp := monthMap[r.Month]
		if dp == nil {
			dp = &store.MonthlyDataPoint{Month: r.Month}
			monthMap[r.Month] = dp
		}
		dp.Views = r.Count
	}

	// Collect and sort by month
	result := make([]store.MonthlyDataPoint, 0, len(monthMap))
	for _, dp := range monthMap {
		result = append(result, *dp)
	}
	// Sort ascending by month string (YYYY-MM sorts lexicographically)
	for i := 0; i < len(result); i++ {
		for j := i + 1; j < len(result); j++ {
			if result[j].Month < result[i].Month {
				result[i], result[j] = result[j], result[i]
			}
		}
	}

	return result, nil
}

// --------------------------------------------------------------------------
// NBTHashStoreImpl
// --------------------------------------------------------------------------

type NBTHashStoreImpl struct {
	q *db.Queries
}

func (s *NBTHashStoreImpl) Create(ctx context.Context, h *store.NBTHash) error {
	return s.q.CreateNBTHash(ctx, db.CreateNBTHashParams{
		ID:          h.ID,
		Hash:        h.Hash,
		SchematicID: h.SchematicID,
		UploadedBy:  h.UploadedBy,
	})
}

func (s *NBTHashStoreImpl) ListByUser(ctx context.Context, userID string) ([]store.NBTHash, error) {
	rows, err := s.q.ListNBTHashesByUser(ctx, &userID)
	if err != nil {
		return nil, err
	}
	result := make([]store.NBTHash, len(rows))
	for i, r := range rows {
		result[i] = store.NBTHash{
			ID:          r.ID,
			Hash:        r.Hash,
			SchematicID: r.SchematicID,
			UploadedBy:  r.UploadedBy,
			Created:     r.Created,
		}
	}
	return result, nil
}

func (s *NBTHashStoreImpl) Delete(ctx context.Context, id, userID string) error {
	return s.q.DeleteNBTHash(ctx, db.DeleteNBTHashParams{
		ID:         id,
		UploadedBy: &userID,
	})
}

func (s *NBTHashStoreImpl) IsBlacklisted(ctx context.Context, hash string) (bool, error) {
	return s.q.CheckHashIsBlacklisted(ctx, hash)
}

// --------------------------------------------------------------------------
// Updated NewStore that uses separate impl types to avoid method collisions
// --------------------------------------------------------------------------

// NewStoreFromPool returns a store.Store backed by PostgreSQL.
func NewStoreFromPool(pool *pgxpool.Pool) *store.Store {
	q := db.New(pool)
	ps := &PostgresStore{q: q, pool: pool}
	return &store.Store{
		Users:          &UserStoreImpl{q: q},
		Sessions:       ps,
		Schematics:     ps,
		Categories:     &CategoryStoreImpl{q: q},
		Tags:           &TagStoreImpl{q: q},
		Comments:       &CommentStoreImpl{q: q},
		Guides:         &GuideStoreImpl{q: q},
		Collections:    &CollectionStoreImpl{q: q},
		Achievements:   &AchievementStoreImpl{q: q},
		Translations:   &TranslationStoreImpl{q: q},
		ViewRatings:    &ViewRatingStoreImpl{q: q},
		Versions:       &VersionStoreImpl{q: q},
		APIKeys:        &APIKeyStoreImpl{q: q},
		Auth:           &AuthStoreImpl{q: q},
		Reports:        &ReportStoreImpl{q: q},
		ModMetadata:    &ModMetadataStoreImpl{q: q},
		VersionLookup:  &VersionLookupStoreImpl{q: q},
		SearchTracking: &SearchTrackingStoreImpl{q: q},
		OutgoingClicks: &OutgoingClickStoreImpl{q: q},
		Contact:        &ContactStoreImpl{q: q},
		Stats:           &StatsStoreImpl{q: q},
		TempUploads:      &TempUploadStoreImpl{q: q},
		TempUploadFiles:  &TempUploadFileStoreImpl{q: q},
		TempUploadImages: &TempUploadImageStoreImpl{q: q},
		NBTHashes:       &NBTHashStoreImpl{q: q},
		DownloadTokens:  &DownloadTokenStoreImpl{q: q},
		SchematicFiles:  &SchematicFileStoreImpl{q: q},
		Webhooks:            &WebhookStoreImpl{q: q},
		SchematicVariations: &SchematicVariationStoreImpl{q: q},
		ModerationChats:     &ModerationChatStoreImpl{q: q},
	}
}

// --------------------------------------------------------------------------
// Additional pointer helpers
// --------------------------------------------------------------------------

func ptrBool(b bool) *bool {
	return &b
}

func ptrInt32(i int32) *int32 {
	return &i
}

func ptrStrNonEmpty(s string) *string {
	if s == "" {
		return nil
	}
	return &s
}

// --------------------------------------------------------------------------
// ModerationChatStoreImpl
// --------------------------------------------------------------------------

type ModerationChatStoreImpl struct{ q *db.Queries }

func (s *ModerationChatStoreImpl) GetThreadByContent(ctx context.Context, contentType, contentID string) (*store.ModerationThread, error) {
	row, err := s.q.GetModerationThreadByContent(ctx, db.GetModerationThreadByContentParams{
		ContentType: contentType,
		ContentID:   contentID,
	})
	if err != nil {
		return nil, err
	}
	return &store.ModerationThread{
		ID:          row.ID,
		ContentType: row.ContentType,
		ContentID:   row.ContentID,
		Status:      row.Status,
		Created:     row.Created,
		Updated:     row.Updated,
	}, nil
}

func (s *ModerationChatStoreImpl) CreateThread(ctx context.Context, contentType, contentID string) (*store.ModerationThread, error) {
	row, err := s.q.CreateModerationThread(ctx, db.CreateModerationThreadParams{
		ContentType: contentType,
		ContentID:   contentID,
	})
	if err != nil {
		return nil, err
	}
	return &store.ModerationThread{
		ID:          row.ID,
		ContentType: row.ContentType,
		ContentID:   row.ContentID,
		Status:      row.Status,
		Created:     row.Created,
		Updated:     row.Updated,
	}, nil
}

func (s *ModerationChatStoreImpl) ListMessages(ctx context.Context, threadID string) ([]store.ModerationMessage, error) {
	rows, err := s.q.ListModerationMessagesByThread(ctx, threadID)
	if err != nil {
		return nil, err
	}
	msgs := make([]store.ModerationMessage, len(rows))
	for i, r := range rows {
		msgs[i] = store.ModerationMessage{
			ID:          r.ID,
			ThreadID:    r.ThreadID,
			AuthorID:    r.AuthorID,
			IsModerator: r.IsModerator,
			Body:        r.Body,
			Created:     r.Created,
		}
	}
	return msgs, nil
}

func (s *ModerationChatStoreImpl) CreateMessage(ctx context.Context, threadID, authorID string, isModerator bool, body string) (*store.ModerationMessage, error) {
	row, err := s.q.CreateModerationMessage(ctx, db.CreateModerationMessageParams{
		ThreadID:    threadID,
		AuthorID:    authorID,
		IsModerator: isModerator,
		Body:        body,
	})
	if err != nil {
		return nil, err
	}
	return &store.ModerationMessage{
		ID:          row.ID,
		ThreadID:    row.ThreadID,
		AuthorID:    row.AuthorID,
		IsModerator: row.IsModerator,
		Body:        row.Body,
		Created:     row.Created,
	}, nil
}

func (s *ModerationChatStoreImpl) CountUserMessagesSinceLastModerator(ctx context.Context, threadID string) (int, error) {
	count, err := s.q.CountUserMessagesSinceLastModerator(ctx, threadID)
	if err != nil {
		return 0, err
	}
	return int(count), nil
}

// Ensure compile-time interface satisfaction for the types on PostgresStore.
var (
	_ store.SessionStore        = (*PostgresStore)(nil)
	_ store.SchematicStore      = (*PostgresStore)(nil)
	_ store.ModerationChatStore = (*ModerationChatStoreImpl)(nil)
)

// Ensure compile-time interface satisfaction for the separate impl types.
var (
	_ store.UserStore          = (*UserStoreImpl)(nil)
	_ store.CategoryStore      = (*CategoryStoreImpl)(nil)
	_ store.TagStore           = (*TagStoreImpl)(nil)
	_ store.CommentStore       = (*CommentStoreImpl)(nil)
	_ store.GuideStore         = (*GuideStoreImpl)(nil)
	_ store.CollectionStore    = (*CollectionStoreImpl)(nil)
	_ store.AchievementStore   = (*AchievementStoreImpl)(nil)
	_ store.TranslationStore   = (*TranslationStoreImpl)(nil)
	_ store.ViewRatingStore    = (*ViewRatingStoreImpl)(nil)
	_ store.VersionStore       = (*VersionStoreImpl)(nil)
	_ store.APIKeyStore        = (*APIKeyStoreImpl)(nil)
	_ store.AuthStore          = (*AuthStoreImpl)(nil)
	_ store.ReportStore        = (*ReportStoreImpl)(nil)
	_ store.ModMetadataStore   = (*ModMetadataStoreImpl)(nil)
	_ store.VersionLookupStore = (*VersionLookupStoreImpl)(nil)
	_ store.SearchTrackingStore = (*SearchTrackingStoreImpl)(nil)
	_ store.OutgoingClickStore = (*OutgoingClickStoreImpl)(nil)
	_ store.ContactStore       = (*ContactStoreImpl)(nil)
	_ store.StatsStore          = (*StatsStoreImpl)(nil)
	_ store.TempUploadStore      = (*TempUploadStoreImpl)(nil)
	_ store.TempUploadFileStore  = (*TempUploadFileStoreImpl)(nil)
	_ store.TempUploadImageStore = (*TempUploadImageStoreImpl)(nil)
	_ store.DownloadTokenStore  = (*DownloadTokenStoreImpl)(nil)
	_ store.SchematicFileStore  = (*SchematicFileStoreImpl)(nil)
	_ store.WebhookStore             = (*WebhookStoreImpl)(nil)
	_ store.SchematicVariationStore  = (*SchematicVariationStoreImpl)(nil)
)

// Ensure unused import is satisfied.
var _ = fmt.Sprintf

// --------------------------------------------------------------------------
// TempUpload Store Implementation
// --------------------------------------------------------------------------

type TempUploadStoreImpl struct{ q *db.Queries }

func (s *TempUploadStoreImpl) Create(ctx context.Context, t *store.TempUpload) error {
	row, err := s.q.CreateTempUpload(ctx, db.CreateTempUploadParams{
		Token:            t.Token,
		UploadedBy:       t.UploadedBy,
		Filename:         t.Filename,
		Description:      t.Description,
		Size:             t.Size,
		Checksum:         t.Checksum,
		BlockCount:       int32(t.BlockCount),
		DimX:             int32(t.DimX),
		DimY:             int32(t.DimY),
		DimZ:             int32(t.DimZ),
		Mods:             t.Mods,
		Materials:        t.Materials,
		MinecraftVersion: t.MinecraftVersion,
		CreatemodVersion: t.CreatemodVersion,
		NbtS3Key:         t.NbtS3Key,
		ImageS3Key:       t.ImageS3Key,
		ParsedSummary:    t.ParsedSummary,
	})
	if err != nil {
		return err
	}
	t.ID = row.ID
	t.Created = row.Created
	t.Updated = row.Updated
	return nil
}

func (s *TempUploadStoreImpl) GetByToken(ctx context.Context, token string) (*store.TempUpload, error) {
	row, err := s.q.GetTempUploadByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	return &store.TempUpload{
		ID:               row.ID,
		Token:            row.Token,
		UploadedBy:       row.UploadedBy,
		Filename:         row.Filename,
		Description:      row.Description,
		Size:             row.Size,
		Checksum:         row.Checksum,
		BlockCount:       int(row.BlockCount),
		DimX:             int(row.DimX),
		DimY:             int(row.DimY),
		DimZ:             int(row.DimZ),
		Mods:             row.Mods,
		Materials:        row.Materials,
		MinecraftVersion: row.MinecraftVersion,
		CreatemodVersion: row.CreatemodVersion,
		NbtS3Key:         row.NbtS3Key,
		ImageS3Key:       row.ImageS3Key,
		ParsedSummary:    row.ParsedSummary,
		Processing:       row.Processing,
		Created:          row.Created,
		Updated:          row.Updated,
	}, nil
}

func (s *TempUploadStoreImpl) GetByChecksum(ctx context.Context, checksum string) (*store.TempUpload, error) {
	row, err := s.q.GetTempUploadByChecksum(ctx, checksum)
	if err != nil {
		return nil, err
	}
	return &store.TempUpload{
		ID:               row.ID,
		Token:            row.Token,
		UploadedBy:       row.UploadedBy,
		Filename:         row.Filename,
		Description:      row.Description,
		Size:             row.Size,
		Checksum:         row.Checksum,
		BlockCount:       int(row.BlockCount),
		DimX:             int(row.DimX),
		DimY:             int(row.DimY),
		DimZ:             int(row.DimZ),
		Mods:             row.Mods,
		Materials:        row.Materials,
		MinecraftVersion: row.MinecraftVersion,
		CreatemodVersion: row.CreatemodVersion,
		NbtS3Key:         row.NbtS3Key,
		ImageS3Key:       row.ImageS3Key,
		ParsedSummary:    row.ParsedSummary,
		Created:          row.Created,
		Updated:          row.Updated,
	}, nil
}

func (s *TempUploadStoreImpl) Update(ctx context.Context, t *store.TempUpload) error {
	return s.q.UpdateTempUpload(ctx, db.UpdateTempUploadParams{
		Token:       t.Token,
		Filename:    t.Filename,
		Description: t.Description,
		NbtS3Key:    t.NbtS3Key,
		ImageS3Key:  t.ImageS3Key,
	})
}

func (s *TempUploadStoreImpl) Claim(ctx context.Context, token string, userID string) error {
	rows, err := s.q.ClaimTempUpload(ctx, db.ClaimTempUploadParams{
		Token:      token,
		UploadedBy: userID,
	})
	if err != nil {
		return err
	}
	if rows == 0 {
		return fmt.Errorf("upload already claimed or not found")
	}
	return nil
}

func (s *TempUploadStoreImpl) MarkProcessing(ctx context.Context, token string) error {
	_, err := s.q.MarkTempUploadProcessing(ctx, token)
	if err != nil {
		return fmt.Errorf("upload already processing or not found")
	}
	return nil
}

func (s *TempUploadStoreImpl) Delete(ctx context.Context, token string) error {
	return s.q.DeleteTempUpload(ctx, token)
}

func (s *TempUploadStoreImpl) DeleteExpired(ctx context.Context, olderThan time.Time) (int64, error) {
	return s.q.DeleteExpiredTempUploads(ctx, olderThan)
}

func (s *TempUploadStoreImpl) ListByUser(ctx context.Context, userID string, limit int, offset int) ([]store.TempUpload, error) {
	rows, err := s.q.ListTempUploadsByUser(ctx, db.ListTempUploadsByUserParams{
		UploadedBy: userID,
		Limit:      int32(limit),
		Offset:     int32(offset),
	})
	if err != nil {
		return nil, err
	}
	result := make([]store.TempUpload, len(rows))
	for i, row := range rows {
		result[i] = store.TempUpload{
			ID:               row.ID,
			Token:            row.Token,
			UploadedBy:       row.UploadedBy,
			Filename:         row.Filename,
			Description:      row.Description,
			Size:             row.Size,
			Checksum:         row.Checksum,
			BlockCount:       int(row.BlockCount),
			DimX:             int(row.DimX),
			DimY:             int(row.DimY),
			DimZ:             int(row.DimZ),
			Mods:             row.Mods,
			Materials:        row.Materials,
			MinecraftVersion: row.MinecraftVersion,
			CreatemodVersion: row.CreatemodVersion,
			NbtS3Key:         row.NbtS3Key,
			ImageS3Key:       row.ImageS3Key,
			ParsedSummary:    row.ParsedSummary,
			Created:          row.Created,
			Updated:          row.Updated,
		}
	}
	return result, nil
}

func (s *TempUploadStoreImpl) ListExpiredUnclaimed(ctx context.Context, olderThan time.Time, limit int) ([]store.TempUpload, error) {
	rows, err := s.q.ListExpiredUnclaimedTempUploads(ctx, db.ListExpiredUnclaimedTempUploadsParams{
		Created: olderThan,
		Limit:   int32(limit),
	})
	if err != nil {
		return nil, err
	}
	result := make([]store.TempUpload, len(rows))
	for i, row := range rows {
		result[i] = store.TempUpload{
			ID:         row.ID,
			Token:      row.Token,
			NbtS3Key:   row.NbtS3Key,
			ImageS3Key: row.ImageS3Key,
		}
	}
	return result, nil
}

func (s *TempUploadStoreImpl) DeleteExpiredUnclaimed(ctx context.Context, olderThan time.Time) (int64, error) {
	return s.q.DeleteExpiredUnclaimedTempUploads(ctx, olderThan)
}

// --------------------------------------------------------------------------
// TempUploadFile Store Implementation
// --------------------------------------------------------------------------

type TempUploadFileStoreImpl struct{ q *db.Queries }

func (s *TempUploadFileStoreImpl) Create(ctx context.Context, f *store.TempUploadFile) error {
	row, err := s.q.CreateTempUploadFile(ctx, db.CreateTempUploadFileParams{
		Token:       f.Token,
		Filename:    f.Filename,
		Description: f.Description,
		Size:        f.Size,
		Checksum:    f.Checksum,
		BlockCount:  int32(f.BlockCount),
		DimX:        int32(f.DimX),
		DimY:        int32(f.DimY),
		DimZ:        int32(f.DimZ),
		Mods:        f.Mods,
		Materials:   f.Materials,
		NbtS3Key:    f.NbtS3Key,
	})
	if err != nil {
		return err
	}
	f.ID = row.ID
	f.Created = row.Created
	return nil
}

func (s *TempUploadFileStoreImpl) ListByToken(ctx context.Context, token string) ([]store.TempUploadFile, error) {
	rows, err := s.q.ListTempUploadFilesByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	result := make([]store.TempUploadFile, len(rows))
	for i, r := range rows {
		result[i] = store.TempUploadFile{
			ID:          r.ID,
			Token:       r.Token,
			Filename:    r.Filename,
			Description: r.Description,
			Size:        r.Size,
			Checksum:    r.Checksum,
			BlockCount:  int(r.BlockCount),
			DimX:        int(r.DimX),
			DimY:        int(r.DimY),
			DimZ:        int(r.DimZ),
			Mods:        r.Mods,
			Materials:   r.Materials,
			NbtS3Key:    r.NbtS3Key,
			Created:     r.Created,
		}
	}
	return result, nil
}

func (s *TempUploadFileStoreImpl) GetByID(ctx context.Context, id string) (*store.TempUploadFile, error) {
	r, err := s.q.GetTempUploadFileByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &store.TempUploadFile{
		ID:          r.ID,
		Token:       r.Token,
		Filename:    r.Filename,
		Description: r.Description,
		Size:        r.Size,
		Checksum:    r.Checksum,
		BlockCount:  int(r.BlockCount),
		DimX:        int(r.DimX),
		DimY:        int(r.DimY),
		DimZ:        int(r.DimZ),
		Mods:        r.Mods,
		Materials:   r.Materials,
		NbtS3Key:    r.NbtS3Key,
		Created:     r.Created,
	}, nil
}

func (s *TempUploadFileStoreImpl) Delete(ctx context.Context, id string) error {
	return s.q.DeleteTempUploadFile(ctx, id)
}

func (s *TempUploadFileStoreImpl) DeleteByToken(ctx context.Context, token string) error {
	return s.q.DeleteTempUploadFilesByToken(ctx, token)
}

// --------------------------------------------------------------------------
// TempUploadImage Store Implementation
// --------------------------------------------------------------------------

type TempUploadImageStoreImpl struct{ q *db.Queries }

func (s *TempUploadImageStoreImpl) Create(ctx context.Context, img *store.TempUploadImage) error {
	row, err := s.q.CreateTempUploadImage(ctx, db.CreateTempUploadImageParams{
		Token:     img.Token,
		Filename:  img.Filename,
		Size:      img.Size,
		S3Key:     img.S3Key,
		SortOrder: int32(img.SortOrder),
	})
	if err != nil {
		return err
	}
	img.ID = row.ID
	img.Created = row.Created
	return nil
}

func (s *TempUploadImageStoreImpl) ListByToken(ctx context.Context, token string) ([]store.TempUploadImage, error) {
	rows, err := s.q.ListTempUploadImagesByToken(ctx, token)
	if err != nil {
		return nil, err
	}
	result := make([]store.TempUploadImage, len(rows))
	for i, r := range rows {
		result[i] = store.TempUploadImage{
			ID:        r.ID,
			Token:     r.Token,
			Filename:  r.Filename,
			Size:      r.Size,
			S3Key:     r.S3Key,
			SortOrder: int(r.SortOrder),
			Created:   r.Created,
		}
	}
	return result, nil
}

func (s *TempUploadImageStoreImpl) Delete(ctx context.Context, id string) error {
	return s.q.DeleteTempUploadImage(ctx, id)
}

func (s *TempUploadImageStoreImpl) DeleteByToken(ctx context.Context, token string) error {
	return s.q.DeleteTempUploadImagesByToken(ctx, token)
}

func (s *TempUploadImageStoreImpl) CountByToken(ctx context.Context, token string) (int, error) {
	count, err := s.q.CountTempUploadImagesByToken(ctx, token)
	return int(count), err
}

// --------------------------------------------------------------------------
// SchematicFile Store Implementation
// --------------------------------------------------------------------------

type SchematicFileStoreImpl struct{ q *db.Queries }

func (sf *SchematicFileStoreImpl) Create(ctx context.Context, f *store.SchematicFile) error {
	row, err := sf.q.CreateSchematicFile(ctx, db.CreateSchematicFileParams{
		SchematicID:  f.SchematicID,
		Filename:     f.Filename,
		OriginalName: f.OriginalName,
		Size:         f.Size,
		MimeType:     f.MimeType,
	})
	if err != nil {
		return err
	}
	f.ID = row.ID
	f.Created = row.Created
	f.Updated = row.Updated
	return nil
}

func (sf *SchematicFileStoreImpl) ListBySchematicID(ctx context.Context, schematicID string) ([]store.SchematicFile, error) {
	rows, err := sf.q.ListSchematicFilesBySchematicID(ctx, schematicID)
	if err != nil {
		return nil, err
	}
	result := make([]store.SchematicFile, len(rows))
	for i, r := range rows {
		result[i] = store.SchematicFile{
			ID:           r.ID,
			SchematicID:  r.SchematicID,
			Filename:     r.Filename,
			OriginalName: r.OriginalName,
			Size:         r.Size,
			MimeType:     r.MimeType,
			Created:      r.Created,
			Updated:      r.Updated,
		}
	}
	return result, nil
}

func (sf *SchematicFileStoreImpl) Delete(ctx context.Context, id string) error {
	return sf.q.DeleteSchematicFile(ctx, id)
}

func (sf *SchematicFileStoreImpl) DeleteBySchematicID(ctx context.Context, schematicID string) error {
	return sf.q.DeleteSchematicFilesBySchematicID(ctx, schematicID)
}

// --------------------------------------------------------------------------
// DownloadToken Store Implementation
// --------------------------------------------------------------------------

type DownloadTokenStoreImpl struct{ q *db.Queries }

func (dt *DownloadTokenStoreImpl) Create(ctx context.Context, t *store.DownloadToken) error {
	row, err := dt.q.CreateDownloadToken(ctx, db.CreateDownloadTokenParams{
		Token:     t.Token,
		Name:      t.Name,
		ExpiresAt: t.ExpiresAt,
	})
	if err != nil {
		return err
	}
	t.ID = row.ID
	t.Created = row.Created
	return nil
}

func (dt *DownloadTokenStoreImpl) GetByID(ctx context.Context, id string) (*store.DownloadToken, error) {
	row, err := dt.q.GetDownloadTokenByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &store.DownloadToken{
		ID:        row.ID,
		Token:     row.Token,
		Name:      row.Name,
		ExpiresAt: row.ExpiresAt,
		Used:      row.Used,
		Created:   row.Created,
	}, nil
}

func (dt *DownloadTokenStoreImpl) Consume(ctx context.Context, token string) (*store.DownloadToken, error) {
	row, err := dt.q.ConsumeDownloadToken(ctx, token)
	if err != nil {
		return nil, err
	}
	return &store.DownloadToken{
		ID:        row.ID,
		Token:     row.Token,
		Name:      row.Name,
		ExpiresAt: row.ExpiresAt,
		Used:      row.Used,
		Created:   row.Created,
	}, nil
}

func (dt *DownloadTokenStoreImpl) CleanupExpired(ctx context.Context) error {
	return dt.q.CleanupExpiredDownloadTokens(ctx)
}

// --------------------------------------------------------------------------
// Webhook Store Implementation
// --------------------------------------------------------------------------

type WebhookStoreImpl struct{ q *db.Queries }

func (ws *WebhookStoreImpl) Create(ctx context.Context, userID, encryptedURL string) error {
	_, err := ws.q.CreateUserWebhook(ctx, db.CreateUserWebhookParams{
		UserID:              userID,
		WebhookUrlEncrypted: encryptedURL,
	})
	return err
}

func (ws *WebhookStoreImpl) Upsert(ctx context.Context, userID, encryptedURL string) error {
	return ws.q.UpsertUserWebhook(ctx, db.UpsertUserWebhookParams{
		UserID:              userID,
		WebhookUrlEncrypted: encryptedURL,
	})
}

func (ws *WebhookStoreImpl) GetByUserID(ctx context.Context, userID string) (*store.UserWebhook, error) {
	row, err := ws.q.GetUserWebhookByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return webhookFromDB(row), nil
}

func (ws *WebhookStoreImpl) UpdateURL(ctx context.Context, userID, encryptedURL string) error {
	return ws.q.UpdateUserWebhookURL(ctx, db.UpdateUserWebhookURLParams{
		UserID:              userID,
		WebhookUrlEncrypted: encryptedURL,
	})
}

func (ws *WebhookStoreImpl) Delete(ctx context.Context, userID string) error {
	return ws.q.DeleteUserWebhook(ctx, userID)
}

func (ws *WebhookStoreImpl) ListActive(ctx context.Context) ([]store.UserWebhook, error) {
	rows, err := ws.q.ListActiveUserWebhooks(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]store.UserWebhook, len(rows))
	for i, r := range rows {
		result[i] = store.UserWebhook{
			ID:                  r.ID,
			UserID:              r.UserID,
			WebhookURLEncrypted: r.WebhookUrlEncrypted,
		}
	}
	return result, nil
}

func (ws *WebhookStoreImpl) IncrementFailure(ctx context.Context, id, message string) error {
	return ws.q.IncrementWebhookFailure(ctx, db.IncrementWebhookFailureParams{
		ID:                 id,
		LastFailureMessage: message,
	})
}

func (ws *WebhookStoreImpl) ResetFailures(ctx context.Context, id string) error {
	return ws.q.ResetWebhookFailures(ctx, id)
}

func webhookFromDB(row db.UserWebhook) *store.UserWebhook {
	w := &store.UserWebhook{
		ID:                  row.ID,
		UserID:              row.UserID,
		WebhookURLEncrypted: row.WebhookUrlEncrypted,
		Active:              row.Active,
		ConsecutiveFailures: int(row.ConsecutiveFailures),
		LastFailureMessage:  row.LastFailureMessage,
		Created:             row.Created,
		Updated:             row.Updated,
	}
	if row.LastFailureAt.Valid {
		t := row.LastFailureAt.Time
		w.LastFailureAt = &t
	}
	return w
}

// --------------------------------------------------------------------------
// SchematicVariation Store Implementation
// --------------------------------------------------------------------------

type SchematicVariationStoreImpl struct{ q *db.Queries }

func (s *SchematicVariationStoreImpl) Create(ctx context.Context, v *store.SchematicVariation) error {
	row, err := s.q.CreateSchematicVariation(ctx, db.CreateSchematicVariationParams{
		SchematicID:  v.SchematicID,
		UserID:       v.UserID,
		Name:         v.Name,
		Replacements: v.Replacements,
		IsPublic:     v.IsPublic,
	})
	if err != nil {
		return err
	}
	v.ID = row.ID
	v.CreatedAt = row.Created
	v.UpdatedAt = row.Updated
	return nil
}

func (s *SchematicVariationStoreImpl) GetByID(ctx context.Context, id string) (*store.SchematicVariation, error) {
	row, err := s.q.GetSchematicVariationByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return variationFromDB(row), nil
}

func (s *SchematicVariationStoreImpl) ListBySchematicAndUser(ctx context.Context, schematicID, userID string) ([]*store.SchematicVariation, error) {
	rows, err := s.q.ListSchematicVariationsBySchematicAndUser(ctx, db.ListSchematicVariationsBySchematicAndUserParams{
		SchematicID: schematicID,
		UserID:      userID,
	})
	if err != nil {
		return nil, err
	}
	result := make([]*store.SchematicVariation, len(rows))
	for i, r := range rows {
		result[i] = variationFromDB(r)
	}
	return result, nil
}

func (s *SchematicVariationStoreImpl) ListPublicBySchematic(ctx context.Context, schematicID string) ([]*store.SchematicVariation, error) {
	rows, err := s.q.ListPublicSchematicVariationsBySchematic(ctx, schematicID)
	if err != nil {
		return nil, err
	}
	result := make([]*store.SchematicVariation, len(rows))
	for i, r := range rows {
		result[i] = variationFromDB(r)
	}
	return result, nil
}

func (s *SchematicVariationStoreImpl) Update(ctx context.Context, v *store.SchematicVariation) error {
	return s.q.UpdateSchematicVariation(ctx, db.UpdateSchematicVariationParams{
		ID:           v.ID,
		Name:         v.Name,
		Replacements: v.Replacements,
		IsPublic:     v.IsPublic,
	})
}

func (s *SchematicVariationStoreImpl) Delete(ctx context.Context, id string) error {
	return s.q.DeleteSchematicVariation(ctx, id)
}

func (s *SchematicVariationStoreImpl) CountBySchematicAndUser(ctx context.Context, schematicID, userID string) (int, error) {
	count, err := s.q.CountSchematicVariationsBySchematicAndUser(ctx, db.CountSchematicVariationsBySchematicAndUserParams{
		SchematicID: schematicID,
		UserID:      userID,
	})
	return int(count), err
}

func (s *SchematicVariationStoreImpl) GetOldestBySchematicAndUser(ctx context.Context, schematicID, userID string) (*store.SchematicVariation, error) {
	row, err := s.q.GetOldestSchematicVariationBySchematicAndUser(ctx, db.GetOldestSchematicVariationBySchematicAndUserParams{
		SchematicID: schematicID,
		UserID:      userID,
	})
	if err != nil {
		return nil, err
	}
	return variationFromDB(row), nil
}

func variationFromDB(row db.SchematicVariation) *store.SchematicVariation {
	return &store.SchematicVariation{
		ID:           row.ID,
		SchematicID:  row.SchematicID,
		UserID:       row.UserID,
		Name:         row.Name,
		Replacements: row.Replacements,
		IsPublic:     row.IsPublic,
		CreatedAt:    row.Created,
		UpdatedAt:    row.Updated,
	}
}
