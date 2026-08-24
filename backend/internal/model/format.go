package model

import (
	"math"
	"strconv"
)

func strconvFormat(v float64) string {
	switch {
	case math.IsNaN(v):
		return "NaN"
	case math.IsInf(v, 1):
		return "+Inf"
	case math.IsInf(v, -1):
		return "-Inf"
	default:
		return strconv.FormatFloat(v, 'f', -1, 64)
	}
}

func ParseFloat(s string) (float64, error) {
	switch s {
	case "NaN", "nan":
		return math.NaN(), nil
	case "+Inf", "Inf", "+inf", "inf":
		return math.Inf(1), nil
	case "-Inf", "-inf":
		return math.Inf(-1), nil
	default:
		return strconv.ParseFloat(s, 64)
	}
}
