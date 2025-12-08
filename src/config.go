package src

import "time"

const (
	BaseAPI            = "https://groupietrackers.herokuapp.com/api"
	ArtistsEndpoint    = BaseAPI + "/artists"
	LocationsEndpoint  = BaseAPI + "/locations"
	DatesEndpoint      = BaseAPI + "/dates"
	RelationsEndpoint  = BaseAPI + "/relation"
	ServerAddress      = ":8080"
	ReadHeaderTimeout  = 5 * time.Second
	ClientTimeout      = 10 * time.Second
	RefreshPath        = "/refresh"
	StaticPrefix       = "/static/"
	TemplatesDirectory = "templates/*.html"
	
	// HTTPS Configuration
	CertFile = "server"
	KeyFile  = "server.key"
	
	// Session Configuration
	SessionName    = "groupietracker-session"
	SessionSecret  = "change-me-to-a-random-secret-key-minimum-32-characters"
	SessionMaxAge  = 86400 * 7 // 7 jours
)
