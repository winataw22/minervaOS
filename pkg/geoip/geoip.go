package geoip

import (
	"encoding/json"
	"errors"
	"net/http"
)

// Location holds the result of a geoip request
type Location struct {
	Longitute float64 `json:"longitude,omitempty"`
	Latitude  float64 `json:"latitude,omitempty"`
	Continent string  `json:"continent,omitempty"`
	Country   string  `json:"country_name,omitempty"`
	City      string  `json:"city,omitempty"`
}

// Fetch retrieves the location of the system calling this function
func Fetch() (Location, error) {
	l := Location{
		Longitute: 0.0,
		Latitude:  0.0,
		Continent: "Uknown",
		Country:   "Uknown",
		City:      "Uknown",
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
