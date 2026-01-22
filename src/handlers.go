package src

import (
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

func (s *Server) HandleIndex(w http.ResponseWriter, r *http.Request) {
	var userProfile *UserProfile
	if IsAuthenticated(r) {
		session, _ := GetSession(r)
		if userID, ok := session.Values["user_id"].(int); ok {
			user, err := GetUserByID(DB, userID)
			if err == nil {
				userProfile = &UserProfile{
					ID:          user.ID,
					Username:    user.Username,
					Email:       user.Email,
					Pseudo:      getStringValue(user.Pseudo),
					Bio:         getStringValue(user.Bio),
					PhotoProfil: getStringValue(user.PhotoProfil),
					Role:        user.Role,
				}
			}
		}
	}

	query := strings.TrimSpace(r.URL.Query().Get("q"))
	artists := s.ListArtists()
	filtered := FilterArtists(artists, query)
	data := IndexPageData{
		Query:   query,
		Count:   len(filtered),
		Total:   len(artists),
		Artists: filtered,
		User:    userProfile,
	}
	s.Render(w, "index.html", data)
}

func (s *Server) HandleProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Méthode non supportée", http.StatusMethodNotAllowed)
		return
	}

	session, err := GetSession(r)
	if err != nil {
		http.Error(w, "Session indisponible", http.StatusUnauthorized)
		return
	}
	userID, ok := session.Values["user_id"].(int)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	user, err := GetUserByID(DB, userID)
	if err != nil {
		http.Error(w, "Utilisateur introuvable", http.StatusNotFound)
		return
	}

	data := IndexPageData{
		User: &UserProfile{
			ID:          user.ID,
			Username:    user.Username,
			Email:       user.Email,
			Pseudo:      getStringValue(user.Pseudo),
			Bio:         getStringValue(user.Bio),
			PhotoProfil: getStringValue(user.PhotoProfil),
			Role:        user.Role,
		},
	}

	s.Render(w, "profile.html", data)
}

func getStringValue(ns sql.NullString) string {
	if ns.Valid {
		return ns.String
	}
	return ""
}

func (s *Server) HandleRoot(w http.ResponseWriter, r *http.Request) {
	if IsAuthenticated(r) {
		http.Redirect(w, r, "/home", http.StatusSeeOther)
		return
	}
	http.Redirect(w, r, "/login", http.StatusSeeOther)
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
		ArtistID int     `json:"artist_id"`
		Location string  `json:"location"`
		Date     string  `json:"date"`
		Quantity int     `json:"quantity"`
		Amount   float64 `json:"amount"`
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

	art, ok := s.FindArtist(req.ArtistID)
	if !ok {
		http.Error(w, "Artiste non trouvé", http.StatusNotFound)
		return
	}

	scheme := "https"
	if r.TLS == nil {
		scheme = "http"
	}
	baseURL := fmt.Sprintf("%s://%s", scheme, r.Host)
	returnURL := fmt.Sprintf("%s/paypal/success", baseURL)
	cancelURL := fmt.Sprintf("%s/artist?id=%d", baseURL, req.ArtistID)

	description := fmt.Sprintf("Billet pour %s - %s (%s)", art.Name, req.Location, req.Date)

	order, err := CreatePayPalOrder(s.client, req.Amount, description, returnURL, cancelURL)
	if err != nil {
		log.Printf("Erreur création commande PayPal: %v", err)
		http.Error(w, "Erreur lors de la création de la commande", http.StatusInternalServerError)
		return
	}

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
	token := r.URL.Query().Get("token")
	orderID := r.URL.Query().Get("order_id")

	if token != "" && orderID == "" {
		orderID = token
	}

	if orderID == "" {
		http.Error(w, "Informations de commande manquantes", http.StatusBadRequest)
		return
	}

	capture, err := CapturePayPalOrder(s.client, orderID)
	if err != nil {
		log.Printf("Erreur capture automatique PayPal: %v", err)
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
	if r.Method == http.MethodGet && IsAuthenticated(r) {
		http.Redirect(w, r, "/home", http.StatusSeeOther)
		return
	}
	if r.Method == http.MethodGet {
		s.Render(w, "login.html", LoginPageData{})
		return
	}

	if err := r.ParseForm(); err != nil {
		s.Render(w, "login.html", LoginPageData{
			Error: "Erreur lors du traitement du formulaire",
		})
		return
	}

	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")

	if email == "" || password == "" {
		s.Render(w, "login.html", LoginPageData{
			Error: "Veuillez remplir tous les champs",
		})
		return
	}

	user, err := GetUserByEmail(DB, email)
	if err != nil {
		s.Render(w, "login.html", LoginPageData{
			Error: "Email ou mot de passe incorrect",
		})
		return
	}
	if err := checkPassword(user.PasswordHash, password); err != nil {
		s.Render(w, "login.html", LoginPageData{
			Error: "Email ou mot de passe incorrect",
		})
		return
	}

	session, err := GetSession(r)
	if err != nil {
		s.Render(w, "login.html", LoginPageData{
			Error: "Erreur de session, veuillez réessayer",
		})
		return
	}
	session.Values["user_id"] = user.ID
	session.Values["email"] = user.Email
	session.Values["username"] = user.Username
	session.Values["role"] = user.Role
	if err := SaveSession(w, r, session); err != nil {
		s.Render(w, "login.html", LoginPageData{
			Error: "Impossible de sauvegarder la session",
		})
		return
	}

	http.Redirect(w, r, "/home", http.StatusSeeOther)
}

func (s *Server) HandleRegister(w http.ResponseWriter, r *http.Request) {
	if r.Method == http.MethodGet {
		s.Render(w, "register.html", nil)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Requête invalide", http.StatusBadRequest)
		return
	}

	email := strings.TrimSpace(r.FormValue("email"))
	password := r.FormValue("password")
	confirm := r.FormValue("confirm_password")

	if password != confirm {
		http.Error(w, "Les mots de passe ne correspondent pas", http.StatusBadRequest)
		return
	}

	if err := CreateUser(DB, email, password); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	user, err := GetUserByEmail(DB, email)
	if err != nil {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}
	session, err := GetSession(r)
	if err == nil {
		session.Values["user_id"] = user.ID
		session.Values["email"] = user.Email
		session.Values["username"] = user.Username
		session.Values["role"] = user.Role
		_ = SaveSession(w, r, session)
	}

	http.Redirect(w, r, "/home", http.StatusSeeOther)
}

func (s *Server) HandleLogout(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non supportée", http.StatusMethodNotAllowed)
		return
	}

	session, err := GetSession(r)
	if err == nil {
		session.Values = make(map[interface{}]interface{})
		session.Options.MaxAge = -1
		_ = SaveSession(w, r, session)
	}

	http.Redirect(w, r, "/login", http.StatusSeeOther)
}

func (s *Server) HandleUpdateProfile(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non supportée", http.StatusMethodNotAllowed)
		return
	}

	session, err := GetSession(r)
	if err != nil {
		http.Error(w, "Session indisponible", http.StatusUnauthorized)
		return
	}

	userID, ok := session.Values["user_id"].(int)
	if !ok {
		http.Error(w, "Non authentifié", http.StatusUnauthorized)
		return
	}

	uploadDir := "static/uploads"
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		log.Printf("Erreur création dossier uploads: %v", err)
	}

	if err := r.ParseMultipartForm(10 << 20); err != nil {
		http.Error(w, "Erreur parsing formulaire", http.StatusBadRequest)
		return
	}

	pseudo := strings.TrimSpace(r.FormValue("pseudo"))
	bio := strings.TrimSpace(r.FormValue("bio"))
	photoProfil := ""

	// Gérer l'upload de la photo
	file, handler, err := r.FormFile("photo_profil")
	if err == nil && handler != nil {
		defer file.Close()

		// Générer un nom de fichier unique
		ext := filepath.Ext(handler.Filename)
		filename := fmt.Sprintf("profile_%d_%d%s", userID, time.Now().Unix(), ext)
		filepath := filepath.Join(uploadDir, filename)

		// Créer le fichier
		dst, err := os.Create(filepath)
		if err != nil {
			log.Printf("Erreur création fichier: %v", err)
		} else {
			defer dst.Close()
			if _, err := io.Copy(dst, file); err != nil {
				log.Printf("Erreur copie fichier: %v", err)
			} else {
				photoProfil = "/static/uploads/" + filename
			}
		}
	}

	if photoProfil == "" {
		user, err := GetUserByID(DB, userID)
		if err == nil && user.PhotoProfil.Valid {
			photoProfil = user.PhotoProfil.String
		}
	}

	// Mettre à jour le profil
	if err := UpdateUserProfile(DB, userID, pseudo, bio, photoProfil); err != nil {
		log.Printf("Erreur mise à jour profil: %v", err)
		http.Error(w, "Erreur lors de la mise à jour du profil", http.StatusInternalServerError)
		return
	}

	if pseudo != "" {
		session.Values["username"] = pseudo
		_ = SaveSession(w, r, session)
	}

	http.Redirect(w, r, "/home", http.StatusSeeOther)
}

func (s *Server) HandleAdminUsers(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, "Méthode non supportée", http.StatusMethodNotAllowed)
		return
	}

	session, err := GetSession(r)
	if err != nil {
		http.Error(w, "Session indisponible", http.StatusUnauthorized)
		return
	}
	adminID, ok := session.Values["user_id"].(int)
	if !ok {
		http.Redirect(w, r, "/login", http.StatusSeeOther)
		return
	}

	admin, err := GetUserByID(DB, adminID)
	if err != nil {
		http.Error(w, "Utilisateur introuvable", http.StatusNotFound)
		return
	}

	users, err := GetAllUsers(DB)
	if err != nil {
		log.Printf("Erreur récupération utilisateurs: %v", err)
		http.Error(w, "Erreur lors de la récupération des utilisateurs", http.StatusInternalServerError)
		return
	}

	// Convertir les User en UserDisplay pour faciliter l'affichage dans les templates
	usersDisplay := make([]UserDisplay, len(users))
	for i, u := range users {
		usersDisplay[i] = UserDisplay{
			ID:          u.ID,
			Username:    u.Username,
			Email:       u.Email,
			Pseudo:      getStringValue(u.Pseudo),
			Bio:         getStringValue(u.Bio),
			PhotoProfil: getStringValue(u.PhotoProfil),
			Role:        u.Role,
			CreatedAt:   u.CreatedAt.Format("02/01/2006"),
		}
	}

	data := AdminUsersPageData{
		Users: usersDisplay,
		User: &UserProfile{
			ID:          admin.ID,
			Username:    admin.Username,
			Email:       admin.Email,
			Pseudo:      getStringValue(admin.Pseudo),
			Bio:         getStringValue(admin.Bio),
			PhotoProfil: getStringValue(admin.PhotoProfil),
			Role:        admin.Role,
		},
	}

	s.Render(w, "admin-users.html", data)
}

// HandleAdminUpdateUserRole met à jour le rôle d'un utilisateur (admin seulement)
func (s *Server) HandleAdminUpdateUserRole(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non supportée", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Requête invalide", http.StatusBadRequest)
		return
	}

	userIDStr := r.FormValue("user_id")
	role := strings.TrimSpace(r.FormValue("role"))

	userID, err := strconv.Atoi(userIDStr)
	if err != nil || userID <= 0 {
		http.Error(w, "ID utilisateur invalide", http.StatusBadRequest)
		return
	}

	if role != "user" && role != "admin" {
		http.Error(w, "Rôle invalide", http.StatusBadRequest)
		return
	}

	// Empêcher un admin de se retirer ses propres droits
	session, _ := GetSession(r)
	if currentUserID, ok := session.Values["user_id"].(int); ok && currentUserID == userID && role == "user" {
		http.Error(w, "Vous ne pouvez pas retirer vos propres droits administrateur", http.StatusForbidden)
		return
	}

	if err := UpdateUserRole(DB, userID, role); err != nil {
		log.Printf("Erreur mise à jour rôle: %v", err)
		http.Error(w, "Erreur lors de la mise à jour du rôle", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

// HandleAdminDeleteUser supprime un utilisateur (admin seulement)
func (s *Server) HandleAdminDeleteUser(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "Méthode non supportée", http.StatusMethodNotAllowed)
		return
	}

	if err := r.ParseForm(); err != nil {
		http.Error(w, "Requête invalide", http.StatusBadRequest)
		return
	}

	userIDStr := r.FormValue("user_id")
	userID, err := strconv.Atoi(userIDStr)
	if err != nil || userID <= 0 {
		http.Error(w, "ID utilisateur invalide", http.StatusBadRequest)
		return
	}

	// Empêcher un admin de se supprimer lui-même
	session, _ := GetSession(r)
	if currentUserID, ok := session.Values["user_id"].(int); ok && currentUserID == userID {
		http.Error(w, "Vous ne pouvez pas supprimer votre propre compte", http.StatusForbidden)
		return
	}

	if err := DeleteUser(DB, userID); err != nil {
		log.Printf("Erreur suppression utilisateur: %v", err)
		http.Error(w, "Erreur lors de la suppression de l'utilisateur", http.StatusInternalServerError)
		return
	}

	http.Redirect(w, r, "/admin/users", http.StatusSeeOther)
}

// HandleLegalConditions affiche la page des conditions générales de vente
func (s *Server) HandleLegalConditions(w http.ResponseWriter, r *http.Request) {
	s.Render(w, "legal-conditions.html", nil)
}

// HandleLegalPrivacy affiche la page de politique de confidentialité
func (s *Server) HandleLegalPrivacy(w http.ResponseWriter, r *http.Request) {
	s.Render(w, "legal-privacy.html", nil)
}

// HandleLegalCookies affiche la page de politique des cookies
func (s *Server) HandleLegalCookies(w http.ResponseWriter, r *http.Request) {
	s.Render(w, "legal-cookies.html", nil)
}

// HandleLegalMentions affiche la page des mentions légales
func (s *Server) HandleLegalMentions(w http.ResponseWriter, r *http.Request) {
	s.Render(w, "legal-mentions.html", nil)
}
