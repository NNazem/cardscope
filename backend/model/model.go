package model

import (
	"errors"
	"time"
)

var ErrNotFound = errors.New("not found")

type Card struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	SetID     string `json:"setId"`
	SetName   string `json:"setName"`
	Number    string `json:"number"`
	ImageURL  string `json:"imageUrl"`
	ImageData []byte `json:"-"`
}

type Listing struct {
	ID          string   `json:"id"`
	URL         string   `json:"url"`
	Title       string   `json:"title"`
	ImageURLs   []string `json:"imageUrls"`
	Marketplace string   `json:"marketplace"`
}

type PolygonPoint struct {
	X int `json:"x"`
	Y int `json:"y"`
}

type MatchCandidate struct {
	Polygon    []PolygonPoint `json:"polygon"`
	Confidence float64        `json:"confidence"`
	Reason     string         `json:"reason"`
}

type ImageAsset struct {
	Data      []byte
	PublicURL string
}

type SearchJob struct {
	ID               string     `json:"id"`
	CardID           string     `json:"cardId"`
	ListingQuery     string     `json:"listingQuery"`
	Status           string     `json:"status"`
	ListingsFound    int        `json:"listingsFound"`
	ImagesAnalyzed   int        `json:"imagesAnalyzed"`
	ImagesTotal      int        `json:"imagesTotal"`
	ConfirmedMatches int        `json:"confirmedMatches"`
	PossibleMatches  int        `json:"possibleMatches"`
	Error            string     `json:"error,omitempty"`
	CreatedAt        time.Time  `json:"createdAt"`
	CompletedAt      *time.Time `json:"completedAt,omitempty"`
}

type SearchResult struct {
	ID             string         `json:"id"`
	JobID          string         `json:"jobId"`
	ListingID      string         `json:"listingId"`
	ListingURL     string         `json:"listingUrl"`
	ListingTitle   string         `json:"listingTitle"`
	SourceImageURL string         `json:"sourceImageUrl"`
	CachedImageURL string         `json:"cachedImageUrl"`
	CropImageURL   string         `json:"cropImageUrl,omitempty"`
	Polygon        []PolygonPoint `json:"polygon"`
	Confidence     float64        `json:"confidence"`
	Bucket         string         `json:"bucket"`
	Reason         string         `json:"reason"`
}

const (
	JobQueued    = "queued"
	JobFetching  = "fetching"
	JobAnalyzing = "analyzing"
	JobCompleted = "completed"
	JobFailed    = "failed"

	BucketConfirmed = "confirmed"
	BucketPossible  = "possible"
)
