package api

import (
	"net/http"

	"reporter/internal/auth"
)

func (s *Server) requireAuth(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		token := bearerToken(r)
		if token == "" {
			http.Error(w, "authentication required", http.StatusUnauthorized)
			return
		}
		claims, err := auth.ParseToken(s.cfg.Auth.JWTSecret, token)
		if err != nil {
			http.Error(w, "invalid token", http.StatusUnauthorized)
			return
		}
		user, ok := s.store.UserByID(claims.Subject)
		if !ok {
			http.Error(w, "user not found", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r.WithContext(withCurrentUser(r.Context(), user)))
	})
}

func (s *Server) withPermission(resource, action string, handler http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		user, ok := currentUser(r)
		if !ok {
			http.Error(w, "not authenticated", http.StatusUnauthorized)
			return
		}
		if !s.authz.Can(user.ID, resource, action) && !s.authz.Can(user.ID, "*", "*") {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		handler(w, r)
	}
}
