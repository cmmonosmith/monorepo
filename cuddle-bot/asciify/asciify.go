package asciify

import (
	"errors"
	"fmt"
	"image"
	"image/color"
	"os"
	"slices"
	"strings"

	// import for initialization side-effects
	_ "image/jpeg"
	_ "image/png"
)

const (
	gradient      = "$@B%8&WM#*oahkbdpqwmZO0QLCJUYXzcvunxrjft/\\|()1{}[]?-_+~<>i!lI;:,\"^`'. "
	gradientRatio = float32(len(gradient)) / 256
)

var (
	extensions = []string{".png", ".jpg", ".jpeg"}
)

// Asciify converts an image to grayscale, then picks pixels at regular intervals to convert to a text character roughly
// corresponding to how dark the pixel is, and builds a multiline string from all those characters in roughly the same aspect
// ratio. The maxWidth and maxHeight parameters are in "character width" and "character height" units respectively, so the
// output height of a square image will be output width / 2.
func Asciify(filename string, maxWidth int, maxHeight int) (string, error) {
	// check some inputs
	extension := strings.ToLower(filename[strings.LastIndex(filename, "."):])
	if !slices.Contains(extensions, extension) {
		return "", fmt.Errorf("file (%s) must be of type png (.png) or jpeg (.jpg/.jpeg)", filename)
	}
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
	inWidth, inHeight := float32(max.X), float32(max.Y)
	if inWidth < 1 || inHeight < 1 {
		return "", errors.New("input image size must be wider/taller than 0")
	}
	outWidth, outHeight := float32(maxWidth), float32(maxHeight)

	// adjust output size to match the aspect ratio of the input image
	if inWidth > inHeight {
		outHeight = outWidth / inWidth * inHeight
	} else if inHeight > inWidth {
		outWidth = outHeight / inHeight * inWidth
	}

	// make sure rounding didn't wreck us somehow
	if outWidth == 0 || outHeight == 0 {
		return "", errors.New("ascii output size must be wider/taller than 0")
	}

	// scan the pixels, convert to grayscale, append corresponding characters to the output string
	var sb strings.Builder
	xMax, yMax := int(outWidth), int(outHeight)
	var yf float32
	for y := 0; y < yMax; y++ {
		yf = float32(y)
		for x := 0; x < xMax; x++ {
			gray := color.GrayModel.Convert(m.At(int(float32(x)/outWidth*inWidth), int(yf/outHeight*inHeight))).(color.Gray)
			sb.WriteByte(gradient[uint8(float32(gray.Y)*gradientRatio)])
		}
		sb.WriteString("\n")
	}

	return sb.String(), nil
}
