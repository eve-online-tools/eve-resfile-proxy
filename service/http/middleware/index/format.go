package index

import (
	"fmt"
	"math"
)

func formatFileSize(size int64) string {
	if size < 0 {
		return "-"
	}
	const unit = 1024
	if size < unit {
		return fmt.Sprintf("%d", size)
	}
	exp := int(math.Log(float64(size)) / math.Log(unit))
	suffix := "KMGTPE"[exp-1 : exp]
	value := float64(size) / math.Pow(unit, float64(exp))
	if value < 10 && value != math.Floor(value) {
		return fmt.Sprintf("%.1f%s", value, suffix)
	}
	return fmt.Sprintf("%.0f%s", value, suffix)
}

func formatCompressionPercent(size, compressed int64) string {
	if size <= 0 || compressed <= 0 {
		return "-"
	}
	pct := 100.0 * (1.0 - float64(compressed)/float64(size))
	if pct < 0 {
		return "0%"
	}
	return fmt.Sprintf("%.0f%%", pct)
}
