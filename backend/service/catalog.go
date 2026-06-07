package service

import (
	"context"
	"pokemon-binder-finder/model"
	"pokemon-binder-finder/pokemontcg"
	"pokemon-binder-finder/repository"
)

type CatalogService struct {
	pokemonClient *pokemontcg.Client
	store         *repository.Store
}

func NewCatalogService(store *repository.Store) *CatalogService {
	return &CatalogService{
		pokemonClient: pokemontcg.New(),
		store:         store,
	}
}

func (c *CatalogService) Sync(ctx context.Context) error {
	for page := 1; ; page++ {
		cards, total, err := c.pokemonClient.FetchPage(ctx, page)
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

func (c *CatalogService) Search(ctx context.Context, query string, limit int) ([]model.Card, error) {
	return c.store.SearchCards(ctx, query, limit)
}

func (c *CatalogService) Get(ctx context.Context, id string) (model.Card, error) {
	return c.store.GetCard(ctx, id)
}
