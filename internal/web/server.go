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
}

// Server is the HTTP server for the web application.
type Server struct {
	router    chi.Router
	server    *http.Server
	templates *Templates
	sessions  *SessionStore
	handlers  *Handlers
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

	// Create session store
	sessions := NewSessionStore()

	// Create handlers
	handlers := NewHandlers(auth, sessions, templates)

	// Create router
	router := chi.NewRouter()

	s := &Server{
		router:    router,
		templates: templates,
		sessions:  sessions,
		handlers:  handlers,
	}

	// Configure middleware
	s.setupMiddleware()

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
func (s *Server) setupMiddleware() {
	s.router.Use(middleware.RequestID)
	s.router.Use(middleware.RealIP)
	s.router.Use(middleware.Logger)
	s.router.Use(middleware.Recoverer)
	s.router.Use(middleware.Compress(5))
}

// setupRoutes configures routes for the application.
func (s *Server) setupRoutes(staticFS fs.FS) {
	// Static files
	fileServer := http.FileServer(http.FS(staticFS))
	s.router.Handle("/static/*", http.StripPrefix("/static/", fileServer))

	// Pages
	s.router.Get("/", s.handlers.Home)

	// Auth routes
	s.router.Get("/auth/login", s.handlers.Login)
	s.router.Get("/callback", s.handlers.Callback)
	s.router.Post("/auth/logout", s.handlers.Logout)
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
