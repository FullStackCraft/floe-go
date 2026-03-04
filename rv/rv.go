// Package rv computes realized volatility from tick-level price observations
// using a quadratic variation estimator.
package rv

import (
	"math"
	"sort"
)

// MillisecondsPerYear for annualization.
const millisecondsPerYear = 31536000000.0

// PriceObservation is a single price tick.
type PriceObservation struct {
	Price     float64
	Timestamp float64 // milliseconds
}

// RealizedVolatilityResult holds the output of the RV computation.
type RealizedVolatilityResult struct {
	RealizedVolatility float64
	AnnualizedVariance float64
	QuadraticVariation float64
	NumObservations    int
	NumReturns         int
	ElapsedMinutes     float64
	ElapsedYears       float64
	FirstObservation   float64
	LastObservation    float64
}

// ComputeRealizedVolatility calculates annualized realized volatility from
// price observations using tick-based quadratic variation:
//
//	QV = Σ (ln(Pi/Pi-1))²
//	σ  = √(QV × year / elapsed)
func ComputeRealizedVolatility(observations []PriceObservation) RealizedVolatilityResult {
	// Filter invalid observations.
	valid := make([]PriceObservation, 0, len(observations))
	for _, obs := range observations {
		if obs.Price > 0 && !math.IsInf(obs.Price, 0) && !math.IsNaN(obs.Price) {
			valid = append(valid, obs)
		}
	}

	if len(valid) < 2 {
		return RealizedVolatilityResult{
			NumObservations: len(valid),
		}
	}

	// Sort by timestamp.
	sort.Slice(valid, func(i, j int) bool {
		return valid[i].Timestamp < valid[j].Timestamp
	})

	// Compute quadratic variation.
	qv := 0.0
	for i := 1; i < len(valid); i++ {
		logReturn := math.Log(valid[i].Price / valid[i-1].Price)
		qv += logReturn * logReturn
	}

	first := valid[0].Timestamp
	last := valid[len(valid)-1].Timestamp
	elapsedMs := last - first
	elapsedYears := elapsedMs / millisecondsPerYear

	if elapsedYears <= 0 {
		return RealizedVolatilityResult{
			NumObservations:  len(valid),
			NumReturns:       len(valid) - 1,
			FirstObservation: first,
			LastObservation:  last,
		}
	}

	annualizedVariance := qv / elapsedYears
	rv := math.Sqrt(annualizedVariance)

	return RealizedVolatilityResult{
		RealizedVolatility: rv,
		AnnualizedVariance: annualizedVariance,
		QuadraticVariation: qv,
		NumObservations:    len(valid),
		NumReturns:         len(valid) - 1,
		ElapsedMinutes:     elapsedMs / 60000.0,
		ElapsedYears:       elapsedYears,
		FirstObservation:   first,
		LastObservation:    last,
	}
}
