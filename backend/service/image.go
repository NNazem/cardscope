package service

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"pokemon-binder-finder/model"
	"pokemon-binder-finder/repository"
	"time"
)

type ImageService struct {
	imageCache  *repository.ImageCache
	imageClient *http.Client
}

func NewImageService(imageCache *repository.ImageCache) *ImageService {
	return &ImageService{
		imageClient: &http.Client{Timeout: 20 * time.Second},
		imageCache:  imageCache,
	}
}

func (i *ImageService) FetchImage(ctx context.Context, sourceURL string) (model.ImageAsset, error) {
	if asset, ok, err := i.imageCache.Get(sourceURL); ok || err != nil {
		return asset, err
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	resp, err := i.imageClient.Do(req)
	if err != nil {
		return model.ImageAsset{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return model.ImageAsset{}, fmt.Errorf("image download returned %s", resp.Status)
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return model.ImageAsset{}, err
	}
	return i.imageCache.Save(sourceURL, data)
}
