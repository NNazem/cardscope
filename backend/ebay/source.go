package ebay

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"pokemon-binder-finder/model"
)

type Source struct {
	client       *http.Client
	clientID     string
	clientSecret string
	tokenURL     string
	searchURL    string
	itemURL      string
	mu           sync.Mutex
	token        string
	tokenExpiry  time.Time
}

func New(clientID, clientSecret string) *Source {
	return &Source{
		client:       &http.Client{Timeout: 30 * time.Second},
		clientID:     clientID,
		clientSecret: clientSecret,
		tokenURL:     "https://api.ebay.com/identity/v1/oauth2/token",
		searchURL:    "https://api.ebay.com/buy/browse/v1/item_summary/search",
		itemURL:      "https://api.ebay.com/buy/browse/v1/item/",
	}
}

func (s *Source) Search(ctx context.Context, query, marketplace string, limit int) ([]model.Listing, error) {
	if s.clientID == "" || s.clientSecret == "" {
		return nil, fmt.Errorf("ebay credentials are not configured")
	}
	token, err := s.accessToken(ctx)
	if err != nil {
		return nil, err
	}
	endpoint, _ := url.Parse(s.searchURL)
	values := endpoint.Query()
	values.Set("q", query)
	values.Set("limit", strconv.Itoa(limit))
	endpoint.RawQuery = values.Encode()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-EBAY-C-MARKETPLACE-ID", marketplace)
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ebay browse api returned %s", resp.Status)
	}
	var payload struct {
		Items []struct {
			ID     string  `json:"itemId"`
			Title  string  `json:"title"`
			URL    string  `json:"itemWebUrl"`
			Image  image   `json:"image"`
			Images []image `json:"additionalImages"`
		} `json:"itemSummaries"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	listings := make([]model.Listing, 0, len(payload.Items))
	for _, item := range payload.Items {
		images := appendImage(nil, item.Image.URL)
		for _, candidate := range item.Images {
			images = appendImage(images, candidate.URL)
		}
		if detailImages, detailErr := s.getImages(ctx, token, marketplace, item.ID); detailErr == nil {
			for _, candidate := range detailImages {
				images = appendImage(images, candidate)
			}
		}
		listings = append(listings, model.Listing{
			ID: item.ID, Title: item.Title, URL: item.URL, ImageURLs: images, Marketplace: marketplace,
		})
	}
	return listings, nil
}

func (s *Source) getImages(ctx context.Context, token, marketplace, itemID string) ([]string, error) {
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, s.itemURL+url.PathEscape(itemID), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-EBAY-C-MARKETPLACE-ID", marketplace)
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("ebay item api returned %s", resp.Status)
	}
	var payload struct {
		Image  image   `json:"image"`
		Images []image `json:"additionalImages"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, err
	}
	images := appendImage(nil, payload.Image.URL)
	for _, candidate := range payload.Images {
		images = appendImage(images, candidate.URL)
	}
	return images, nil
}

type image struct {
	URL string `json:"imageUrl"`
}

func appendImage(images []string, candidate string) []string {
	if candidate == "" {
		return images
	}
	for _, existing := range images {
		if existing == candidate {
			return images
		}
	}
	return append(images, candidate)
}

func (s *Source) accessToken(ctx context.Context) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.token != "" && time.Now().Before(s.tokenExpiry.Add(-time.Minute)) {
		return s.token, nil
	}
	body := strings.NewReader("grant_type=client_credentials&scope=" + url.QueryEscape("https://api.ebay.com/oauth/api_scope"))
	req, _ := http.NewRequestWithContext(ctx, http.MethodPost, s.tokenURL, body)
	req.Header.Set("Authorization", "Basic "+base64.StdEncoding.EncodeToString([]byte(s.clientID+":"+s.clientSecret)))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := s.client.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("ebay oauth returned %s", resp.Status)
	}
	var payload struct {
		Token     string `json:"access_token"`
		ExpiresIn int    `json:"expires_in"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return "", err
	}
	s.token = payload.Token
	s.tokenExpiry = time.Now().Add(time.Duration(payload.ExpiresIn) * time.Second)
	return s.token, nil
}
