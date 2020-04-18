package utils

import (
	"net/url"
)

func IsValidUrl(str string) bool {
	u, err := url.Parse(str)
	if err != nil || u.Scheme == "" || u.Host == "" {
		return false
	}

	return true
}

// Floor returns the floor nearest to a multiple of m
func Floor(x, m int) int {
	return x / m * m
}

// Floor0 returns the floor nearest to a multiple of m
// With the exception that ZERO above a block will be floored to the previous block
func Floor0(x, m int) int {
	return (x - 1) / m * m
}

func Ceil(x, m int) int {
	return (x + m - 1) / m * m
}
