package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"pokemon-binder-finder/config"
	"pokemon-binder-finder/ebay"
	"pokemon-binder-finder/repository"
	"pokemon-binder-finder/service"
	"pokemon-binder-finder/vision"
	"pokemon-binder-finder/web"
)

func main() {
	cfg := config.Load()
	if err := os.MkdirAll(filepath.Dir(cfg.DatabasePath), 0o755); err != nil {
		log.Fatal(err)
	}
	store, err := repository.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()
	catalog := service.NewCatalogService(store)
	imageCache, err := repository.NewImageCache(cfg.ImageCacheDir)
	if err != nil {
		log.Fatal(err)
	}
	searches := service.NewSearchService(
		store,
		catalog,
		ebay.New(cfg.EbayClientID, cfg.EbayClientSecret),
		imageCache,
		service.NewVisionService(vision.NewMatcher(), store, cfg.MatchConfirmedThreshold, cfg.MatchPossibleThreshold),
		service.SearchConfig{
			Marketplaces: cfg.EbayMarketplaceIDs,
			ResultLimit:  cfg.ListingResultLimit,
		},
	)
	log.Printf("listening on %s", cfg.HTTPAddr)
	router := web.NewRouter(catalog, searches, cfg.ImageCacheDir)
	if err := http.ListenAndServe(cfg.HTTPAddr, router); err != nil {
		log.Fatal(err)
	}
}
