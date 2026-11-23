module bindu-plugin/bindu-edge-cv-plugin

go 1.25.0

require (
	bindunet v0.0.0
	github.com/bluenviron/gortsplib/v4 v4.15.0
	github.com/pion/rtp v1.8.20
	github.com/vladimirvivien/go4vl v0.3.0
	github.com/yalue/onnxruntime_go v1.27.0
)

require (
	github.com/bluenviron/mediacommon/v2 v2.3.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/pion/logging v0.2.3 // indirect
	github.com/pion/randutil v0.1.0 // indirect
	github.com/pion/rtcp v1.2.15 // indirect
	github.com/pion/sdp/v3 v3.0.14 // indirect
	github.com/pion/srtp/v3 v3.0.6 // indirect
	github.com/pion/transport/v3 v3.0.7 // indirect
	golang.org/x/net v0.56.0 // indirect
	golang.org/x/sys v0.47.0 // indirect
)

replace bindunet => ../../../../..
