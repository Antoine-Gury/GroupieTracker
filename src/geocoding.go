package src

import (
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"
)

// Coordinates représente des coordonnées géographiques
type Coordinates struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

// LocationWithCoords représente un lieu avec ses coordonnées
type LocationWithCoords struct {
	Location    string     `json:"location"`
	Coordinates Coordinates `json:"coordinates"`
	Dates       []string   `json:"dates"`
}

var (
	// Cache pour stocker les coordonnées géocodées
	geocodeCache = make(map[string]Coordinates)
	cacheMutex   sync.RWMutex
)

// GeocodeLocation convertit une adresse en coordonnées géographiques
// Utilise Nominatim (OpenStreetMap) qui est gratuit et ne nécessite pas de clé API
func GeocodeLocation(address string) (Coordinates, error) {
	// Vérifier le cache d'abord
	cacheMutex.RLock()
	if coords, exists := geocodeCache[address]; exists {
		cacheMutex.RUnlock()
		return coords, nil
	}
	cacheMutex.RUnlock()

	// Nettoyer l'adresse (remplacer _ par des espaces, formater)
	cleanAddr := CleanAddressForGeocoding(address)
	
	// Construire l'URL de l'API Nominatim
	baseURL := "https://nominatim.openstreetmap.org/search"
	params := url.Values{}
	params.Set("q", cleanAddr)
	params.Set("format", "json")
	params.Set("limit", "1")
	
	reqURL := fmt.Sprintf("%s?%s", baseURL, params.Encode())
	
	// Créer la requête HTTP
	req, err := http.NewRequest("GET", reqURL, nil)
	if err != nil {
		return Coordinates{}, fmt.Errorf("erreur création requête: %v", err)
	}
	
	// Ajouter un User-Agent (requis par Nominatim)
	req.Header.Set("User-Agent", "GroupieTracker/1.0")
	
	// Client HTTP avec timeout
	client := &http.Client{
		Timeout: 10 * time.Second,
	}
	
	// Effectuer la requête
	resp, err := client.Do(req)
	if err != nil {
		return Coordinates{}, fmt.Errorf("erreur requête geocoding: %v", err)
	}
	defer resp.Body.Close()
	
	// Vérifier le code de statut
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return Coordinates{}, fmt.Errorf("erreur HTTP %d: %s", resp.StatusCode, string(body))
	}
	
	// Lire la réponse JSON
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return Coordinates{}, fmt.Errorf("erreur lecture réponse: %v", err)
	}
	
	// Parser la réponse JSON
	var results []struct {
		Lat string `json:"lat"`
		Lon string `json:"lon"`
	}
	
	if err := json.Unmarshal(body, &results); err != nil {
		return Coordinates{}, fmt.Errorf("erreur parsing JSON: %v", err)
	}
	
	// Vérifier si des résultats ont été trouvés
	if len(results) == 0 {
		log.Printf("Aucun résultat de geocoding pour: %s", address)
		return Coordinates{}, fmt.Errorf("adresse non trouvée: %s", address)
	}
	
	// Convertir les chaînes en float64
	var coords Coordinates
	if _, err := fmt.Sscanf(results[0].Lat, "%f", &coords.Latitude); err != nil {
		return Coordinates{}, fmt.Errorf("erreur parsing latitude: %v", err)
	}
	if _, err := fmt.Sscanf(results[0].Lon, "%f", &coords.Longitude); err != nil {
		return Coordinates{}, fmt.Errorf("erreur parsing longitude: %v", err)
	}
	
	// Stocker dans le cache
	cacheMutex.Lock()
	geocodeCache[address] = coords
	cacheMutex.Unlock()
	
	log.Printf("Geocodé: %s -> (%.6f, %.6f)", address, coords.Latitude, coords.Longitude)
	return coords, nil
}

// CleanAddressForGeocoding nettoie et formate l'adresse pour le geocoding
func CleanAddressForGeocoding(address string) string {
	// Remplacer les underscores par des espaces
	cleaned := strings.ReplaceAll(address, "_", " ")
	
	// Supprimer les espaces multiples
	cleaned = strings.TrimSpace(cleaned)
	
	// Remplacer les séparateurs multiples par un seul espace
	for strings.Contains(cleaned, "  ") {
		cleaned = strings.ReplaceAll(cleaned, "  ", " ")
	}
	
	return cleaned
}

// GeocodeLocations geocode plusieurs adresses en parallèle
func GeocodeLocations(locations []string, relations map[string][]string) []LocationWithCoords {
	if len(relations) == 0 {
		return nil
	}
	
	resultsChan := make(chan LocationWithCoords, len(relations))
	var wg sync.WaitGroup
	
	// Geocoder en parallèle (avec limite de concurrence)
	semaphore := make(chan struct{}, 5) // Maximum 5 requêtes simultanées
	
	for location, dates := range relations {
		wg.Add(1)
		go func(loc string, dts []string) {
			defer wg.Done()
			
			// Acquérir un slot
			semaphore <- struct{}{}
			defer func() { <-semaphore }()
			
			coords, err := GeocodeLocation(loc)
			if err != nil {
				log.Printf("Erreur geocoding pour %s: %v", loc, err)
				// Continuer même en cas d'erreur (coordonnées à 0,0)
			}
			
			resultsChan <- LocationWithCoords{
				Location:    FormatLocation(loc),
				Coordinates: coords,
				Dates:       CleanDates(dts),
			}
		}(location, dates)
	}
	
	// Fermer le channel une fois toutes les goroutines terminées
	go func() {
		wg.Wait()
		close(resultsChan)
	}()
	
	// Collecter les résultats
	results := make([]LocationWithCoords, 0, len(relations))
	for res := range resultsChan {
		results = append(results, res)
	}
	
	return results
}

