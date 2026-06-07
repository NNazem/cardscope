package pokemontcg

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strconv"
	"time"

	"pokemon-binder-finder/model"
)

type Client struct {
	baseURL string
	client  *http.Client
}

type FetchCardResponse struct {
	Data       []FetchCardData `json:"data"`
	TotalCount int             `json:"totalCount"`
}

type FetchCardData struct {
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
}

func New() *Client {
	return &Client{
		baseURL: "https://api.pokemontcg.io/v2",
		client:  &http.Client{Timeout: 30 * time.Second},
	}
}

func (c *Client) FetchPage(ctx context.Context, page int) ([]model.Card, int, error) {
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

	var fetchCardResponse = &FetchCardResponse{}
	if err := json.NewDecoder(resp.Body).Decode(&fetchCardResponse); err != nil {
		return nil, 0, err
	}
	cards := make([]model.Card, 0, len(fetchCardResponse.Data))
	for _, item := range fetchCardResponse.Data {
		imageURL := item.Images.Large
		if imageURL == "" {
			imageURL = item.Images.Small
		}
		cards = append(cards, model.Card{
			ID: item.ID, Name: item.Name, Number: item.Number,
			SetID: item.Set.ID, SetName: item.Set.Name, ImageURL: imageURL,
		})
	}
	return cards, fetchCardResponse.TotalCount, nil
}
