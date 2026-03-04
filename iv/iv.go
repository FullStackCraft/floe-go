// Package iv computes model-free implied volatility using the CBOE variance
// swap methodology (the same approach used to compute the VIX index).
package iv

import (
	"math"
	"sort"

	floe "github.com/FullStackCraft/floe-go"
)

const millisecondsPerYear = 31536000000.0

// VarianceSwapResult holds the result for a single expiration.
type VarianceSwapResult struct {
	ImpliedVolatility  float64
	AnnualizedVariance float64
	Forward            float64
	K0                 float64
	TimeToExpiry       float64
	Expiration         int64
	NumStrikes         int
	PutContribution    float64
	CallContribution   float64
}

// ImpliedVolatilityResult holds the final IV, possibly interpolated between
// two terms in the CBOE VIX style.
type ImpliedVolatilityResult struct {
	ImpliedVolatility float64
	NearTerm          VarianceSwapResult
	FarTerm           *VarianceSwapResult
	TargetDays        *int
	IsInterpolated    bool
}

// ComputeVarianceSwapIV computes the model-free implied variance for a single
// expiration using the CBOE variance swap methodology.
func ComputeVarianceSwapIV(
	options []floe.NormalizedOption,
	spot float64,
	riskFreeRate float64,
	asOfTimestamp int64,
) VarianceSwapResult {
	if len(options) == 0 || spot <= 0 {
		return VarianceSwapResult{}
	}

	// Determine expiration from first option.
	expiration := options[0].ExpirationTimestamp
	T := float64(expiration-asOfTimestamp) / millisecondsPerYear
	if T <= 0 {
		return VarianceSwapResult{Expiration: expiration}
	}

	// Pair calls and puts by strike.
	type strikePair struct {
		strike  float64
		callMid float64
		putMid  float64
		hasBoth bool
	}

	callsByStrike := make(map[float64]float64)
	putsByStrike := make(map[float64]float64)
	strikeSet := make(map[float64]bool)

	for _, opt := range options {
		mid := midPrice(opt)
		if mid <= 0 {
			continue
		}
		strikeSet[opt.Strike] = true
		if opt.OptionType == floe.Call {
			callsByStrike[opt.Strike] = mid
		} else {
			putsByStrike[opt.Strike] = mid
		}
	}

	strikes := make([]float64, 0, len(strikeSet))
	for k := range strikeSet {
		strikes = append(strikes, k)
	}
	sort.Float64s(strikes)

	if len(strikes) < 2 {
		return VarianceSwapResult{Expiration: expiration, TimeToExpiry: T}
	}

	// Find K0: strike where |call - put| is minimized (both must exist).
	k0 := strikes[0]
	minDiff := math.MaxFloat64
	for _, k := range strikes {
		c, hc := callsByStrike[k]
		p, hp := putsByStrike[k]
		if hc && hp {
			diff := math.Abs(c - p)
			if diff < minDiff {
				minDiff = diff
				k0 = k
			}
		}
	}

	// Forward price: F = K0 + e^(rT) * (call(K0) - put(K0))
	ert := math.Exp(riskFreeRate * T)
	callK0, _ := callsByStrike[k0]
	putK0, _ := putsByStrike[k0]
	F := k0 + ert*(callK0-putK0)

	// Compute variance sum using OTM options.
	var putContrib, callContrib float64
	var numStrikes int

	// Walk puts downward from K0 (below K0).
	zeroCount := 0
	for i := len(strikes) - 1; i >= 0; i-- {
		k := strikes[i]
		if k > k0 {
			continue
		}
		p, ok := putsByStrike[k]
		if !ok || p <= 0 {
			zeroCount++
			if zeroCount >= 2 {
				break
			}
			continue
		}
		zeroCount = 0

		deltaK := computeDeltaK(strikes, i)
		q := p
		if k == k0 {
			c, hc := callsByStrike[k]
			if hc {
				q = (c + p) / 2
			}
		}
		contribution := (deltaK / (k * k)) * ert * q
		putContrib += contribution
		numStrikes++
	}

	// Walk calls upward from K0 (above K0).
	zeroCount = 0
	for i := 0; i < len(strikes); i++ {
		k := strikes[i]
		if k < k0 {
			continue
		}
		c, ok := callsByStrike[k]
		if !ok || c <= 0 {
			zeroCount++
			if zeroCount >= 2 {
				break
			}
			continue
		}
		zeroCount = 0

		deltaK := computeDeltaK(strikes, i)
		q := c
		if k == k0 {
			p, hp := putsByStrike[k]
			if hp {
				q = (c + p) / 2
			}
		}
		contribution := (deltaK / (k * k)) * ert * q
		callContrib += contribution
		numStrikes++
	}

	// σ² = (2/T) * Σ - (1/T) * (F/K0 - 1)²
	variance := (2.0/T)*(putContrib+callContrib) - (1.0/T)*math.Pow(F/k0-1, 2)
	if variance < 0 {
		variance = 0
	}

	return VarianceSwapResult{
		ImpliedVolatility:  math.Sqrt(variance),
		AnnualizedVariance: variance,
		Forward:            F,
		K0:                 k0,
		TimeToExpiry:       T,
		Expiration:         expiration,
		NumStrikes:         numStrikes,
		PutContribution:    putContrib,
		CallContribution:   callContrib,
	}
}

// ComputeImpliedVolatility computes model-free IV, optionally interpolating
// between two expirations using the CBOE VIX methodology.
//
// If farTermOptions is nil or empty, a single-term result is returned.
func ComputeImpliedVolatility(
	nearTermOptions []floe.NormalizedOption,
	spot, riskFreeRate float64,
	asOfTimestamp int64,
	farTermOptions []floe.NormalizedOption,
	targetDays *int,
) ImpliedVolatilityResult {
	near := ComputeVarianceSwapIV(nearTermOptions, spot, riskFreeRate, asOfTimestamp)

	if len(farTermOptions) == 0 || targetDays == nil {
		return ImpliedVolatilityResult{
			ImpliedVolatility: near.ImpliedVolatility,
			NearTerm:          near,
			IsInterpolated:    false,
		}
	}

	far := ComputeVarianceSwapIV(farTermOptions, spot, riskFreeRate, asOfTimestamp)

	// CBOE VIX-style interpolation in variance space.
	T1 := near.TimeToExpiry
	T2 := far.TimeToExpiry
	N1 := T1 * 365.0 * 1440.0 // minutes in near term
	N2 := T2 * 365.0 * 1440.0 // minutes in far term
	Ntarget := float64(*targetDays) * 1440.0
	N365 := 365.0 * 1440.0

	denom := N2 - N1
	if denom <= 0 {
		return ImpliedVolatilityResult{
			ImpliedVolatility: near.ImpliedVolatility,
			NearTerm:          near,
			FarTerm:           &far,
			TargetDays:        targetDays,
			IsInterpolated:    false,
		}
	}

	w1 := (N2 - Ntarget) / denom
	w2 := (Ntarget - N1) / denom

	interpolatedVariance := (T1*near.AnnualizedVariance*w1 + T2*far.AnnualizedVariance*w2) * N365 / Ntarget
	if interpolatedVariance < 0 {
		interpolatedVariance = 0
	}

	return ImpliedVolatilityResult{
		ImpliedVolatility: math.Sqrt(interpolatedVariance),
		NearTerm:          near,
		FarTerm:           &far,
		TargetDays:        targetDays,
		IsInterpolated:    true,
	}
}

// ---- helpers ----

func midPrice(opt floe.NormalizedOption) float64 {
	if opt.Bid > 0 && opt.Ask > 0 {
		return (opt.Bid + opt.Ask) / 2
	}
	if opt.Mark > 0 {
		return opt.Mark
	}
	return 0
}

func computeDeltaK(strikes []float64, i int) float64 {
	n := len(strikes)
	if n == 1 {
		return 1
	}
	if i == 0 {
		return strikes[1] - strikes[0]
	}
	if i == n-1 {
		return strikes[n-1] - strikes[n-2]
	}
	return (strikes[i+1] - strikes[i-1]) / 2
}
