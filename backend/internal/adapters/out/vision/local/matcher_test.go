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
	reference := image.NewRGBA(image.Rect(0, 0, 60, 84))
	for y := 0; y < 84; y++ {
		for x := 0; x < 60; x++ {
			reference.Set(x, y, color.RGBA{R: uint8(x * 4), G: uint8(y * 3), B: uint8((x + y) * 2), A: 255})
		}
	}
	scene := image.NewRGBA(image.Rect(0, 0, 180, 252))
	for y := 0; y < 84; y++ {
		for x := 0; x < 60; x++ {
			scene.Set(60+x, 84+y, reference.At(x, y))
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
