package pages

import (
	"image"
	"io"

	"github.com/gen2brain/webp"
)

// webpQuality is the lossy quality used for all generated WebP images.
const webpQuality = 80

// encodeWebP writes img as lossy WebP. Do not use imgconv for WebP output:
// its nativewebp backend is lossless-only and silently ignores the quality
// option, producing files 5-20x larger than lossy at equal visual quality.
func encodeWebP(w io.Writer, img image.Image) error {
	return webp.Encode(w, img, webp.Options{Quality: webpQuality, Method: webp.DefaultMethod})
}
