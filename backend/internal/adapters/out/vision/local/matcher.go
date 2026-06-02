//go:build !gocv

package local

import (
	"bytes"
	"context"
	"fmt"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"

	"pokemon-binder-finder/internal/domain"
)

type Matcher struct{}

func New() *Matcher { return &Matcher{} }

func (m *Matcher) Match(_ context.Context, reference, candidate []byte) ([]domain.MatchCandidate, error) {
	ref, _, err := image.Decode(bytes.NewReader(reference))
	if err != nil {
		return nil, fmt.Errorf("decode reference: %w", err)
	}
	scene, _, err := image.Decode(bytes.NewReader(candidate))
	if err != nil {
		return nil, fmt.Errorf("decode listing image: %w", err)
	}
	refRatio := float64(ref.Bounds().Dx()) / float64(ref.Bounds().Dy())
	sceneBounds := scene.Bounds()
	bestScore := 0.0
	var best image.Rectangle
	for _, height := range []int{sceneBounds.Dy() / 5, sceneBounds.Dy() / 4, sceneBounds.Dy() / 3, sceneBounds.Dy() / 2, sceneBounds.Dy()} {
		width := int(float64(height) * refRatio)
		if width < 12 || height < 12 || width > sceneBounds.Dx() {
			continue
		}
		step := max(4, min(width, height)/6)
		for y := sceneBounds.Min.Y; y+height <= sceneBounds.Max.Y; y += step {
			for x := sceneBounds.Min.X; x+width <= sceneBounds.Max.X; x += step {
				rect := image.Rect(x, y, x+width, y+height)
				score := compare(ref, scene, rect)
				if score > bestScore {
					bestScore, best = score, rect
				}
			}
		}
	}
	if bestScore < 0.25 {
		return nil, nil
	}
	return []domain.MatchCandidate{{
		Polygon: []domain.PolygonPoint{
			{X: best.Min.X, Y: best.Min.Y}, {X: best.Max.X, Y: best.Min.Y},
			{X: best.Max.X, Y: best.Max.Y}, {X: best.Min.X, Y: best.Max.Y},
		},
		Confidence: bestScore,
		Reason:     "local visual similarity matched",
	}}, nil
}

func compare(reference, scene image.Image, rect image.Rectangle) float64 {
	const grid = 8
	var difference float64
	for row := 0; row < grid; row++ {
		for column := 0; column < grid; column++ {
			refX := reference.Bounds().Min.X + (column*reference.Bounds().Dx()+reference.Bounds().Dx()/2)/grid
			refY := reference.Bounds().Min.Y + (row*reference.Bounds().Dy()+reference.Bounds().Dy()/2)/grid
			x := rect.Min.X + (column*rect.Dx()+rect.Dx()/2)/grid
			y := rect.Min.Y + (row*rect.Dy()+rect.Dy()/2)/grid
			r1, g1, b1, _ := reference.At(refX, refY).RGBA()
			r2, g2, b2, _ := scene.At(x, y).RGBA()
			difference += math.Abs(float64(r1)-float64(r2)) + math.Abs(float64(g1)-float64(g2)) + math.Abs(float64(b1)-float64(b2))
		}
	}
	return math.Max(0, 1-difference/(grid*grid*3*65535))
}
