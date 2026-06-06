package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"pokemon-binder-finder/internal/application/ports"
	"pokemon-binder-finder/internal/domain"
)

type Dependencies struct {
	Catalog   ports.CardCatalog
	Searches  SearchJobs
	AssetsDir string
}

type SearchJobs interface {
	Create(context.Context, string, string) (domain.SearchJob, error)
	Get(context.Context, string) (domain.SearchJob, error)
	Results(context.Context, string, string) ([]domain.SearchResult, error)
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
		mux.HandleFunc("GET /api/cards", getCards(dependencies))
		mux.HandleFunc("POST /api/catalog/sync", syncCatalog(dependencies))
	}
	if dependencies.Searches != nil {
		mux.HandleFunc("POST /api/search-jobs", searchJob(dependencies))
		mux.HandleFunc("GET /api/search-jobs/{id}", getJob(dependencies))
		mux.HandleFunc("GET /api/search-jobs/{id}/results", getJobResult(dependencies))
	}
	if dependencies.AssetsDir != "" {
		mux.Handle("/api/assets/", http.StripPrefix("/api/assets/", http.FileServer(http.Dir(dependencies.AssetsDir))))
	}
	return withCORS(mux)
}

func getJobResult(dependencies Dependencies) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		bucket := r.URL.Query().Get("bucket")
		if bucket != domain.BucketConfirmed && bucket != domain.BucketPossible {
			writeError(w, http.StatusBadRequest, errors.New("bucket must be confirmed or possible"))
			return
		}
		results, err := dependencies.Searches.Results(r.Context(), r.PathValue("id"), bucket)
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, results)
	}
}

func getJob(dependencies Dependencies) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		job, err := dependencies.Searches.Get(r.Context(), r.PathValue("id"))
		if err != nil {
			writeStoreError(w, err)
			return
		}
		writeJSON(w, http.StatusOK, job)
	}
}

func searchJob(dependencies Dependencies) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		var input struct {
			CardID       string `json:"cardId"`
			ListingQuery string `json:"listingQuery"`
		}
		if err := json.NewDecoder(r.Body).Decode(&input); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		job, err := dependencies.Searches.Create(r.Context(), input.CardID, input.ListingQuery)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		writeJSON(w, http.StatusAccepted, job)
	}
}

func syncCatalog(dependencies Dependencies) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		if err := dependencies.Catalog.Sync(r.Context()); err != nil {
			writeError(w, http.StatusBadGateway, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "synced"})
	}
}

func getCards(dependencies Dependencies) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
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
	}
}

func writeStoreError(w http.ResponseWriter, err error) {
	if errors.Is(err, domain.ErrNotFound) {
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
