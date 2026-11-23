package main

import (
	"math"
	"sync"
	"time"
)

// RawObservation represents a single physical BLE advertising observation from the scanner.
type RawObservation struct {
	BeaconID string
	Protocol string
	RSSI     int
	TxPower  int
	UUID     string
	Major    uint16
	Minor    uint16
	SeenAt   time.Time
}

// PresenceTransition represents a state change event (SEEN or LOST).
type PresenceTransition struct {
	Type              string // "SEEN" or "LOST"
	BeaconID          string
	Protocol          string
	Status            string // "SEEN" or "LOST"
	SeenAt            time.Time
	LostAt            time.Time
	LastSeenAt        time.Time
	RSSIRaw           int
	RSSIFiltered      float64
	DistanceEstimateM float64
	TxPower           int
	UUID              string
	Major             uint16
	Minor             uint16
	ObservationCount  int
	AbsenceWindowMS   int
	Reason            string
}

type beaconTrack struct {
	beaconID         string
	protocol         string
	uuid             string
	major            uint16
	minor            uint16
	txPower          int
	present          bool
	firstCandidateAt time.Time
	lastSeenAt       time.Time
	observationCount int
	lastRawRSSI      int
	filteredRSSI     float64
	initialized      bool
}

// PresenceEngine manages the temporal state machine and RSSI filtering.
type PresenceEngine struct {
	mu               sync.Mutex
	presenceWindow   time.Duration
	absenceWindow    time.Duration
	pathLossExponent float64
	tracks           map[string]*beaconTrack
	onTransition     func(trans PresenceTransition)
}

// NewPresenceEngine creates a new presence engine with configured time windows.
func NewPresenceEngine(presenceWindowMS, absenceWindowMS int, onTransition func(trans PresenceTransition)) *PresenceEngine {
	if presenceWindowMS <= 0 {
		presenceWindowMS = 1000
	}
	if absenceWindowMS <= 0 {
		absenceWindowMS = 5000
	}
	return &PresenceEngine{
		presenceWindow:   time.Duration(presenceWindowMS) * time.Millisecond,
		absenceWindow:    time.Duration(absenceWindowMS) * time.Millisecond,
		pathLossExponent: 2.0,
		tracks:           make(map[string]*beaconTrack),
		onTransition:     onTransition,
	}
}

// Ingest processes a single raw BLE packet and evaluates the presence candidate threshold.
func (e *PresenceEngine) Ingest(obs RawObservation) {
	e.mu.Lock()
	defer e.mu.Unlock()

	now := obs.SeenAt
	if now.IsZero() {
		now = time.Now().UTC()
	}

	track, exists := e.tracks[obs.BeaconID]
	if !exists {
		track = &beaconTrack{
			beaconID:         obs.BeaconID,
			protocol:         obs.Protocol,
			uuid:             obs.UUID,
			major:            obs.Major,
			minor:            obs.Minor,
			txPower:          obs.TxPower,
			filteredRSSI:     float64(obs.RSSI),
			initialized:      true,
			firstCandidateAt: now,
		}
		e.tracks[obs.BeaconID] = track
	}

	// Exponential Moving Average (EMA) for RSSI smoothing (alpha = 0.25)
	alpha := 0.25
	track.filteredRSSI = (alpha * float64(obs.RSSI)) + ((1.0 - alpha) * track.filteredRSSI)
	track.lastRawRSSI = obs.RSSI
	track.lastSeenAt = now
	track.observationCount++
	track.protocol = obs.Protocol
	track.uuid = obs.UUID
	track.major = obs.Major
	track.minor = obs.Minor
	track.txPower = obs.TxPower

	if track.firstCandidateAt.IsZero() {
		track.firstCandidateAt = now
	}

	// If already stably present, no new SEEN transition is triggered
	if track.present {
		return
	}

	// Verify candidate dwell time
	dwell := now.Sub(track.firstCandidateAt)
	if dwell >= e.presenceWindow && track.observationCount >= 2 {
		track.present = true
		dist := e.estimateDistance(track.txPower, track.filteredRSSI)

		if e.onTransition != nil {
			e.onTransition(PresenceTransition{
				Type:              "SEEN",
				BeaconID:          track.beaconID,
				Protocol:          track.protocol,
				Status:            "SEEN",
				SeenAt:            now,
				RSSIRaw:           track.lastRawRSSI,
				RSSIFiltered:      math.Round(track.filteredRSSI*10) / 10,
				DistanceEstimateM: math.Round(dist*100) / 100,
				TxPower:           track.txPower,
				UUID:              track.uuid,
				Major:             track.major,
				Minor:             track.minor,
				ObservationCount:  track.observationCount,
				Reason:            "STABLE_DWELL_REACHED",
			})
		}
	}
}

// Sweep checks for beacon absence when silence exceeds the absence threshold.
func (e *PresenceEngine) Sweep(now time.Time) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if now.IsZero() {
		now = time.Now().UTC()
	}

	for _, track := range e.tracks {
		if !track.present {
			continue
		}

		if now.Sub(track.lastSeenAt) > e.absenceWindow {
			track.present = false
			track.firstCandidateAt = time.Time{}
			track.observationCount = 0

			if e.onTransition != nil {
				e.onTransition(PresenceTransition{
					Type:            "LOST",
					BeaconID:        track.beaconID,
					Protocol:        track.protocol,
					Status:          "LOST",
					LastSeenAt:      track.lastSeenAt,
					LostAt:          now,
					AbsenceWindowMS: int(e.absenceWindow / time.Millisecond),
					RSSIFiltered:    math.Round(track.filteredRSSI*10) / 10,
					UUID:            track.uuid,
					Major:           track.major,
					Minor:           track.minor,
					Reason:          "ABSENCE_TIMEOUT",
				})
			}
		}
	}
}

// estimateDistance calculates metric distance using the log-distance path loss model.
func (e *PresenceEngine) estimateDistance(txPower int, rssi float64) float64 {
	if txPower == 0 {
		txPower = -59 // Standard iBeacon calibrated power at 1 meter
	}
	ratio := (float64(txPower) - rssi) / (10.0 * e.pathLossExponent)
	dist := math.Pow(10.0, ratio)
	if dist < 0.1 {
		return 0.1
	}
	return dist
}
