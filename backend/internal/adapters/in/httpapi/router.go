package httpapi

import (
	"encoding/json"
	"net/http"
	"strconv"

	"pokemon-binder-finder/internal/application/ports"
)

type Dependencies struct {
	Catalog ports.CardCatalog
}

func NewRouter(deps ...Dependencies) http.Handler {
	var dependencies Dependencies
	if len(deps) > 0 {
		dependencies = deps[0]
	}
	mux := http.NewServeMux()
	mux.HandleFunc("GET /api/health", func(w http.ResponseWriter, _ *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	if dependencies.Catalog != nil {
		mux.HandleFunc("GET /api/cards", func(w http.ResponseWriter, r *http.Request) {
			limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))
			if limit <= 0 || limit > 50 {
				limit = 20
			}
			cards, err := dependencies.Catalog.Search(r.Context(), r.URL.Query().Get("query"), limit)
			if err != nil {
				writeError(w, http.StatusInternalServerError, err)
				return
			}
			writeJSON(w, http.StatusOK, cards)
		})
		mux.HandleFunc("POST /api/catalog/sync", func(w http.ResponseWriter, r *http.Request) {
			if err := dependencies.Catalog.Sync(r.Context()); err != nil {
				writeError(w, http.StatusBadGateway, err)
				return
			}
			writeJSON(w, http.StatusOK, map[string]string{"status": "synced"})
		})
	}
	return withCORS(mux)
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
