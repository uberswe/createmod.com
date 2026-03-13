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
	ExternalURL        string
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
	ID     string
	Key    string
	Name   string
	Public bool
}

// Tag represents a schematic tag.
type Tag struct {
	ID     string
	Key    string
	Name   string
	Public bool
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
	Excerpt     string
	Content     string
	Slug        string
	VideoURL    string
	WikiURL     string
	UploadLink  string
	BannerURL   string
	Views       int
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
	CollageURL  string
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

// SchematicCategoryInfo holds category info for batch enrichment.
type SchematicCategoryInfo struct {
	ID   string
	Key  string
	Name string
}

// SchematicTagInfo holds tag info for batch enrichment.
type SchematicTagInfo struct {
	ID   string
	Key  string
	Name string
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

// GameVersion represents a Minecraft or Create mod version entry.
type GameVersion struct {
	ID      string
	Version string
	Created time.Time
}

// TrendingData holds pre-fetched engagement signals for computing trending scores.
type TrendingData struct {
	SchematicIDs     []string
	SchematicCreated map[string]time.Time
	RecentViews      map[string]float64 // views in the recent period
	TotalViews       map[string]float64 // all-time views
	RatingSum        map[string]float64
	RatingCount      map[string]float64
	RecentDownloads  map[string]float64 // downloads in the recent period
	TotalDownloads   map[string]float64 // all-time downloads
}

// HourlyStat holds an hourly aggregation row.
type HourlyStat struct {
	Hour  string
	Count int64
}

// MonthlyDataPoint holds monthly stats for a user.
type MonthlyDataPoint struct {
	Month     string
	Uploads   int64
	Downloads int64
	Views     int64
}

// SearchEntry represents a recorded search query.
type SearchEntry struct {
	ID           string
	Query        string
	ResultsCount int
	UserID       string
	IPAddress    string
	Created      time.Time
}

// SitemapSchematic is a lightweight schematic entry for sitemap generation.
type SitemapSchematic struct {
	ID      string
	Name    string
	Updated time.Time
}

// SitemapUser is a lightweight user entry for sitemap generation.
type SitemapUser struct {
	ID       string
	Username string
	Updated  time.Time
}

// SitemapGuide is a lightweight guide entry for sitemap generation.
type SitemapGuide struct {
	ID      string
	Slug    string
	Updated time.Time
}

// ModMetadata represents mod info from Modrinth/CurseForge.
type ModMetadata struct {
	ID                 string
	Namespace          string
	DisplayName        string
	Description        string
	IconURL            string
	ModrinthSlug       string
	ModrinthURL        string
	CurseforgeID       string
	CurseforgeURL      string
	SourceURL          string
	LastFetched        *time.Time
	ManuallySet        bool
	BlocksitemsMatched bool
}

// ModCount represents a mod namespace with its schematic count.
type ModCount struct {
	ModName string
	Count   int
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
	ListForSitemap(ctx context.Context) ([]SitemapUser, error)
	ListAdminEmails(ctx context.Context) ([]string, error)
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
	// Extended queries for migration
	ListApprovedWithVideo(ctx context.Context, limit, offset int) ([]Schematic, error)
	ListRandomApproved(ctx context.Context, limit int) ([]Schematic, error)
	ListByCategoryIDs(ctx context.Context, categoryIDs []string, excludeIDs []string, limit int) ([]Schematic, error)
	ListHighestRated(ctx context.Context, limit, offset int) ([]Schematic, error)
	ListForSitemap(ctx context.Context) ([]SitemapSchematic, error)
	CountByAuthor(ctx context.Context, authorID string) (int64, error)
	GetByChecksum(ctx context.Context, checksum string) (string, error) // returns schematic ID
	UpdateName(ctx context.Context, id, name string) error
	ListByNamePattern(ctx context.Context, pattern string, limit int) ([]Schematic, error)
	// Admin queries
	ListForAdmin(ctx context.Context, filter string, limit, offset int) ([]Schematic, error)
	CountForAdmin(ctx context.Context, filter string) (int64, error)
	GetByIDAdmin(ctx context.Context, id string) (*Schematic, error)
	// Pre-computed aggregates
	UpdateTrendingScore(ctx context.Context, id string, score float64) error
	UpdateRatingAggregates(ctx context.Context, id string, avgRating float64, ratingCount int) error
	RefreshRatingAggregates(ctx context.Context, id string) error
	// Batch enrichment
	BatchGetCategoriesForSchematics(ctx context.Context, ids []string) (map[string][]SchematicCategoryInfo, error)
	BatchGetTagsForSchematics(ctx context.Context, ids []string) (map[string][]SchematicTagInfo, error)
	// Mod-related queries
	ListModCounts(ctx context.Context) ([]ModCount, error)
	CountVanilla(ctx context.Context) (int, error)
	ListByMod(ctx context.Context, mod string, limit, offset int) ([]Schematic, int, error)
	ListVanilla(ctx context.Context, limit, offset int) ([]Schematic, int, error)
	UpdateDetectedLanguage(ctx context.Context, id, lang string) error
}

// CategoryStore handles categories.
type CategoryStore interface {
	List(ctx context.Context) ([]Category, error)
	GetByID(ctx context.Context, id string) (*Category, error)
	GetByIDs(ctx context.Context, ids []string) ([]Category, error)
	Create(ctx context.Context, c *Category) error
	ListAll(ctx context.Context) ([]Category, error)
	ListPending(ctx context.Context) ([]Category, error)
	Approve(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
	GetByKey(ctx context.Context, key string) (*Category, error)
}

// TagStore handles tags.
type TagStore interface {
	List(ctx context.Context) ([]Tag, error)
	GetByID(ctx context.Context, id string) (*Tag, error)
	GetByIDs(ctx context.Context, ids []string) ([]Tag, error)
	ListWithCount(ctx context.Context) ([]TagWithCount, error)
	Create(ctx context.Context, t *Tag) error
	ListAll(ctx context.Context) ([]Tag, error)
	ListPending(ctx context.Context) ([]Tag, error)
	Approve(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
	GetByKey(ctx context.Context, key string) (*Tag, error)
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
	IncrementViews(ctx context.Context, id string) error
	ListForSitemap(ctx context.Context) ([]SitemapGuide, error)
}

// CollectionStore handles collections.
type CollectionStore interface {
	GetByID(ctx context.Context, id string) (*Collection, error)
	GetBySlug(ctx context.Context, slug string) (*Collection, error)
	List(ctx context.Context, limit, offset int) ([]Collection, error)
	ListByAuthor(ctx context.Context, authorID string) ([]Collection, error)
	ListFeatured(ctx context.Context, limit int) ([]Collection, error)
	ListPublished(ctx context.Context, limit, offset int) ([]Collection, error)
	Create(ctx context.Context, c *Collection) error
	Update(ctx context.Context, c *Collection) error
	SoftDelete(ctx context.Context, id string) error
	GetSchematicIDs(ctx context.Context, collectionID string) ([]string, error)
	AddSchematic(ctx context.Context, collectionID, schematicID string, position int) error
	RemoveSchematic(ctx context.Context, collectionID, schematicID string) error
	ClearSchematics(ctx context.Context, collectionID string) error
	IncrementViews(ctx context.Context, id string) error
	CountByUser(ctx context.Context, userID string) (int64, error)
	ListForSitemap(ctx context.Context) ([]SitemapCollection, error)
	UpdateCollageURL(ctx context.Context, id, collageURL string) error
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
	// RecordView upserts view counts for all period types (daily, weekly, monthly, yearly, total).
	RecordView(ctx context.Context, schematicID string) error
	// GetTotalViewCount returns the total (all-time) view count from the type=4 row.
	GetTotalViewCount(ctx context.Context, schematicID string) (int, error)
	// FetchTrendingData returns bulk engagement signals for computing trending scores.
	FetchTrendingData(ctx context.Context, recentDays int) (*TrendingData, error)
	// Batch enrichment
	BatchGetViewCounts(ctx context.Context, ids []string) (map[string]int, error)
	BatchGetDownloadCounts(ctx context.Context, ids []string) (map[string]int, error)
	BatchGetRatings(ctx context.Context, ids []string) (map[string]*SchematicRating, error)
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
	DeleteByTarget(ctx context.Context, targetType, targetID string) (int64, error)
}

// ModMetadataStore handles mod metadata from Modrinth/CurseForge.
type ModMetadataStore interface {
	GetByNamespace(ctx context.Context, namespace string) (*ModMetadata, error)
	Upsert(ctx context.Context, m *ModMetadata) error
	ListAll(ctx context.Context) ([]ModMetadata, error)
	ListStale(ctx context.Context, limit int) ([]ModMetadata, error)
}

// VersionLookupStore handles game version lists.
type VersionLookupStore interface {
	ListMinecraftVersions(ctx context.Context) ([]GameVersion, error)
	ListCreatemodVersions(ctx context.Context) ([]GameVersion, error)
	GetMinecraftVersionByID(ctx context.Context, id string) (*GameVersion, error)
	GetCreatemodVersionByID(ctx context.Context, id string) (*GameVersion, error)
}

// SearchTrackingStore handles search query persistence.
type SearchTrackingStore interface {
	RecordSearch(ctx context.Context, query string, resultsCount int, userID, ip string) error
	ListTopSearches(ctx context.Context, limit int) ([]SearchEntry, error)
}

// OutgoingClickStore handles external link click tracking.
type OutgoingClickStore interface {
	RecordClick(ctx context.Context, url, source, sourceID string, userID *string) error
}

// ContactStore handles contact form submissions.
type ContactStore interface {
	CreateSubmission(ctx context.Context, authorID *string, title, content, name string) error
}

// StatsStore handles aggregation queries for dashboards.
type StatsStore interface {
	HourlyStats(ctx context.Context, table string, cutoff time.Time) ([]HourlyStat, error)
	MonthlyUserStats(ctx context.Context, userID string, months int) ([]MonthlyDataPoint, error)
}

// NBTHash represents a stored NBT file hash for duplicate/blacklist detection.
type NBTHash struct {
	ID          string
	Hash        string
	SchematicID *string
	UploadedBy  *string
	Created     time.Time
}

// NBTHashStore handles NBT hash operations for blacklisting.
type NBTHashStore interface {
	Create(ctx context.Context, h *NBTHash) error
	ListByUser(ctx context.Context, userID string) ([]NBTHash, error)
	Delete(ctx context.Context, id, userID string) error
	IsBlacklisted(ctx context.Context, hash string) (bool, error)
}

// DownloadToken represents a one-time download token.
type DownloadToken struct {
	ID        string
	Token     string
	Name      string // schematic name
	ExpiresAt time.Time
	Used      bool
	Created   time.Time
}

// DownloadTokenStore handles download token persistence.
type DownloadTokenStore interface {
	Create(ctx context.Context, dt *DownloadToken) error
	Consume(ctx context.Context, token string) (*DownloadToken, error) // get + mark used atomically
	CleanupExpired(ctx context.Context) error
}

// SitemapCollection is a lightweight collection entry for sitemap generation.
type SitemapCollection struct {
	ID      string
	Slug    string
	Updated time.Time
}

// UserWebhook represents a user's Discord webhook configuration.
type UserWebhook struct {
	ID                  string
	UserID              string
	WebhookURLEncrypted string
	Active              bool
	ConsecutiveFailures int
	LastFailureAt       *time.Time
	LastFailureMessage  string
	Created             time.Time
	Updated             time.Time
}

// WebhookStore handles user webhook persistence.
type WebhookStore interface {
	Create(ctx context.Context, userID, encryptedURL string) error
	Upsert(ctx context.Context, userID, encryptedURL string) error
	GetByUserID(ctx context.Context, userID string) (*UserWebhook, error)
	UpdateURL(ctx context.Context, userID, encryptedURL string) error
	Delete(ctx context.Context, userID string) error
	ListActive(ctx context.Context) ([]UserWebhook, error)
	IncrementFailure(ctx context.Context, id, message string) error
	ResetFailures(ctx context.Context, id string) error
}

// Store aggregates all sub-stores for dependency injection.
// TempUpload represents a temporarily uploaded schematic awaiting publishing.
type TempUpload struct {
	ID               string
	Token            string
	UploadedBy       string
	Filename         string
	Description      string
	Size             int64
	Checksum         string
	BlockCount       int
	DimX             int
	DimY             int
	DimZ             int
	Mods             json.RawMessage
	Materials        json.RawMessage
	MinecraftVersion string
	CreatemodVersion string
	NbtS3Key         string
	ImageS3Key       string
	ParsedSummary    string
	Processing       bool
	Created          time.Time
	Updated          time.Time
}

// TempUploadFile represents an additional NBT file attached to a temp upload.
type TempUploadFile struct {
	ID          string
	Token       string
	Filename    string
	Description string
	Size        int64
	Checksum    string
	BlockCount  int
	DimX        int
	DimY        int
	DimZ        int
	Mods        json.RawMessage
	Materials   json.RawMessage
	NbtS3Key    string
	Created     time.Time
}

// SchematicFile represents an additional file (variation) attached to a published schematic.
type SchematicFile struct {
	ID           string
	SchematicID  string
	Filename     string
	OriginalName string
	Size         int64
	MimeType     string
	Created      time.Time
	Updated      time.Time
}

// SchematicFileStore manages additional files for published schematics.
type SchematicFileStore interface {
	Create(ctx context.Context, f *SchematicFile) error
	ListBySchematicID(ctx context.Context, schematicID string) ([]SchematicFile, error)
	Delete(ctx context.Context, id string) error
	DeleteBySchematicID(ctx context.Context, schematicID string) error
}

// TempUploadStore manages temporary upload persistence.
type TempUploadStore interface {
	Create(ctx context.Context, t *TempUpload) error
	GetByToken(ctx context.Context, token string) (*TempUpload, error)
	GetByChecksum(ctx context.Context, checksum string) (*TempUpload, error)
	Update(ctx context.Context, t *TempUpload) error
	Claim(ctx context.Context, token string, userID string) error
	MarkProcessing(ctx context.Context, token string) error
	Delete(ctx context.Context, token string) error
	DeleteExpired(ctx context.Context, olderThan time.Time) (int64, error)
	ListByUser(ctx context.Context, userID string, limit int, offset int) ([]TempUpload, error)
	ListExpiredUnclaimed(ctx context.Context, olderThan time.Time, limit int) ([]TempUpload, error)
	DeleteExpiredUnclaimed(ctx context.Context, olderThan time.Time) (int64, error)
}

// TempUploadFileStore manages additional files for temp uploads.
type TempUploadFileStore interface {
	Create(ctx context.Context, f *TempUploadFile) error
	ListByToken(ctx context.Context, token string) ([]TempUploadFile, error)
	GetByID(ctx context.Context, id string) (*TempUploadFile, error)
	Delete(ctx context.Context, id string) error
	DeleteByToken(ctx context.Context, token string) error
}

type Store struct {
	Users           UserStore
	Sessions        SessionStore
	Schematics      SchematicStore
	Categories      CategoryStore
	Tags            TagStore
	Comments        CommentStore
	Guides          GuideStore
	Collections     CollectionStore
	Achievements    AchievementStore
	Translations    TranslationStore
	ViewRatings     ViewRatingStore
	Versions        VersionStore
	APIKeys         APIKeyStore
	Auth            AuthStore
	Reports         ReportStore
	ModMetadata     ModMetadataStore
	VersionLookup   VersionLookupStore
	SearchTracking  SearchTrackingStore
	OutgoingClicks  OutgoingClickStore
	Contact         ContactStore
	Stats           StatsStore
	TempUploads     TempUploadStore
	TempUploadFiles TempUploadFileStore
	NBTHashes       NBTHashStore
	DownloadTokens  DownloadTokenStore
	SchematicFiles  SchematicFileStore
	Webhooks        WebhookStore
}
