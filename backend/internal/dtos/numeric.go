package dtos

import "strconv"

func numericToFloat(s string) float64 {
	f, _ := strconv.ParseFloat(s, 64)
	return f
}
