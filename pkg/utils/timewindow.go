package utils

import (
	"errors"
	"math"
	"strconv"
	"time"
)

func ParseTimeWindow(s string) (time.Duration, error) {
	if s == "max" {
		return time.Duration(math.MaxInt64), nil
	}

	i, err := strconv.ParseInt(s[:len(s)-1], 10, 64)
	if err != nil {
		return 0, err
	}

	switch s[len(s)-1] {
	case 'h':
		return time.Duration(i) * time.Hour, nil
	case 'w':
		return time.Duration(i) * 7 * 24 * time.Hour, nil
	case 'm':
		return time.Duration(i) * 30 * 24 * time.Hour, nil
	case 'y':
		return time.Duration(i) * 265 * 24 * time.Hour, nil
	default:
		return 0, errors.New("invalid time window")
	}
}
