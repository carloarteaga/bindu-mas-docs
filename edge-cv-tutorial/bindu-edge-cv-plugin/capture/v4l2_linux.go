//go:build linux

package capture

import (
	"bytes"
	"context"
	"fmt"
	"image"
	"image/color"
	"image/jpeg"
	"strings"
	"sync"
	"time"

	"github.com/vladimirvivien/go4vl/device"
	"github.com/vladimirvivien/go4vl/v4l2"
)

type V4L2Source struct {
	spec        CameraSpec
	devicePath  string
	fps         int
	device      *device.Device
	pixelFormat v4l2.FourCCType
	width       int
	height      int

	ctx      context.Context
	cancel   context.CancelFunc
	mu       sync.RWMutex
	latest   image.Image
	latestAt time.Time
	lastErr  error
	notify   chan struct{}
	closed   bool
}

func OpenV4L2Source(spec CameraSpec, targetFPS int) (*V4L2Source, error) {
	devicePath := strings.TrimSpace(spec.SourceRef)
	if devicePath == "" {
		devicePath = "/dev/video0"
	}
	if targetFPS <= 0 {
		targetFPS = 5
	}

	options := []device.Option{
		device.WithVideoCaptureEnabled(),
		device.WithBufferSize(2),
		device.WithFPS(uint32(targetFPS)),
	}

	dev, err := device.Open(devicePath, options...)
	if err != nil {
		return nil, fmt.Errorf("open v4l2 device %s: %w", devicePath, err)
	}

	pixFmt, err := dev.GetPixFormat()
	if err != nil {
		_ = dev.Close()
		return nil, fmt.Errorf("get pix format on %s: %w", devicePath, err)
	}

	// Try MJPEG or JPEG format for efficiency
	targetFmt := pixFmt
	formats, _ := dev.GetFormatDescriptions()
	for _, f := range formats {
		if f.PixelFormat == v4l2.PixelFmtMJPEG || f.PixelFormat == v4l2.PixelFmtJPEG {
			targetFmt.PixelFormat = f.PixelFormat
			break
		}
	}
	if targetFmt.PixelFormat != pixFmt.PixelFormat {
		_ = dev.SetPixFormat(targetFmt)
		pixFmt, _ = dev.GetPixFormat()
	}

	_ = dev.GetFrames() // register streaming mode

	ctx, cancel := context.WithCancel(context.Background())
	if err := dev.Start(ctx); err != nil {
		cancel()
		_ = dev.Close()
		return nil, fmt.Errorf("start v4l2 stream on %s: %w", devicePath, err)
	}

	frames := dev.GetFrames()
	if frames == nil {
		cancel()
		_ = dev.Stop()
		_ = dev.Close()
		return nil, fmt.Errorf("no frames channel exposed by %s", devicePath)
	}

	src := &V4L2Source{
		spec:        spec,
		devicePath:  devicePath,
		fps:         targetFPS,
		device:      dev,
		pixelFormat: pixFmt.PixelFormat,
		width:       int(pixFmt.Width),
		height:      int(pixFmt.Height),
		ctx:         ctx,
		cancel:      cancel,
		notify:      make(chan struct{}),
	}

	go src.streamLoop(frames)
	return src, nil
}

func (s *V4L2Source) CameraID() string  { return s.spec.CameraID }
func (s *V4L2Source) SourceRef() string { return s.devicePath }

func (s *V4L2Source) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.cancel()
	s.signalLocked()
	s.mu.Unlock()

	if s.device != nil {
		_ = s.device.Stop()
		return s.device.Close()
	}
	return nil
}

func (s *V4L2Source) CaptureFrame(ctx context.Context) (image.Image, time.Time, error) {
	for {
		s.mu.RLock()
		img := s.latest
		capturedAt := s.latestAt
		err := s.lastErr
		notify := s.notify
		closed := s.closed
		s.mu.RUnlock()

		if img != nil && !capturedAt.IsZero() {
			return img, capturedAt, nil
		}
		if closed {
			return nil, time.Time{}, fmt.Errorf("v4l2 source %s closed", s.devicePath)
		}
		if err != nil {
			return nil, time.Time{}, err
		}

		select {
		case <-ctx.Done():
			return nil, time.Time{}, ctx.Err()
		case <-notify:
		}
	}
}

func (s *V4L2Source) streamLoop(frames <-chan *device.Frame) {
	for {
		select {
		case <-s.ctx.Done():
			return
		case frame, ok := <-frames:
			if !ok {
				s.setErr(fmt.Errorf("frame stream closed"))
				return
			}
			if frame == nil {
				continue
			}

			img, err := s.decode(frame.Data)
			now := time.Now().UTC()
			frame.Release()

			if err != nil {
				s.setErr(err)
				continue
			}

			s.mu.Lock()
			s.latest = img
			s.latestAt = now
			s.lastErr = nil
			s.signalLocked()
			s.mu.Unlock()
		}
	}
}

func (s *V4L2Source) decode(raw []byte) (image.Image, error) {
	switch s.pixelFormat {
	case v4l2.PixelFmtMJPEG, v4l2.PixelFmtJPEG:
		return jpeg.Decode(bytes.NewReader(raw))
	case v4l2.PixelFmtYUYV:
		return decodeYUYV(raw, s.width, s.height)
	default:
		// Attempt JPEG fallback
		img, err := jpeg.Decode(bytes.NewReader(raw))
		if err == nil {
			return img, nil
		}
		return nil, fmt.Errorf("unsupported pixel format: %v", s.pixelFormat)
	}
}

func (s *V4L2Source) setErr(err error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.lastErr = err
	s.signalLocked()
}

func (s *V4L2Source) signalLocked() {
	close(s.notify)
	s.notify = make(chan struct{})
}

func decodeYUYV(raw []byte, width, height int) (image.Image, error) {
	if width <= 0 || height <= 0 {
		return nil, fmt.Errorf("invalid dimensions %dx%d", width, height)
	}
	expected := width * height * 2
	if len(raw) < expected {
		return nil, fmt.Errorf("short buffer for yuyv")
	}
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x += 2 {
			offset := (y*width + x) * 2
			y0 := raw[offset+0]
			u := raw[offset+1]
			y1 := raw[offset+2]
			v := raw[offset+3]

			r0, g0, b0 := yuv2rgb(y0, u, v)
			img.Set(x, y, color.RGBA{R: r0, G: g0, B: b0, A: 255})
			if x+1 < width {
				r1, g1, b1 := yuv2rgb(y1, u, v)
				img.Set(x+1, y, color.RGBA{R: r1, G: g1, B: b1, A: 255})
			}
		}
	}
	return img, nil
}

func yuv2rgb(y, u, v uint8) (uint8, uint8, uint8) {
	c := int(y) - 16
	d := int(u) - 128
	e := int(v) - 128
	r := clampInt((298*c + 409*e + 128) >> 8)
	g := clampInt((298*c - 100*d - 208*e + 128) >> 8)
	b := clampInt((298*c + 516*d + 128) >> 8)
	return uint8(r), uint8(g), uint8(b)
}

func clampInt(val int) int {
	if val < 0 {
		return 0
	}
	if val > 255 {
		return 255
	}
	return val
}
