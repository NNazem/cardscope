package pokemontcg

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"pokemon-binder-finder/internal/application/ports"
	"pokemon-binder-finder/internal/domain"
)

type Catalog struct {
	baseURL string
	client  *http.Client
	store   ports.Store
}

func New(store ports.Store) *Catalog {
	return &Catalog{
		baseURL: "https://api.pokemontcg.io/v2",
		client:  &http.Client{Timeout: 30 * time.Second},
		store:   store,
	}
}

func (c *Catalog) Sync(ctx context.Context) error {
	for page := 1; ; page++ {
		cards, total, err := c.fetchPage(ctx, page)
		if err != nil {
			return err
		}
		if err := c.store.UpsertCards(ctx, cards); err != nil {
			return err
		}
		if page*250 >= total || len(cards) == 0 {
			return nil
		}
	}
}

func (c *Catalog) Search(ctx context.Context, query string, limit int) ([]domain.Card, error) {
	return c.store.SearchCards(ctx, query, limit)
}

func (c *Catalog) Get(ctx context.Context, id string) (domain.Card, error) {
	return c.store.GetCard(ctx, id)
}

func (c *Catalog) fetchPage(ctx context.Context, page int) ([]domain.Card, int, error) {
	endpoint, _ := url.Parse(c.baseURL + "/cards")
	values := endpoint.Query()
	values.Set("page", strconv.Itoa(page))
	values.Set("pageSize", "250")
	values.Set("select", "id,name,number,set,images")
	endpoint.RawQuery = values.Encode()
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	resp, err := c.client.Do(req)
	if err != nil {
		return nil, 0, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("pokemon tcg api returned %s", resp.Status)
	}
	var payload struct {
		Data []struct {
			ID     string `json:"id"`
			Name   string `json:"name"`
			Number string `json:"number"`
			Set    struct {
				ID   string `json:"id"`
				Name string `json:"name"`
			} `json:"set"`
			Images struct {
				Large string `json:"large"`
				Small string `json:"small"`
			} `json:"images"`
		} `json:"data"`
		TotalCount int `json:"totalCount"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		return nil, 0, err
	}
	cards := make([]domain.Card, 0, len(payload.Data))
	for _, item := range payload.Data {
		imageURL := item.Images.Large
		if imageURL == "" {
			imageURL = item.Images.Small
		}
		cards = append(cards, domain.Card{
			ID: item.ID, Name: item.Name, Number: item.Number,
			SetID: item.Set.ID, SetName: item.Set.Name, ImageURL: imageURL,
		})
	}
	return cards, payload.TotalCount, nil
}
