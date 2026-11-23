package inference

import (
	"fmt"
	"image"
	"image/color"
)

// PreprocessImage performs 640x640 letterbox resizing and float32 normalization.
func PreprocessImage(img image.Image, inputWidth, inputHeight int) ([]float32, PreprocessedFrame, error) {
	if img == nil {
		return nil, PreprocessedFrame{}, fmt.Errorf("image is nil")
	}
	if inputWidth <= 0 || inputHeight <= 0 {
		return nil, PreprocessedFrame{}, fmt.Errorf("invalid input size %dx%d", inputWidth, inputHeight)
	}

	bounds := img.Bounds()
	srcW := bounds.Dx()
	srcH := bounds.Dy()
	if srcW <= 0 || srcH <= 0 {
		return nil, PreprocessedFrame{}, fmt.Errorf("invalid image bounds")
	}

	scaleX := float32(inputWidth) / float32(srcW)
	scaleY := float32(inputHeight) / float32(srcH)
	scale := scaleX
	if scaleY < scale {
		scale = scaleY
	}
	scaledW := int(float32(srcW) * scale)
	scaledH := int(float32(srcH) * scale)
	if scaledW < 1 {
		scaledW = 1
	}
	if scaledH < 1 {
		scaledH = 1
	}

	total := inputWidth * inputHeight
	data := make([]float32, total*3)

	// YOLOX 114.0 fill padding
	for i := 0; i < total; i++ {
		data[i] = 114.0
		data[total+i] = 114.0
		data[(2*total)+i] = 114.0
	}

	for y := 0; y < scaledH; y++ {
		srcY := bounds.Min.Y + (y*srcH)/scaledH
		for x := 0; x < scaledW; x++ {
			srcX := bounds.Min.X + (x*srcW)/scaledW
			r8, g8, b8 := rgbAt(img.At(srcX, srcY))
			idx := y*inputWidth + x
			data[idx] = float32(b8)             // B channel
			data[total+idx] = float32(g8)       // G channel
			data[(2*total)+idx] = float32(r8)   // R channel
		}
	}

	return data, PreprocessedFrame{
		Scale: scale,
		PadX:  0,
		PadY:  0,
		OrigW: srcW,
		OrigH: srcH,
		Img:   img,
	}, nil
}

func rgbAt(c color.Color) (uint8, uint8, uint8) {
	if c == nil {
		return 0, 0, 0
	}
	r, g, b, _ := c.RGBA()
	return uint8(r >> 8), uint8(g >> 8), uint8(b >> 8)
}
