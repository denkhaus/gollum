package main

import (
	"strconv"
)

// Atoi converts a string to an integer
func Atoi(s string) (int, error) {
	return strconv.Atoi(s)
}

// ParseInt parses a string to an integer with given base and bit size
func ParseInt(s string, base, bitSize int) (int64, error) {
	return strconv.ParseInt(s, base, bitSize)
}

// ParseFloat parses a string to a float
func ParseFloat(s string, bitSize int) (float64, error) {
	return strconv.ParseFloat(s, bitSize)
}

// FormatInt formats an integer as a string
func FormatInt(i int64, base int) string {
	return strconv.FormatInt(i, base)
}

// FormatFloat formats a float as a string
// format is a byte constant like 'f', 'e', 'E', 'g', 'G'
func FormatFloat(f float64, format byte, prec, bitSize int) string {
	return strconv.FormatFloat(f, format, prec, bitSize)
}

// Itoa formats an integer as a string
func Itoa(i int) string {
	return strconv.Itoa(i)
}
