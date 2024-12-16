# Geoip Module

The `geoip` module provides a simple way to determine the geographical location of the system executing the code.
This includes details such as longitude, latitude, country, city, and continent.

## Features

- **Failover Mechanism:** The module attempts to fetch location data from multiple services to ensure high availability. If one URL fails, it logs the error and retries with the next URL.
- **Structured Location Data:** Returns structured data in a `Location` struct for easy integration.

```go
type Location struct {
	Longitude   float64 `json:"longitude"`
	Latitude    float64 `json:"latitude"`
	Continent   string  `json:"continent"`
	Country     string  `json:"country_name"`
	CountryCode string  `json:"country_code"`
	City        string  `json:"city_name"`
}
```

## Usage

The module provides a single exported function: `Fetch`. 

The `Fetch` function retrieves the geographical location of the system calling the function. 
