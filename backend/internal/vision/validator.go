package vision

import (
	"bytes"
	"context"
	"errors"
	"image"
	_ "image/jpeg"
	_ "image/png"
	"math"
	"net/http"
	"strings"
	"unicode"
	"unicode/utf8"

	_ "golang.org/x/image/webp"
)

const (
	MaxImageBytes   = 5 << 20
	MaxUploadBytes  = MaxImageBytes + (64 << 10)
	MaxReasonLength = 300
	maxImagePixels  = 20_000_000
)

type Validator interface {
	Validate(context.Context, ValidationRequest) (ValidationResult, error)
}

// CheckImage ignores the filename and claimed MIME type. DecodeConfig bounds
// allocations before a full decode catches truncated or corrupt image data.
func CheckImage(data []byte) (string, error) {
	if len(data) == 0 || len(data) > MaxImageBytes {
		return "", errors.New("photo vide ou supérieure à 5 Mo")
	}
	mediaType := http.DetectContentType(data)
	formats := map[string]string{"image/jpeg": "jpeg", "image/png": "png", "image/webp": "webp"}
	want, ok := formats[mediaType]
	if !ok {
		return "", errors.New("utilise une photo JPEG, PNG ou WebP")
	}
	config, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil || format != want || config.Width <= 0 || config.Height <= 0 || int64(config.Width)*int64(config.Height) > maxImagePixels {
		return "", errors.New("photo illisible ou trop grande (20 mégapixels maximum)")
	}
	if _, _, err = image.Decode(bytes.NewReader(data)); err != nil {
		return "", errors.New("photo endommagée ou incomplète")
	}
	return mediaType, nil
}

func CheckResult(result ValidationResult) error {
	if result.Verdict != Valid && result.Verdict != Invalid && result.Verdict != Uncertain {
		return errors.New("invalid vision verdict")
	}
	if math.IsNaN(result.Confidence) || math.IsInf(result.Confidence, 0) || result.Confidence < 0 || result.Confidence > 1 {
		return errors.New("invalid vision confidence")
	}
	if !utf8.ValidString(result.Reason) || strings.TrimSpace(result.Reason) == "" || utf8.RuneCountInString(result.Reason) > MaxReasonLength {
		return errors.New("invalid vision reason")
	}
	for _, r := range result.Reason {
		if unicode.IsControl(r) {
			return errors.New("invalid vision reason")
		}
	}
	return nil
}
