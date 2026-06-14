package vision

import (
	"context"
	"fmt"
	"math"

	"gocv.io/x/gocv"

	"pokemon-binder-finder/model"
)

const (
	nearestMatchCount = 2
	loweRatio         = 0.75
	minGoodMatches    = 8
)

type Matcher struct{}

func NewMatcher() *Matcher { return &Matcher{} }

func (m *Matcher) Match(_ context.Context, reference, candidate []byte) ([]model.MatchCandidate, error) {
	referenceImage, err := readImage(reference)
	if err != nil {
		return nil, err
	}
	defer referenceImage.Close()
	candidateImage, err := readImage(candidate)
	if err != nil {
		return nil, fmt.Errorf("decode listing image: %w", err)
	}
	defer candidateImage.Close()

	refPoints, refDescriptors := detectKeypoints(referenceImage)
	defer refDescriptors.Close()
	candidatePoints, candidateDescriptors := detectKeypoints(candidateImage)
	defer candidateDescriptors.Close()
	if refDescriptors.Empty() || candidateDescriptors.Empty() {
		return nil, nil
	}

	good := findGoodDescriptorsMatches(refDescriptors, candidateDescriptors)
	if len(good) < minGoodMatches {
		return nil, nil
	}

	homography, inliers := findHomography(good, refPoints, candidatePoints)
	defer inliers.Close()
	defer homography.Close()
	if homography.Empty() {
		return nil, nil
	}

	corners := extractPoints(referenceImage)
	defer corners.Close()

	polygonPoints := projectReferencePolygon(corners, homography, candidateImage)
	if polygonPoints == nil {
		return nil, nil
	}

	inlierRatio := float64(gocv.CountNonZero(inliers)) / float64(len(good))
	matchRatio := float64(len(good)) / float64(len(refPoints))
	confidence := math.Min(0.99, 0.25+inlierRatio*0.45+matchRatio*1.5)

	return []model.MatchCandidate{{
		Polygon: polygonPoints, Confidence: confidence,
		Reason: "ORB features and RANSAC homography matched",
	}}, nil
}

func projectReferencePolygon(corners gocv.Mat, homography gocv.Mat, candidateImage gocv.Mat) []model.PolygonPoint {
	projected := gocv.NewMat()
	defer projected.Close()
	err := gocv.PerspectiveTransform(corners, &projected, homography)
	if err != nil {
		return nil
	}
	polygon := make([]model.PolygonPoint, 4)
	for index := range polygon {
		polygon[index] = model.PolygonPoint{X: int(projected.GetFloatAt(index, 0)), Y: int(projected.GetFloatAt(index, 1))}
		if polygon[index].X < 0 || polygon[index].Y < 0 || polygon[index].X > candidateImage.Cols() || polygon[index].Y > candidateImage.Rows() {
			return nil
		}
	}
	return polygon
}

func extractPoints(referenceImage gocv.Mat) gocv.Mat {
	width := float32(referenceImage.Cols())
	height := float32(referenceImage.Rows())

	corners := gocv.NewPoint2fVectorFromPoints([]gocv.Point2f{
		gocv.NewPoint2f(0, 0),
		gocv.NewPoint2f(width, 0),
		gocv.NewPoint2f(width, height),
		gocv.NewPoint2f(0, height),
	})

	defer corners.Close()

	return gocv.NewMatFromPoint2fVector(corners, true)
}

func findHomography(good []gocv.DMatch, refPoints []gocv.KeyPoint, candidatePoints []gocv.KeyPoint) (homography gocv.Mat, inliers gocv.Mat) {
	src := gocv.NewMatWithSize(len(good), 1, gocv.MatTypeCV64FC2)
	defer src.Close()
	dst := gocv.NewMatWithSize(len(good), 1, gocv.MatTypeCV64FC2)
	defer dst.Close()
	for index, match := range good {
		src.SetDoubleAt(index, 0, refPoints[match.QueryIdx].X)
		src.SetDoubleAt(index, 1, refPoints[match.QueryIdx].Y)
		dst.SetDoubleAt(index, 0, candidatePoints[match.TrainIdx].X)
		dst.SetDoubleAt(index, 1, candidatePoints[match.TrainIdx].Y)
	}
	inliers = gocv.NewMat()
	homography = gocv.FindHomography(src, dst, gocv.HomographyMethodRANSAC, 5, &inliers, 2000, 0.995)
	return homography, inliers
}

func findGoodDescriptorsMatches(refDescriptors gocv.Mat, candidateDescriptors gocv.Mat) []gocv.DMatch {
	matcher := gocv.NewBFMatcherWithParams(gocv.NormHamming, false)
	defer matcher.Close()
	var good []gocv.DMatch
	for _, matches := range matcher.KnnMatch(refDescriptors, candidateDescriptors, nearestMatchCount) {
		if len(matches) == nearestMatchCount && matches[0].Distance < loweRatio*matches[1].Distance {
			good = append(good, matches[0])
		}
	}
	return good
}

func detectKeypoints(referenceImage gocv.Mat) ([]gocv.KeyPoint, gocv.Mat) {
	orb := gocv.NewORB()
	defer orb.Close()
	mask := gocv.NewMat()
	defer mask.Close()

	return orb.DetectAndCompute(referenceImage, mask)
}

func readImage(reference []byte) (gocv.Mat, error) {
	ref, err := gocv.IMDecode(reference, gocv.IMReadGrayScale)
	if err != nil {
		return gocv.Mat{}, fmt.Errorf("decode reference: %w", err)
	}

	if ref.Empty() {
		return gocv.Mat{}, fmt.Errorf("decoded image is empty")
	}
	return ref, err
}
