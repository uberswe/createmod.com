package server

import (
	"context"
	"createmod/internal/aidescription"
	"createmod/internal/auth"
	"createmod/internal/cache"
	"createmod/internal/discord"
	"createmod/internal/jobs"
	appmailer "createmod/internal/mailer"
	"createmod/internal/moderation"
	"createmod/internal/modmeta"
	"createmod/internal/pages"
	"createmod/internal/pointlog"
	irouter "createmod/internal/router"
	"createmod/internal/search"
	"createmod/internal/session"
	"createmod/internal/sitemap"
	"createmod/internal/storage"
	"createmod/internal/store"
	"createmod/internal/translation"
	"fmt"
	"github.com/jackc/pgx/v5/pgxpool"
	"log"
	"log/slog"
	"math/rand"
	"net/http"
	"os"
	"os/signal"
	"regexp"
	"strings"
	"sync/atomic"
	"syscall"
	"time"
	"unicode"
)

type Config struct {
	AutoMigrate         bool
	CreateAdmin         bool
	DiscordWebhookUrl   string
	OpenAIApiKey        string
	CurseForgeApiKey    string
	Dev                 bool
	DatabaseURL         string
	Store               *store.Store
	Pool                *pgxpool.Pool
	Storage             *storage.Service
	DiscordClientID     string
	DiscordClientSecret string
	GithubClientID      string
	GithubClientSecret  string
	BaseURL             string
	MaintenanceMode     *atomic.Bool // runtime-togglable maintenance flag
}

type Server struct {
	conf                 Config
	store                *store.Store
	pool                 *pgxpool.Pool
	storageService       *storage.Service
	sessionStore         *session.Store
	searchService        *search.Service
	sitemapService       *sitemap.Service
	cacheService         *cache.Service
	discordService       *discord.Service
	moderationService    *moderation.Service
	aiDescriptionService *aidescription.Service
	translationService   *translation.Service
	pointLogService      *pointlog.Service
	modMetaService       *modmeta.Service
	jobWorker            *jobs.Worker
	discordOAuth         *auth.OAuthProvider
	githubOAuth          *auth.OAuthProvider
}

// detectLanguageFromRequest returns a normalized language code based on the
// incoming request Accept-Language header. Falls back to "en".
func detectLanguageFromRequest(r *http.Request) string {
	if r == nil {
		return "en"
	}
	al := strings.TrimSpace(strings.ToLower(r.Header.Get("Accept-Language")))
	if al == "" {
		return "en"
	}
	// take first token before comma
	if idx := strings.Index(al, ","); idx >= 0 {
		al = al[:idx]
	}
	al = strings.TrimSpace(al)
	switch {
	case strings.HasPrefix(al, "pt-br"):
		return "pt-BR"
	case strings.HasPrefix(al, "pt-pt"):
		return "pt-PT"
	case al == "pt" || strings.HasPrefix(al, "pt-"):
		return "pt-PT"
	case strings.HasPrefix(al, "es"):
		return "es"
	case strings.HasPrefix(al, "de"):
		return "de"
	case strings.HasPrefix(al, "pl"):
		return "pl"
	case strings.HasPrefix(al, "ru"):
		return "ru"
	case strings.HasPrefix(al, "zh"):
		return "zh-Hans"
	default:
		return "en"
	}
}

func New(conf Config) *Server {
	sitemapService := sitemap.New(conf.Dev)
	discordService := discord.New(conf.DiscordWebhookUrl)
	moderationService := moderation.NewService(conf.OpenAIApiKey, slog.Default())
	aiDescriptionService := aidescription.New(conf.OpenAIApiKey, slog.Default())
	translationService := translation.New(conf.OpenAIApiKey, slog.Default(), conf.Store)

	srv := &Server{
		conf:                 conf,
		store:                conf.Store,
		pool:                 conf.Pool,
		storageService:       conf.Storage,
		sitemapService:       sitemapService,
		cacheService:         cache.New(),
		discordService:       discordService,
		moderationService:    moderationService,
		aiDescriptionService: aiDescriptionService,
		translationService:   translationService,
		pointLogService:      pointlog.New(conf.Store),
		modMetaService:       modmeta.New(conf.CurseForgeApiKey, conf.Store),
	}

	srv.sessionStore = session.NewStore(conf.Pool)
	pages.SetPasswordResetPool(conf.Pool)

	// Build OAuth providers if credentials are configured
	if conf.DiscordClientID != "" && conf.DiscordClientSecret != "" {
		srv.discordOAuth = auth.NewDiscordProvider(
			conf.DiscordClientID, conf.DiscordClientSecret,
			conf.BaseURL+"/auth/discord/callback",
		)
	}
	if conf.GithubClientID != "" && conf.GithubClientSecret != "" {
		srv.githubOAuth = auth.NewGitHubProvider(
			conf.GithubClientID, conf.GithubClientSecret,
			conf.BaseURL+"/auth/github/callback",
		)
	}

	return srv
}

func (s *Server) Start() {
	log.Println("Launching...")

	// Initialise the search service. It attempts to load a cached index
	// snapshot from storage (S3) so the server can serve search
	// requests immediately. A background goroutine then does a full
	// rebuild from the database to pick up any recent changes.
	log.Println("Starting Search Server")
	s.searchService = search.New(s.storageService)

	// Full index rebuild + trending scores + sitemap in the background
	// so the server is available to handle requests right away.
	go func() {
		slog.Info("search: background index rebuild starting")
		s.rebuildSearchIndexFromStore()
		s.sitemapService.Generate(s.store)
		slog.Info("search: background index rebuild complete")
	}()

	// Warm page caches so the first visitor never waits
	pages.WarmIndexCache(s.cacheService, s.store)
	pages.WarmVideosCache(s.cacheService, s.store)

	// Background: repair schematics (validate NBT, fill missing stats, soft-delete bad entries)
	go pages.RepairSchematics(s.storageService, s.store)

	// Background: periodically clean up expired temp uploads from PostgreSQL
	pages.StartTempUploadCleanup(s.store)

	// Start River job worker for periodic background jobs.
	jobCtx := context.Background()
	w, err := jobs.New(jobCtx, jobs.Config{
		Pool: s.pool,
		Deps: jobs.Deps{
			Store:        s.store,
			Storage:      s.storageService,
			Search:       s.searchService,
			Cache:        s.cacheService,
			Sitemap:      s.sitemapService,
			AIDesc:       s.aiDescriptionService,
			Translation:  s.translationService,
			PointLog:     s.pointLogService,
			ModMeta:      s.modMetaService,
			SessionStore: s.sessionStore,
		},
	})
	if err != nil {
		slog.Error("failed to create River job worker", "error", err)
	} else {
		if err := w.Start(jobCtx); err != nil {
			slog.Error("failed to start River job worker", "error", err)
		} else {
			s.jobWorker = w
			slog.Info("River job worker started")
		}
	}

	// ROUTES

	mailService := appmailer.New()

	chiRouter := irouter.Register(irouter.RegisterParams{
		SearchService:      s.searchService,
		CacheService:       s.cacheService,
		DiscordService:     s.discordService,
		ModerationService:  s.moderationService,
		TranslationService: s.translationService,
		ModMetaService:     s.modMetaService,
		AppStore:           s.store,
		SessionStore:       s.sessionStore,
		StorageService:     s.storageService,
		DiscordOAuth:       s.discordOAuth,
		GithubOAuth:        s.githubOAuth,
		MailService:        mailService,
		MaintenanceMode:    s.conf.MaintenanceMode,
	})

	// Wrap the chi router with the language prefix stripper
	handler := irouter.LangPrefixHandler(chiRouter)

	// Determine listen address
	addr := os.Getenv("LISTEN_ADDR")
	if addr == "" {
		if port := os.Getenv("PORT"); port != "" {
			addr = ":" + port
		} else {
			addr = ":8090"
		}
	}

	httpServer := &http.Server{
		Addr:         addr,
		Handler:      handler,
		ReadTimeout:  30 * time.Second,
		WriteTimeout: 60 * time.Second,
		IdleTimeout:  120 * time.Second,
	}

	// Graceful shutdown
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		log.Println("Shutting down...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if s.jobWorker != nil {
			if err := s.jobWorker.Stop(shutdownCtx); err != nil {
				slog.Error("failed to stop job worker", "error", err)
			}
		}

		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			slog.Error("HTTP server shutdown error", "error", err)
		}
	}()

	log.Printf("CreateMod.com Running on %s", addr)

	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("HTTP server error: %v", err)
	}
}

// ToYoutubeEmbedUrl extracts a YouTube video ID from a URL and returns
// the embed URL format. Returns empty string if no valid ID is found.
func ToYoutubeEmbedUrl(url string) string {
	r, err := regexp.Compile("(?:youtube\\.com\\/(?:[^\\/]+\\/.+\\/|(?:v|e(?:mbed)?)\\/|.*[?&]v=)|youtu\\.be\\/)([^\"&?\\/\\s]{11})")
	if err != nil {
		panic(err)
	}
	matches := r.FindAllStringSubmatch(url, 1)
	if len(matches) == 1 && len(matches[0]) == 2 {
		return fmt.Sprintf("https://www.youtube.com/embed/%s", matches[0][1])
	}
	return ""
}

func uniqueSlug(appStore *store.Store, s string) string {
	schem, err := appStore.Schematics.GetByName(context.Background(), s)
	if err != nil {
		// GetByName returns error when not found — slug is available
		return s
	}
	if schem != nil {
		return uniqueSlug(appStore, fmt.Sprintf("%s%s", s, randSeq(4)))
	}
	return s
}

func anyLetter(r rune) bool {
	return unicode.IsLetter(r)
}

// PostMigrationRebuild rebuilds the search index, sitemaps, and page caches
// after a data migration has populated the database. Call this when
// maintenance mode is disabled following a SQLite-to-PostgreSQL migration.
func (s *Server) PostMigrationRebuild() {
	slog.Info("post-migration: rebuilding search index, sitemaps, and caches")
	s.rebuildSearchIndexFromStore()
	s.sitemapService.Generate(s.store)
	pages.WarmIndexCache(s.cacheService, s.store)
	pages.WarmVideosCache(s.cacheService, s.store)
	slog.Info("post-migration: rebuild complete")
}

// rebuildSearchIndexFromStore rebuilds the search index using the PostgreSQL store.
func (s *Server) rebuildSearchIndexFromStore() {
	ctx := context.Background()
	storeSchematics, err := s.store.Schematics.ListAllForIndex(ctx)
	if err != nil {
		slog.Warn("search rebuild: failed to list schematics", "error", err)
		return
	}
	mapped := pages.MapStoreSchematics(s.store, storeSchematics, s.cacheService)
	slog.Debug("search service mapped schematics", "mapped schematic count", len(mapped))
	s.searchService.BuildIndex(mapped)
	if scores := pages.ComputeTrendingScoresFromStore(s.store); scores != nil {
		s.searchService.SetTrendingScores(scores)
	}
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

func randSeq(n int) string {
	letters := []rune("bcdfghjklmnpqrstvwxz")
	b := make([]rune, n)
	for i := range b {
		b[i] = letters[rand.Intn(len(letters))]
	}
	return string(b)
}
