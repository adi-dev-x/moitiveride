package config

import (
	"fmt"
	"os"
	"strconv"
)

type Config struct {
	// Server
	Port string

	// Database
	DatabaseURL string

	// Redis
	RedisAddr     string
	RedisPassword string

	// Routing engine
	DepotLat float64
	DepotLng float64

	// Google Maps (optional — uses offline fallback when empty)
	GoogleMapsAPIKey string

	// Feature flags
	LastFemaleDropPolicy bool
}

func Load() (*Config, error) {
	depotLat, err := parseFloat("DEPOT_LAT", 12.9716) // default: Bangalore city centre
	if err != nil {
		return nil, err
	}
	depotLng, err := parseFloat("DEPOT_LNG", 77.5946)
	if err != nil {
		return nil, err
	}

	dbURL := os.Getenv("DATABASE_URL")
	if dbURL == "" {
		dbURL = "postgres://postgres:postgres@localhost:5432/motive_ride?sslmode=disable"
	}

	return &Config{
		Port:                 getEnv("PORT", "8080"),
		DatabaseURL:          dbURL,
		RedisAddr:            getEnv("REDIS_ADDR", "localhost:6379"),
		RedisPassword:        os.Getenv("REDIS_PASSWORD"),
		DepotLat:             depotLat,
		DepotLng:             depotLng,
		GoogleMapsAPIKey:     os.Getenv("GOOGLE_MAPS_API_KEY"),
		LastFemaleDropPolicy: getEnv("LAST_FEMALE_DROP_POLICY", "false") == "true",
	}, nil
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func parseFloat(key string, fallback float64) (float64, error) {
	v := os.Getenv(key)
	if v == "" {
		return fallback, nil
	}
	f, err := strconv.ParseFloat(v, 64)
	if err != nil {
		return 0, fmt.Errorf("config %s: %w", key, err)
	}
	return f, nil
}
