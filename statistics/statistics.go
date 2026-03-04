// Package statistics provides mathematical utility functions for options analytics.
package statistics

import "math"

// CumulativeNormalDistribution computes the CDF of the standard normal distribution
// using the Abramowitz and Stegun approximation.
func CumulativeNormalDistribution(x float64) float64 {
	t := 1.0 / (1.0 + 0.2316419*math.Abs(x))
	d := 0.3989423 * math.Exp(-x*x/2.0)
	probability := d * t * (0.3193815 +
		t*(-0.3565638+
			t*(1.781478+
				t*(-1.821256+
					t*1.330274))))

	if x > 0 {
		return 1.0 - probability
	}
	return probability
}

// NormalPDF computes the probability density function of the standard normal distribution.
func NormalPDF(x float64) float64 {
	return math.Exp(-x*x/2.0) / math.Sqrt(2.0*math.Pi)
}
