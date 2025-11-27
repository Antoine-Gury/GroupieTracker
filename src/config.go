package src

import "time"

const (
	BaseAPI            = "https://groupietrackers.herokuapp.com/api"
	ArtistsEndpoint    = BaseAPI + "/artists"
	LocationsEndpoint  = BaseAPI + "/locations"
	DatesEndpoint      = BaseAPI + "/dates"
	RelationsEndpoint  = BaseAPI + "/relation"
	ServerAddress      = ":3030"
	ReadHeaderTimeout  = 5 * time.Second
	ClientTimeout      = 10 * time.Second
	RefreshPath        = "/refresh"
	StaticPrefix       = "/static/"
	TemplatesDirectory = "templates/*.html"
)
