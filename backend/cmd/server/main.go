package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"pokemon-binder-finder/internal/adapters/in/httpapi"
	"pokemon-binder-finder/internal/adapters/out/assets/filesystem"
	"pokemon-binder-finder/internal/adapters/out/catalog/pokemontcg"
	"pokemon-binder-finder/internal/adapters/out/listings/ebay"
	sqlitestore "pokemon-binder-finder/internal/adapters/out/persistence/sqlite"
	localvision "pokemon-binder-finder/internal/adapters/out/vision/local"
	"pokemon-binder-finder/internal/adapters/out/vision/ollama"
	"pokemon-binder-finder/internal/application/usecases"
	"pokemon-binder-finder/internal/platform/config"
)

func main() {
	cfg := config.Load()
	if err := os.MkdirAll(filepath.Dir(cfg.DatabasePath), 0o755); err != nil {
		log.Fatal(err)
	}
	store, err := sqlitestore.Open(cfg.DatabasePath)
	if err != nil {
		log.Fatal(err)
	}
	defer store.Close()
	catalog := pokemontcg.New(store)
	fetcher, err := filesystem.New(cfg.ImageCacheDir)
	if err != nil {
		log.Fatal(err)
	}
	searches := usecases.NewSearchService(
		store,
		catalog,
		ebay.New(cfg.EbayClientID, cfg.EbayClientSecret),
		fetcher,
		localvision.New(),
		ollama.New(cfg.OllamaBaseURL, cfg.OllamaModel),
		usecases.SearchConfig{
			Marketplaces:       cfg.EbayMarketplaceIDs,
			ResultLimit:        cfg.ListingResultLimit,
			ConfirmedThreshold: cfg.MatchConfirmedThreshold,
			PossibleThreshold:  cfg.MatchPossibleThreshold,
		},
	)
	log.Printf("listening on %s", cfg.HTTPAddr)
	deps := httpapi.Dependencies{Catalog: catalog, Searches: searches, AssetsDir: cfg.ImageCacheDir}
	if err := http.ListenAndServe(cfg.HTTPAddr, httpapi.NewRouter(deps)); err != nil {
		log.Fatal(err)
	}
}
