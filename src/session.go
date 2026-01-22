package src

import (
	"net/http"

	"github.com/gorilla/sessions"
)

func GetSession(r *http.Request) (*sessions.Session, error) {
	return store.Get(r, SessionName)
}

func SaveSession(w http.ResponseWriter, r *http.Request, session *sessions.Session) error {
	return session.Save(r, w)
}

func SessionMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		next.ServeHTTP(w, r)
	}
}

func IsAuthenticated(r *http.Request) bool {
	session, err := GetSession(r)
	if err != nil {
		return false
	}
	_, ok := session.Values["user_id"]
	return ok
}

func RequireAuth(next http.HandlerFunc) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		if !IsAuthenticated(r) {
			http.Redirect(w, r, "/login", http.StatusSeeOther)
			return
		}
		next.ServeHTTP(w, r)
	}
}

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

