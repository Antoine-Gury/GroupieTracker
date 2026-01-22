package src

type Artist struct {
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

type LocationsPayload struct {
	Index []struct {
		ID        int      `json:"id"`
		Locations []string `json:"locations"`
	} `json:"index"`
}

type DatesPayload struct {
	Index []struct {
		ID    int      `json:"id"`
		Dates []string `json:"dates"`
	} `json:"index"`
}

type RelationsPayload struct {
	Index []struct {
		ID             int                 `json:"id"`
		DatesLocations map[string][]string `json:"datesLocations"`
	} `json:"index"`
}

type IndexPageData struct {
	Query   string
	Count   int
	Total   int
	Artists []Artist
	User    *UserProfile // Informations de l'utilisateur connecté
}

type UserProfile struct {
	ID          int
	Username    string
	Email       string
	Pseudo      string
	Bio         string
	PhotoProfil string
	Role        string
}

type AdminUsersPageData struct {
	Users []UserDisplay
	User  *UserProfile // Utilisateur connecté (admin)
}

type UserDisplay struct {
	ID          int
	Username    string
	Email       string
	Pseudo      string
	Bio         string
	PhotoProfil string
	Role        string
	CreatedAt   string
}

type LocationDates struct {
	Raw    string
	Pretty string
	Dates  []string
	Count  int
}

type ArtistPageData struct {
	Artist          Artist
	LocationDates   []LocationDates
	LocationsCoords []LocationWithCoords
	PayPalClientID  string
}

type LoginPageData struct {
	Error   string
	Message string
}
