package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"pokemon-binder-finder/model"
	"pokemon-binder-finder/pokemontcg"
	"pokemon-binder-finder/service"
)

func NewRouter(catalog *pokemontcg.Catalog, searchService *service.SearchService, assetsDir string) http.Handler {
	mux := http.NewServeMux()
	searchHandler := NewSearchHandler(catalog, searchService)

	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.HandleFunc("GET /api/cards", searchHandler.GetCards)
	mux.HandleFunc("POST /api/catalog/sync", searchHandler.SyncCatalog)
	mux.HandleFunc("POST /api/search-jobs", searchHandler.SearchCardJob)
	mux.HandleFunc("GET /api/search-jobs/{id}", searchHandler.GetCardJobById)
	mux.HandleFunc("GET /api/search-jobs/{id}/results", searchHandler.GetCardJobResultById)
	if assetsDir != "" {
		mux.Handle("/api/assets/", http.StripPrefix("/api/assets/", http.FileServer(http.Dir(assetsDir))))
	}
	return withCORS(mux)
}

func writeStoreError(w http.ResponseWriter, err error) {
	if errors.Is(err, model.ErrNotFound) {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeError(w, http.StatusInternalServerError, err)
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}
