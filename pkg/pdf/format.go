package pdf

import (
	"fmt"
	"strings"
)

// FormatRupiah formats an integer amount to Indonesian Rupiah currency format (e.g. Rp 150.000)
func FormatRupiah(amount int) string {
	if amount == 0 {
		return "Rp 0"
	}

	isNegative := false
	if amount < 0 {
		isNegative = true
		amount = -amount
	}

	str := fmt.Sprintf("%d", amount)
	n := len(str)
	if n <= 3 {
		if isNegative {
			return "-Rp " + str
		}
		return "Rp " + str
	}

	var parts []string
	rem := n % 3
	if rem > 0 {
		parts = append(parts, str[:rem])
	}
	for i := rem; i < n; i += 3 {
		parts = append(parts, str[i:i+3])
	}

	result := strings.Join(parts, ".")
	if isNegative {
		return "-Rp " + result
	}
	return "Rp " + result
}
