package main

import (
	"encoding/json"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	baseAPI            = "https://groupietrackers.herokuapp.com/api"
	artistsEndpoint    = baseAPI + "/artists"
	locationsEndpoint  = baseAPI + "/locations"
	datesEndpoint      = baseAPI + "/dates"
	relationsEndpoint  = baseAPI + "/relation"
	serverAddress      = ":3030"
	readHeaderTimeout  = 5 * time.Second
	clientTimeout      = 10 * time.Second
	refreshPath        = "/refresh"
	staticPrefix       = "/static/"
	templatesDirectory = "templates/*.html"
)

type artist struct {
	ID              int                 `json:"id"`
	Image           string              `json:"image"`
	Name            string              `json:"name"`
	Members         []string            `json:"members"`
	CreationDate    int                 `json:"creationDate"`
	FirstAlbum      string              `json:"firstAlbum"`
	LocationsURL    string              `json:"locations"`
	ConcertDatesURL string              `json:"concertDates"`
	RelationsURL    string              `json:"relations"`
	Locations       []string            `json:"-"`
	ConcertDates    []string            `json:"-"`
	DatesLocations  map[string][]string `json:"-"`
}

type locationsPayload struct {
	Index []struct {
		ID        int      `json:"id"`
		Locations []string `json:"locations"`
	} `json:"index"`
}

type datesPayload struct {
	Index []struct {
		ID    int      `json:"id"`
		Dates []string `json:"dates"`
	} `json:"index"`
}

type relationsPayload struct {
	Index []struct {
		ID             int                 `json:"id"`
		DatesLocations map[string][]string `json:"datesLocations"`
	} `json:"index"`
}

type indexPageData struct {
	Query   string
	Count   int
	Total   int
	Artists []artist
}

type locationDates struct {
	Raw    string
	Pretty string
	Dates  []string
	Count  int
}

type artistPageData struct {
	Artist        artist
	LocationDates []locationDates
}

type server struct {
	client    *http.Client
	templates *template.Template
	mu        sync.RWMutex
	artists   []artist
}

func newServer() (*server, error) {
	funcMap := template.FuncMap{
		"formatDate":     formatDate,
		"formatLocation": formatLocation,
		"joinMembers": func(members []string) string {
			return strings.Join(members, ", ")
		},
	}
	tmpl := template.Must(template.New("pages").Funcs(funcMap).ParseGlob(templatesDirectory))
	srv := &server{
		client: &http.Client{
			Timeout: clientTimeout,
		},
		templates: tmpl,
	}
	if err := srv.refreshData(); err != nil {
		return nil, err
	}
	return srv, nil
}

func (s *server) refreshData() error {
	artists, err := fetchArtistsData(s.client)
	if err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.artists = artists
	return nil
}

func (s *server) listArtists() []artist {
	s.mu.RLock()
	defer s.mu.RUnlock()
	snapshot := make([]artist, len(s.artists))
	copy(snapshot, s.artists)
	return snapshot
}

func (s *server) findArtist(id int) (artist, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	for _, art := range s.artists {
		if art.ID == id {
			return art, true
		}
	}
	return artist{}, false
}

func (s *server) render(w http.ResponseWriter, name string, data interface{}) {
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := s.templates.ExecuteTemplate(w, name, data); err != nil {
		log.Printf("template %s failed: %v", name, err)
		http.Error(w, "Une erreur est survenue", http.StatusInternalServerError)
	}
}

func (s *server) handleIndex(w http.ResponseWriter, r *http.Request) {
	query := strings.TrimSpace(r.URL.Query().Get("q"))
	artists := s.listArtists()
	filtered := filterArtists(artists, query)
	data := indexPageData{
		Query:   query,
		Count:   len(filtered),
		Total:   len(artists),
		Artists: filtered,
	}
	s.render(w, "index.html", data)
}

func (s *server) handleArtist(w http.ResponseWriter, r *http.Request) {
	idParam := r.URL.Query().Get("id")
	id, err := strconv.Atoi(idParam)
	if err != nil || id <= 0 {
		http.Error(w, "Identifiant invalide", http.StatusBadRequest)
		return
	}
	art, ok := s.findArtist(id)
	if !ok {
		http.NotFound(w, r)
		return
	}
	locDates := buildLocationDates(art.DatesLocations)
	data := artistPageData{
		Artist:        art,
		LocationDates: locDates,
	}
	s.render(w, "artist.html", data)
}

func (s *server) handleRefresh(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non supportée", http.StatusMethodNotAllowed)
		return
	}
	if err := s.refreshData(); err != nil {
		log.Printf("refresh failed: %v", err)
		http.Error(w, "Impossible d'actualiser les données", http.StatusBadGateway)
		return
	}
	http.Redirect(w, r, "/", http.StatusSeeOther)
}

func buildLocationDates(relations map[string][]string) []locationDates {
	if len(relations) == 0 {
		return nil
	}
	result := make([]locationDates, 0, len(relations))
	for location, dates := range relations {
		cleaned := cleanDates(dates)
		result = append(result, locationDates{
			Raw:    location,
			Pretty: formatLocation(location),
			Dates:  cleaned,
			Count:  len(cleaned),
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Pretty < result[j].Pretty
	})
	return result
}

func filterArtists(artists []artist, query string) []artist {
	if query == "" {
		return artists
	}
	lower := strings.ToLower(query)
	matches := make([]artist, 0, len(artists))
	for _, art := range artists {
		if artistMatches(art, lower) {
			matches = append(matches, art)
		}
	}
	return matches
}

func artistMatches(art artist, needle string) bool {
	if strings.Contains(strings.ToLower(art.Name), needle) {
		return true
	}
	for _, member := range art.Members {
		if strings.Contains(strings.ToLower(member), needle) {
			return true
		}
	}
	if strings.Contains(strconv.Itoa(art.CreationDate), needle) {
		return true
	}
	if strings.Contains(strings.ToLower(art.FirstAlbum), needle) {
		return true
	}
	for _, location := range art.Locations {
		if strings.Contains(strings.ToLower(location), needle) {
			return true
		}
	}
	return false
}

func fetchArtistsData(client *http.Client) ([]artist, error) {
	var artists []artist
	if err := fetchJSON(client, artistsEndpoint, &artists); err != nil {
		return nil, err
	}
	locMap, err := fetchLocations(client)
	if err != nil {
		return nil, err
	}
	dateMap, err := fetchDates(client)
	if err != nil {
		return nil, err
	}
	relMap, err := fetchRelations(client)
	if err != nil {
		return nil, err
	}
	for i := range artists {
		id := artists[i].ID
		artists[i].Locations = locMap[id]
		artists[i].ConcertDates = cleanDates(dateMap[id])
		artists[i].DatesLocations = relMap[id]
	}
	return artists, nil
}

func fetchLocations(client *http.Client) (map[int][]string, error) {
	var payload locationsPayload
	if err := fetchJSON(client, locationsEndpoint, &payload); err != nil {
		return nil, err
	}
	result := make(map[int][]string, len(payload.Index))
	for _, entry := range payload.Index {
		result[entry.ID] = entry.Locations
	}
	return result, nil
}

func fetchDates(client *http.Client) (map[int][]string, error) {
	var payload datesPayload
	if err := fetchJSON(client, datesEndpoint, &payload); err != nil {
		return nil, err
	}
	result := make(map[int][]string, len(payload.Index))
	for _, entry := range payload.Index {
		result[entry.ID] = entry.Dates
	}
	return result, nil
}

func fetchRelations(client *http.Client) (map[int]map[string][]string, error) {
	var payload relationsPayload
	if err := fetchJSON(client, relationsEndpoint, &payload); err != nil {
		return nil, err
	}
	result := make(map[int]map[string][]string, len(payload.Index))
	for _, entry := range payload.Index {
		result[entry.ID] = entry.DatesLocations
	}
	return result, nil
}

func fetchJSON(client *http.Client, url string, target interface{}) error {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("appel %s renvoie %d", url, resp.StatusCode)
	}
	return json.NewDecoder(resp.Body).Decode(target)
}

func cleanDates(values []string) []string {
	if len(values) == 0 {
		return values
	}
	result := make([]string, 0, len(values))
	for _, val := range values {
		val = strings.TrimSpace(val)
		val = strings.TrimPrefix(val, "*")
		if val != "" {
			result = append(result, val)
		}
	}
	return result
}

func formatDate(value string) string {
	parts := strings.Split(value, "-")
	if len(parts) != 3 {
		return value
	}
	return fmt.Sprintf("%s/%s/%s", parts[0], parts[1], parts[2])
}

func formatLocation(raw string) string {
	if raw == "" {
		return raw
	}
	parts := strings.Split(raw, "-")
	city := capitalize(strings.ReplaceAll(parts[0], "_", " "))
	if len(parts) == 1 {
		return city
	}
	country := strings.ToUpper(strings.ReplaceAll(parts[1], "_", " "))
	return fmt.Sprintf("%s (%s)", city, country)
}

func capitalize(input string) string {
	if input == "" {
		return input
	}
	words := strings.Fields(input)
	for i, w := range words {
		words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
	}
	return strings.Join(words, " ")
}

func main() {
	srv, err := newServer()
	if err != nil {
		log.Fatalf("initialisation impossible: %v", err)
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/", srv.handleIndex)
	mux.HandleFunc("/artist", srv.handleArtist)
	mux.HandleFunc(refreshPath, srv.handleRefresh)
	fileServer := http.FileServer(http.Dir("static"))
	mux.Handle(staticPrefix, http.StripPrefix(staticPrefix, fileServer))
	server := &http.Server{
		Addr:              serverAddress,
		Handler:           mux,
		ReadHeaderTimeout: readHeaderTimeout,
	}
	log.Printf("Serveur démarré sur http://localhost%s", serverAddress)
	if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		log.Fatalf("serveur arrêté: %v", err)
	}
}
