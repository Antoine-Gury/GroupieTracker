package src

import (
	"os"
	"time"
)

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
	DefaultDBHost      = "127.0.0.1"
	DefaultDBPort      = "3306"
	DefaultDBUser      = "root"
	DefaultDBPassword  = ""
	DefaultDBName      = "groupietracker"
	CertFile           = "server"
	KeyFile            = "server.key"
	SessionName        = "groupietracker-session"
	SessionSecret      = "change-me-to-a-random-secret-key-minimum-32-characters"
	SessionMaxAge      = 86400 * 7
	DefaultTicketPrice = 50.00
)

var (
	PayPalClientID = getEnvOrDefault("PAYPAL_CLIENT_ID", "AYZTk4mq-RDQ1wx_cV8_OL8x6Z7DLwdIlVgh9VA1-hxIpVl90W0CsIx0LOPnPJhbZUUXtMYGl3005mPi")
	PayPalSecret   = getEnvOrDefault("PAYPAL_SECRET", "EN_zEbAcKwJluLRQOUJEZbqUmVgRFYxtuy3gD5WoTuLozW8ptEQyp_6uqd3-_6NGQUQxI3h7-88jc-gq")
	PayPalMode     = getEnvOrDefault("PAYPAL_MODE", "sandbox")
	PayPalBaseURL  = "https://api-m.sandbox.paypal.com"
)

func init() {
	if PayPalMode == "live" {
		PayPalBaseURL = "https://api-m.paypal.com"
	}
}

func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}
