package capture

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"image"
	"image/jpeg"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// RTSPSource captures frames from RTSP network cameras using ffmpeg in pipe mode.
type RTSPSource struct {
	spec     CameraSpec
	fps      int
	mu       sync.RWMutex
	latest   image.Image
	latestAt time.Time
	lastErr  error
	seq      uint64
	notify   chan struct{}
	closed   bool
	cancel   context.CancelFunc
}

// OpenRTSPSource starts background worker to stream frames from an RTSP camera.
func OpenRTSPSource(spec CameraSpec, targetFPS int) (*RTSPSource, error) {
	urlStr := strings.TrimSpace(spec.SourceRef)
	if urlStr == "" {
		return nil, fmt.Errorf("empty RTSP source_ref for camera %s", spec.CameraID)
	}
	if targetFPS <= 0 {
		targetFPS = 1
	}

	ctx, cancel := context.WithCancel(context.Background())
	src := &RTSPSource{
		spec:   spec,
		fps:    targetFPS,
		cancel: cancel,
		notify: make(chan struct{}),
	}

	go src.streamWorker(ctx)
	return src, nil
}

func (s *RTSPSource) streamWorker(ctx context.Context) {
	transport := "tcp"
	if strings.ToLower(s.spec.RTSPTransport) == "udp" {
		transport = "udp"
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
		}

		fpsArg := fmt.Sprintf("%d", s.fps)
		cmd := exec.CommandContext(ctx, "ffmpeg",
			"-loglevel", "error",
			"-rtsp_transport", transport,
			"-timeout", "5000000",
			"-fflags", "nobuffer",
			"-flags", "low_delay",
			"-i", s.spec.SourceRef,
			"-r", fpsArg,
			"-f", "image2pipe",
			"-vcodec", "mjpeg",
			"-q:v", "5",
			"-an",
			"-",
		)

		var stderrBuf bytes.Buffer
		cmd.Stderr = &stderrBuf

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			s.setError(fmt.Errorf("ffmpeg stdout pipe creation failed: %w", err))
			time.Sleep(2 * time.Second)
			continue
		}

		if err := cmd.Start(); err != nil {
			s.setError(fmt.Errorf("failed to start ffmpeg for %s: %w", s.spec.CameraID, err))
			time.Sleep(3 * time.Second)
			continue
		}

		var lastFrameUnix int64
		atomic.StoreInt64(&lastFrameUnix, time.Now().Unix())

		watchdogCtx, cancelWatchdog := context.WithCancel(ctx)
		go func() {
			ticker := time.NewTicker(2 * time.Second)
			defer ticker.Stop()
			for {
				select {
				case <-watchdogCtx.Done():
					return
				case <-ticker.C:
					last := atomic.LoadInt64(&lastFrameUnix)
					if time.Now().Unix()-last > 8 {
						if cmd.Process != nil {
							_ = cmd.Process.Kill()
						}
						return
					}
				}
			}
		}()

		reader := bufio.NewReaderSize(stdout, 128*1024)
		for {
			select {
			case <-ctx.Done():
				cancelWatchdog()
				_ = cmd.Process.Kill()
				return
			default:
			}

			// Scan for JPEG SOI marker (0xFF, 0xD8)
			b, err := reader.ReadByte()
			if err != nil {
				break
			}
			if b != 0xFF {
				continue
			}
			b2, err := reader.ReadByte()
			if err != nil {
				break
			}
			if b2 != 0xD8 {
				continue
			}

			var buf bytes.Buffer
			buf.WriteByte(0xFF)
			buf.WriteByte(0xD8)

			foundEOI := false
			for {
				nb, err := reader.ReadByte()
				if err != nil {
					break
				}
				buf.WriteByte(nb)
				if nb == 0xD9 && buf.Len() > 2 {
					raw := buf.Bytes()
					if raw[len(raw)-2] == 0xFF {
						foundEOI = true
						break
					}
				}
			}

			if !foundEOI {
				break
			}

			img, err := jpeg.Decode(bytes.NewReader(buf.Bytes()))
			if err != nil {
				s.setError(fmt.Errorf("jpeg decode error on camera %s: %w", s.spec.CameraID, err))
				continue
			}

			atomic.StoreInt64(&lastFrameUnix, time.Now().Unix())

			s.mu.Lock()
			s.latest = img
			s.latestAt = time.Now().UTC()
			s.seq++
			s.lastErr = nil
			s.signalLocked()
			s.mu.Unlock()
		}

		cancelWatchdog()
		_ = cmd.Wait()
		if stderrStr := strings.TrimSpace(stderrBuf.String()); stderrStr != "" {
			s.setError(fmt.Errorf("ffmpeg error on camera %s: %s", s.spec.CameraID, stderrStr))
		}

		select {
		case <-ctx.Done():
			return
		case <-time.After(2 * time.Second):
		}
	}
}

func (s *RTSPSource) setError(err error) {
	s.mu.Lock()
	s.lastErr = err
	s.signalLocked()
	s.mu.Unlock()
}

func (s *RTSPSource) CameraID() string  { return s.spec.CameraID }
func (s *RTSPSource) SourceRef() string { return s.spec.SourceRef }

// CaptureFrame awaits the next available frame or returns the latest frame if fresh.
func (s *RTSPSource) CaptureFrame(ctx context.Context) (image.Image, time.Time, error) {
	for {
		s.mu.RLock()
		img := s.latest
		capturedAt := s.latestAt
		err := s.lastErr
		notify := s.notify
		closed := s.closed
		s.mu.RUnlock()

		if closed {
			return nil, time.Time{}, fmt.Errorf("rtsp source %s closed", s.spec.CameraID)
		}
		if img != nil && !capturedAt.IsZero() && time.Since(capturedAt) < 3*time.Second {
			return img, capturedAt, nil
		}
		if err != nil {
			return nil, time.Time{}, err
		}

		select {
		case <-ctx.Done():
			if img != nil && !capturedAt.IsZero() && time.Since(capturedAt) < 3*time.Second {
				return img, capturedAt, nil
			}
			return nil, time.Time{}, ctx.Err()
		case <-notify:
		}
	}
}

// Close releases ffmpeg process and resources.
func (s *RTSPSource) Close() error {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return nil
	}
	s.closed = true
	s.cancel()
	s.signalLocked()
	s.mu.Unlock()
	return nil
}

func (s *RTSPSource) signalLocked() {
	close(s.notify)
	s.notify = make(chan struct{})
}
