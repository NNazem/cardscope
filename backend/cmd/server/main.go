package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"pokemon-binder-finder/internal/adapters/in/httpapi"
	"pokemon-binder-finder/internal/adapters/out/catalog/pokemontcg"
	sqlitestore "pokemon-binder-finder/internal/adapters/out/persistence/sqlite"
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
	log.Printf("listening on %s", cfg.HTTPAddr)
	if err := http.ListenAndServe(cfg.HTTPAddr, httpapi.NewRouter(httpapi.Dependencies{Catalog: catalog})); err != nil {
		log.Fatal(err)
	}
}
