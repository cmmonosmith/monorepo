package asciify

import (
	"errors"
	"image"
	"image/color"
	"os"
	"strings"

	// import for initialization side-effects
	//_ "image/jpeg"
	_ "image/png"
)

const (
	chars             = "$@B%8&WM#*oahkbdpqwmZO0QLCJUYXzcvunxrjft/\\|()1{}[]?-_+~<>i!lI;:,\"^`'. "
	pigeonhole_factor = float32(len(chars)) / 256
)

// Asciify converts an image to grayscale, then picks pixels at regular intervals to convert to a text character roughly
// corresponding to how dark the pixel is, and builds a multiline string from all those characters in roughly the same aspect
// ratio
func Asciify(filename string, maxWidth int, maxHeight int) (string, error) {
	if maxWidth == 0 || maxHeight == 0 {
		return "", errors.New("ascii max size must be wider/taller than 0")
	}

	// open up the image from disk
	reader, err := os.Open(filename)
	if err != nil {
		return "", err
	}
	defer reader.Close()
	m, _, err := image.Decode(reader)
	if err != nil {
		return "", err
	}

	// figure out how wide and tall the text output will actually be
	max := m.Bounds().Max
	inWidth, inHeight := max.X, max.Y
	if inWidth == 0 || inHeight == 0 {
		return "", errors.New("input image size must be wider/taller than 0")
	}
	// for particularly small images, cap the character width/height at the pixel width/height
	if inWidth < maxWidth {
		maxWidth = inWidth
	}
	if inHeight < maxHeight {
		maxHeight = inHeight
	}
	outWidth, outHeight := maxWidth, maxHeight
	// adjust output size to match the aspect ratio of the input image
	if inWidth > inHeight {
		outHeight = int(float32(outWidth) / float32(inWidth) * float32(inHeight))
	} else if inHeight > inWidth {
		outWidth = int(float32(outHeight) / float32(inHeight) * float32(inWidth))
	}
	outHeight /= 2 // Discord codeblock text is generally twice as tall as it is wide, so cut the character height in half
	// make sure rounding didn't wreck us
	if outWidth == 0 || outHeight == 0 {
		return "", errors.New("ascii output size must be wider/taller than 0")
	}
	// how many pixels does each character represent
	pixelWidth := float32(inWidth) / float32(outWidth)
	pixelHeight := float32(inHeight) / float32(outHeight)

	// scan the pixels, convert to grayscale, append corresponding characters to the output string
	var sb strings.Builder
	for y := 0; y < outHeight; y++ {
		for x := 0; x < outWidth; x++ {
			gray := color.GrayModel.Convert(m.At(int(float32(x)*pixelWidth), int(float32(y)*pixelHeight))).(color.Gray)
			sb.WriteByte(chars[uint8(float32(gray.Y)*pigeonhole_factor)])
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}
