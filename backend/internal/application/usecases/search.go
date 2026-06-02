package usecases

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"time"

	"pokemon-binder-finder/internal/application/ports"
	"pokemon-binder-finder/internal/domain"
)

type SearchService struct {
	store              ports.Store
	catalog            ports.CardCatalog
	source             ports.ListingSource
	fetcher            ports.ImageFetcher
	matcher            ports.ImageMatcher
	verifier           ports.VisionVerifier
	marketplaces       []string
	resultLimit        int
	confirmedThreshold float64
	possibleThreshold  float64
}

type SearchConfig struct {
	Marketplaces       []string
	ResultLimit        int
	ConfirmedThreshold float64
	PossibleThreshold  float64
}

func NewSearchService(store ports.Store, catalog ports.CardCatalog, source ports.ListingSource, fetcher ports.ImageFetcher, matcher ports.ImageMatcher, verifier ports.VisionVerifier, cfg SearchConfig) *SearchService {
	return &SearchService{
		store: store, catalog: catalog, source: source, fetcher: fetcher, matcher: matcher, verifier: verifier,
		marketplaces: cfg.Marketplaces, resultLimit: cfg.ResultLimit,
		confirmedThreshold: cfg.ConfirmedThreshold, possibleThreshold: cfg.PossibleThreshold,
	}
}

func (s *SearchService) Create(ctx context.Context, cardID, query string) (domain.SearchJob, error) {
	if cardID == "" || query == "" {
		return domain.SearchJob{}, fmt.Errorf("cardId and listingQuery are required")
	}
	if _, err := s.catalog.Get(ctx, cardID); err != nil {
		return domain.SearchJob{}, fmt.Errorf("unknown card: %w", err)
	}
	job := domain.SearchJob{ID: id(), CardID: cardID, ListingQuery: query, Status: domain.JobQueued, CreatedAt: time.Now().UTC()}
	if err := s.store.CreateJob(ctx, job); err != nil {
		return domain.SearchJob{}, err
	}
	go s.run(context.Background(), job)
	return job, nil
}

func (s *SearchService) Get(ctx context.Context, id string) (domain.SearchJob, error) {
	return s.store.GetJob(ctx, id)
}

func (s *SearchService) Results(ctx context.Context, id, bucket string) ([]domain.SearchResult, error) {
	return s.store.ListResults(ctx, id, bucket)
}

func (s *SearchService) run(ctx context.Context, job domain.SearchJob) {
	card, err := s.catalog.Get(ctx, job.CardID)
	if err != nil {
		s.fail(ctx, &job, err)
		return
	}
	reference, err := s.fetcher.Fetch(ctx, card.ImageURL)
	if err != nil {
		s.fail(ctx, &job, fmt.Errorf("download reference image: %w", err))
		return
	}
	job.Status = domain.JobFetching
	_ = s.store.UpdateJob(ctx, job)
	listings := make(map[string]domain.Listing)
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
	job.Status = domain.JobAnalyzing
	job.VisionDegraded = s.verifier == nil || !s.verifier.Available(ctx)
	_ = s.store.UpdateJob(ctx, job)
	for _, listing := range listings {
		for _, imageURL := range listing.ImageURLs {
			s.analyze(ctx, &job, listing, imageURL, reference.Data)
			job.ImagesAnalyzed++
			_ = s.store.UpdateJob(ctx, job)
		}
	}
	now := time.Now().UTC()
	job.Status, job.CompletedAt = domain.JobCompleted, &now
	_ = s.store.UpdateJob(ctx, job)
}

func (s *SearchService) analyze(ctx context.Context, job *domain.SearchJob, listing domain.Listing, imageURL string, reference []byte) {
	asset, err := s.fetcher.Fetch(ctx, imageURL)
	if err != nil {
		return
	}
	candidates, err := s.matcher.Match(ctx, reference, asset.Data)
	if err != nil {
		return
	}
	for index, candidate := range candidates {
		if candidate.Confidence >= s.possibleThreshold && candidate.Confidence < s.confirmedThreshold && !job.VisionDegraded {
			if confidence, reason, err := s.verifier.Verify(ctx, reference, asset.Data, candidate); err == nil {
				candidate.Confidence, candidate.Reason = confidence, reason
			}
		}
		bucket := classify(candidate.Confidence, s.confirmedThreshold, s.possibleThreshold)
		if bucket == "" {
			continue
		}
		if bucket == domain.BucketConfirmed {
			job.ConfirmedMatches++
		} else {
			job.PossibleMatches++
		}
		result := domain.SearchResult{
			ID: id() + fmt.Sprintf("-%d", index), JobID: job.ID, ListingID: listing.ID,
			ListingURL: listing.URL, ListingTitle: listing.Title, SourceImageURL: imageURL,
			CachedImageURL: asset.PublicURL, Polygon: candidate.Polygon, Confidence: candidate.Confidence,
			Bucket: bucket, Reason: candidate.Reason,
		}
		_ = s.store.SaveResult(ctx, result)
	}
}

func (s *SearchService) fail(ctx context.Context, job *domain.SearchJob, err error) {
	now := time.Now().UTC()
	job.Status, job.Error, job.CompletedAt = domain.JobFailed, err.Error(), &now
	_ = s.store.UpdateJob(ctx, *job)
}

func classify(confidence, confirmed, possible float64) string {
	if confidence >= confirmed {
		return domain.BucketConfirmed
	}
	if confidence >= possible {
		return domain.BucketPossible
	}
	return ""
}

func id() string {
	value := make([]byte, 16)
	_, _ = rand.Read(value)
	return hex.EncodeToString(value)
}
