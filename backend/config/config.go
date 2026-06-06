package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	HTTPAddr                string
	DatabasePath            string
	ImageCacheDir           string
	ImageCacheRetentionDays int
	ListingResultLimit      int
	EbayClientID            string
	EbayClientSecret        string
	EbayMarketplaceIDs      []string
	MatchConfirmedThreshold float64
	MatchPossibleThreshold  float64
}

func Load() Config {
	return Config{
		HTTPAddr:                env("HTTP_ADDR", ":8080"),
		DatabasePath:            env("DATABASE_PATH", "./data/app.db"),
		ImageCacheDir:           env("IMAGE_CACHE_DIR", "./data/images"),
		ImageCacheRetentionDays: envInt("IMAGE_CACHE_RETENTION_DAYS", 7),
		ListingResultLimit:      envInt("LISTING_RESULT_LIMIT_PER_MARKETPLACE", 100),
		EbayClientID:            os.Getenv("EBAY_CLIENT_ID"),
		EbayClientSecret:        os.Getenv("EBAY_CLIENT_SECRET"),
		EbayMarketplaceIDs:      splitCSV(env("EBAY_MARKETPLACE_IDS", "EBAY_IT")),
		MatchConfirmedThreshold: envFloat("MATCH_CONFIRMED_THRESHOLD", 0.80),
		MatchPossibleThreshold:  envFloat("MATCH_POSSIBLE_THRESHOLD", 0.45),
	}
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func envInt(key string, fallback int) int {
	value, err := strconv.Atoi(env(key, strconv.Itoa(fallback)))
	if err != nil {
		return fallback
	}
	return value
}

func envFloat(key string, fallback float64) float64 {
	value, err := strconv.ParseFloat(env(key, strconv.FormatFloat(fallback, 'f', -1, 64)), 64)
	if err != nil {
		return fallback
	}
	return value
}

func splitCSV(value string) []string {
	var values []string
	for _, item := range strings.Split(value, ",") {
		if item = strings.TrimSpace(item); item != "" {
			values = append(values, item)
		}
	}
	return values
}
