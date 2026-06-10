package service

import (
	"context"

	"pokemon-binder-finder/ebay"
	"pokemon-binder-finder/model"
)

type EbayService struct {
	client *ebay.EbayClient
}

func NewEbayService(client *ebay.EbayClient) *EbayService {
	return &EbayService{client: client}
}

func (s *EbayService) Search(ctx context.Context, query, marketplace string, limit int) ([]model.Listing, error) {
	return s.client.Search(ctx, query, marketplace, limit)
}
