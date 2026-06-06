package web

import (
	"encoding/json"
	"errors"
	"net/http"
	"pokemon-binder-finder/model"
	"pokemon-binder-finder/pokemontcg"
	"pokemon-binder-finder/service"
	"strconv"
)

type SearchHandler struct {
	catalog       *pokemontcg.Catalog
	searchService *service.SearchService
}

type SearchJobInput struct {
	CardID       string `json:"cardId"`
	ListingQuery string `json:"listingQuery"`
}

func NewSearchHandler(catalog *pokemontcg.Catalog, searchService *service.SearchService) *SearchHandler {
	return &SearchHandler{
		catalog:       catalog,
		searchService: searchService,
	}
}

func (sh *SearchHandler) GetCards(w http.ResponseWriter, r *http.Request) {
	limit, _ := strconv.Atoi(r.URL.Query().Get("limit"))

	if limit < 0 {
		writeError(w, http.StatusInternalServerError, errors.New("limit should be a positive value greater or equal of 0"))
		return
	}

	cards, err := sh.catalog.Search(r.Context(), r.URL.Query().Get("query"), limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}

	writeJSON(w, http.StatusOK, cards)
}

func (sh *SearchHandler) SyncCatalog(w http.ResponseWriter, r *http.Request) {
	err := sh.catalog.Sync(r.Context())

	if err != nil {
		writeError(w, http.StatusBadGateway, err)
		return
	}

	writeJSON(w, http.StatusOK, map[string]string{"status": "Synced successfully."})
}

func (sh *SearchHandler) SearchCardJob(w http.ResponseWriter, r *http.Request) {
	var input = &SearchJobInput{}
	err := json.NewDecoder(r.Body).Decode(&input)

	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	job, err := sh.searchService.Create(r.Context(), input.CardID, input.ListingQuery)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	writeJSON(w, http.StatusAccepted, job)
}

func (sh *SearchHandler) GetCardJobById(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")

	job, err := sh.searchService.Get(r.Context(), id)

	if err != nil {
		writeStoreError(w, err)
		return
	}

	writeJSON(w, http.StatusOK, job)
}

func (sh *SearchHandler) GetCardJobResultById(w http.ResponseWriter, r *http.Request) {
	bucket := r.URL.Query().Get("bucket")
	if bucket != model.BucketConfirmed && bucket != model.BucketPossible {
		writeError(w, http.StatusBadRequest, errors.New("bucket must be confirmed or possible"))
		return
	}
	results, err := sh.searchService.Results(r.Context(), r.PathValue("id"), bucket)
	if err != nil {
		writeStoreError(w, err)
		return
	}
	writeJSON(w, http.StatusOK, results)
}
