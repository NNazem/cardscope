package sqlite

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"pokemon-binder-finder/internal/domain"
)

func TestStorePersistsCardsAndJobs(t *testing.T) {
	store, err := Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	ctx := context.Background()
	card := domain.Card{ID: "base1-4", Name: "Charizard", SetID: "base1", SetName: "Base", Number: "4", ImageURL: "https://example/card.png"}
	if err := store.UpsertCards(ctx, []domain.Card{card}); err != nil {
		t.Fatal(err)
	}
	cards, err := store.SearchCards(ctx, "Char", 10)
	if err != nil || len(cards) != 1 || cards[0].ID != card.ID {
		t.Fatalf("unexpected cards: %#v, %v", cards, err)
	}
	job := domain.SearchJob{ID: "job-1", CardID: card.ID, ListingQuery: "binder", Status: domain.JobQueued, CreatedAt: time.Now()}
	if err := store.CreateJob(ctx, job); err != nil {
		t.Fatal(err)
	}
	found, err := store.GetJob(ctx, job.ID)
	if err != nil || found.Status != domain.JobQueued {
		t.Fatalf("unexpected job: %#v, %v", found, err)
	}
}
