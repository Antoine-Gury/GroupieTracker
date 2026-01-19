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

// IsAuthenticated vérifie si un utilisateur est connecté.
func IsAuthenticated(r *http.Request) bool {
	session, err := GetSession(r)
	if err != nil {
		return false
	}
	_, ok := session.Values["user_id"]
	return ok
}

// RequireAuth redirige vers /login si l'utilisateur n'est pas connecté.
func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !IsAuthenticated(r) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	}
}

// IsAdmin vérifie si l'utilisateur connecté est admin.
func IsAdmin(r *http.Request) bool {
	session, err := GetSession(r)
	if err != nil {
		return false
	}
	userID, ok := session.Values["user_id"].(int)
	if !ok {
		return false
	}
	user, err := GetUserByID(DB, userID)
	if err != nil {
		return false
	}
	return user.Role == "admin"
}

// RequireAdmin redirige vers /home si l'utilisateur n'est pas admin.
func RequireAdmin(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !IsAuthenticated(r) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		if !IsAdmin(r) {
			http.Error(w, "Accès refusé: droits administrateur requis", http.StatusForbidden)
			return
		}
		next.ServeHTTP(w, r)
	}
}

