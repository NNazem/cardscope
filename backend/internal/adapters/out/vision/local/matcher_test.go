package local

import (
	"bytes"
	"context"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func TestMatchFindsReferenceInScene(t *testing.T) {
	reference := image.NewRGBA(image.Rect(0, 0, 180, 252))
	for y := 0; y < 252; y++ {
		for x := 0; x < 180; x++ {
			value := uint8((x*31 + y*17 + x*y) % 255)
			if (x/12+y/12)%2 == 0 {
				value = 255 - value
			}
			reference.Set(x, y, color.RGBA{R: value, G: uint8(x*y) ^ value, B: uint8(x*7+y*13) ^ value, A: 255})
		}
	}
	scene := image.NewRGBA(image.Rect(0, 0, 540, 756))
	for y := 0; y < 252; y++ {
		for x := 0; x < 180; x++ {
			scene.Set(180+x, 252+y, reference.At(x, y))
		}
	}
	var referenceData, sceneData bytes.Buffer
	_ = png.Encode(&referenceData, reference)
	_ = png.Encode(&sceneData, scene)
	matches, err := New().Match(context.Background(), referenceData.Bytes(), sceneData.Bytes())
	if err != nil || len(matches) == 0 {
		t.Fatalf("expected a match, got %#v: %v", matches, err)
	}
}
