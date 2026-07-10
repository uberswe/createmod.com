// Package store defines the persistence interfaces for the application.
// Implementations live in internal/database (PostgreSQL via sqlc).
package store

import (
	"context"
	"encoding/json"
	"time"
)

// Moderation state constants for the schematic state machine.
const (
	ModerationAutoReview = "auto_review"
	ModerationPublished  = "published"
	ModerationFlagged    = "flagged"
	ModerationApproved   = "approved"
	ModerationRejected   = "rejected"
	ModerationDeleted    = "deleted"
)

// IsPublicState returns true if the moderation state means the schematic is publicly visible.
func IsPublicState(state string) bool {
	return state == ModerationPublished || state == ModerationApproved
}

// User represents a user account.
type User struct {
	ID             string
	Email          string
	Username       string
	PasswordHash   string
	OldPassword    string
	Avatar         string
	Points         int
	Verified       bool
	IsAdmin        bool
	Deleted        *time.Time
	Created        time.Time
	Updated        time.Time
	FollowerCount  int
	FollowingCount int
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
	RotationImages     []string
	RotationDisabled   bool
	ShortCode          string
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
	ExternalURL        string
	Featured           bool
	AIDescription      string
	ModerationState    string
	ModerationReason   string
	ScheduledAt        *time.Time
	Deleted            *time.Time
	OldID              *int
	Status             string
	Type               string
	Created            time.Time
	Updated            time.Time
	CreatedOverride    *time.Time // when set, Update overwrites the created timestamp
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
	Deleted        *time.Time
	SchematicName  string
	SchematicTitle string
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
	Video       string
	Featured    bool
	Views       int
	Published   bool
	PublishedAt *time.Time
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
	ReferenceID string
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

// CommentTranslation represents translated comment content.
type CommentTranslation struct {
	ID        string
	CommentID string
	Language  string
	Content   string
	Created   time.Time
	Updated   time.Time
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

// SchematicChange reports a schematic that was updated or removed, derived from
// the schematic_versions history and the deleted timestamp.
type SchematicChange struct {
	Name string
	Kind string // "updated" or "removed"
	At   time.Time
}

// SchematicStat holds the volatile counters for a schematic, fetched
// separately from content so caches can refresh them on a short timer.
type SchematicStat struct {
	Name         string
	Views        int
	Downloads    int
	AvgRating    float64
	RatingCount  int
	CommentCount int
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
	ID                    string
	UserID                string
	Provider              string
	ProviderID            string
	AccessTokenEncrypted  string
	RefreshTokenEncrypted string
	TokenExpiry           *time.Time
	Username              string
	AvatarURL             string
	Metadata              json.RawMessage
	Created               time.Time
}

// SchematicVersion represents a version snapshot.
type SchematicVersion struct {
	ID            string
	SchematicID   string
	Version       int
	Snapshot      string
	Note          string
	SchematicFile string
	Changelog     string
	BlockCount    int
	DimX          int
	DimY          int
	DimZ          int
	Materials     json.RawMessage
	Created       time.Time
}

// Badge represents an earnable badge.
type Badge struct {
	ID          string
	Key         string
	Title       string
	Description string
	Icon        string
	Category    string // "achievement", "supporter", "verified"
	Threshold   int
	MultiEarn   bool
}

// UserBadge represents a badge earned by a user.
type UserBadge struct {
	ID      string
	UserID  string
	BadgeID string
	Count   int
	Badge   Badge
}

// DisplayBadge represents a badge chosen for display on a user profile.
type DisplayBadge struct {
	UserID   string
	BadgeID  string
	Position int
	Badge    Badge
}

// SocialLink represents a user's social platform link.
type SocialLink struct {
	ID       string
	UserID   string
	Platform string
	URL      string
	Username string
	Verified bool
	Created  time.Time
	Updated  time.Time
}

// UserFollow represents a unified follow (user, category, feed, search, mod).
type UserFollow struct {
	ID               string
	UserID           string
	FollowType       string // "user","category","latest","trending","highest_rated","search","mod"
	TargetID         string
	EmailFrequency   string // "realtime","daily","weekly","off"
	UnsubscribeToken string
	LastNotified     *time.Time
	Created          time.Time
}

// SchematicVideo represents a video linked to a schematic.
type SchematicVideo struct {
	ID          string
	SchematicID string
	VideoURL    string
	VideoType   string // "showcase", "tutorial", "timelapse"
	Title       string
	Position    int
	Created     time.Time
}

// SchematicReference represents an "inspired by" reference link.
type SchematicReference struct {
	ID           string
	SchematicID  string
	URL          string
	SourceType   string
	Title        string
	ThumbnailURL string
	AuthorName   string
	LastFetched  *time.Time
	Created      time.Time
	Updated      time.Time
}

// Modpack represents a modpack from Modrinth.
type Modpack struct {
	ID          string
	ModrinthID  string
	Slug        string
	Name        string
	Description string
	IconURL     string
	ModrinthURL string
	Downloads   int
	LastFetched *time.Time
	Created     time.Time
	Updated     time.Time
}

// RedditLink represents a Reddit post linked to a schematic.
type RedditLink struct {
	ID           string
	SchematicID  string
	RedditURL    string
	Subreddit    string
	PostTitle    string
	Upvotes      int
	CommentCount int
	ThumbnailURL string
	LastFetched  *time.Time
	Created      time.Time
	Updated      time.Time
}

// Notification represents a user notification.
type Notification struct {
	ID          string
	UserID      string
	Type        string
	Title       string
	Body        string
	URL         string
	ActorID     string
	ReferenceID string
	Read        bool
	Created     time.Time
}

// NotificationPreference represents user notification preferences per category.
type NotificationPreference struct {
	ID       string
	UserID   string
	Category string
	Email    string // "immediate", "daily", "weekly", "off"
	Web      bool
	Created  time.Time
	Updated  time.Time
}

// NewsletterSubscriber represents an email subscription.
type NewsletterSubscriber struct {
	ID               string
	Email            string
	UserID           *string
	Type             string
	Frequency        string
	Confirmed        bool
	ConfirmToken     string
	UnsubscribeToken string
	Created          time.Time
	Updated          time.Time
}

// NewsletterIssue represents a sent newsletter.
type NewsletterIssue struct {
	ID       string
	Type     string
	Subject  string
	HTMLBody string
	Slug     string
	SentAt   *time.Time
	Created  time.Time
}

// SearchAlert represents a saved search alert.
type SearchAlert struct {
	ID               string
	UserID           string
	Query            string
	Filters          json.RawMessage
	Frequency        string
	LastChecked      *time.Time
	LastNotified     *time.Time
	Active           bool
	UnsubscribeToken string
	Created          time.Time
	Updated          time.Time
}

// ZeroResultSuggestion represents a suggestion for zero-result queries.
type ZeroResultSuggestion struct {
	ID         string
	Query      string
	Suggestion string
	Auto       bool
	Created    time.Time
	Updated    time.Time
}

// KnownIP represents a user's recognized IP address.
type KnownIP struct {
	ID        string
	UserID    string
	IPAddress string
	UserAgent string
	Verified  bool
	LastSeen  time.Time
	Created   time.Time
}

// IPVerificationCode represents a pending IP verification.
type IPVerificationCode struct {
	ID        string
	UserID    string
	IPAddress string
	CodeHash  string
	ExpiresAt time.Time
	Used      bool
	Created   time.Time
}

// UserTOTP represents TOTP configuration for a user.
type UserTOTP struct {
	ID              string
	UserID          string
	SecretEncrypted string
	Enabled         bool
	Verified        bool
	Created         time.Time
	Updated         time.Time
}

// TOTPBackupCode represents a TOTP backup code.
type TOTPBackupCode struct {
	ID       string
	UserID   string
	CodeHash string
	Used     bool
	Created  time.Time
}

// Passkey represents a WebAuthn credential.
type Passkey struct {
	ID              string
	UserID          string
	CredentialID    []byte
	PublicKey       []byte
	AttestationType string
	Transport       []string
	AAGUID          []byte
	SignCount       int
	FriendlyName    string
	LastUsed        *time.Time
	Created         time.Time
}

// SecuritySettings represents per-user security configuration.
type SecuritySettings struct {
	ID                string
	UserID            string
	NewIPVerification bool
	TOTPEnabled       bool
	PasskeysEnabled   bool
	Created           time.Time
	Updated           time.Time
}

// APIKey represents a user API key.
type APIKey struct {
	ID      string
	UserID  string
	KeyHash string
	Label   string
	Last8   string
	Created time.Time
	// RateLimitPerMinute is an admin-assigned override applied to every API
	// endpoint. 0 means "use the endpoint's default limit".
	RateLimitPerMinute int
}

// AdminAPIKey is an API key with owner and usage information for the admin
// overview page. LastUsed is the zero time when the key has never been used.
type AdminAPIKey struct {
	APIKey
	Username   string
	UsageTotal int64
	Usage24h   int64
	Usage7d    int64
	LastUsed   time.Time
}

// APIKeyEndpointUsage is a per-endpoint request count for one API key.
type APIKeyEndpointUsage struct {
	APIKeyID string
	Endpoint string
	Requests int64
	LastUsed time.Time
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

// Event type constants for schematic_events.
const (
	EventVideoPlay    = 1
	EventYouTubeClick = 2
	EventTimeOnPage   = 3
	EventBlockModify  = 4
	EventLayerViewer  = 5
)

// SchematicStatsSummary holds per-schematic summary stats for the user dashboard.
type SchematicStatsSummary struct {
	ID            string
	Name          string
	Title         string
	FeaturedImage string
	Views         int
	Downloads     int
	Created       time.Time
}

// MonthlyDataPoint holds monthly stats for a user.
type MonthlyDataPoint struct {
	Month     string
	Uploads   int64
	Downloads int64
	Views     int64
}

// ZeroResultQuery represents a search query that returned zero results.
type ZeroResultQuery struct {
	Query           string
	ZeroResultCount int64
}

// SearchEntry represents a recorded search query.
type SearchEntry struct {
	ID              string
	Query           string
	ResultsCount    int
	ZeroResultCount int
	UserID          string
	IPAddress       string
	Created         time.Time
}

// CachedTwitchStream holds stream data stored in cache by the job worker and read by the page handler.
type CachedTwitchStream struct {
	UserName     string
	UserLogin    string
	Title        string
	ViewerCount  int
	ThumbnailURL string
}

// TopViewedSchematic holds a schematic's identity and total views for ranking.
type TopViewedSchematic struct {
	ID            string
	Name          string
	Title         string
	FeaturedImage string
	TotalViews    int64
}

// DailyCount holds a day label and an aggregate count.
type DailyCount struct {
	Day   string
	Count int64
}

// SearchTermDailyCount holds a per-term daily count.
type SearchTermDailyCount struct {
	Query string
	Day   string
	Count int64
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
	RestoreUser(ctx context.Context, id string) error
	CascadeSoftDelete(ctx context.Context, id string) error
	CascadeRestore(ctx context.Context, id string) error
	IsContributor(ctx context.Context, userID string) (bool, error)
	ListUsers(ctx context.Context, limit, offset int) ([]User, error)
	CountUsers(ctx context.Context) (int64, error)
	ListForSitemap(ctx context.Context) ([]SitemapUser, error)
	ListAdminEmails(ctx context.Context) ([]string, error)
	// Admin queries
	GetByIDIncludingDeleted(ctx context.Context, id string) (*User, error)
	ListForAdmin(ctx context.Context, filter, search string, limit, offset int) ([]User, error)
	CountForAdmin(ctx context.Context, filter, search string) (int64, error)
	ListTopByPoints(ctx context.Context, limit, offset int) ([]User, error)
	CountActive(ctx context.Context) (int64, error)
	GetUserPointsRank(ctx context.Context, userID string) (int64, error)
}

// SessionStore handles session persistence.
type SessionStore interface {
	CreateSession(ctx context.Context, s *Session) error
	GetSession(ctx context.Context, id string) (*Session, error)
	DeleteSession(ctx context.Context, id string) error
	DeleteUserSessions(ctx context.Context, userID string) error
	CleanupExpired(ctx context.Context) error
}

// ModerationThread represents a moderation discussion thread.
type ModerationThread struct {
	ID          string
	ContentType string
	ContentID   string
	Status      string
	Created     time.Time
	Updated     time.Time
}

// ModerationMessage represents a single message in a moderation thread.
type ModerationMessage struct {
	ID          string
	ThreadID    string
	AuthorID    string
	IsModerator bool
	Body        string
	Created     time.Time
}

// ModerationChatStore handles moderation discussion threads and messages.
type ModerationChatStore interface {
	GetThreadByContent(ctx context.Context, contentType, contentID string) (*ModerationThread, error)
	CreateThread(ctx context.Context, contentType, contentID string) (*ModerationThread, error)
	ListMessages(ctx context.Context, threadID string) ([]ModerationMessage, error)
	CreateMessage(ctx context.Context, threadID, authorID string, isModerator bool, body string) (*ModerationMessage, error)
	CountUserMessagesSinceLastModerator(ctx context.Context, threadID string) (int, error)
	MarkResolved(ctx context.Context, threadID, resolvedBy string) error
	MarkCreatorRead(ctx context.Context, threadID string) error
	ListUnreadByCreator(ctx context.Context, userID string) ([]ModerationThread, error)
	ListByModerator(ctx context.Context, limit, offset int) ([]ModerationThread, error)
}

// SchematicStore handles schematic persistence.
type SchematicStore interface {
	GetByID(ctx context.Context, id string) (*Schematic, error)
	GetByName(ctx context.Context, name string) (*Schematic, error)
	ChangesSince(ctx context.Context, sinceAt time.Time, sinceName, sinceKind string, limit int) ([]SchematicChange, error)
	StatsByNames(ctx context.Context, names []string) ([]SchematicStat, error)
	GetByShortCode(ctx context.Context, code string) (*Schematic, error)
	SetShortCode(ctx context.Context, id, code string) error
	ShortCodeExists(ctx context.Context, code string) (bool, error)
	NameExists(ctx context.Context, name string) (bool, error)
	ListApproved(ctx context.Context, limit, offset int) ([]Schematic, error)
	CountApproved(ctx context.Context) (int64, error)
	ListByAuthor(ctx context.Context, authorID string, limit, offset int) ([]Schematic, error)
	ListByAuthorExcluding(ctx context.Context, authorID, excludeID string, limit int) ([]Schematic, error)
	ListByIDs(ctx context.Context, ids []string) ([]Schematic, error)
	ListFeatured(ctx context.Context, limit int) ([]Schematic, error)
	ListAllForIndex(ctx context.Context) ([]Schematic, error)
	Create(ctx context.Context, s *Schematic) error
	Update(ctx context.Context, s *Schematic) error
	SetModerationState(ctx context.Context, id, state, reason string) error
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
	CountSoftDeletedByAuthor(ctx context.Context, authorID string) (int64, error)
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
	BatchUpdateTrendingScores(ctx context.Context, ids []string, scores []float64) error
	BatchUpdateRatingAggregates(ctx context.Context, ids []string, avgRatings []float64, ratingCounts []int) error
	RefreshRatingAggregates(ctx context.Context, id string) error
	// Batch enrichment
	BatchGetCategoriesForSchematics(ctx context.Context, ids []string) (map[string][]SchematicCategoryInfo, error)
	BatchGetTagsForSchematics(ctx context.Context, ids []string) (map[string][]SchematicTagInfo, error)
	// Mod-related queries
	ListModCounts(ctx context.Context) ([]ModCount, error)
	CountVanilla(ctx context.Context) (int, error)
	ListByMod(ctx context.Context, mod string, limit, offset int) ([]Schematic, int, error)
	CountByMod(ctx context.Context, mod string) (int, error)
	ListByModPaginated(ctx context.Context, mod string, limit, offset int) ([]Schematic, error)
	ListVanilla(ctx context.Context, limit, offset int) ([]Schematic, int, error)
	UpdateDetectedLanguage(ctx context.Context, id, lang string) error
	// ListByAuthorAll returns all schematics by an author regardless of moderation state (except deleted).
	ListByAuthorAll(ctx context.Context, authorID string, limit, offset int) ([]Schematic, error)
	// CountByAuthorAll counts all schematics by an author regardless of moderation state (except deleted).
	CountByAuthorAll(ctx context.Context, authorID string) (int64, error)
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
	Disapprove(ctx context.Context, id string) error
	Delete(ctx context.Context, id string) error
	Restore(ctx context.Context, id string) error
	CountByUser(ctx context.Context, userID string) (int64, error)
	// Admin queries
	ListForAdmin(ctx context.Context, filter, search string, limit, offset int) ([]Comment, error)
	CountForAdmin(ctx context.Context, filter, search string) (int64, error)
}

// GuideStore handles guides.
type GuideStore interface {
	GetByID(ctx context.Context, id string) (*Guide, error)
	GetBySlug(ctx context.Context, slug string) (*Guide, error)
	List(ctx context.Context, limit, offset int) ([]Guide, error)
	ListCreatedSince(ctx context.Context, since, until time.Time) ([]Guide, error)
	Create(ctx context.Context, g *Guide) error
	Update(ctx context.Context, g *Guide) error
	Delete(ctx context.Context, id string) error
	CountByUser(ctx context.Context, userID string) (int64, error)
	IncrementViews(ctx context.Context, id string) error
	ListForSitemap(ctx context.Context) ([]SitemapGuide, error)
	ListForAdmin(ctx context.Context, filter string, limit, offset int) ([]Guide, error)
	CountForAdmin(ctx context.Context, filter string) (int64, error)
	GetByIDAdmin(ctx context.Context, id string) (*Guide, error)
	SoftDelete(ctx context.Context, id string) error
}

// CollectionStore handles collections.
type CollectionStore interface {
	GetByID(ctx context.Context, id string) (*Collection, error)
	GetBySlug(ctx context.Context, slug string) (*Collection, error)
	List(ctx context.Context, limit, offset int) ([]Collection, error)
	ListByAuthor(ctx context.Context, authorID string) ([]Collection, error)
	ListFeatured(ctx context.Context, limit int) ([]Collection, error)
	ListPublished(ctx context.Context, limit, offset int) ([]Collection, error)
	ListPublishedSince(ctx context.Context, since, until time.Time) ([]Collection, error)
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
	ListForAdmin(ctx context.Context, filter string, limit, offset int) ([]Collection, error)
	CountForAdmin(ctx context.Context, filter string) (int64, error)
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
	CountPointLogByReason(ctx context.Context, userID, reason string) (int, error)
}

// BadgeStore handles badge management and awards.
type BadgeStore interface {
	GetByKey(ctx context.Context, key string) (*Badge, error)
	List(ctx context.Context) ([]Badge, error)
	ListUserBadges(ctx context.Context, userID string) ([]UserBadge, error)
	AwardBadge(ctx context.Context, userID, badgeID string) error
	IncrementBadge(ctx context.Context, userID, badgeID string) error
	RemoveBadge(ctx context.Context, userID, badgeID string) error
	SetDisplayedBadges(ctx context.Context, userID string, badgeIDs []string) error
	GetDisplayedBadges(ctx context.Context, userID string) ([]DisplayBadge, error)
	BatchGetDisplayedBadges(ctx context.Context, userIDs []string) (map[string][]DisplayBadge, error)
}

// SocialLinkStore handles user social platform links.
type SocialLinkStore interface {
	Upsert(ctx context.Context, link *SocialLink) error
	ListByUser(ctx context.Context, userID string) ([]SocialLink, error)
	GetByUserAndPlatform(ctx context.Context, userID, platform string) (*SocialLink, error)
	Delete(ctx context.Context, userID, platform string) error
	ListByPlatform(ctx context.Context, platform string) ([]SocialLink, error)
}

// FollowStore handles unified follow relationships (users, categories, feeds, searches, mods).
type FollowStore interface {
	Follow(ctx context.Context, userID, followType, targetID, emailFrequency string) error
	Unfollow(ctx context.Context, userID, followType, targetID string) error
	UpdateFrequency(ctx context.Context, userID, followType, targetID, emailFrequency string) error
	IsFollowing(ctx context.Context, userID, followType, targetID string) (bool, error)
	GetFollow(ctx context.Context, userID, followType, targetID string) (*UserFollow, error)
	ListByUser(ctx context.Context, userID string) ([]UserFollow, error)
	ListByUserAndType(ctx context.Context, userID, followType string) ([]UserFollow, error)
	ListByTarget(ctx context.Context, followType, targetID string) ([]UserFollow, error)
	ListByFrequency(ctx context.Context, emailFrequency string) ([]UserFollow, error)
	Unsubscribe(ctx context.Context, unsubscribeToken string) error
	UpdateLastNotified(ctx context.Context, id string) error
	// User-specific helpers for profile follower/following counts and lists.
	ListFollowerUsers(ctx context.Context, userID string, limit, offset int) ([]User, error)
	ListFollowingUsers(ctx context.Context, userID string, limit, offset int) ([]User, error)
	CountFollowers(ctx context.Context, userID string) (int, error)
	ListFollowedUserIDs(ctx context.Context, userID string) ([]string, error)
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
	GetCommentTranslation(ctx context.Context, commentID, lang string) (*CommentTranslation, error)
	UpsertCommentTranslation(ctx context.Context, commentID string, t *CommentTranslation) error
	ListCommentsWithoutTranslation(ctx context.Context, lang string, limit int) ([]Comment, error)
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
	GetByID(ctx context.Context, id string) (*SchematicVersion, error)
	GetBySchematicAndVersion(ctx context.Context, schematicID string, version int) (*SchematicVersion, error)
}

// APIKeyStore handles API key management.
type APIKeyStore interface {
	GetByLast8(ctx context.Context, last8 string) (*APIKey, error)
	ListByUser(ctx context.Context, userID string) ([]APIKey, error)
	Create(ctx context.Context, k *APIKey) error
	Delete(ctx context.Context, id, userID string) error
	LogUsage(ctx context.Context, apiKeyID, endpoint string) error
	// ListAllWithUsage returns every API key with owner username and usage
	// aggregates, newest first. For the admin overview page.
	ListAllWithUsage(ctx context.Context) ([]AdminAPIKey, error)
	// SetRateLimit sets the per-minute rate limit override for a key
	// (0 clears the override so endpoint defaults apply).
	SetRateLimit(ctx context.Context, id string, perMinute int) error
	// UsageByEndpoint returns per-endpoint request counts for all keys over
	// the last 30 days.
	UsageByEndpoint(ctx context.Context) ([]APIKeyEndpointUsage, error)
}

// AuthStore handles external auth providers (OAuth).
type AuthStore interface {
	GetByProvider(ctx context.Context, provider, providerID string) (*ExternalAuth, error)
	Create(ctx context.Context, ea *ExternalAuth) error
	ListByUser(ctx context.Context, userID string) ([]ExternalAuth, error)
	DeleteByProvider(ctx context.Context, userID, provider string) error
	GetByUserAndProvider(ctx context.Context, userID, provider string) (*ExternalAuth, error)
	ListByProvider(ctx context.Context, provider string) ([]ExternalAuth, error)
}

// ReportStore handles content reports.
type ReportStore interface {
	Create(ctx context.Context, r *Report) error
	List(ctx context.Context, limit, offset int) ([]Report, error)
	ListSince(ctx context.Context, since, until time.Time) ([]Report, error)
	Delete(ctx context.Context, id string) error
	DeleteByTarget(ctx context.Context, targetType, targetID string) (int64, error)
	CountUnresolvedCommentReportsByAuthor(ctx context.Context, authorID string) (int64, error)
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
	RecordSearchClick(ctx context.Context, query, resultID string, position int, userID, ip string) error
	RecordSearchConversion(ctx context.Context, query, schematicID, userID, ip string) error
	ListTopSearches(ctx context.Context, limit int) ([]SearchEntry, error)
	RefreshSearchQueryCounts(ctx context.Context) error
	PruneOldSearches(ctx context.Context) (int64, error)
	HasRecentApprovedUpload(ctx context.Context, userID string, since time.Time) (bool, error)
	ListTopSearchesSince(ctx context.Context, since time.Time, limit int) ([]SearchEntry, error)
	ListTopViewedSchematicsSince(ctx context.Context, since time.Time, limit int) ([]TopViewedSchematic, error)
	DailySearchVolume(ctx context.Context, since time.Time) ([]DailyCount, error)
	DailySearchTermVolume(ctx context.Context, since time.Time, terms []string) ([]SearchTermDailyCount, error)
	UpsertSearchTermModeration(ctx context.Context, query string, isClean bool) error
	ListCleanSearchTerms(ctx context.Context, terms []string) ([]string, error)
	ListDirtySearchTerms(ctx context.Context, terms []string) ([]string, error)
	ListUncheckedSearchTerms(ctx context.Context, terms []string, since time.Time) ([]string, error)
	ListTopZeroResultQueries(ctx context.Context, limit int) ([]ZeroResultQuery, error)
	ListTopSuccessfulQueries(ctx context.Context, limit int) ([]string, error)
}

// OutgoingClickStore handles external link click tracking.
type OutgoingClickStore interface {
	RecordClick(ctx context.Context, url, source, sourceID string, userID *string) error
}

// AdClickStat represents a click count for an ad unit in a given period.
type AdClickStat struct {
	AdUnit string
	Dest   string
	Period string
	Count  int64
}

// AdClickStore tracks clicks on ad units (NitroAds).
type AdClickStore interface {
	RecordClick(ctx context.Context, adUnit, dest string) error
	ListDaily(ctx context.Context) ([]AdClickStat, error)
	ListMonthly(ctx context.Context) ([]AdClickStat, error)
	RollupAndClean(ctx context.Context, cutoffDay string) error
}

// ContactStore handles contact form submissions.
type ContactStore interface {
	CreateSubmission(ctx context.Context, authorID *string, title, content, name string) error
}

// StatsStore handles aggregation queries for dashboards.
type StatsStore interface {
	HourlyStats(ctx context.Context, table string, cutoff time.Time) ([]HourlyStat, error)
	MonthlyUserStats(ctx context.Context, userID string, months int) ([]MonthlyDataPoint, error)
	RecordEvent(ctx context.Context, schematicID string, eventType int, value int) error
	HourlySchematicViews(ctx context.Context, schematicID string, since time.Time) ([]HourlyStat, error)
	HourlySchematicDownloads(ctx context.Context, schematicID string, since time.Time) ([]HourlyStat, error)
	HourlySchematicEvents(ctx context.Context, schematicID string, eventType int, since time.Time) ([]HourlyStat, error)
	HourlySchematicEventAvg(ctx context.Context, schematicID string, eventType int, since time.Time) ([]HourlyStat, error)
	HourlyUserViews(ctx context.Context, userID string, since time.Time) ([]HourlyStat, error)
	HourlyUserDownloads(ctx context.Context, userID string, since time.Time) ([]HourlyStat, error)
	HourlyUserEvents(ctx context.Context, userID string, eventType int, since time.Time) ([]HourlyStat, error)
	HourlyUserEventAvg(ctx context.Context, userID string, eventType int, since time.Time) ([]HourlyStat, error)
	ListSchematicStats(ctx context.Context, userID string, limit, offset int) ([]SchematicStatsSummary, error)
	CountUserSchematics(ctx context.Context, userID string) (int, error)
	GetSiteAvgVDRatio(ctx context.Context) (float64, error)
	GetSiteAvgVDRatioSinceCutoff(ctx context.Context, since time.Time) (float64, error)
	GetSchematicVDRatioSinceCutoff(ctx context.Context, schematicID string, since time.Time) (views int64, downloads int64, err error)
	GetUserVDRatioSinceCutoff(ctx context.Context, userID string, since time.Time) (views int64, downloads int64, err error)
	DeleteOldEvents(ctx context.Context, before time.Time) (int64, error)
	DailySchematicUploads(ctx context.Context, since time.Time) ([]DailyCount, error)
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
// SchematicSafety is one schematic's content-safety scan result.
type SchematicSafety struct {
	SchematicID     string
	Checksum        string
	FileSafe        bool
	Manifest        []byte // JSON schematic.Manifest
	PipelineVersion int
	ScannedAt       time.Time
}

// EditorSession is a server-authoritative schematic editing session: a
// source reference plus an operation log with an undo cursor.
type EditorSession struct {
	ID         string
	UserID     string
	SourceKind string // "schematic", "upload" or "blank"
	SourceRef  string
	Ops        []byte // JSON []schematic.Op
	Cursor     int
	Created    time.Time
	Updated    time.Time
}

// EditorSessionStore persists editor sessions.
type EditorSessionStore interface {
	Create(ctx context.Context, userID, sourceKind, sourceRef string) (string, error)
	GetByID(ctx context.Context, id string) (*EditorSession, error)
	UpdateOps(ctx context.Context, id string, ops []byte, cursor int) error
	DeleteExpired(ctx context.Context, before time.Time) (int64, error)
}

// SchematicFingerprint is one schematic's stored similarity fingerprint.
type SchematicFingerprint struct {
	SchematicID string
	FP          []byte // JSON schematic.Fingerprint
	Version     int
	ComputedAt  time.Time
}

// SchematicFingerprintStore persists similarity fingerprints.
type SchematicFingerprintStore interface {
	Upsert(ctx context.Context, f *SchematicFingerprint) error
	GetBySchematicID(ctx context.Context, schematicID string) (*SchematicFingerprint, error)
	ListNeedingCompute(ctx context.Context, version int, limit int) ([]string, error)
	ListAll(ctx context.Context, version int) ([]SchematicFingerprint, error)
	Delete(ctx context.Context, schematicID string) error
}

// SchematicSafetyStore persists content-safety scan results.
type SchematicSafetyStore interface {
	Upsert(ctx context.Context, s *SchematicSafety) error
	GetBySchematicID(ctx context.Context, schematicID string) (*SchematicSafety, error)
	ListNeedingScan(ctx context.Context, pipelineVersion int, limit int) ([]string, error)
	Delete(ctx context.Context, schematicID string) error
}

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
	UseCount  int
	Created   time.Time
}

// DownloadTokenStore handles download token persistence.
type DownloadTokenStore interface {
	Create(ctx context.Context, dt *DownloadToken) error
	GetByID(ctx context.Context, id string) (*DownloadToken, error)
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
	Description  string
	Size         int64
	MimeType     string
	Created      time.Time
	Updated      time.Time
}

// SchematicFileStore manages additional files for published schematics.
type SchematicFileStore interface {
	Create(ctx context.Context, f *SchematicFile) error
	GetByID(ctx context.Context, id string) (*SchematicFile, error)
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
	ListAll(ctx context.Context, limit int, offset int) ([]TempUpload, error)
	CountAll(ctx context.Context) (int64, error)
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

// TempUploadImage represents an image file attached to a temp upload.
type TempUploadImage struct {
	ID        string
	Token     string
	Filename  string
	Size      int64
	S3Key     string
	SortOrder int
	Category  string
	Created   time.Time
}

// TempUploadImageStore manages images attached to temp uploads.
type TempUploadImageStore interface {
	Create(ctx context.Context, img *TempUploadImage) error
	ListByToken(ctx context.Context, token string) ([]TempUploadImage, error)
	ListByTokenAndCategory(ctx context.Context, token, category string) ([]TempUploadImage, error)
	Delete(ctx context.Context, id string) error
	DeleteByToken(ctx context.Context, token string) error
	CountByToken(ctx context.Context, token string) (int, error)
}

// SchematicVariation represents a saved block replacement configuration for a schematic.
type SchematicVariation struct {
	ID           string
	SchematicID  string
	UserID       string
	Name         string
	Replacements json.RawMessage // JSON array of {original, replacement}
	IsPublic     bool
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

// SchematicVariationStore handles schematic variation persistence.
type SchematicVariationStore interface {
	Create(ctx context.Context, v *SchematicVariation) error
	GetByID(ctx context.Context, id string) (*SchematicVariation, error)
	ListBySchematicAndUser(ctx context.Context, schematicID, userID string) ([]*SchematicVariation, error)
	ListPublicBySchematic(ctx context.Context, schematicID string) ([]*SchematicVariation, error)
	Update(ctx context.Context, v *SchematicVariation) error
	Delete(ctx context.Context, id string) error
	CountBySchematicAndUser(ctx context.Context, schematicID, userID string) (int, error)
	GetOldestBySchematicAndUser(ctx context.Context, schematicID, userID string) (*SchematicVariation, error)
}

// ModerationLogEntry represents a single moderation action on a schematic.
type ModerationLogEntry struct {
	ID            string
	SchematicID   string
	ActorID       string
	ActorType     string // "admin", "system", "user"
	ActorUsername string
	Action        string // "state_change", "upload", "soft_delete"
	OldState      string
	NewState      string
	Reason        string
	CreatedAt     time.Time
}

// AutoApprovedSchematic is a schematic auto-approved by the system within a
// time window, sourced from the moderation log.
type AutoApprovedSchematic struct {
	SchematicID string
	Title       string
	Name        string
	ApprovedAt  time.Time
}

// ModerationLogStore handles moderation audit log persistence.
type ModerationLogStore interface {
	Create(ctx context.Context, entry *ModerationLogEntry) error
	ListBySchematic(ctx context.Context, schematicID string) ([]ModerationLogEntry, error)
	ListAutoApprovedSince(ctx context.Context, since, until time.Time) ([]AutoApprovedSchematic, error)
}

// SchematicVideoStore handles videos linked to schematics.
type SchematicVideoStore interface {
	Create(ctx context.Context, v *SchematicVideo) error
	ListBySchematic(ctx context.Context, schematicID string) ([]SchematicVideo, error)
	Delete(ctx context.Context, id, schematicID string) error
	DeleteBySchematic(ctx context.Context, schematicID string) error
	UpdatePosition(ctx context.Context, id string, position int) error
	BatchGetBySchematicIDs(ctx context.Context, ids []string) (map[string][]SchematicVideo, error)
}

// ReferenceStore handles "inspired by" build references.
type ReferenceStore interface {
	Create(ctx context.Context, ref *SchematicReference) error
	ListBySchematic(ctx context.Context, schematicID string) ([]SchematicReference, error)
	Delete(ctx context.Context, id, schematicID string) error
	ListStale(ctx context.Context, limit int) ([]SchematicReference, error)
	UpdateMetadata(ctx context.Context, id, title, thumbnailURL, authorName string) error
}

// ModpackStore handles modpack management and associations.
type ModpackStore interface {
	Upsert(ctx context.Context, m *Modpack) error
	GetByID(ctx context.Context, id string) (*Modpack, error)
	GetBySlug(ctx context.Context, slug string) (*Modpack, error)
	List(ctx context.Context) ([]Modpack, error)
	Search(ctx context.Context, query string, limit int) ([]Modpack, error)
	SetSchematicModpacks(ctx context.Context, schematicID string, modpackIDs []string) error
	GetSchematicModpacks(ctx context.Context, schematicID string) ([]Modpack, error)
	BatchGetSchematicModpacks(ctx context.Context, schematicIDs []string) (map[string][]Modpack, error)
	ListSchematicsByModpack(ctx context.Context, modpackID string, limit, offset int) ([]Schematic, error)
	CountSchematicsByModpack(ctx context.Context, modpackID string) (int64, error)
}

// RedditLinkStore handles Reddit post links on schematics.
type RedditLinkStore interface {
	Create(ctx context.Context, link *RedditLink) error
	GetBySchematic(ctx context.Context, schematicID string) ([]RedditLink, error)
	Delete(ctx context.Context, id, schematicID string) error
	ListStale(ctx context.Context, limit int) ([]RedditLink, error)
	UpdateMetadata(ctx context.Context, id, postTitle string, upvotes, commentCount int, thumbnailURL string) error
}

// NotificationStore handles user notifications.
type NotificationStore interface {
	Create(ctx context.Context, n *Notification) error
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]Notification, error)
	ListRecent(ctx context.Context, userID string, limit int) ([]Notification, error)
	CountUnread(ctx context.Context, userID string) (int, error)
	MarkRead(ctx context.Context, id, userID string) error
	MarkAllRead(ctx context.Context, userID string) error
	DeleteOld(ctx context.Context, before time.Time) error
	GetPreferences(ctx context.Context, userID string) ([]NotificationPreference, error)
	UpsertPreference(ctx context.Context, pref *NotificationPreference) error
	GetPreference(ctx context.Context, userID, category string) (*NotificationPreference, error)
	ListUsersWithDigestPreference(ctx context.Context, frequency string) ([]string, error)
	ListUnreadSince(ctx context.Context, userID string, since time.Time) ([]Notification, error)
}

// NewsletterStore handles newsletter subscriptions and issues.
type NewsletterStore interface {
	Subscribe(ctx context.Context, sub *NewsletterSubscriber) error
	Confirm(ctx context.Context, confirmToken string) error
	Unsubscribe(ctx context.Context, unsubscribeToken string) error
	ListConfirmed(ctx context.Context, subType string) ([]NewsletterSubscriber, error)
	ListConfirmedByFrequency(ctx context.Context, subType, frequency string) ([]NewsletterSubscriber, error)
	GetByEmail(ctx context.Context, email, subType string) (*NewsletterSubscriber, error)
	CreateIssue(ctx context.Context, issue *NewsletterIssue) error
	GetIssueBySlug(ctx context.Context, slug string) (*NewsletterIssue, error)
	ListIssues(ctx context.Context, issueType string, limit, offset int) ([]NewsletterIssue, error)
	UpdateIssueSentAt(ctx context.Context, id string) error
}

// SearchAlertStore handles saved search alerts.
type SearchAlertStore interface {
	Create(ctx context.Context, alert *SearchAlert) error
	ListByUser(ctx context.Context, userID string) ([]SearchAlert, error)
	ListActive(ctx context.Context, limit int) ([]SearchAlert, error)
	Delete(ctx context.Context, id, userID string) error
	Unsubscribe(ctx context.Context, unsubscribeToken string) error
	UpdateLastChecked(ctx context.Context, id string) error
	UpdateLastNotified(ctx context.Context, id string) error
}

// ZeroResultStore handles zero-result search suggestions.
type ZeroResultStore interface {
	Upsert(ctx context.Context, s *ZeroResultSuggestion) error
	Get(ctx context.Context, query string) (*ZeroResultSuggestion, error)
	List(ctx context.Context, limit, offset int) ([]ZeroResultSuggestion, error)
	Delete(ctx context.Context, id string) error
}

// SecurityStore handles account security features.
type SecurityStore interface {
	UpsertKnownIP(ctx context.Context, ip *KnownIP) error
	GetKnownIP(ctx context.Context, userID, ipAddress string) (*KnownIP, error)
	ListKnownIPs(ctx context.Context, userID string) ([]KnownIP, error)
	VerifyKnownIP(ctx context.Context, userID, ipAddress string) error
	DeleteKnownIP(ctx context.Context, id, userID string) error
	CreateIPVerificationCode(ctx context.Context, code *IPVerificationCode) error
	GetIPVerificationCode(ctx context.Context, userID, ipAddress string) (*IPVerificationCode, error)
	MarkIPVerificationCodeUsed(ctx context.Context, id string) error
	CleanupExpiredIPCodes(ctx context.Context) error
	UpsertTOTP(ctx context.Context, totp *UserTOTP) error
	GetTOTP(ctx context.Context, userID string) (*UserTOTP, error)
	EnableTOTP(ctx context.Context, userID string) error
	DisableTOTP(ctx context.Context, userID string) error
	DeleteTOTP(ctx context.Context, userID string) error
	CreateTOTPBackupCode(ctx context.Context, userID, codeHash string) error
	ListTOTPBackupCodes(ctx context.Context, userID string) ([]TOTPBackupCode, error)
	MarkBackupCodeUsed(ctx context.Context, id string) error
	DeleteTOTPBackupCodes(ctx context.Context, userID string) error
	CreatePasskey(ctx context.Context, pk *Passkey) error
	GetPasskeyByCredentialID(ctx context.Context, credentialID []byte) (*Passkey, error)
	ListPasskeys(ctx context.Context, userID string) ([]Passkey, error)
	UpdatePasskeySignCount(ctx context.Context, id string, signCount int) error
	DeletePasskey(ctx context.Context, id, userID string) error
	UpsertSecuritySettings(ctx context.Context, settings *SecuritySettings) error
	GetSecuritySettings(ctx context.Context, userID string) (*SecuritySettings, error)
}

// ModSecret is a shared HMAC secret for the mod / partner API, managed in the
// database via the admin area. The secret value is stored in plaintext because
// HMAC verification needs the raw value (and the values are treated as public).
type ModSecret struct {
	ID        string
	Label     string // short name
	Note      string // free-text description of what it's used for
	Secret    string // the shared secret value
	Active    bool
	CreatedBy string
	Created   time.Time
}

// ModSecretStore manages admin-defined HMAC shared secrets.
type ModSecretStore interface {
	// ListActive returns the secret values of all active entries (hot path).
	ListActive(ctx context.Context) ([]string, error)
	// List returns all entries (for the admin UI), newest first.
	List(ctx context.Context) ([]ModSecret, error)
	Create(ctx context.Context, s *ModSecret) error
	SetActive(ctx context.Context, id string, active bool) error
	Delete(ctx context.Context, id string) error
}

type Store struct {
	Users               UserStore
	Sessions            SessionStore
	Schematics          SchematicStore
	Categories          CategoryStore
	Tags                TagStore
	Comments            CommentStore
	Guides              GuideStore
	Collections         CollectionStore
	Achievements        AchievementStore
	Translations        TranslationStore
	ViewRatings         ViewRatingStore
	Versions            VersionStore
	APIKeys             APIKeyStore
	Auth                AuthStore
	Reports             ReportStore
	ModMetadata         ModMetadataStore
	VersionLookup       VersionLookupStore
	SearchTracking      SearchTrackingStore
	OutgoingClicks      OutgoingClickStore
	Contact             ContactStore
	Stats               StatsStore
	TempUploads         TempUploadStore
	TempUploadFiles     TempUploadFileStore
	TempUploadImages    TempUploadImageStore
	NBTHashes           NBTHashStore
	SchematicSafety     SchematicSafetyStore
	Fingerprints        SchematicFingerprintStore
	EditorSessions      EditorSessionStore
	DownloadTokens      DownloadTokenStore
	SchematicFiles      SchematicFileStore
	Webhooks            WebhookStore
	SchematicVariations SchematicVariationStore
	ModerationChats     ModerationChatStore
	ModerationLog       ModerationLogStore
	Badges              BadgeStore
	SocialLinks         SocialLinkStore
	Follows             FollowStore
	SchematicVideos     SchematicVideoStore
	References          ReferenceStore
	Modpacks            ModpackStore
	RedditLinks         RedditLinkStore
	Notifications       NotificationStore
	Newsletters         NewsletterStore
	SearchAlerts        SearchAlertStore
	ZeroResults         ZeroResultStore
	Security            SecurityStore
	AdClicks            AdClickStore
	ModSecrets          ModSecretStore
}
