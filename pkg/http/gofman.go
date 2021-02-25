package http

import (
	"net/http"

	"github.com/dhenkes/gofman/pkg/gofman"
	"github.com/gorilla/mux"
)

// registerDebugRoutes is a helper function for registering all system related
// debug routes.
func (s *Server) registerDebugRoutes(r *mux.Router) {
	r.HandleFunc("/version", s.handleVersion).Methods("GET")
	r.HandleFunc("/commit", s.handleCommit).Methods("GET")
}

// handleVersion displays the deployed version.
func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(gofman.Version))
}

// handleVersion displays the deployed commit.
func (s *Server) handleCommit(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/plain")
	w.Write([]byte(gofman.Commit))
}
