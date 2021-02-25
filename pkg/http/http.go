package http

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/dhenkes/gofman/pkg/gofman"
	"github.com/gorilla/mux"
)

//go:embed assets/css/*.css
var assetsFS embed.FS

// HTTP constants.
const (
	ShutdownTimeout = 1 * time.Second
)

// Server represents an HTTP server.
type Server struct {
	ln     net.Listener
	server *http.Server
	router *mux.Router

	// Bind address & port for the server's listener.
	Address string
	Port    int

	// Servics used by the various HTTP routes.
	ActorService         gofman.ActorService
	FileService          gofman.FileService
	SessionService       gofman.SessionService
	SetupService         gofman.SetupService
	TagService           gofman.TagService
	UserService          gofman.UserService
	AuthService          gofman.AuthService
	PathTraversalService gofman.PathTraversalService
}

// NewServer returns a new instance of Server.
func NewServer() *Server {
	s := &Server{
		server: &http.Server{},
		router: mux.NewRouter(),
	}

	s.router.Use(s.handlePanic)

	s.server.Handler = http.HandlerFunc(s.router.ServeHTTP)

	s.router.NotFoundHandler = http.HandlerFunc(s.handleNotFound)

	if assetsHTTPFS, err := fs.Sub(assetsFS, "assets"); err == nil {
		s.router.PathPrefix("/assets/").
			Handler(http.StripPrefix("/assets/", s.handleAssets(http.FS(assetsHTTPFS))))
	}

	{
		r := s.router.PathPrefix("/debug").Subrouter()

		s.registerDebugRoutes(r)
	}

	{
		r := s.router.PathPrefix("/").Subrouter()
		r.Use(s.authenticate)

		s.registerSetupRoutes(r)
	}

	{
		r := s.router.PathPrefix("/").Subrouter()
		r.Use(s.authenticate)
		r.Use(s.requireAuth)

		s.registerActorRoutes(r)
		s.registerFileRoutes(r)
		s.registerSessionRoutes(r)
		s.registerTagRoutes(r)
		s.registerUserRoutes(r)
	}

	return s
}

// URL returns the local base URL of the running server.
func (s *Server) URL() string {
	return fmt.Sprintf("%s:%d", s.Address, s.Port)
}

// Open begins listening on the bind address.
func (s *Server) Open() (err error) {
	if s.ln, err = net.Listen("tcp", s.URL()); err != nil {
		return err
	}

	go s.server.Serve(s.ln)

	return nil
}

// Close gracefully shuts down the server.
func (s *Server) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), ShutdownTimeout)
	defer cancel()
	return s.server.Shutdown(ctx)
}

// handlePanic is middleware for catching panics.
func (s *Server) handlePanic(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			if err := recover(); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				fmt.Fprintf(w, "500")
			}
		}()

		next.ServeHTTP(w, r)
	})
}

// handleAssets handles request to publicly accessible assets. It checks if the
// asset exists and if that is the case it will return it. If the asset is a
// directory or it does not exist our default not found handler will be called.
func (s *Server) handleAssets(root http.FileSystem) http.Handler {
	fs := http.FileServer(root)

	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if !strings.HasPrefix(path, "/") {
			path = "/" + path
			r.URL.Path = path
		}

		file, err := root.Open(path)
		if err != nil {
			s.handleNotFound(w, r)
			return
		}

		stats, err := file.Stat()
		if err != nil {
			file.Close()
			s.handleNotFound(w, r)
			return
		}

		if stats.IsDir() {
			file.Close()
			s.handleNotFound(w, r)
			return
		}

		if err == nil {
			file.Close()
		}

		fs.ServeHTTP(w, r)
	})
}

// handleNotFound handles requests to routes that don't exist.
func (s *Server) handleNotFound(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, "404")
}

// handleMethodNotAllowed handles requests to routes that did not implement
// the requested method.
func (s *Server) handleMethodNotAllowed(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprintf(w, "404")
}
