// Package store defines the persistence interfaces for the application.
// Implementations live in internal/database (PostgreSQL via sqlc).
package store

import (
	"context"
	"encoding/json"
	"time"
)

// User represents a user account.
type User struct {
	ID           string
	Email        string
	Username     string
	PasswordHash string
	OldPassword  string
	Avatar       string
	Points       int
	Verified     bool
	IsAdmin      bool
	Deleted      *time.Time
	Created      time.Time
	Updated      time.Time
}

// Session represents an authenticated user session.
type Session struct {
	ID        string
	UserID    string
	ExpiresAt time.Time
	Created   time.Time
	User      *User // populated on lookup
}

// Schematic represents a Create mod schematic.
type Schematic struct {
	ID                 string
	AuthorID           string
	Name               string
	Title              string
	Description        string
	Excerpt            string
	Content            string
	Postdate           *time.Time
	Modified           *time.Time
	DetectedLanguage   string
	FeaturedImage      string
	Gallery            []string
	SchematicFile      string
	Video              string
	HasDependencies    bool
	Dependencies       string
	CreatemodVersionID *string
	MinecraftVersionID *string
	Views              int
	Downloads          int
	BlockCount         int
	DimX               int
	DimY               int
	DimZ               int
	Materials          json.RawMessage
	Mods               json.RawMessage
	Paid               bool
	Featured           bool
	AIDescription      string
	Moderated          bool
	ModerationReason   string
	Blacklisted        bool
	ScheduledAt        *time.Time
	Deleted            *time.Time
	OldID              *int
	Status             string
	Type               string
	Created            time.Time
	Updated            time.Time
}

// Category represents a schematic category.
type Category struct {
	ID   string
	Key  string
	Name string
}

// Tag represents a schematic tag.
type Tag struct {
	ID   string
	Key  string
	Name string
}

// TagWithCount is a tag with the number of schematics using it.
type TagWithCount struct {
	ID    string
	Key   string
	Name  string
	Count int64
}

// Comment represents a comment on a schematic.
type Comment struct {
	ID             string
	AuthorID       *string
	SchematicID    *string
	ParentID       *string
	Content        string
	Published      *time.Time
	Approved       bool
	Type           string
	Karma          int
	AuthorUsername string
	AuthorAvatar   string
	Created        time.Time
	Updated        time.Time
}

// Guide represents a community guide.
type Guide struct {
	ID          string
	AuthorID    *string
	Title       string
	Description string
	Content     string
	Slug        string
	UploadLink  string
	Created     time.Time
	Updated     time.Time
}

// Collection represents a user-curated collection of schematics.
type Collection struct {
	ID          string
	AuthorID    *string
	Title       string
	Name        string
	Slug        string
	Description string
	BannerURL   string
	Featured    bool
	Views       int
	Published   bool
	Deleted     string
	Created     time.Time
	Updated     time.Time
}

// Achievement represents a badge/achievement.
type Achievement struct {
	ID          string
	Key         string
	Title       string
	Description string
	Icon        string
}

// PointLogEntry represents a points transaction.
type PointLogEntry struct {
	ID          string
	UserID      string
	Points      int
	Reason      string
	Description string
	EarnedAt    time.Time
}

// Translation represents translated content.
type Translation struct {
	ID          string
	Language    string
	Title       string
	Description string
	Content     string
}

// Report represents a content moderation report.
type Report struct {
	ID         string
	TargetType string
	TargetID   string
	Reason     string
	Reporter   string
	Created    time.Time
}

// SchematicRating holds aggregated rating data.
type SchematicRating struct {
	AvgRating   float64
	RatingCount int
}

// ExternalAuth represents an OAuth provider link.
type ExternalAuth struct {
	ID         string
	UserID     string
	Provider   string
	ProviderID string
	Created    time.Time
}

// SchematicVersion represents a version snapshot.
type SchematicVersion struct {
	ID          string
	SchematicID string
	Version     int
	Snapshot    string
	Note        string
	Created     time.Time
}

// APIKey represents a user API key.
type APIKey struct {
	ID       string
	UserID   string
	KeyHash  string
	Label    string
	Last8    string
	Created  time.Time
}

// ModMetadata represents mod info from Modrinth/CurseForge.
type ModMetadata struct {
	ID            string
	Namespace     string
	DisplayName   string
	Description   string
	IconURL       string
	ModrinthSlug  string
	ModrinthURL   string
	CurseforgeID  string
	CurseforgeURL string
	SourceURL     string
	LastFetched   *time.Time
	ManuallySet   bool
}

// UserStore handles user persistence.
type UserStore interface {
	GetUserByID(ctx context.Context, id string) (*User, error)
	GetUserByEmail(ctx context.Context, email string) (*User, error)
	GetUserByUsername(ctx context.Context, username string) (*User, error)
	CreateUser(ctx context.Context, u *User) error
	UpdateUser(ctx context.Context, u *User) error
	UpdateUserPoints(ctx context.Context, id string, points int) error
	UpdateUserPassword(ctx context.Context, id string, hash string) error
	UpdateUserAvatar(ctx context.Context, id string, avatar string) error
	SoftDeleteUser(ctx context.Context, id string) error
	IsContributor(ctx context.Context, userID string) (bool, error)
	ListUsers(ctx context.Context, limit, offset int) ([]User, error)
	CountUsers(ctx context.Context) (int64, error)
}

// SessionStore handles session persistence.
type SessionStore interface {
	CreateSession(ctx context.Context, s *Session) error
	GetSession(ctx context.Context, id string) (*Session, error)
	DeleteSession(ctx context.Context, id string) error
	DeleteUserSessions(ctx context.Context, userID string) error
	CleanupExpired(ctx context.Context) error
}

// SchematicStore handles schematic persistence.
type SchematicStore interface {
	GetByID(ctx context.Context, id string) (*Schematic, error)
	GetByName(ctx context.Context, name string) (*Schematic, error)
	ListApproved(ctx context.Context, limit, offset int) ([]Schematic, error)
	CountApproved(ctx context.Context) (int64, error)
	ListByAuthor(ctx context.Context, authorID string, limit, offset int) ([]Schematic, error)
	ListByAuthorExcluding(ctx context.Context, authorID, excludeID string, limit int) ([]Schematic, error)
	ListByIDs(ctx context.Context, ids []string) ([]Schematic, error)
	ListFeatured(ctx context.Context, limit int) ([]Schematic, error)
	ListAllForIndex(ctx context.Context) ([]Schematic, error)
	Create(ctx context.Context, s *Schematic) error
	Update(ctx context.Context, s *Schematic) error
	SoftDelete(ctx context.Context, id string) error
	// Relations
	GetCategoryIDs(ctx context.Context, schematicID string) ([]string, error)
	GetTagIDs(ctx context.Context, schematicID string) ([]string, error)
	SetCategories(ctx context.Context, schematicID string, categoryIDs []string) error
	SetTags(ctx context.Context, schematicID string, tagIDs []string) error
}

// CategoryStore handles categories.
type CategoryStore interface {
	List(ctx context.Context) ([]Category, error)
	GetByID(ctx context.Context, id string) (*Category, error)
	GetByIDs(ctx context.Context, ids []string) ([]Category, error)
	Create(ctx context.Context, c *Category) error
}

// TagStore handles tags.
type TagStore interface {
	List(ctx context.Context) ([]Tag, error)
	GetByID(ctx context.Context, id string) (*Tag, error)
	GetByIDs(ctx context.Context, ids []string) ([]Tag, error)
	ListWithCount(ctx context.Context) ([]TagWithCount, error)
	Create(ctx context.Context, t *Tag) error
}

// CommentStore handles comments.
type CommentStore interface {
	GetByID(ctx context.Context, id string) (*Comment, error)
	ListBySchematic(ctx context.Context, schematicID string) ([]Comment, error)
	CountBySchematic(ctx context.Context, schematicID string) (int64, error)
	Create(ctx context.Context, c *Comment) error
	Approve(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
	CountByUser(ctx context.Context, userID string) (int64, error)
}

// GuideStore handles guides.
type GuideStore interface {
	GetByID(ctx context.Context, id string) (*Guide, error)
	GetBySlug(ctx context.Context, slug string) (*Guide, error)
	List(ctx context.Context, limit, offset int) ([]Guide, error)
	Create(ctx context.Context, g *Guide) error
	Update(ctx context.Context, g *Guide) error
	Delete(ctx context.Context, id string) error
	CountByUser(ctx context.Context, userID string) (int64, error)
}

// CollectionStore handles collections.
type CollectionStore interface {
	GetByID(ctx context.Context, id string) (*Collection, error)
	GetBySlug(ctx context.Context, slug string) (*Collection, error)
	List(ctx context.Context, limit, offset int) ([]Collection, error)
	ListByAuthor(ctx context.Context, authorID string) ([]Collection, error)
	ListFeatured(ctx context.Context, limit int) ([]Collection, error)
	Create(ctx context.Context, c *Collection) error
	Update(ctx context.Context, c *Collection) error
	SoftDelete(ctx context.Context, id string) error
	GetSchematicIDs(ctx context.Context, collectionID string) ([]string, error)
	AddSchematic(ctx context.Context, collectionID, schematicID string, position int) error
	RemoveSchematic(ctx context.Context, collectionID, schematicID string) error
	ClearSchematics(ctx context.Context, collectionID string) error
	IncrementViews(ctx context.Context, id string) error
	CountByUser(ctx context.Context, userID string) (int64, error)
}

// AchievementStore handles achievements and points.
type AchievementStore interface {
	GetByKey(ctx context.Context, key string) (*Achievement, error)
	List(ctx context.Context) ([]Achievement, error)
	ListUserAchievements(ctx context.Context, userID string) ([]Achievement, error)
	Award(ctx context.Context, userID, achievementID string) error
	HasAchievement(ctx context.Context, userID, achievementID string) (bool, error)
	CreatePointLog(ctx context.Context, entry *PointLogEntry) error
	GetPointLog(ctx context.Context, userID string) ([]PointLogEntry, error)
	SumUserPoints(ctx context.Context, userID string) (int, error)
}

// TranslationStore handles translations for all content types.
type TranslationStore interface {
	GetSchematicTranslation(ctx context.Context, schematicID, lang string) (*Translation, error)
	ListSchematicTranslations(ctx context.Context, schematicID string) ([]Translation, error)
	UpsertSchematicTranslation(ctx context.Context, schematicID string, t *Translation) error
	ListSchematicsWithoutTranslation(ctx context.Context, lang string, limit int) ([]Schematic, error)
	GetGuideTranslation(ctx context.Context, guideID, lang string) (*Translation, error)
	UpsertGuideTranslation(ctx context.Context, guideID string, t *Translation) error
	GetCollectionTranslation(ctx context.Context, collectionID, lang string) (*Translation, error)
	UpsertCollectionTranslation(ctx context.Context, collectionID string, t *Translation) error
}

// ViewRatingStore handles view counts, downloads, and ratings.
type ViewRatingStore interface {
	GetViewCount(ctx context.Context, schematicID string) (int, error)
	GetDownloadCount(ctx context.Context, schematicID string) (int, error)
	RecordDownload(ctx context.Context, schematicID string, userID *string) error
	GetRating(ctx context.Context, schematicID string) (*SchematicRating, error)
	UpsertRating(ctx context.Context, userID, schematicID string, rating float64) error
}

// VersionStore handles schematic version history.
type VersionStore interface {
	Create(ctx context.Context, v *SchematicVersion) error
	ListBySchematic(ctx context.Context, schematicID string) ([]SchematicVersion, error)
	GetLatestVersion(ctx context.Context, schematicID string) (int, error)
}

// APIKeyStore handles API key management.
type APIKeyStore interface {
	GetByLast8(ctx context.Context, last8 string) (*APIKey, error)
	ListByUser(ctx context.Context, userID string) ([]APIKey, error)
	Create(ctx context.Context, k *APIKey) error
	Delete(ctx context.Context, id, userID string) error
	LogUsage(ctx context.Context, apiKeyID, endpoint string) error
}

// AuthStore handles external auth providers (OAuth).
type AuthStore interface {
	GetByProvider(ctx context.Context, provider, providerID string) (*ExternalAuth, error)
	Create(ctx context.Context, ea *ExternalAuth) error
	ListByUser(ctx context.Context, userID string) ([]ExternalAuth, error)
}

// ReportStore handles content reports.
type ReportStore interface {
	Create(ctx context.Context, r *Report) error
	List(ctx context.Context, limit, offset int) ([]Report, error)
	Delete(ctx context.Context, id string) error
}

// ModMetadataStore handles mod metadata from Modrinth/CurseForge.
type ModMetadataStore interface {
	GetByNamespace(ctx context.Context, namespace string) (*ModMetadata, error)
	Upsert(ctx context.Context, m *ModMetadata) error
	ListStale(ctx context.Context, limit int) ([]ModMetadata, error)
}

// Store aggregates all sub-stores for dependency injection.
type Store struct {
	Users        UserStore
	Sessions     SessionStore
	Schematics   SchematicStore
	Categories   CategoryStore
	Tags         TagStore
	Comments     CommentStore
	Guides       GuideStore
	Collections  CollectionStore
	Achievements AchievementStore
	Translations TranslationStore
	ViewRatings  ViewRatingStore
	Versions     VersionStore
	APIKeys      APIKeyStore
	Auth         AuthStore
	Reports      ReportStore
	ModMetadata  ModMetadataStore
}
