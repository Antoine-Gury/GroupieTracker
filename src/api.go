package src

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func FetchArtistsData(client *http.Client) ([]Artist, error) {
	var artists []Artist
	if err := FetchJSON(client, ArtistsEndpoint, &artists); err != nil {
		return nil, err
	}
	locMap, err := FetchLocations(client)
	if err != nil {
		return nil, err
	}
	dateMap, err := FetchDates(client)
	if err != nil {
		return nil, err
	}
	relMap, err := FetchRelations(client)
	if err != nil {
		return nil, err
	}
	for i := range artists {
		id := artists[i].ID
		artists[i].Locations = locMap[id]
		artists[i].ConcertDates = CleanDates(dateMap[id])
		artists[i].DatesLocations = relMap[id]
	}
	return artists, nil
}

func FetchLocations(client *http.Client) (map[int][]string, error) {
	var payload LocationsPayload
	if err := FetchJSON(client, LocationsEndpoint, &payload); err != nil {
		return nil, err
	}
	result := make(map[int][]string, len(payload.Index))
	for _, entry := range payload.Index {
		result[entry.ID] = entry.Locations
	}
	return result, nil
}

func FetchDates(client *http.Client) (map[int][]string, error) {
	var payload DatesPayload
	if err := FetchJSON(client, DatesEndpoint, &payload); err != nil {
		return nil, err
	}
	result := make(map[int][]string, len(payload.Index))
	for _, entry := range payload.Index {
		result[entry.ID] = entry.Dates
	}
	return result, nil
}

func FetchRelations(client *http.Client) (map[int]map[string][]string, error) {
	var payload RelationsPayload
	if err := FetchJSON(client, RelationsEndpoint, &payload); err != nil {
		return nil, err
	}
	result := make(map[int]map[string][]string, len(payload.Index))
	for _, entry := range payload.Index {
		result[entry.ID] = entry.DatesLocations
	}
	return result, nil
}

func FetchJSON(client *http.Client, url string, target interface{}) error {
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
