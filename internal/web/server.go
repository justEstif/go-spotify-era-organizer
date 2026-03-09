package web

import (
	"context"
	"fmt"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	spotifyauth "github.com/zmb3/spotify/v2/auth"

	"github.com/justestif/go-spotify-era-organizer/internal/analysis"
	"github.com/justestif/go-spotify-era-organizer/internal/db"
	"github.com/justestif/go-spotify-era-organizer/internal/eras"
	"github.com/justestif/go-spotify-era-organizer/internal/jobs"
	"github.com/justestif/go-spotify-era-organizer/internal/lastfm"
	"github.com/justestif/go-spotify-era-organizer/internal/ratelimit"
	syncpkg "github.com/justestif/go-spotify-era-organizer/internal/sync"
	"github.com/justestif/go-spotify-era-organizer/internal/tags"
)

const (
	// DefaultAddr is the default server address.
	DefaultAddr = "127.0.0.1:8080"

	// RedirectURI must match the Spotify app configuration.
	RedirectURI = "http://127.0.0.1:8080/callback"
)

// ServerConfig holds server configuration.
type ServerConfig struct {
	Addr         string
	ClientID     string
	ClientSecret string
	TemplatesFS  fs.FS
	StaticFS     fs.FS
	DB           *db.DB // Optional - if nil, uses in-memory sessions
	LastFMAPIKey string // Optional - if empty, tag fetching is disabled
	AdminToken   string // Optional - if empty, admin endpoint disabled
}

// Server is the HTTP server for the web application.
type Server struct {
	router          chi.Router
	server          *http.Server
	templates       *Templates
	sessions        SessionManager
	handlers        *Handlers
	db              *db.DB
	syncService     *syncpkg.Service
	eraService      *eras.Service
	analysisService *analysis.Service
	jobQueue        *jobs.Queue
	adminToken      string
}

// NewServer creates a new web server.
func NewServer(cfg ServerConfig) (*Server, error) {
	// Create Spotify authenticator
	auth := spotifyauth.New(
		spotifyauth.WithClientID(cfg.ClientID),
		spotifyauth.WithClientSecret(cfg.ClientSecret),
		spotifyauth.WithRedirectURL(RedirectURI),
		spotifyauth.WithScopes(
			spotifyauth.ScopeUserLibraryRead,
			spotifyauth.ScopePlaylistModifyPublic,
			spotifyauth.ScopePlaylistModifyPrivate,
		),
	)

	// Create template manager
	templates, err := NewTemplates(cfg.TemplatesFS)
	if err != nil {
		return nil, fmt.Errorf("loading templates: %w", err)
	}

	// Create session store (DB-backed or in-memory)
	var sessions SessionManager
	if cfg.DB != nil {
		sessions = NewDBSessionStore(cfg.DB)
	} else {
		sessions = NewSessionStore()
	}

	// Create services (only if DB is available)
	var syncService *syncpkg.Service
	var eraService *eras.Service
	var analysisService *analysis.Service
	if cfg.DB != nil {
		syncService = syncpkg.New(cfg.DB)
		eraService = eras.New(cfg.DB)

		// Create tag service if Last.fm API key is available
		var tagService *tags.Service
		if cfg.LastFMAPIKey != "" {
			lastfmClient := lastfm.NewClient(&lastfm.Config{APIKey: cfg.LastFMAPIKey})
			tagService = tags.NewService(lastfmClient)
		}

		// Create analysis service (orchestrates sync → tags → eras pipeline)
		analysisService = analysis.New(cfg.DB, syncService, eraService, tagService)
	}

	// Create job queue and register handlers
	jobQueue := jobs.New()
	if analysisService != nil {
		factory := &jobs.AuthClientFactory{Auth: auth}
		jobQueue.Register("sync", jobs.SyncHandler(analysisService, factory))
		jobQueue.Register("analyze", jobs.AnalyzeHandler(analysisService, factory))
		jobQueue.Register("recluster", jobs.ReclusterHandler(analysisService))
	}

	// Create handlers
	handlers := NewHandlers(HandlerDeps{
		Auth:            auth,
		Sessions:        sessions,
		Templates:       templates,
		DB:              cfg.DB,
		SyncService:     syncService,
		EraService:      eraService,
		AnalysisService: analysisService,
		AdminToken:      cfg.AdminToken,
		JobQueue:        jobQueue,
	})

	// Create per-user rate limiter (10 req/s, burst 20)
	limiter := ratelimit.New(10, 20)

	// Start periodic cleanup goroutine
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			limiter.Cleanup()
		}
	}()

	// Create router
	router := chi.NewRouter()

	s := &Server{
		router:          router,
		templates:       templates,
		sessions:        sessions,
		handlers:        handlers,
		db:              cfg.DB,
		syncService:     syncService,
		eraService:      eraService,
		analysisService: analysisService,
		jobQueue:        jobQueue,
		adminToken:      cfg.AdminToken,
	}

	// Configure middleware
	s.setupMiddleware(limiter)

	// Configure routes
	s.setupRoutes(cfg.StaticFS)

	// Create HTTP server
	s.server = &http.Server{
		Addr:         cfg.Addr,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	return s, nil
}

// setupMiddleware configures middleware for the router.
func (s *Server) setupMiddleware(limiter *ratelimit.Limiter) {
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.Compress(5))
	s.router.Use(RateLimitMiddleware(limiter, s.sessions))
}

// setupRoutes configures routes for the application.
func (s *Server) setupRoutes(staticFS fs.FS) {
	// Static files
	fileServer := http.FileServer(http.FS(staticFS))
	s.router.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	// Pages
	s.router.Get("/", s.handlers.Home)
	s.router.Get("/eras", s.handlers.Eras)
	s.router.Get("/search", s.handlers.Search)
	s.router.Get("/stats", s.handlers.Stats)
	s.router.Get("/timeline", s.handlers.Timeline)
	s.router.Get("/eras/{id}/tracks", s.handlers.EraTracks)
	s.router.Put("/eras/{id}/name", s.handlers.RenameEra)
	s.router.Post("/eras/{id}/playlist", s.handlers.ExportPlaylist)

	// Auth routes
	s.router.Get("/auth/login", s.handlers.Login)
	s.router.Get("/callback", s.handlers.Callback)
	s.router.Post("/auth/logout", s.handlers.Logout)

	// API routes
	s.router.Post("/api/analyze", s.handlers.Analyze)
	s.router.Get("/api/eras", s.handlers.GetEras)
	s.router.Get("/api/eras/{id}/tracks", s.handlers.GetEraTracksAPI)
	s.router.Post("/api/recluster", s.handlers.Recluster)
	s.router.Post("/api/outliers/assign", s.handlers.AssignOutlier)
	s.router.Post("/api/sync", s.handlers.SyncLibrary)
	s.router.Get("/api/sync/status", s.handlers.GetSyncStatus)

	// Job queue routes
	s.router.Post("/api/jobs/sync", s.handlers.SubmitSyncJob)
	s.router.Post("/api/jobs/analyze", s.handlers.SubmitAnalyzeJob)
	s.router.Get("/api/jobs/{id}/progress", s.handlers.JobProgress)
	s.router.Get("/api/jobs/active", s.handlers.ActiveJobs)

	// Admin routes
	s.router.Get("/api/admin/status", s.handlers.AdminStatus)
}

// Start starts the HTTP server.
func (s *Server) Start() error {
	log.Printf("Starting server at http://%s", s.server.Addr)
	return s.server.ListenAndServe()
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.server.Shutdown(ctx)
}

// Run starts the server and handles graceful shutdown on interrupt signals.
func (s *Server) Run() error {
	// Channel to receive shutdown signals
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, os.Interrupt, syscall.SIGTERM)

	// Start server in goroutine
	errCh := make(chan error, 1)
	go func() {
		if err := s.Start(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()

	// Wait for interrupt or error
	select {
	case err := <-errCh:
		return err
	case <-stop:
		log.Println("Shutting down server...")
	}

	// Graceful shutdown with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := s.Shutdown(ctx); err != nil {
		return fmt.Errorf("server shutdown: %w", err)
	}

	log.Println("Server stopped")
	return nil
}
