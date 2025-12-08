package src

import (
	"log"
	"net/http"
	"strconv"
	"strings"
	"time"
)

func (s *Server) HandleIndex(w http.ResponseWriter, r *http.Request) {
	// Exemple d'utilisation des sessions Gorilla
	session, err := GetSession(r)
	if err != nil {
		log.Printf("Erreur session: %v", err)
	}

	// Compter les visites de l'utilisateur
	var visits int
	if v, ok := session.Values["visits"].(int); ok {
		visits = v + 1
	} else {
		visits = 1
	}
	session.Values["visits"] = visits
	session.Values["last_visit"] = time.Now().Format(time.RFC3339)

	// Sauvegarder la session
	if err := SaveSession(w, r, session); err != nil {
		log.Printf("Erreur sauvegarde session: %v", err)
	}

	// Log pour debug (optionnel)
	if visits%10 == 0 {
		log.Printf("Utilisateur a visité %d fois", visits)
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	artists := s.ListArtists()
	filtered := FilterArtists(artists, query)
	data := IndexPageData{
		Query:   query,
		Count:   len(filtered),
		Total:   len(artists),
		Artists: filtered,
	}
	s.Render(w, "index.html", data)
}

func (s *Server) HandleArtist(w http.ResponseWriter, r *http.Request) {
	idParam := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idParam)
	if err != nil || id <= 0 {
		http.Error(w, "Identifiant invalide", http.StatusBadRequest)
		return
	}
	art, ok := s.FindArtist(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	locDates := BuildLocationDates(art.DatesLocations)
	data := ArtistPageData{
		Artist:        art,
		LocationDates: locDates,
	}
	s.Render(w, "artist.html", data)
}

func (s *Server) HandleRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non supportée", http.StatusMethodNotAllowed)
		return
	}
	if err := s.RefreshData(); err != nil {
		http.Error(w, "Impossible d'actualiser les données", http.StatusBadGateway)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}
