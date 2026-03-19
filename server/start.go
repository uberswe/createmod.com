package server

import (
	"context"
	"createmod/internal/abtest"
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
	"createmod/internal/sitemap"
	"createmod/internal/storage"
	"createmod/internal/store"
	"createmod/internal/translation"
	"fmt"
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
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := redisClient.Ping(ctx).Err(); err != nil {
			log.Fatalf("Failed to connect to Redis: %v", err)
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

	// Load trending A/B test configuration
	trendingCfg := abtest.LoadTrendingConfig()
	var trendingWindowDays []int
	if trendingCfg.Enabled {
		trendingWindowDays = trendingCfg.AllWindowDays()
	} else {
		trendingWindowDays = []int{30}
	}

	// A/B test configuration — load before startup goroutine so Meilisearch
	// client is available for the initial index sync.
	abCfg := abtest.LoadConfig()

	// Build variant router with search engines.
	bleveAI := search.NewBleveEngine(s.searchService, false)   // variant B: with AI (current behavior)
	bleveBase := search.NewBleveEngine(s.searchService, true)  // variant A: base only

	engines := map[string]search.SearchEngine{
		"A": bleveBase,
		"B": bleveAI,
	}

	// Initialize Meilisearch if configured.
	if abCfg.MeilisearchURL != "" {
		s.meiliClient = meilisearch.New(abCfg.MeilisearchURL, meilisearch.WithAPIKey(abCfg.MeilisearchKey))
		if _, err := s.meiliClient.Health(); err != nil {
			slog.Warn("Meilisearch not reachable, variants C/D/E will fall back to Bleve", "error", err)
		} else {
			slog.Info("Connected to Meilisearch", "url", abCfg.MeilisearchURL)
			if err := search.EnsureMeiliIndexes(s.meiliClient); err != nil {
				slog.Error("Failed to configure Meilisearch indexes", "error", err)
			}
			engines["C"] = search.NewMeiliEngine(s.meiliClient, search.MeiliIndexBase, s.searchService)
			engines["D"] = search.NewMeiliEngine(s.meiliClient, search.MeiliIndexAI, s.searchService)
			engines["E"] = search.NewMeiliEngine(s.meiliClient, search.MeiliIndexFull, s.searchService)
		}
	}

	variantRouter := abtest.NewVariantRouter(engines, bleveAI)

	if !migrating {
		// Warm per-pod in-memory caches in the background so startup isn't
		// blocked. Handlers tolerate cold caches (they compute on miss).
		go func() {
			pages.WarmIndexCache(s.cacheService, s.store, trendingWindowDays)
			pages.WarmVideosCache(s.cacheService, s.store)
		}()

		// Build search index per-pod. The index is in-memory so each pod
		// needs its own copy; River's deduplication means only one pod
		// would run the periodic job, leaving other pods with empty indexes.
		// After building Bleve, also sync to Meilisearch if available.
		go func() {
			slog.Info("per-pod search index build starting")
			s.searchService.WarmFromStorage()
			storeSchematics, err := s.store.Schematics.ListAllForIndex(context.Background())
			if err != nil {
				slog.Error("per-pod search index build failed", "error", err)
				return
			}
			mapped := pages.MapStoreSchematics(s.store, storeSchematics, s.cacheService)
			s.searchService.BuildIndex(mapped)
			if scores := pages.ComputeTrendingScoresFromStore(s.store); scores != nil {
				s.searchService.SetTrendingScores(scores)
			}
			slog.Info("per-pod search index build complete", "count", len(mapped))

			// Sync to Meilisearch so variants C/D/E have data immediately.
			if s.meiliClient != nil {
				filterIndex := s.searchService.GetIndex()
				if len(filterIndex) > 0 {
					docs := search.MapToMeiliDocuments(filterIndex, nil)
					for _, uid := range []string{search.MeiliIndexBase, search.MeiliIndexAI, search.MeiliIndexFull} {
						if err := search.SyncMeiliIndex(s.meiliClient, uid, docs); err != nil {
							slog.Error("per-pod meili sync failed", "index", uid, "error", err)
						} else {
							slog.Info("per-pod meili sync complete", "index", uid, "docs", len(docs))
						}
					}
				}
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

	chiRouter := irouter.Register(irouter.RegisterParams{
		SearchService:      s.searchService,
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
		MailService:        s.mailService,
		JobWorker:          s.jobWorker,
		MaintenanceMode:    s.conf.MaintenanceMode,
		VariantRouter:      variantRouter,
		ABTestConfig:       abCfg,
		TrendingConfig:     trendingCfg,
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
		Pool:       s.pool,
		WindowDays: windowDays,
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
			Moderation:   s.moderationService,
			Mail:         s.mailService,
			MeiliClient:  s.meiliClient,
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
