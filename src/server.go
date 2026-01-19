package src

import (
	"html/template"
	"log"
	"net/http"
	"strings"
	"sync"

	"github.com/gorilla/sessions"
)

var (
	store *sessions.CookieStore
)

func init() {
	// Initialise le store de sessions avec une clé secrète
	store = sessions.NewCookieStore([]byte(SessionSecret))
	store.Options = &sessions.Options{
		Path:     "/",
		MaxAge:   SessionMaxAge,
		HttpOnly: true,
		Secure:   true, // Requiert HTTPS
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
			// Helper pour extraire la valeur d'un sql.NullString dans les templates
			// On ne peut pas directement accéder aux champs Valid/String dans les templates
			// Cette fonction sera utilisée différemment - on va plutôt créer un type wrapper
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

	// Handlers avec support de sessions (accessible via GetSession() dans les handlers)
	// Redirige la racine vers la page de login pour forcer l'auth en premier.
	mux.HandleFunc("/", s.HandleRoot)
	mux.HandleFunc("/login", s.HandleLogin)
	mux.HandleFunc("/register", s.HandleRegister)
	mux.HandleFunc("/home", RequireAuth(s.HandleIndex))
	mux.HandleFunc("/profile", RequireAuth(s.HandleProfile))
	mux.HandleFunc("/artist", RequireAuth(s.HandleArtist))
	mux.HandleFunc(RefreshPath, RequireAuth(s.HandleRefresh))
	mux.HandleFunc("/api/geocode", RequireAuth(s.HandleGeocode))
	
	// Handlers PayPal
	mux.HandleFunc("/api/paypal/create-order", RequireAuth(s.HandleCreateOrder))
	mux.HandleFunc("/api/paypal/capture-order", RequireAuth(s.HandleCaptureOrder))
	mux.HandleFunc("/paypal/success", RequireAuth(s.HandlePayPalSuccess))
	
	// Handler gestion de profil
	mux.HandleFunc("/profile/update", RequireAuth(s.HandleUpdateProfile))
	mux.HandleFunc("/logout", s.HandleLogout)
	
	// Handlers admin (gestion utilisateurs)
	mux.HandleFunc("/admin/users", RequireAdmin(s.HandleAdminUsers))
	mux.HandleFunc("/admin/users/update-role", RequireAdmin(s.HandleAdminUpdateUserRole))
	mux.HandleFunc("/admin/users/delete", RequireAdmin(s.HandleAdminDeleteUser))
	
	fileServer := http.FileServer(http.Dir("static"))
	mux.Handle(StaticPrefix, http.StripPrefix(StaticPrefix, fileServer))

	server := &http.Server{
		Addr:              ServerAddress,
		Handler:           mux,
		ReadHeaderTimeout: ReadHeaderTimeout,
	}

	log.Printf("Serveur HTTPS démarré sur https://localhost%s", ServerAddress)
	log.Printf("Certificats utilisés: %s (cert) et %s (key)", CertFile, KeyFile)
	log.Printf("Sessions Gorilla activées (nom: %s)", SessionName)
	return server.ListenAndServeTLS(CertFile, KeyFile)
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
