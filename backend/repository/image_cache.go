package repository

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"

	"pokemon-binder-finder/model"
)

type ImageCache struct {
	dir string
}

func NewImageCache(dir string) (*ImageCache, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &ImageCache{dir: dir}, nil
}

func (c *ImageCache) Get(sourceURL string) (model.ImageAsset, bool, error) {
	name, path := c.cachePath(sourceURL)
	data, err := os.ReadFile(path)
	if err == nil {
		return model.ImageAsset{Data: data, PublicURL: "/api/assets/" + name}, true, nil
	}
	if os.IsNotExist(err) {
		return model.ImageAsset{}, false, nil
	}
	return model.ImageAsset{}, false, err
}

func (c *ImageCache) Save(sourceURL string, data []byte) (model.ImageAsset, error) {
	name, path := c.cachePath(sourceURL)
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return model.ImageAsset{}, err
	}
	return model.ImageAsset{Data: data, PublicURL: "/api/assets/" + name}, nil
}

func (c *ImageCache) cachePath(sourceURL string) (string, string) {
	sum := sha256.Sum256([]byte(sourceURL))
	name := hex.EncodeToString(sum[:]) + ".img"
	return name, filepath.Join(c.dir, name)
}
