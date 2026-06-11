package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"

	"pokemon-binder-finder/model"
	"pokemon-binder-finder/repository"
)

type JobService struct {
	store *repository.Store
}

func NewJobService(store *repository.Store) *JobService {
	return &JobService{
		store: store,
	}
}

func (j *JobService) Get(ctx context.Context, id string) (model.SearchJob, error) {
	return j.store.GetJob(ctx, id)
}

func (j *JobService) Results(ctx context.Context, id, bucket string) ([]model.SearchResult, error) {
	return j.store.ListResults(ctx, id, bucket)
}

func (j *JobService) CreateJob(ctx context.Context, cardID, listingQuery string) (model.SearchJob, error) {
	if cardID == "" || listingQuery == "" {
		return model.SearchJob{}, fmt.Errorf("cardId and listingQuery are required")
	}

	searchJob := model.SearchJob{
		ID:           uuid.NewString(),
		CardID:       cardID,
		ListingQuery: listingQuery,
		Status:       model.JobQueued,
		CreatedAt:    time.Now().UTC(),
	}
	if err := j.store.CreateJob(ctx, searchJob); err != nil {
		return model.SearchJob{}, err
	}
	return searchJob, nil
}

func (j *JobService) UpdateJob(ctx context.Context, searchJob model.SearchJob) error {
	return j.store.UpdateJob(ctx, searchJob)
}

func (j *JobService) SaveMatchResult(ctx context.Context, jobID string, listing model.Listing, targetImageURL string, target model.ImageAsset, match VisionMatch, index int) (model.SearchResult, error) {
	searchResult := model.SearchResult{
		ID:             uuid.NewString(),
		JobID:          jobID,
		ListingID:      listing.ID,
		ListingURL:     listing.URL,
		ListingTitle:   listing.Title,
		SourceImageURL: targetImageURL,
		CachedImageURL: target.PublicURL,
		Polygon:        match.Polygon,
		Confidence:     match.Confidence,
		Bucket:         match.Bucket,
		Reason:         match.Reason,
	}
	if err := j.store.SaveResult(ctx, searchResult); err != nil {
		return model.SearchResult{}, err
	}
	return searchResult, nil
}

func (j *JobService) FailJob(ctx context.Context, job *model.SearchJob, err error) {
	now := time.Now().UTC()
	job.Status, job.Error, job.CompletedAt = model.JobFailed, err.Error(), &now
	_ = j.UpdateJob(ctx, *job)
}
