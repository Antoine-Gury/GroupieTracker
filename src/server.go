package src

import (
	"html/template"
	"log"
	"net/http"
	"strings"
	"sync"
)

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
	mux.HandleFunc("/", s.HandleIndex)
	mux.HandleFunc("/artist", s.HandleArtist)
	mux.HandleFunc(RefreshPath, s.HandleRefresh)
	fileServer := http.FileServer(http.Dir("static"))
	mux.Handle(StaticPrefix, http.StripPrefix(StaticPrefix, fileServer))

	server := &http.Server{
		Addr:              ServerAddress,
		Handler:           mux,
		ReadHeaderTimeout: ReadHeaderTimeout,
	}
	log.Printf("Serveur démarré sur http://localhost%s", ServerAddress)
	return server.ListenAndServe()
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
