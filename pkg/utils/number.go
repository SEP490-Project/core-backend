package utils

import "math"

// RoundFloat rounds a float64 to a specified number of decimal places.
func RoundFloat(val float64, precision int) float64 {
	if precision < 0 {
		return val
	}
	factor := math.Pow(10, float64(precision))
	return math.Round(val*factor) / factor
}
