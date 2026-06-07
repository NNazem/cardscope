package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"

	"pokemon-binder-finder/ebay"
	"pokemon-binder-finder/model"
	"pokemon-binder-finder/repository"
)

type SearchService struct {
	store          *repository.Store
	catalogService *CatalogService
	source         *ebay.Source
	imageCache     *repository.ImageCache
	imageClient    *http.Client
	visionService  *VisionService
	marketplaces   []string
	resultLimit    int
}

type SearchConfig struct {
	Marketplaces []string
	ResultLimit  int
}

func NewSearchService(store *repository.Store, catalogService *CatalogService, source *ebay.Source, imageCache *repository.ImageCache, visionService *VisionService, cfg SearchConfig) *SearchService {
	return &SearchService{
		store: store, catalogService: catalogService, source: source, imageCache: imageCache, imageClient: &http.Client{Timeout: 20 * time.Second}, visionService: visionService,
		marketplaces: cfg.Marketplaces, resultLimit: cfg.ResultLimit,
	}
}

func (s *SearchService) Create(ctx context.Context, cardID, query string) (model.SearchJob, error) {
	if cardID == "" || query == "" {
		return model.SearchJob{}, fmt.Errorf("cardId and listingQuery are required")
	}
	if _, err := s.catalogService.Get(ctx, cardID); err != nil {
		return model.SearchJob{}, fmt.Errorf("unknown card: %w", err)
	}
	job := model.SearchJob{ID: id(), CardID: cardID, ListingQuery: query, Status: model.JobQueued, CreatedAt: time.Now().UTC()}
	if err := s.store.CreateJob(ctx, job); err != nil {
		return model.SearchJob{}, err
	}
	go s.run(context.Background(), job)
	return job, nil
}

func (s *SearchService) Get(ctx context.Context, id string) (model.SearchJob, error) {
	return s.store.GetJob(ctx, id)
}

func (s *SearchService) Results(ctx context.Context, id, bucket string) ([]model.SearchResult, error) {
	return s.store.ListResults(ctx, id, bucket)
}

func (s *SearchService) run(ctx context.Context, job model.SearchJob) {
	card, err := s.catalogService.Get(ctx, job.CardID)
	if err != nil {
		s.fail(ctx, &job, err)
		return
	}
	reference, err := s.fetchImage(ctx, card.ImageURL)
	if err != nil {
		s.fail(ctx, &job, fmt.Errorf("download reference image: %w", err))
		return
	}
	job.Status = model.JobFetching
	_ = s.store.UpdateJob(ctx, job)
	listings := make(map[string]model.Listing)
	for _, marketplace := range s.marketplaces {
		found, searchErr := s.source.Search(ctx, job.ListingQuery, marketplace, s.resultLimit)
		if searchErr != nil {
			s.fail(ctx, &job, searchErr)
			return
		}
		for _, listing := range found {
			listings[listing.ID] = listing
		}
	}
	job.ListingsFound = len(listings)
	for _, listing := range listings {
		job.ImagesTotal += len(listing.ImageURLs)
	}
	job.Status = model.JobAnalyzing
	_ = s.store.UpdateJob(ctx, job)
	for _, listing := range listings {
		for _, imageURL := range listing.ImageURLs {
			s.analyze(ctx, &job, listing, imageURL, reference.Data)
			job.ImagesAnalyzed++
			_ = s.store.UpdateJob(ctx, job)
		}
	}
	now := time.Now().UTC()
	job.Status, job.CompletedAt = model.JobCompleted, &now
	_ = s.store.UpdateJob(ctx, job)
}

func (s *SearchService) analyze(ctx context.Context, job *model.SearchJob, listing model.Listing, imageURL string, reference []byte) {
	asset, err := s.fetchImage(ctx, imageURL)
	if err != nil {
		return
	}
	s.visionService.AnalyzeCandidates(ctx, job, listing, asset, imageURL, reference)
}

func (s *SearchService) fail(ctx context.Context, job *model.SearchJob, err error) {
	now := time.Now().UTC()
	job.Status, job.Error, job.CompletedAt = model.JobFailed, err.Error(), &now
	_ = s.store.UpdateJob(ctx, *job)
}

func (s *SearchService) fetchImage(ctx context.Context, sourceURL string) (model.ImageAsset, error) {
	if asset, ok, err := s.imageCache.Get(sourceURL); ok || err != nil {
		return asset, err
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	resp, err := s.imageClient.Do(req)
	if err != nil {
		return model.ImageAsset{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return model.ImageAsset{}, fmt.Errorf("image download returned %s", resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return model.ImageAsset{}, err
	}
	return s.imageCache.Save(sourceURL, data)
}

func id() string {
	value := make([]byte, 16)
	_, _ = rand.Read(value)
	return hex.EncodeToString(value)
}
