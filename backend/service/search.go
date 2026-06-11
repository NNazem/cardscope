package service

import (
	"context"
	"fmt"
	"time"

	"pokemon-binder-finder/model"
)

type SearchService struct {
	jobService     *JobService
	catalogService *CatalogService
	ebayService    *EbayService
	imageService   *ImageService
	visionService  *VisionService
	marketplaces   []string
	resultLimit    int
}

type SearchConfig struct {
	Marketplaces []string
	ResultLimit  int
}

func NewSearchService(jobService *JobService, catalogService *CatalogService, ebayService *EbayService, imageService *ImageService, visionService *VisionService, cfg SearchConfig) *SearchService {
	return &SearchService{
		jobService: jobService, catalogService: catalogService, ebayService: ebayService, imageService: imageService, visionService: visionService,
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

	job, err := s.jobService.CreateJob(ctx, cardID, query)
	if err != nil {
		return model.SearchJob{}, err
	}
	go s.executeJob(context.Background(), job, referenceCard)
	return job, nil
}

func (s *SearchService) executeJob(ctx context.Context, job model.SearchJob, referenceCard model.Card) {
	reference, err := s.imageService.FetchImage(ctx, referenceCard.ImageURL)
	if err != nil {
		s.jobService.FailJob(ctx, &job, fmt.Errorf("download reference image: %w", err))
		return
	}

	job.Status = model.JobFetching
	_ = s.jobService.UpdateJob(ctx, job)

	listings := s.fetchListings(ctx, job)
	job.ListingsFound = len(listings)
	for _, listing := range listings {
		job.ImagesTotal += len(listing.ImageURLs)
	}

	job.Status = model.JobAnalyzing
	_ = s.jobService.UpdateJob(ctx, job)

	s.analyzeListings(ctx, job, listings, reference)

	now := time.Now().UTC()
	job.Status, job.CompletedAt = model.JobCompleted, &now
	_ = s.jobService.UpdateJob(ctx, job)
}

func (s *SearchService) analyzeListings(ctx context.Context, job model.SearchJob, listings map[string]model.Listing, reference model.ImageAsset) {
	for _, listing := range listings {
		for _, targetImageURL := range listing.ImageURLs {
			s.analyze(ctx, &job, listing, targetImageURL, reference.Data)
			job.ImagesAnalyzed++
			_ = s.jobService.UpdateJob(ctx, job)
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
	target, err := s.imageService.FetchImage(ctx, targetImageURL)
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

		_, _ = s.jobService.SaveMatchResult(ctx, job.ID, listing, targetImageURL, target, visionMatch, index)
	}

}
