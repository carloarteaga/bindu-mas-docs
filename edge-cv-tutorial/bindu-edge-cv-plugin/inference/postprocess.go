package inference

import (
	"fmt"
	"image"
	"math"
	"sort"
)

// DecodeYOLOXOutput parses the raw tensor floats into verified Detection structs.
func DecodeYOLOXOutput(output []float32, labels []string, prep PreprocessedFrame, opts InferenceOptions) ([]Detection, error) {
	const (
		featuresPerAnchor = 85
		numClasses        = 80
	)
	if len(output)%featuresPerAnchor != 0 {
		return nil, fmt.Errorf("unexpected output tensor size %d", len(output))
	}
	numAnchors := len(output) / featuresPerAnchor
	inputWidth := int(float32(prep.OrigW) * prep.Scale)
	if inputWidth < 640 {
		inputWidth = 640
	}
	inputHeight := 640

	grids, strides, err := yoloxGrid(numAnchors, inputWidth, inputHeight)
	if err != nil {
		return nil, err
	}

	confThresh := float32(opts.ConfidenceThreshold)
	if confThresh <= 0 {
		confThresh = 0.40
	}

	classFilter := map[string]struct{}{}
	for _, c := range opts.ClassesOfInterest {
		classFilter[c] = struct{}{}
	}

	var candidates []Detection
	for i := 0; i < numAnchors; i++ {
		base := i * featuresPerAnchor
		cx := (output[base+0] + grids[i][0]) * strides[i]
		cy := (output[base+1] + grids[i][1]) * strides[i]
		w := float32(math.Exp(float64(output[base+2]))) * strides[i]
		h := float32(math.Exp(float64(output[base+3]))) * strides[i]
		obj := output[base+4]
		if obj <= 0 {
			continue
		}

		bestClass, bestScore := -1, float32(0)
		for c := 0; c < numClasses; c++ {
			score := obj * output[base+5+c]
			if score > bestScore {
				bestScore = score
				bestClass = c
			}
		}

		if bestClass < 0 || bestScore < confThresh {
			continue
		}

		label := ""
		if bestClass < len(labels) {
			label = labels[bestClass]
		}
		if len(classFilter) > 0 {
			if _, ok := classFilter[label]; !ok {
				continue
			}
		}

		// Undo letterbox to original pixel coordinates
		x1 := (cx - (w / 2)) / prep.Scale
		y1 := (cy - (h / 2)) / prep.Scale
		x2 := (cx + (w / 2)) / prep.Scale
		y2 := (cy + (h / 2)) / prep.Scale

		x1 = clamp(x1, 0, float32(prep.OrigW))
		y1 = clamp(y1, 0, float32(prep.OrigH))
		x2 = clamp(x2, 0, float32(prep.OrigW))
		y2 = clamp(y2, 0, float32(prep.OrigH))

		if x2 <= x1 || y2 <= y1 {
			continue
		}

		// Calculate normalized coordinates [0.0..1.0]
		normX1 := float64(x1) / float64(prep.OrigW)
		normY1 := float64(y1) / float64(prep.OrigH)
		normX2 := float64(x2) / float64(prep.OrigW)
		normY2 := float64(y2) / float64(prep.OrigH)

		normW := normX2 - normX1
		normH := normY2 - normY1
		area := normW * normH
		centroidX := normX1 + (normW / 2.0)
		centroidY := normY1 + (normH / 2.0)

		colorName := "UNKNOWN"
		if opts.ColorExtractionEnabled && prep.Img != nil {
			colorName = extractDominantColor(prep.Img, int(x1), int(y1), int(x2), int(y2))
		}

		candidates = append(candidates, Detection{
			ClassName:     label,
			ClassID:       bestClass,
			Confidence:    math.Round(float64(bestScore)*100) / 100,
			BBox:          [4]float64{normX1, normY1, normX2, normY2},
			BBoxArea:      math.Round(area*1000) / 1000,
			Centroid:      [2]float64{centroidX, centroidY},
			DominantColor: colorName,
		})
	}

	return nms(candidates, 0.45), nil
}

func yoloxGrid(numAnchors, inputWidth, inputHeight int) ([][2]float32, []float32, error) {
	strideList := []int{8, 16, 32}
	var grids [][2]float32
	var strides []float32
	for _, stride := range strideList {
		w := inputWidth / stride
		h := inputHeight / stride
		for y := 0; y < h; y++ {
			for x := 0; x < w; x++ {
				grids = append(grids, [2]float32{float32(x), float32(y)})
				strides = append(strides, float32(stride))
			}
		}
	}
	if len(grids) != numAnchors {
		return nil, nil, fmt.Errorf("grid mismatch: expected %d, got %d", numAnchors, len(grids))
	}
	return grids, strides, nil
}

func clamp(v, minV, maxV float32) float32 {
	if v < minV {
		return minV
	}
	if v > maxV {
		return maxV
	}
	return v
}

func nms(dets []Detection, iouThresh float64) []Detection {
	if len(dets) == 0 {
		return nil
	}
	sort.Slice(dets, func(i, j int) bool {
		return dets[i].Confidence > dets[j].Confidence
	})

	var kept []Detection
	for len(dets) > 0 {
		best := dets[0]
		kept = append(kept, best)
		var remaining []Detection
		for _, cand := range dets[1:] {
			if calcIoU(best.BBox, cand.BBox) < iouThresh {
				remaining = append(remaining, cand)
			}
		}
		dets = remaining
	}
	return kept
}

func calcIoU(a, b [4]float64) float64 {
	x1 := math.Max(a[0], b[0])
	y1 := math.Max(a[1], b[1])
	x2 := math.Min(a[2], b[2])
	y2 := math.Min(a[3], b[3])

	interW := math.Max(0, x2-x1)
	interH := math.Max(0, y2-y1)
	interArea := interW * interH

	areaA := (a[2] - a[0]) * (a[3] - a[1])
	areaB := (b[2] - b[0]) * (b[3] - b[1])
	unionArea := areaA + areaB - interArea
	if unionArea <= 0 {
		return 0
	}
	return interArea / unionArea
}

func extractDominantColor(img image.Image, x1, y1, x2, y2 int) string {
	var totalR, totalG, totalB, count int64
	stepX := int(math.Max(1, float64(x2-x1)/10))
	stepY := int(math.Max(1, float64(y2-y1)/10))

	for y := y1; y < y2; y += stepY {
		for x := x1; x < x2; x += stepX {
			r, g, b, _ := img.At(x, y).RGBA()
			totalR += int64(r >> 8)
			totalG += int64(g >> 8)
			totalB += int64(b >> 8)
			count++
		}
	}
	if count == 0 {
		return "DARK"
	}
	avgR := totalR / count
	avgG := totalG / count
	avgB := totalB / count

	if avgR < 60 && avgG < 60 && avgB < 60 {
		return "DARK"
	}
	if avgR > 180 && avgG > 180 && avgB > 180 {
		return "WHITE"
	}
	if avgB > avgR+30 && avgB > avgG+20 {
		return "NAVY_BLUE"
	}
	if avgR > avgG+30 && avgR > avgB+30 {
		return "RED"
	}
	if avgG > avgR+20 && avgG > avgB+20 {
		return "GREEN"
	}
	return "NEUTRAL"
}
