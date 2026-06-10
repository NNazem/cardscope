package service

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"time"

	"pokemon-binder-finder/model"
	"pokemon-binder-finder/repository"
)

type SearchService struct {
	store          *repository.Store
	catalogService *CatalogService
	ebayService    *EbayService
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

func NewSearchService(store *repository.Store, catalogService *CatalogService, ebayService *EbayService, imageCache *repository.ImageCache, visionService *VisionService, cfg SearchConfig) *SearchService {
	return &SearchService{
		store: store, catalogService: catalogService, ebayService: ebayService, imageCache: imageCache, imageClient: &http.Client{Timeout: 20 * time.Second}, visionService: visionService,
		marketplaces: cfg.Marketplaces, resultLimit: cfg.ResultLimit,
	}
}

func (s *SearchService) SearchCard(ctx context.Context, cardID, query string) (model.SearchJob, error) {
	if cardID == "" || query == "" {
		return model.SearchJob{}, fmt.Errorf("cardId and listingQuery are required")
	}

	referenceCard, err := s.catalogService.GetCard(ctx, cardID)
	if err != nil {
		return model.SearchJob{}, fmt.Errorf("unknown card: %w", err)
	}

	job := model.SearchJob{ID: id(), CardID: cardID, ListingQuery: query, Status: model.JobQueued, CreatedAt: time.Now().UTC()}
	if err := s.store.CreateJob(ctx, job); err != nil {
		return model.SearchJob{}, err
	}
	go s.executeJob(context.Background(), job, referenceCard)
	return job, nil
}

func (s *SearchService) Get(ctx context.Context, id string) (model.SearchJob, error) {
	return s.store.GetJob(ctx, id)
}

func (s *SearchService) Results(ctx context.Context, id, bucket string) ([]model.SearchResult, error) {
	return s.store.ListResults(ctx, id, bucket)
}

func (s *SearchService) executeJob(ctx context.Context, job model.SearchJob, referenceCard model.Card) {
	reference, err := s.fetchImage(ctx, referenceCard.ImageURL)
	if err != nil {
		s.fail(ctx, &job, fmt.Errorf("download reference image: %w", err))
		return
	}

	job.Status = model.JobFetching
	_ = s.store.UpdateJob(ctx, job)

	listings := s.fetchListings(ctx, job)
	job.ListingsFound = len(listings)
	for _, listing := range listings {
		job.ImagesTotal += len(listing.ImageURLs)
	}

	job.Status = model.JobAnalyzing
	_ = s.store.UpdateJob(ctx, job)

	s.analyzeListings(ctx, job, listings, reference)

	now := time.Now().UTC()
	job.Status, job.CompletedAt = model.JobCompleted, &now
	_ = s.store.UpdateJob(ctx, job)
}

func (s *SearchService) analyzeListings(ctx context.Context, job model.SearchJob, listings map[string]model.Listing, reference model.ImageAsset) {
	for _, listing := range listings {
		for _, targetImageURL := range listing.ImageURLs {
			s.analyze(ctx, &job, listing, targetImageURL, reference.Data)
			job.ImagesAnalyzed++
			_ = s.store.UpdateJob(ctx, job)
		}
	}
}

func (s *SearchService) fetchListings(ctx context.Context, job model.SearchJob) map[string]model.Listing {
	listings := make(map[string]model.Listing)
	for _, marketplace := range s.marketplaces {
		found, searchErr := s.ebayService.Search(ctx, job.ListingQuery, marketplace, s.resultLimit)
		if searchErr != nil {
			continue
		}
		for _, listing := range found {
			listings[listing.ID] = listing
		}
	}
	return listings
}

func (s *SearchService) analyze(ctx context.Context, job *model.SearchJob, listing model.Listing, targetImageURL string, reference []byte) {
	target, err := s.fetchImage(ctx, targetImageURL)
	if err != nil {
		return
	}
	visionMatches, err := s.visionService.AnalyzeImage(ctx, target, reference)
	if err != nil {
		return
	}

	for index, visionMatch := range visionMatches {
		if visionMatch.Bucket == "" {
			continue
		}

		if visionMatch.Bucket == model.BucketConfirmed {
			job.ConfirmedMatches++
		} else {
			job.PossibleMatches++
		}

		result := model.SearchResult{
			ID: id() + fmt.Sprintf("-%d", index), JobID: job.ID, ListingID: listing.ID,
			ListingURL: listing.URL, ListingTitle: listing.Title, SourceImageURL: targetImageURL,
			CachedImageURL: target.PublicURL, Polygon: visionMatch.Polygon, Confidence: visionMatch.Confidence,
			Bucket: visionMatch.Bucket, Reason: visionMatch.Reason,
		}
		_ = s.store.SaveResult(ctx, result)
	}

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
