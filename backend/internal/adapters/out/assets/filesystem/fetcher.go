package filesystem

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"pokemon-binder-finder/internal/domain"
)

type Fetcher struct {
	dir    string
	client *http.Client
}

func New(dir string) (*Fetcher, error) {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Fetcher{dir: dir, client: &http.Client{Timeout: 20 * time.Second}}, nil
}

func (f *Fetcher) Fetch(ctx context.Context, sourceURL string) (domain.ImageAsset, error) {
	sum := sha256.Sum256([]byte(sourceURL))
	name := hex.EncodeToString(sum[:]) + ".img"
	path := filepath.Join(f.dir, name)
	data, err := os.ReadFile(path)
	if err == nil {
		return domain.ImageAsset{Data: data, PublicURL: "/api/assets/" + name}, nil
	}
	req, _ := http.NewRequestWithContext(ctx, http.MethodGet, sourceURL, nil)
	resp, err := f.client.Do(req)
	if err != nil {
		return domain.ImageAsset{}, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return domain.ImageAsset{}, fmt.Errorf("image download returned %s", resp.Status)
	}
	data, err = io.ReadAll(resp.Body)
	if err != nil {
		return domain.ImageAsset{}, err
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return domain.ImageAsset{}, err
	}
	return domain.ImageAsset{Data: data, PublicURL: "/api/assets/" + name}, nil
}
