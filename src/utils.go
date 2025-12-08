package src

import (
	"sort"
	"strconv"
	"strings"
)

func BuildLocationDates(relations map[string][]string) []LocationDates {
	if len(relations) == 0 {
		return nil
	}
	result := make([]LocationDates, 0, len(relations))
	for location, dates := range relations {
		cleaned := CleanDates(dates)
		result = append(result, LocationDates{
			Raw:    location,
			Pretty: FormatLocation(location),
			Dates:  cleaned,
			Count:  len(cleaned),
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].Pretty < result[j].Pretty
	})
	return result
}

func FilterArtists(artists []Artist, query string) []Artist {
	if query == "" {
		return artists
	}
	lower := strings.ToLower(query)
	matches := make([]Artist, 0, len(artists))
	for _, art := range artists {
		if ArtistMatches(art, lower) {
			matches = append(matches, art)
		}
	}
	return matches
}

func ArtistMatches(art Artist, needle string) bool {
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

func CleanDates(values []string) []string {
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

func FormatDate(value string) string {
	parts := strings.Split(value, "-")
	if len(parts) != 3 {
		return value
	}
	return parts[0] + "/" + parts[1] + "/" + parts[2]
}

func FormatLocation(raw string) string {
	if raw == "" {
		return raw
	}
	parts := strings.Split(raw, "-")
	city := Capitalize(strings.ReplaceAll(parts[0], "_", " "))
	if len(parts) == 1 {
		return city
	}
	country := strings.ToUpper(strings.ReplaceAll(parts[1], "_", " "))
	return city + " (" + country + ")"
}

func Capitalize(input string) string {
	if input == "" {
		return input
	}
	words := strings.Fields(input)
	for i, w := range words {
		words[i] = strings.ToUpper(w[:1]) + strings.ToLower(w[1:])
	}
	return strings.Join(words, " ")
}
