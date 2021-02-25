package http

import (
	"net/http"

	"github.com/dhenkes/gofman/pkg/gofman"
)

// authenticate is middleware for loading session data from a cookie.
func (s *Server) authenticate(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var sessionid string
		var token string

		if cookie, err := r.Cookie("Session"); err == http.ErrNoCookie || err != nil || cookie == nil {
			next.ServeHTTP(w, r)
			return
		} else {
			sessionid = cookie.Value
		}

		if cookie, err := r.Cookie("Token"); err == http.ErrNoCookie || err != nil || cookie == nil {
			next.ServeHTTP(w, r)
			return
		} else {
			token = cookie.Value
		}

		session, err := s.SessionService.FindSessionForToken(r.Context(), sessionid, token)
		if err != nil || session == nil {
			next.ServeHTTP(w, r)
			return
		}

		user, err := s.UserService.FindUserByID(r.Context(), session.UserID)
		if err != nil || user == nil {
			next.ServeHTTP(w, r)
			return
		}

		r = r.WithContext(gofman.NewContextWithSession(r.Context(), session))
		r = r.WithContext(gofman.NewContextWithUser(r.Context(), user))

		next.ServeHTTP(w, r)
	})
}

// requireAuth is middleware for requiring authentication.
func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		userid := gofman.UserIDFromContext(r.Context())
		if userid == "" {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		session := gofman.SessionFromContext(r.Context())
		if session == nil || session.ID == "" {
			http.Redirect(w, r, "/login", http.StatusFound)
			return
		}

		next.ServeHTTP(w, r)
	})
}
