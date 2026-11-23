package inference

import (
	"image"
)

// Detection represents a single verified object bounding box and class.
type Detection struct {
	ClassName     string     `json:"class_name"`
	ClassID       int        `json:"class_id"`
	Confidence    float64    `json:"confidence"`
	BBox          [4]float64 `json:"bbox"` // [xmin, ymin, xmax, ymax] normalized [0.0..1.0]
	BBoxArea      float64    `json:"bbox_area"`
	Centroid      [2]float64 `json:"centroid"` // [cx, cy] normalized [0.0..1.0]
	DominantColor string     `json:"dominant_color"`
}

// InferenceOptions contains dynamic runtime options configured via EdgeCVSet.
type InferenceOptions struct {
	ConfidenceThreshold    float64
	ClassesOfInterest      []string
	ColorExtractionEnabled bool
}

// PreprocessedFrame holds letterbox geometry metadata for bbox unscaling.
type PreprocessedFrame struct {
	Scale float32
	PadX  float32
	PadY  float32
	OrigW int
	OrigH int
	Img   image.Image
}
