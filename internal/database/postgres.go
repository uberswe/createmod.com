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
		Featured:           s.Featured,
		AIDescription:      s.AiDescription,
		Moderated:          s.Moderated,
		ModerationReason:   s.ModerationReason,
		Blacklisted:        s.Blacklisted,
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
		ID:            m.ID,
		Namespace:     m.Namespace,
		DisplayName:   m.DisplayName,
		Description:   m.Description,
		IconURL:       m.IconUrl,
		ModrinthSlug:  m.ModrinthSlug,
		ModrinthURL:   m.ModrinthUrl,
		CurseforgeID:  m.CurseforgeID,
		CurseforgeURL: m.CurseforgeUrl,
		SourceURL:     m.SourceUrl,
		LastFetched:   fromPgTimestamptz(m.LastFetched),
		ManuallySet:   m.ManuallySet,
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
// UserStore implementation
// ============================================================================

func (ps *PostgresStore) GetUserByID(ctx context.Context, id string) (*store.User, error) {
	u, err := ps.q.GetUserByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return userFromDB(u), nil
}

func (ps *PostgresStore) GetUserByEmail(ctx context.Context, email string) (*store.User, error) {
	u, err := ps.q.GetUserByEmail(ctx, email)
	if err != nil {
		return nil, err
	}
	return userFromDB(u), nil
}

func (ps *PostgresStore) GetUserByUsername(ctx context.Context, username string) (*store.User, error) {
	u, err := ps.q.GetUserByUsername(ctx, username)
	if err != nil {
		return nil, err
	}
	return userFromDB(u), nil
}

func (ps *PostgresStore) CreateUser(ctx context.Context, u *store.User) error {
	if u.ID == "" {
		u.ID = generateID()
	}
	created, err := ps.q.CreateUser(ctx, db.CreateUserParams{
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

func (ps *PostgresStore) UpdateUser(ctx context.Context, u *store.User) error {
	_, err := ps.q.UpdateUser(ctx, db.UpdateUserParams{
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

func (ps *PostgresStore) UpdateUserPoints(ctx context.Context, id string, points int) error {
	return ps.q.UpdateUserPoints(ctx, db.UpdateUserPointsParams{
		ID:     id,
		Points: int32(points),
	})
}

func (ps *PostgresStore) UpdateUserPassword(ctx context.Context, id string, hash string) error {
	return ps.q.UpdateUserPassword(ctx, db.UpdateUserPasswordParams{
		ID:           id,
		PasswordHash: hash,
	})
}

func (ps *PostgresStore) UpdateUserAvatar(ctx context.Context, id string, avatar string) error {
	return ps.q.UpdateUserAvatar(ctx, db.UpdateUserAvatarParams{
		ID:     id,
		Avatar: avatar,
	})
}

func (ps *PostgresStore) SoftDeleteUser(ctx context.Context, id string) error {
	return ps.q.SoftDeleteUser(ctx, id)
}

func (ps *PostgresStore) IsContributor(ctx context.Context, userID string) (bool, error) {
	return ps.q.GetUserIsContributor(ctx, &userID)
}

func (ps *PostgresStore) ListUsers(ctx context.Context, limit, offset int) ([]store.User, error) {
	rows, err := ps.q.ListUsers(ctx, db.ListUsersParams{
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

func (ps *PostgresStore) CountUsers(ctx context.Context) (int64, error) {
	return ps.q.CountUsers(ctx)
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
		Moderated:          s.Moderated,
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
		Moderated:          ptrBool(s.Moderated),
		ModerationReason:   ptrStr(s.ModerationReason),
		Blacklisted:        ptrBool(s.Blacklisted),
		Featured:           ptrBool(s.Featured),
		ScheduledAt:        toPgTimestamptz(s.ScheduledAt),
		BlockCount:         ptrInt32(int32(s.BlockCount)),
		DimX:               ptrInt32(int32(s.DimX)),
		DimY:               ptrInt32(int32(s.DimY)),
		DimZ:               ptrInt32(int32(s.DimZ)),
		Materials:          s.Materials,
		Mods:               s.Mods,
	})
	return err
}

func (ps *PostgresStore) SoftDelete(ctx context.Context, id string) error {
	return ps.q.SoftDeleteSchematic(ctx, id)
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
		result[i] = store.Category{ID: r.ID, Key: r.Key, Name: r.Name}
	}
	return result, nil
}

func (cs *CategoryStoreImpl) GetByID(ctx context.Context, id string) (*store.Category, error) {
	row, err := cs.q.GetCategoryByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &store.Category{ID: row.ID, Key: row.Key, Name: row.Name}, nil
}

func (cs *CategoryStoreImpl) GetByIDs(ctx context.Context, ids []string) ([]store.Category, error) {
	rows, err := cs.q.GetCategoriesByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	result := make([]store.Category, len(rows))
	for i, r := range rows {
		result[i] = store.Category{ID: r.ID, Key: r.Key, Name: r.Name}
	}
	return result, nil
}

func (cs *CategoryStoreImpl) Create(ctx context.Context, c *store.Category) error {
	if c.ID == "" {
		c.ID = generateID()
	}
	_, err := cs.q.CreateCategory(ctx, db.CreateCategoryParams{
		ID:   c.ID,
		Key:  c.Key,
		Name: c.Name,
	})
	return err
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
		result[i] = store.Tag{ID: r.ID, Key: r.Key, Name: r.Name}
	}
	return result, nil
}

func (ts *TagStoreImpl) GetByID(ctx context.Context, id string) (*store.Tag, error) {
	row, err := ts.q.GetTagByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &store.Tag{ID: row.ID, Key: row.Key, Name: row.Name}, nil
}

func (ts *TagStoreImpl) GetByIDs(ctx context.Context, ids []string) ([]store.Tag, error) {
	rows, err := ts.q.GetTagsByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	result := make([]store.Tag, len(rows))
	for i, r := range rows {
		result[i] = store.Tag{ID: r.ID, Key: r.Key, Name: r.Name}
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
		ID:   t.ID,
		Key:  t.Key,
		Name: t.Name,
	})
	return err
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
	})
	return err
}

func (gs *GuideStoreImpl) Delete(ctx context.Context, id string) error {
	return gs.q.DeleteGuide(ctx, id)
}

func (gs *GuideStoreImpl) CountByUser(ctx context.Context, userID string) (int64, error) {
	return gs.q.CountUserGuides(ctx, &userID)
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
		Featured:    ptrBool(c.Featured),
		Published:   ptrBool(c.Published),
	})
	return err
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
		ID:            m.ID,
		Namespace:     m.Namespace,
		DisplayName:   m.DisplayName,
		Description:   m.Description,
		IconUrl:       m.IconURL,
		ModrinthSlug:  m.ModrinthSlug,
		ModrinthUrl:   m.ModrinthURL,
		CurseforgeID:  m.CurseforgeID,
		CurseforgeUrl: m.CurseforgeURL,
		SourceUrl:     m.SourceURL,
		LastFetched:   toPgTimestamptz(m.LastFetched),
		ManuallySet:   m.ManuallySet,
	})
	return err
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

// --------------------------------------------------------------------------
// Updated NewStore that uses separate impl types to avoid method collisions
// --------------------------------------------------------------------------

// NewStoreFromPool returns a store.Store backed by PostgreSQL.
func NewStoreFromPool(pool *pgxpool.Pool) *store.Store {
	q := db.New(pool)
	ps := &PostgresStore{q: q, pool: pool}
	return &store.Store{
		Users:        ps,
		Sessions:     ps,
		Schematics:   ps,
		Categories:   &CategoryStoreImpl{q: q},
		Tags:         &TagStoreImpl{q: q},
		Comments:     &CommentStoreImpl{q: q},
		Guides:       &GuideStoreImpl{q: q},
		Collections:  &CollectionStoreImpl{q: q},
		Achievements: &AchievementStoreImpl{q: q},
		Translations: &TranslationStoreImpl{q: q},
		ViewRatings:  &ViewRatingStoreImpl{q: q},
		Versions:     &VersionStoreImpl{q: q},
		APIKeys:      &APIKeyStoreImpl{q: q},
		Auth:         &AuthStoreImpl{q: q},
		Reports:      &ReportStoreImpl{q: q},
		ModMetadata:  &ModMetadataStoreImpl{q: q},
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

// Ensure compile-time interface satisfaction for the types on PostgresStore.
var (
	_ store.UserStore      = (*PostgresStore)(nil)
	_ store.SessionStore   = (*PostgresStore)(nil)
	_ store.SchematicStore = (*PostgresStore)(nil)
)

// Ensure compile-time interface satisfaction for the separate impl types.
var (
	_ store.CategoryStore    = (*CategoryStoreImpl)(nil)
	_ store.TagStore         = (*TagStoreImpl)(nil)
	_ store.CommentStore     = (*CommentStoreImpl)(nil)
	_ store.GuideStore       = (*GuideStoreImpl)(nil)
	_ store.CollectionStore  = (*CollectionStoreImpl)(nil)
	_ store.AchievementStore = (*AchievementStoreImpl)(nil)
	_ store.TranslationStore = (*TranslationStoreImpl)(nil)
	_ store.ViewRatingStore  = (*ViewRatingStoreImpl)(nil)
	_ store.VersionStore     = (*VersionStoreImpl)(nil)
	_ store.APIKeyStore      = (*APIKeyStoreImpl)(nil)
	_ store.AuthStore        = (*AuthStoreImpl)(nil)
	_ store.ReportStore      = (*ReportStoreImpl)(nil)
	_ store.ModMetadataStore = (*ModMetadataStoreImpl)(nil)
)

// Ensure unused import is satisfied.
var _ = fmt.Sprintf
