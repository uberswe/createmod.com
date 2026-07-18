package server

import (
	"context"
	"createmod/internal/aidescription"
	"createmod/internal/auth"
	"createmod/internal/cache"
	"createmod/internal/discord"
	"createmod/internal/jobs"
	"createmod/internal/slowlog"
	appmailer "createmod/internal/mailer"
	"createmod/internal/moderation"
	"createmod/internal/modmeta"
	"createmod/internal/pages"
	"createmod/internal/pointlog"
	"createmod/internal/ratelimit"
	irouter "createmod/internal/router"
	"createmod/internal/search"
	"createmod/internal/session"
	"createmod/internal/similarity"
	"createmod/internal/sitemap"
	"createmod/internal/storage"
	"createmod/internal/store"
	"createmod/internal/translation"
	"fmt"
	"runtime"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/meilisearch/meilisearch-go"
	"github.com/redis/go-redis/v9"
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
	RedisURL            string
	Store               *store.Store
	Pool                *pgxpool.Pool
	Storage             *storage.Service
	DiscordClientID     string
	DiscordClientSecret string
	GithubClientID      string
	GithubClientSecret  string
	TwitchClientID      string
	TwitchClientSecret  string
	PatreonClientID     string
	PatreonClientSecret string
	RedditClientID      string
	RedditClientSecret  string
	GoogleClientID      string
	GoogleClientSecret  string
	MicrosoftClientID   string
	MicrosoftClientSecret string
	SteamAPIKey         string
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
	rateLimiter          ratelimit.Limiter
	redisClient          *redis.Client
	discordService       *discord.Service
	moderationService    *moderation.Service
	aiDescriptionService *aidescription.Service
	translationService   *translation.Service
	pointLogService      *pointlog.Service
	modMetaService       *modmeta.Service
	mailService          *appmailer.Service
	jobWorker            *jobs.Worker
	discordOAuth         *auth.OAuthProvider
	githubOAuth          *auth.OAuthProvider
	twitchOAuth          *auth.OAuthProvider
	patreonOAuth         *auth.OAuthProvider
	redditOAuth          *auth.OAuthProvider
	googleOAuth          *auth.OAuthProvider
	microsoftOAuth       *auth.OAuthProvider
	steamAuth            *auth.SteamProvider
	meiliClient          meilisearch.ServiceManager
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
	sitemapService := sitemap.New(conf.Dev, conf.Storage)
	discordService := discord.New(conf.DiscordWebhookUrl)
	moderationService := moderation.NewService(conf.OpenAIApiKey, slog.Default())
	aiDescriptionService := aidescription.New(conf.OpenAIApiKey, slog.Default())
	translationService := translation.New(conf.OpenAIApiKey, slog.Default(), conf.Store)

	// Initialize shared Redis client, rate limiter, and cache.
	var rl ratelimit.Limiter
	var redisClient *redis.Client
	var cacheService *cache.Service
	if conf.RedisURL != "" {
		opts, err := redis.ParseURL(conf.RedisURL)
		if err != nil {
			log.Fatalf("Failed to parse Redis URL: %v", err)
		}
		redisClient = redis.NewClient(opts)
		var redisConnected bool
		for attempt := 1; attempt <= 10; attempt++ {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			if err := redisClient.Ping(ctx).Err(); err != nil {
				log.Printf("Redis connection attempt %d/10 failed: %v", attempt, err)
				cancel()
				time.Sleep(time.Duration(attempt) * time.Second)
				continue
			}
			cancel()
			redisConnected = true
			break
		}
		if !redisConnected {
			log.Fatalf("Failed to connect to Redis after 10 attempts")
		}
		redisClient.AddHook(&slowlog.RedisHook{})
		rl = ratelimit.NewRedisFromClient(redisClient)
		cacheService = cache.NewWithRedis(redisClient)
		log.Println("Connected to Redis (shared client for rate limiting + caching)")
	} else {
		rl = ratelimit.NewMemory()
		cacheService = cache.New()
		log.Println("WARNING: REDIS_URL not set, rate limiting and caching use per-pod in-memory stores")
	}

	srv := &Server{
		conf:                 conf,
		store:                conf.Store,
		pool:                 conf.Pool,
		storageService:       conf.Storage,
		sitemapService:       sitemapService,
		cacheService:         cacheService,
		rateLimiter:          rl,
		redisClient:          redisClient,
		discordService:       discordService,
		moderationService:    moderationService,
		aiDescriptionService: aiDescriptionService,
		translationService:   translationService,
		pointLogService:      pointlog.New(conf.Store),
		modMetaService:       modmeta.New(conf.CurseForgeApiKey, conf.Store),
		mailService:          appmailer.New(),
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
	if conf.TwitchClientID != "" && conf.TwitchClientSecret != "" {
		srv.twitchOAuth = auth.NewTwitchProvider(
			conf.TwitchClientID, conf.TwitchClientSecret,
			conf.BaseURL+"/auth/twitch/callback",
		)
	}
	if conf.PatreonClientID != "" && conf.PatreonClientSecret != "" {
		srv.patreonOAuth = auth.NewPatreonProvider(
			conf.PatreonClientID, conf.PatreonClientSecret,
			conf.BaseURL+"/auth/patreon/callback",
		)
	}
	if conf.RedditClientID != "" && conf.RedditClientSecret != "" {
		srv.redditOAuth = auth.NewRedditProvider(
			conf.RedditClientID, conf.RedditClientSecret,
			conf.BaseURL+"/auth/reddit/callback",
		)
	}
	if conf.GoogleClientID != "" && conf.GoogleClientSecret != "" {
		srv.googleOAuth = auth.NewGoogleProvider(
			conf.GoogleClientID, conf.GoogleClientSecret,
			conf.BaseURL+"/auth/google/callback",
		)
	}
	if conf.MicrosoftClientID != "" && conf.MicrosoftClientSecret != "" {
		srv.microsoftOAuth = auth.NewMicrosoftProvider(
			conf.MicrosoftClientID, conf.MicrosoftClientSecret,
			conf.BaseURL+"/auth/microsoft/callback",
		)
	}
	if conf.SteamAPIKey != "" {
		srv.steamAuth = auth.NewSteamProvider(
			conf.SteamAPIKey,
			conf.BaseURL+"/auth/steam/callback",
		)
	}

	return srv
}

func (s *Server) Start() {
	log.Println("Launching...")

	// Initialise the search service. The S3 cache load and full index
	// rebuild happen via River's SearchIndexWorker (RunOnStart: true),
	// so the server can start accepting HTTP traffic immediately.
	log.Println("Starting Search Server")
	s.searchService = search.NewEmpty(s.storageService)

	// When maintenance mode is active, skip heavy background jobs.
	migrating := s.conf.MaintenanceMode != nil && s.conf.MaintenanceMode.Load()

	trendingWindowDays := []int{7}

	// Initialize Meilisearch.
	meiliURL := os.Getenv("MEILISEARCH_URL")
	meiliKey := os.Getenv("MEILISEARCH_KEY")
	if meiliKey == "" {
		meiliKey = os.Getenv("MEILI_MASTER_KEY")
	}
	var searchEngine search.SearchEngine
	if meiliURL != "" {
		s.meiliClient = meilisearch.New(meiliURL, meilisearch.WithAPIKey(meiliKey))
		if _, err := s.meiliClient.Health(); err != nil {
			slog.Error("Meilisearch not reachable", "error", err)
		} else {
			slog.Info("Connected to Meilisearch", "url", meiliURL)
			if err := search.EnsureMeiliIndexes(s.meiliClient); err != nil {
				slog.Error("Failed to configure Meilisearch indexes", "error", err)
			}
			searchEngine = search.NewMeiliEngine(s.meiliClient, search.MeiliIndex, s.searchService)
		}
	}
	if searchEngine == nil {
		slog.Warn("Meilisearch not configured; search will return empty results")
		searchEngine = search.NewNoopEngine()
	}

	if !migrating {
		// Warm per-pod in-memory caches in the background so startup isn't
		// blocked. Handlers tolerate cold caches (they compute on miss).
		go func() {
			pages.WarmIndexCache(s.cacheService, s.store, trendingWindowDays)
			pages.WarmVideosCache(s.cacheService, s.store)
		}()

		// Pre-warm site stats search cache on every pod. Uses a random
		// initial delay (0–5 min) so pods don't all hit the DB at once,
		// then refreshes every 30 minutes.
		go func() {
			jitter := time.Duration(rand.Intn(300)) * time.Second
			time.Sleep(jitter)
			ctx := context.Background()
			pages.WarmSearchStatsCache(ctx, s.cacheService, s.store, "30d")
			pages.WarmSearchStatsCache(ctx, s.cacheService, s.store, "7d")
			slog.Info("site stats cache warmed (initial)", "jitter", jitter.Round(time.Second))

			ticker := time.NewTicker(30 * time.Minute)
			defer ticker.Stop()
			for range ticker.C {
				ctx := context.Background()
				pages.WarmSearchStatsCache(ctx, s.cacheService, s.store, "30d")
				pages.WarmSearchStatsCache(ctx, s.cacheService, s.store, "7d")
				slog.Info("site stats cache warmed (periodic)")
			}
		}()

		// Load in-memory search index from S3 cache for fast startup.
		// The full DB rebuild and Meilisearch sync are handled by the
		// SearchIndexWorker River job (RunOnStart: true, every 10 min),
		// which runs on one pod and saves the result to S3. All pods
		// periodically reload from S3 to stay current without each
		// doing the expensive DB rebuild themselves.
		go func() {
			s.searchService.WarmFromStorage()

			// Periodically reload the in-memory index from S3 so this
			// pod picks up rebuilds done by whichever pod ran the River
			// job. This is cheap (~1s for ~4k docs) compared to a full
			// DB rebuild (~12s).
			ticker := time.NewTicker(10 * time.Minute)
			defer ticker.Stop()
			for range ticker.C {
				s.searchService.WarmFromStorage()
			}
		}()

		// All other periodic work (search index rebuild, sitemap generation,
		// schematic repair, temp upload cleanup, trending scores, etc.) is
		// handled by River periodic jobs with UniqueOpts deduplication, so
		// only one pod executes each job even when running multiple replicas.
		s.startJobWorker(trendingWindowDays)
	} else {
		slog.Info("maintenance mode active — deferring background jobs until migration completes")
	}

	// ROUTES

	// Similarity fingerprint index: per-pod, loaded in the background so
	// boot isn't blocked, refreshed every 10 minutes.
	similarityService := similarity.New(s.store)
	go similarityService.Start(context.Background())

	chiRouter := irouter.Register(irouter.RegisterParams{
		SimilarityService:  similarityService,
		SearchService:      s.searchService,
		SearchEngine:       searchEngine,
		CacheService:       s.cacheService,
		RateLimiter:        s.rateLimiter,
		DiscordService:     s.discordService,
		ModerationService:  s.moderationService,
		TranslationService: s.translationService,
		ModMetaService:     s.modMetaService,
		AppStore:           s.store,
		SessionStore:       s.sessionStore,
		StorageService:     s.storageService,
		DiscordOAuth:       s.discordOAuth,
		GithubOAuth:        s.githubOAuth,
		TwitchOAuth:        s.twitchOAuth,
		PatreonOAuth:       s.patreonOAuth,
		RedditOAuth:        s.redditOAuth,
		GoogleOAuth:        s.googleOAuth,
		MicrosoftOAuth:     s.microsoftOAuth,
		SteamAuth:          s.steamAuth,
		MailService:        s.mailService,
		JobWorker:          s.jobWorker,
		MaintenanceMode:    s.conf.MaintenanceMode,
		DBPool:             s.pool,
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

	// Periodic runtime stats for observability.
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			var m runtime.MemStats
			runtime.ReadMemStats(&m)
			slog.Info("runtime",
				"heap_alloc_mb", m.HeapAlloc/1024/1024,
				"heap_sys_mb", m.HeapSys/1024/1024,
				"goroutines", runtime.NumGoroutine(),
				"gc_cycles", m.NumGC,
			)
		}
	}()

	httpServer := &http.Server{
		Addr:              addr,
		Handler:           handler,
		ReadTimeout:       5 * time.Minute,
		ReadHeaderTimeout: 10 * time.Second,
		WriteTimeout:      5 * time.Minute,
		IdleTimeout:       120 * time.Second,
	}

	// Graceful shutdown — ordered: HTTP → jobs → cache → rate limiter → Redis → PostgreSQL
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
		<-sigCh

		log.Println("Shutting down...")

		shutdownCtx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		// 1. Stop accepting new HTTP requests and drain in-flight ones
		if err := httpServer.Shutdown(shutdownCtx); err != nil {
			slog.Error("HTTP server shutdown error", "error", err)
		}

		// 2. Stop background jobs
		if s.jobWorker != nil {
			if err := s.jobWorker.Stop(shutdownCtx); err != nil {
				slog.Error("failed to stop job worker", "error", err)
			}
		}

		// 3. Stop cache pub/sub subscription
		if s.cacheService != nil {
			s.cacheService.Close()
		}

		// 4. Close rate limiter (no-op when using shared client)
		if s.rateLimiter != nil {
			if err := s.rateLimiter.Close(); err != nil {
				slog.Error("failed to close rate limiter", "error", err)
			}
		}

		// 5. Close shared Redis connection
		if s.redisClient != nil {
			if err := s.redisClient.Close(); err != nil {
				slog.Error("failed to close Redis client", "error", err)
			}
		}

		// 6. Close PostgreSQL pool
		if s.pool != nil {
			s.pool.Close()
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


// startJobWorker initialises and starts the River background job worker.
func (s *Server) startJobWorker(windowDays []int) {
	jobCtx := context.Background()
	w, err := jobs.New(jobCtx, jobs.Config{
		Pool: s.pool,
		Deps: jobs.Deps{
			Store:              s.store,
			Storage:            s.storageService,
			Search:             s.searchService,
			Cache:              s.cacheService,
			Sitemap:            s.sitemapService,
			AIDesc:             s.aiDescriptionService,
			Translation:        s.translationService,
			PointLog:           s.pointLogService,
			ModMeta:            s.modMetaService,
			SessionStore:       s.sessionStore,
			Moderation:         s.moderationService,
			Mail:               s.mailService,
			MeiliClient:        s.meiliClient,
			TwitchClientID:     s.conf.TwitchClientID,
			TwitchClientSecret: s.conf.TwitchClientSecret,
		},
	})
	if err != nil {
		slog.Error("failed to create River job worker", "error", err)
		return
	}
	if err := w.Start(jobCtx); err != nil {
		slog.Error("failed to start River job worker", "error", err)
		return
	}
	s.jobWorker = w
	slog.Info("River job worker started")
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
