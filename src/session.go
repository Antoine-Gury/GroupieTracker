package src

import (
	"net/http"

	"github.com/gorilla/sessions"
)

// GetSession récupère la session de la requête
func GetSession(r *http.Request) (*sessions.Session, error) {
	return store.Get(r, SessionName)
}

// SaveSession sauvegarde la session dans la réponse
func SaveSession(w http.ResponseWriter, r *http.Request, session *sessions.Session) error {
	return session.Save(r, w)
}

// SessionMiddleware est un middleware pour gérer les sessions
func SessionMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// La session est accessible via GetSession() dans les handlers
		next.ServeHTTP(w, r)
	}
}

