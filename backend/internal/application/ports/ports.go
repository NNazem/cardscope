package ports

import (
	"context"

	"pokemon-binder-finder/internal/domain"
)

type CardCatalog interface {
	Sync(context.Context) error
	Search(context.Context, string, int) ([]domain.Card, error)
	Get(context.Context, string) (domain.Card, error)
}

type ListingSource interface {
	Search(context.Context, string, string, int) ([]domain.Listing, error)
}

type ImageFetcher interface {
	Fetch(context.Context, string) (domain.ImageAsset, error)
}

type ImageMatcher interface {
	Match(context.Context, []byte, []byte) ([]domain.MatchCandidate, error)
}

type VisionVerifier interface {
	Verify(context.Context, []byte, []byte, domain.MatchCandidate) (float64, string, error)
	Available(context.Context) bool
}

type Store interface {
	UpsertCards(context.Context, []domain.Card) error
	SearchCards(context.Context, string, int) ([]domain.Card, error)
	GetCard(context.Context, string) (domain.Card, error)
	CreateJob(context.Context, domain.SearchJob) error
	UpdateJob(context.Context, domain.SearchJob) error
	GetJob(context.Context, string) (domain.SearchJob, error)
	SaveResult(context.Context, domain.SearchResult) error
	ListResults(context.Context, string, string) ([]domain.SearchResult, error)
}
