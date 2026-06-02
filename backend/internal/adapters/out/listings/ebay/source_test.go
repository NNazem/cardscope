package ebay

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"
)

func TestSearchMapsListingsAndImages(t *testing.T) {
	source := New("id", "secret")
	source.client.Transport = roundTripFunc(func(req *http.Request) (*http.Response, error) {
		payload := `{"access_token":"token","expires_in":3600}`
		if req.Method == http.MethodGet {
			if got := req.Header.Get("X-EBAY-C-MARKETPLACE-ID"); got != "EBAY_IT" {
				t.Fatalf("unexpected marketplace: %s", got)
			}
			payload = `{"itemSummaries":[{"itemId":"item-1","title":"Binder","itemWebUrl":"https://example/listing","image":{"imageUrl":"https://example/1.jpg"},"additionalImages":[{"imageUrl":"https://example/2.jpg"}]}]}`
		}
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader(payload)), Header: make(http.Header)}, nil
	})
	listings, err := source.Search(context.Background(), "binder", "EBAY_IT", 100)
	if err != nil {
		t.Fatal(err)
	}
	if len(listings) != 1 || len(listings[0].ImageURLs) != 2 {
		t.Fatalf("unexpected listings: %#v", listings)
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(req *http.Request) (*http.Response, error) { return f(req) }
