package src

import (
	"html/template"
	"log"
	"net/http"
	"os"
	"strings"
	"sync"

	"github.com/gorilla/sessions"
)

var (
	store *sessions.CookieStore
)

func init() {
	store = sessions.NewCookieStore([]byte(SessionSecret))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   SessionMaxAge,
		HttpOnly: true,
		Secure:   true,
		SameSite: http.SameSiteLaxMode,
	}
}

type Server struct {
	client    *http.Client
	templates *template.Template
	mu        sync.RWMutex
	artists   []Artist
}

func NewServer() (*Server, error) {
	funcMap := template.FuncMap{
		"formatDate":     FormatDate,
		"formatLocation": FormatLocation,
		"joinMembers": func(members []string) string {
			return strings.Join(members, ", ")
		},
		"sub": func(a, b int) int {
			return a - b
		},
		"substr": func(s string, start, length int) string {
			if start >= len(s) {
				return ""
			}
			end := start + length
			if end > len(s) {
				end = len(s)
			}
			return s[start:end]
		},
		"upper": strings.ToUpper,
		"getString": func(ns interface{}) string {
			return ""
		},
	}
	tmpl := template.Must(template.New("pages").Funcs(funcMap).ParseGlob(TemplatesDirectory))
	srv := &Server{
		client: &http.Client{
			Timeout: ClientTimeout,
		},
		templates: tmpl,
	}
	if err := srv.RefreshData(); err != nil {
		return nil, err
	}
	return srv, nil
}

func (s *Server) Start() error {
	mux := http.NewServeMux()

	mux.HandleFunc("/", s.HandleRoot)
	mux.HandleFunc("/login", s.HandleLogin)
	mux.HandleFunc("/register", s.HandleRegister)
	mux.HandleFunc("/home", RequireAuth(s.HandleIndex))
	mux.HandleFunc("/profile", RequireAuth(s.HandleProfile))
	mux.HandleFunc("/artist", RequireAuth(s.HandleArtist))
	mux.HandleFunc(RefreshPath, RequireAuth(s.HandleRefresh))
	mux.HandleFunc("/api/geocode", RequireAuth(s.HandleGeocode))
	mux.HandleFunc("/api/paypal/create-order", RequireAuth(s.HandleCreateOrder))
	mux.HandleFunc("/api/paypal/capture-order", RequireAuth(s.HandleCaptureOrder))
	mux.HandleFunc("/paypal/success", RequireAuth(s.HandlePayPalSuccess))
	mux.HandleFunc("/profile/update", RequireAuth(s.HandleUpdateProfile))
	mux.HandleFunc("/logout", s.HandleLogout)
	mux.HandleFunc("/admin/users", RequireAdmin(s.HandleAdminUsers))
	mux.HandleFunc("/admin/users/update-role", RequireAdmin(s.HandleAdminUpdateUserRole))
	mux.HandleFunc("/admin/users/delete", RequireAdmin(s.HandleAdminDeleteUser))
	mux.HandleFunc("/legal/conditions", s.HandleLegalConditions)
	mux.HandleFunc("/legal/privacy", s.HandleLegalPrivacy)
	mux.HandleFunc("/legal/cookies", s.HandleLegalCookies)
	mux.HandleFunc("/legal/mentions", s.HandleLegalMentions)
	
	fileServer := http.FileServer(http.Dir("static"))
	mux.Handle(StaticPrefix, http.StripPrefix(StaticPrefix, fileServer))

	server := &http.Server{
		Addr:              ServerAddress,
		Handler:           mux,
		ReadHeaderTimeout: ReadHeaderTimeout,
	}

	certExists := fileExists(CertFile)
	keyExists := fileExists(KeyFile)
	
	if certExists && keyExists {
		store.Options.Secure = true
		log.Printf("Serveur lancé: https://localhost%s", ServerAddress)
		return server.ListenAndServeTLS(CertFile, KeyFile)
	} else {
		store.Options.Secure = false
		log.Printf("Serveur lancé: http://localhost%s", ServerAddress)
		return server.ListenAndServe()
	}
}

func (s *Server) RefreshData() error {
	artists, err := FetchArtistsData(s.client)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.artists = artists
	return nil
}

func (s *Server) ListArtists() []Artist {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snapshot := make([]Artist, len(s.artists))
	copy(snapshot, s.artists)
	return snapshot
}

func (s *Server) FindArtist(id int) (Artist, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, art := range s.artists {
		if art.ID == id {
			return art, true
		}
	}
	return Artist{}, false
}

func (s *Server) Render(w http.ResponseWriter, name string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("template %s failed: %v", name, err)
		http.Error(w, "Une erreur est survenue", http.StatusInternalServerError)
	}
}

func fileExists(filename string) bool {
	_, err := os.Stat(filename)
	return err == nil
}
