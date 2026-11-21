package appCore

import (
	"fmt"
	"os"
	"strconv"
)

func retentionDays(name string, fallback int) (int, error) {
	raw := os.Getenv(name)
	if raw == "" {
		return fallback, nil
	}
	v, err := strconv.Atoi(raw)
	if err != nil || v < 1 || v > 3650 {
		return 0, fmt.Errorf("%s must be 1..3650 days", name)
	}
	return v, nil
}
