package capture

import (
	"context"
	"fmt"
	"image"
	"strings"
	"time"
)

// CameraSpec matches the dynamic camera definition from EdgeCVSet.
type CameraSpec struct {
	CameraID           string   `json:"camera_id"`
	StationID          string   `json:"station_id"`
	Name               string   `json:"name"`
	SourceKind         string   `json:"source_kind"` // "usb" or "rtsp"/"tcp_ip"
	SourceRef          string   `json:"source_ref"`  // e.g. "/dev/video0" or "rtsp://..."
	RTSPTransport      string   `json:"rtsp_transport,omitempty"`
	AdjacentCameraIDs  []string `json:"adjacent_camera_ids"`
	MinPresenceFrames  int      `json:"min_presence_frames"`
	LostDebounceFrames int      `json:"lost_debounce_frames"`
	Enabled            bool     `json:"enabled"`
}

// CameraSource provides raw frame acquisition from physical or network cameras.
type CameraSource interface {
	CaptureFrame(ctx context.Context) (image.Image, time.Time, error)
	Close() error
	CameraID() string
	SourceRef() string
}

// OpenCameraSource instantiates the appropriate camera capture driver based on EdgeCVSet configuration.
func OpenCameraSource(spec CameraSpec, fps int) (CameraSource, error) {
	kind := strings.ToLower(strings.TrimSpace(spec.SourceKind))
	switch kind {
	case "usb", "v4l2":
		return OpenV4L2Source(spec, fps)
	case "rtsp", "tcp_ip":
		return OpenRTSPSource(spec, fps)
	default:
		return nil, fmt.Errorf("unsupported source_kind %q for camera %s", spec.SourceKind, spec.CameraID)
	}
}
