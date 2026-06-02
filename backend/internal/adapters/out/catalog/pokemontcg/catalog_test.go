package pokemontcg

import (
	"context"
	"io"
	"net/http"
	"path/filepath"
	"strings"
	"testing"

	sqlitestore "pokemon-binder-finder/internal/adapters/out/persistence/sqlite"
)

func TestSyncAndSearch(t *testing.T) {
	store, err := sqlitestore.Open(filepath.Join(t.TempDir(), "test.db"))
	if err != nil {
		t.Fatal(err)
	}
	defer store.Close()
	catalog := New(store)
	catalog.client.Transport = roundTripFunc(func(_ *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body: io.NopCloser(strings.NewReader(
				`{"data":[{"id":"base1-4","name":"Charizard","number":"4","set":{"id":"base1","name":"Base"},"images":{"large":"https://example/card.png"}}],"totalCount":1}`,
			)),
			Header: make(http.Header),
		}, nil
	})
	if err := catalog.Sync(context.Background()); err != nil {
		t.Fatal(err)
	}
	cards, err := catalog.Search(context.Background(), "Char", 10)
	if err != nil || len(cards) != 1 || cards[0].ID != "base1-4" {
		t.Fatalf("unexpected cards: %#v, %v", cards, err)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
