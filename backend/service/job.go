package service

import (
	"context"
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

func (j *JobService) CreateJob(ctx context.Context, searchJob model.SearchJob) error {
	return j.store.CreateJob(ctx, searchJob)
}

func (j *JobService) UpdateJob(ctx context.Context, searchJob model.SearchJob) error {
	return j.store.UpdateJob(ctx, searchJob)
}

func (j *JobService) SaveResult(ctx context.Context, searchResult model.SearchResult) error {
	return j.store.SaveResult(ctx, searchResult)
}
