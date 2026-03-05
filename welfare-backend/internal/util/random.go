package util

import (
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"math"
	"math/big"
)

func RandomToken(size int) (string, error) {
	if size <= 0 {
		return "", fmt.Errorf("invalid token size")
	}
	buf := make([]byte, size)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func RandomDecimalInRange(min, max float64, scale int) (float64, error) {
	if scale < 0 || scale > 4 {
		return 0, fmt.Errorf("scale must be between 0 and 4")
	}
	if max < min {
		return 0, fmt.Errorf("max must >= min")
	}
	factor := math.Pow10(scale)
	minI := int64(math.Round(min * factor))
	maxI := int64(math.Round(max * factor))
	if maxI < minI {
		return 0, fmt.Errorf("invalid range after rounding")
	}
	delta := maxI - minI + 1
	n, err := rand.Int(rand.Reader, big.NewInt(delta))
	if err != nil {
		return 0, err
	}
	value := minI + n.Int64()
	return float64(value) / factor, nil
}
