package service

import (
	"context"
	"pokemon-binder-finder/model"
	"pokemon-binder-finder/vision"
)

type VisionService struct {
	matcher            *vision.Matcher
	confirmedThreshold float64
	possibleThreshold  float64
}

func NewVisionService(matcher *vision.Matcher, confirmedThreshold float64, possibleThreshold float64) *VisionService {
	return &VisionService{
		matcher:            matcher,
		confirmedThreshold: confirmedThreshold,
		possibleThreshold:  possibleThreshold,
	}
}

type VisionMatch struct {
	Polygon    []model.PolygonPoint
	Confidence float64
	Bucket     string
	Reason     string
}

func (v *VisionService) AnalyzeImage(ctx context.Context, target model.ImageAsset, reference []byte) ([]VisionMatch, error) {
	candidates, err := v.matcher.Match(ctx, reference, target.Data)
	if err != nil {
		return nil, err
	}

	var matches []VisionMatch

	for _, candidate := range candidates {
		bucket := classify(candidate.Confidence, v.confirmedThreshold, v.possibleThreshold)
		match := VisionMatch{
			Polygon:    candidate.Polygon,
			Confidence: candidate.Confidence,
			Bucket:     bucket,
			Reason:     candidate.Reason,
		}
		matches = append(matches, match)
	}

	return matches, nil
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
