//go:build gocv

package local

import (
	"context"
	"fmt"
	"math"

	"gocv.io/x/gocv"

	"pokemon-binder-finder/internal/domain"
)

type Matcher struct{}

func New() *Matcher { return &Matcher{} }

func (m *Matcher) Match(_ context.Context, reference, candidate []byte) ([]domain.MatchCandidate, error) {
	ref, err := gocv.IMDecode(reference, gocv.IMReadGrayScale)
	if err != nil {
		return nil, fmt.Errorf("decode reference: %w", err)
	}
	defer ref.Close()
	scene, err := gocv.IMDecode(candidate, gocv.IMReadGrayScale)
	if err != nil {
		return nil, fmt.Errorf("decode listing image: %w", err)
	}
	defer scene.Close()
	if ref.Empty() || scene.Empty() {
		return nil, fmt.Errorf("decoded image is empty")
	}

	orb := gocv.NewORB()
	defer orb.Close()
	mask := gocv.NewMat()
	defer mask.Close()
	refPoints, refDescriptors := orb.DetectAndCompute(ref, mask)
	defer refDescriptors.Close()
	scenePoints, sceneDescriptors := orb.DetectAndCompute(scene, mask)
	defer sceneDescriptors.Close()
	if refDescriptors.Empty() || sceneDescriptors.Empty() {
		return nil, nil
	}

	matcher := gocv.NewBFMatcherWithParams(gocv.NormHamming, false)
	defer matcher.Close()
	var good []gocv.DMatch
	for _, matches := range matcher.KnnMatch(refDescriptors, sceneDescriptors, 2) {
		if len(matches) == 2 && matches[0].Distance < 0.75*matches[1].Distance {
			good = append(good, matches[0])
		}
	}
	if len(good) < 8 {
		return nil, nil
	}

	src := gocv.NewMatWithSize(len(good), 1, gocv.MatTypeCV64FC2)
	defer src.Close()
	dst := gocv.NewMatWithSize(len(good), 1, gocv.MatTypeCV64FC2)
	defer dst.Close()
	for index, match := range good {
		src.SetDoubleAt(index, 0, refPoints[match.QueryIdx].X)
		src.SetDoubleAt(index, 1, refPoints[match.QueryIdx].Y)
		dst.SetDoubleAt(index, 0, scenePoints[match.TrainIdx].X)
		dst.SetDoubleAt(index, 1, scenePoints[match.TrainIdx].Y)
	}
	inliers := gocv.NewMat()
	defer inliers.Close()
	homography := gocv.FindHomography(src, &dst, gocv.HomograpyMethodRANSAC, 5, &inliers, 2000, 0.995)
	defer homography.Close()
	if homography.Empty() {
		return nil, nil
	}

	corners := gocv.NewMatWithSize(4, 1, gocv.MatTypeCV32F+gocv.MatChannels2)
	defer corners.Close()
	corners.SetFloatAt(0, 0, 0)
	corners.SetFloatAt(0, 1, 0)
	corners.SetFloatAt(1, 0, float32(ref.Cols()))
	corners.SetFloatAt(1, 1, 0)
	corners.SetFloatAt(2, 0, float32(ref.Cols()))
	corners.SetFloatAt(2, 1, float32(ref.Rows()))
	corners.SetFloatAt(3, 0, 0)
	corners.SetFloatAt(3, 1, float32(ref.Rows()))
	projected := gocv.NewMat()
	defer projected.Close()
	gocv.PerspectiveTransform(corners, &projected, homography)
	polygon := make([]domain.PolygonPoint, 4)
	for index := range polygon {
		polygon[index] = domain.PolygonPoint{X: int(projected.GetFloatAt(index, 0)), Y: int(projected.GetFloatAt(index, 1))}
		if polygon[index].X < 0 || polygon[index].Y < 0 || polygon[index].X > scene.Cols() || polygon[index].Y > scene.Rows() {
			return nil, nil
		}
	}
	inlierRatio := float64(gocv.CountNonZero(inliers)) / float64(len(good))
	matchRatio := float64(len(good)) / float64(len(refPoints))
	confidence := math.Min(0.99, 0.25+inlierRatio*0.45+matchRatio*1.5)
	if confidence < 0.25 {
		return nil, nil
	}
	return []domain.MatchCandidate{{
		Polygon: polygon, Confidence: confidence,
		Reason: "ORB features and RANSAC homography matched",
	}}, nil
}
