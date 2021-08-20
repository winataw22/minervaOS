package geoip

import (
	"encoding/json"
	"errors"
	"net/http"
)

// Location holds the result of a geoip request
type Location struct {
	Longitute   float64 `json:"longitude"`
	Latitude    float64 `json:"latitude"`
	Continent   string  `json:"continent"`
	Country     string  `json:"country_name"`
	CountryCode string  `json:"country_code"`
	City        string  `json:"city_name"`
}

// Fetch retrieves the location of the system calling this function
func Fetch() (Location, error) {
	l := Location{
		Longitute: 0.0,
		Latitude:  0.0,
		Continent: "Unknown",
		Country:   "Unknown",
		City:      "Unknown",
	}

	resp, err := http.Get("https://geoip.grid.tf")
	if err != nil {
		return l, err
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		return l, errors.New("error fetch location")
	}

	if err := json.NewDecoder(resp.Body).Decode(&l); err != nil {
		return l, err
	}

	return l, err
}
