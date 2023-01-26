package price

// IntPriceMultiplier is the multiplier used to
// convert price and budget representation
// from floating point to fixed point with scaling
// factor 1/1000000.
const IntPriceMultiplier = 1000000

// ToInt converts price from floating point to fixed point.
func ToInt(f float64) int64 {
	return int64(f * IntPriceMultiplier)
}

// ToFloat converts price from fixed point to floating point.
func ToFloat(f int64) float64 {
	return float64(f) / IntPriceMultiplier
}
