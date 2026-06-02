package main

import (
	"log"
	"net/http"

	"pokemon-binder-finder/internal/adapters/in/httpapi"
	"pokemon-binder-finder/internal/platform/config"
)

func main() {
	cfg := config.Load()
	log.Printf("listening on %s", cfg.HTTPAddr)
	if err := http.ListenAndServe(cfg.HTTPAddr, httpapi.NewRouter()); err != nil {
		log.Fatal(err)
	}
}
