package tracking

import (
	"fmt"
	"math"
	"sync"
	"time"

	"bindu-plugin/bindu-edge-cv-plugin/capture"
	"bindu-plugin/bindu-edge-cv-plugin/inference"
)

// TrackedObject represents an active entity tracked in 2D image coordinates.
type TrackedObject struct {
	TrackID         string
	CameraID        string
	StationID       string
	ClassName       string
	FirstSeenAt     time.Time
	LastSeenAt      time.Time
	ConsecutiveHits int
	LostFramesCount int
	AppearedEmitted bool
	MaxBBoxArea     float64
	SumConfidence   float64
	DetectionsCount int
	Centroid        [2]float64
	DominantColor   string
}

// TrackCallbacks defines the emission hooks for state transitions.
type TrackCallbacks struct {
	OnAppeared    func(track *TrackedObject, avgConf float64)
	OnDisappeared func(track *TrackedObject, dwellDurationSeconds float64, avgConf float64)
}

// CameraTracker manages 2D tracks for one physical camera stream.
type CameraTracker struct {
	mu           sync.Mutex
	spec         capture.CameraSpec
	activeTracks map[string]*TrackedObject // key: trackID
	nextID       int64
	callbacks    TrackCallbacks
}

func NewCameraTracker(spec capture.CameraSpec, callbacks TrackCallbacks) *CameraTracker {
	return &CameraTracker{
		spec:         spec,
		activeTracks: make(map[string]*TrackedObject),
		callbacks:    callbacks,
	}
}

// ProcessDetections matches new bounding boxes against existing active tracks.
func (t *CameraTracker) ProcessDetections(dets []inference.Detection, observedAt time.Time) {
	t.mu.Lock()
	defer t.mu.Unlock()

	matchedTrackIDs := map[string]bool{}

	for _, det := range dets {
		// Find closest active track of same class
		var bestTrack *TrackedObject
		minDist := 0.25 // max normalized centroid distance to associate track

		for _, track := range t.activeTracks {
			if track.ClassName != det.ClassName {
				continue
			}
			dist := centroidDist(track.Centroid, det.Centroid)
			if dist < minDist {
				minDist = dist
				bestTrack = track
			}
		}

		if bestTrack != nil {
			// Update existing track
			bestTrack.LastSeenAt = observedAt
			bestTrack.ConsecutiveHits++
			bestTrack.LostFramesCount = 0
			bestTrack.DetectionsCount++
			bestTrack.SumConfidence += det.Confidence
			if det.BBoxArea > bestTrack.MaxBBoxArea {
				bestTrack.MaxBBoxArea = det.BBoxArea
			}
			bestTrack.Centroid = det.Centroid
			if det.DominantColor != "" && det.DominantColor != "UNKNOWN" {
				bestTrack.DominantColor = det.DominantColor
			}
			matchedTrackIDs[bestTrack.TrackID] = true
		} else {
			// Create new track
			t.nextID++
			trackID := fmt.Sprintf("track-%s-%d", t.spec.CameraID, t.nextID)
			newTrack := &TrackedObject{
				TrackID:         trackID,
				CameraID:        t.spec.CameraID,
				StationID:       t.spec.StationID,
				ClassName:       det.ClassName,
				FirstSeenAt:     observedAt,
				LastSeenAt:      observedAt,
				ConsecutiveHits: 1,
				MaxBBoxArea:     det.BBoxArea,
				SumConfidence:   det.Confidence,
				DetectionsCount: 1,
				Centroid:        det.Centroid,
				DominantColor:   det.DominantColor,
			}
			t.activeTracks[trackID] = newTrack
			bestTrack = newTrack
			matchedTrackIDs[trackID] = true
		}

		// Check min_presence_frames threshold dynamically from EdgeCVSet
		minPresence := t.spec.MinPresenceFrames
		if minPresence <= 0 {
			minPresence = 3
		}

		if bestTrack.ConsecutiveHits >= minPresence && !bestTrack.AppearedEmitted {
			bestTrack.AppearedEmitted = true
			avgConf := bestTrack.SumConfidence / float64(bestTrack.DetectionsCount)
			if t.callbacks.OnAppeared != nil {
				t.callbacks.OnAppeared(bestTrack, avgConf)
			}
		}
	}

	// Increment lost frame counter for tracks not matched in this frame
	maxLost := t.spec.LostDebounceFrames
	if maxLost <= 0 {
		maxLost = 5
	}

	for id, track := range t.activeTracks {
		if matchedTrackIDs[id] {
			continue
		}
		track.LostFramesCount++
		if track.LostFramesCount >= maxLost {
			if track.AppearedEmitted {
				// Exact real measured dwell duration
				dwellDuration := track.LastSeenAt.Sub(track.FirstSeenAt).Seconds()
				if dwellDuration < 0.5 {
					dwellDuration = 0.5
				}
				avgConf := track.SumConfidence / float64(math.Max(1, float64(track.DetectionsCount)))
				if t.callbacks.OnDisappeared != nil {
					t.callbacks.OnDisappeared(track, math.Round(dwellDuration*10)/10, avgConf)
				}
			}
			delete(t.activeTracks, id)
		}
	}
}

func centroidDist(a, b [2]float64) float64 {
	dx := a[0] - b[0]
	dy := a[1] - b[1]
	return math.Sqrt(dx*dx + dy*dy)
}
