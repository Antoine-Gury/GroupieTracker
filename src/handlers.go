package src

import (
	"encoding/json"
	"fmt"
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
	
	// Géocoder les emplacements pour la carte
	locationsCoords := GeocodeLocations(art.Locations, art.DatesLocations)
	
	data := ArtistPageData{
		Artist:          art,
		LocationDates:   locDates,
		LocationsCoords: locationsCoords,
		PayPalClientID:  PayPalClientID,
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

func (s *Server) HandleGeocode(w http.ResponseWriter, r *http.Request) {
	// API endpoint pour géocoder une adresse (optionnel, pour utilisation AJAX)
	if r.Method != http.MethodGet {
		http.Error(w, "Méthode non supportée", http.StatusMethodNotAllowed)
		return
	}
	
	address := r.URL.Query().Get("address")
	if address == "" {
		http.Error(w, "Paramètre 'address' manquant", http.StatusBadRequest)
		return
	}
	
	coords, err := GeocodeLocation(address)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(coords)
}

func (s *Server) HandleCreateOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non supportée", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		ArtistID    int     `json:"artist_id"`
		Location    string  `json:"location"`
		Date        string  `json:"date"`
		Quantity    int     `json:"quantity"`
		Amount      float64 `json:"amount"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Requête invalide", http.StatusBadRequest)
		return
	}

	if req.Quantity <= 0 {
		req.Quantity = 1
	}
	if req.Amount <= 0 {
		req.Amount = DefaultTicketPrice * float64(req.Quantity)
	}

	// Récupérer les informations de l'artiste
	art, ok := s.FindArtist(req.ArtistID)
	if !ok {
		http.Error(w, "Artiste non trouvé", http.StatusNotFound)
		return
	}

	// Construire les URLs de retour
	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, r.Host)
	returnURL := fmt.Sprintf("%s/paypal/success", baseURL)
	cancelURL := fmt.Sprintf("%s/artist?id=%d", baseURL, req.ArtistID)

	description := fmt.Sprintf("Billet pour %s - %s (%s)", art.Name, req.Location, req.Date)

	// Créer la commande PayPal
	order, err := CreatePayPalOrder(s.client, req.Amount, description, returnURL, cancelURL)
	if err != nil {
		log.Printf("Erreur création commande PayPal: %v", err)
		http.Error(w, "Erreur lors de la création de la commande", http.StatusInternalServerError)
		return
	}

	// Trouver l'URL d'approbation
	var approveURL string
	for _, link := range order.Links {
		if link.Rel == "approve" {
			approveURL = link.Href
			break
		}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"order_id":    order.ID,
		"status":      order.Status,
		"approve_url": approveURL,
	})
}

func (s *Server) HandleCaptureOrder(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non supportée", http.StatusMethodNotAllowed)
		return
	}

	var req struct {
		OrderID string `json:"order_id"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Requête invalide", http.StatusBadRequest)
		return
	}

	if req.OrderID == "" {
		http.Error(w, "order_id manquant", http.StatusBadRequest)
		return
	}

	// Capturer le paiement
	capture, err := CapturePayPalOrder(s.client, req.OrderID)
	if err != nil {
		log.Printf("Erreur capture PayPal: %v", err)
		http.Error(w, "Erreur lors de la capture du paiement", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(capture)
}

func (s *Server) HandlePayPalSuccess(w http.ResponseWriter, r *http.Request) {
	// PayPal redirige avec 'token' dans les paramètres, pas 'order_id'
	token := r.URL.Query().Get("token")
	orderID := r.URL.Query().Get("order_id")
	
	// Si on a un token, on doit le convertir en order_id (le token EST l'order_id)
	if token != "" && orderID == "" {
		orderID = token
	}
	
	if orderID == "" {
		// Essayer de capturer depuis la session ou autres moyens
		http.Error(w, "Informations de commande manquantes", http.StatusBadRequest)
		return
	}

	// Capturer automatiquement le paiement
	capture, err := CapturePayPalOrder(s.client, orderID)
	if err != nil {
		log.Printf("Erreur capture automatique PayPal: %v", err)
		// Continuer quand même pour afficher la page
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	
	statusMessage := "traitée avec succès"
	if capture != nil && capture.Status == "COMPLETED" {
		statusMessage = "payée et confirmée"
	} else if err != nil {
		statusMessage = "en attente de confirmation"
	}
	
	fmt.Fprintf(w, `
<!doctype html>
<html lang="fr">
<head>
	<meta charset="utf-8">
	<meta name="viewport" content="width=device-width, initial-scale=1">
	<title>Paiement réussi - Groupie Tracker</title>
	<link rel="stylesheet" href="/static/CSS/styles.css">
</head>
<body>
	<main class="container" style="padding: 4rem 2rem; text-align: center;">
		<h1 style="color: var(--gold); margin-bottom: 1rem;">✅ Paiement réussi !</h1>
		<p style="color: var(--muted); margin-bottom: 2rem;">Votre commande #%s a été %s.</p>
		<p style="color: var(--foreground); margin-bottom: 2rem;">Vous recevrez un email de confirmation sous peu.</p>
		<a href="/" style="display: inline-block; padding: 0.75rem 2rem; background: var(--gradient-gold); color: var(--bg); text-decoration: none; border-radius: 0.75rem; font-weight: 600; margin-top: 1rem;">Retour à l'accueil</a>
	</main>
</body>
</html>
	`, orderID, statusMessage)
}

func (s *Server) HandleLogin(w http.ResponseWriter, r *http.Request) {
	s.Render(w, "login.html", nil)
}

func (s *Server) HandleRegister(w http.ResponseWriter, r *http.Request) {
	s.Render(w, "register.html", nil)
}
