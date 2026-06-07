package service

import (
	"context"
	"fmt"
	"pokemon-binder-finder/model"
	"pokemon-binder-finder/repository"
	"pokemon-binder-finder/vision"
)

type VisionService struct {
	matcher            *vision.Matcher
	store              *repository.Store
	confirmedThreshold float64
	possibleThreshold  float64
}

func NewVisionService(matcher *vision.Matcher, store *repository.Store, confirmedTreshold float64, possibleTreshold float64) *VisionService {
	return &VisionService{
		matcher:            matcher,
		store:              store,
		confirmedThreshold: confirmedTreshold,
		possibleThreshold:  possibleTreshold,
	}
}

func (v *VisionService) AnalyzeCandidates(ctx context.Context, job *model.SearchJob, listing model.Listing, asset model.ImageAsset, imageURL string, reference []byte) {
	candidates, err := v.matcher.Match(ctx, reference, asset.Data)
	if err != nil {
		return
	}

	for index, candidate := range candidates {
		bucket := classify(candidate.Confidence, v.confirmedThreshold, v.possibleThreshold)
		if bucket == "" {
			continue
		}
		if bucket == model.BucketConfirmed {
			job.ConfirmedMatches++
		} else {
			job.PossibleMatches++
		}
		result := model.SearchResult{
			ID: id() + fmt.Sprintf("-%d", index), JobID: job.ID, ListingID: listing.ID,
			ListingURL: listing.URL, ListingTitle: listing.Title, SourceImageURL: imageURL,
			CachedImageURL: asset.PublicURL, Polygon: candidate.Polygon, Confidence: candidate.Confidence,
			Bucket: bucket, Reason: candidate.Reason,
		}
		_ = v.store.SaveResult(ctx, result)
	}
}

func classify(confidence, confirmed, possible float64) string {
	if confidence >= confirmed {
		return model.BucketConfirmed
	}
	if confidence >= possible {
		return model.BucketPossible
	}
	return ""
}
