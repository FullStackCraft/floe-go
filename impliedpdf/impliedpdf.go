// Package impliedpdf estimates risk-neutral probability distributions from
// option prices using Breeden-Litzenberger numerical differentiation, and
// provides exposure-adjusted variants that account for dealer positioning.
package impliedpdf

import (
	"math"
	"sort"

	floe "github.com/FullStackCraft/floe-go"
)

// StrikeProbability holds the probability density at a single strike.
type StrikeProbability struct {
	Strike      float64
	Probability float64
}

// ImpliedProbabilityDistribution is the full implied PDF with summary stats.
type ImpliedProbabilityDistribution struct {
	Symbol                         string
	ExpiryDate                     int64
	CalculationTimestamp           int64
	UnderlyingPrice                float64
	StrikeProbabilities            []StrikeProbability
	MostLikelyPrice                float64
	MedianPrice                    float64
	ExpectedValue                  float64
	ExpectedMove                   float64
	TailSkew                       float64
	CumulativeProbabilityAboveSpot float64
	CumulativeProbabilityBelowSpot float64
}

// ImpliedPDFResult is the result of estimating an implied PDF.
type ImpliedPDFResult struct {
	Success      bool
	Error        string
	Distribution *ImpliedProbabilityDistribution
}

// EstimateImpliedProbabilityDistribution computes the implied PDF for a single
// expiry using Breeden-Litzenberger (d²C/dK²) from call option prices.
func EstimateImpliedProbabilityDistribution(
	symbol string,
	underlyingPrice float64,
	callOptions []floe.NormalizedOption,
	calculationTimestamp int64,
) ImpliedPDFResult {
	sorted := make([]floe.NormalizedOption, len(callOptions))
	copy(sorted, callOptions)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].Strike < sorted[j].Strike })

	n := len(sorted)
	if n < 3 {
		return ImpliedPDFResult{Success: false, Error: "Not enough data points (need at least 3 call options)"}
	}

	expiryDate := sorted[0].ExpirationTimestamp
	probs := make([]StrikeProbability, n)
	probs[0] = StrikeProbability{Strike: sorted[0].Strike, Probability: 0}
	probs[n-1] = StrikeProbability{Strike: sorted[n-1].Strike, Probability: 0}

	for i := 1; i < n-1; i++ {
		kPrev := sorted[i-1].Strike
		kNext := sorted[i+1].Strike

		midPrev := (sorted[i-1].Bid + sorted[i-1].Ask) / 2
		midCurr := (sorted[i].Bid + sorted[i].Ask) / 2
		midNext := (sorted[i+1].Bid + sorted[i+1].Ask) / 2

		strikeDiff := kNext - kPrev
		if math.Abs(strikeDiff) < 1e-9 {
			probs[i] = StrikeProbability{Strike: sorted[i].Strike, Probability: 0}
			continue
		}

		d2 := (midNext - 2*midCurr + midPrev) / (strikeDiff * strikeDiff)
		probs[i] = StrikeProbability{Strike: sorted[i].Strike, Probability: math.Max(d2, 0)}
	}

	// Normalize.
	sum := 0.0
	for _, sp := range probs {
		sum += sp.Probability
	}
	if sum < 1e-9 {
		return ImpliedPDFResult{Success: false, Error: "Insufficient probability mass to normalize"}
	}
	for i := range probs {
		probs[i].Probability /= sum
	}

	// Mode.
	mostLikelyPrice := probs[0].Strike
	maxProb := 0.0
	for _, sp := range probs {
		if sp.Probability > maxProb {
			maxProb = sp.Probability
			mostLikelyPrice = sp.Strike
		}
	}

	// Median.
	cum := 0.0
	medianPrice := probs[n/2].Strike
	for _, sp := range probs {
		cum += sp.Probability
		if cum >= 0.5 {
			medianPrice = sp.Strike
			break
		}
	}

	// Mean.
	mean := 0.0
	for _, sp := range probs {
		mean += sp.Strike * sp.Probability
	}

	// Variance / expected move.
	variance := 0.0
	for _, sp := range probs {
		d := sp.Strike - mean
		variance += d * d * sp.Probability
	}
	expectedMove := math.Sqrt(variance)

	// Tail skew.
	var leftTail, rightTail float64
	for _, sp := range probs {
		if sp.Strike < mean {
			leftTail += sp.Probability
		} else {
			rightTail += sp.Probability
		}
	}
	tailSkew := rightTail / math.Max(leftTail, 1e-9)

	// Cumulative above/below spot.
	var belowSpot, aboveSpot float64
	for _, sp := range probs {
		if sp.Strike < underlyingPrice {
			belowSpot += sp.Probability
		} else if sp.Strike > underlyingPrice {
			aboveSpot += sp.Probability
		}
	}

	return ImpliedPDFResult{
		Success: true,
		Distribution: &ImpliedProbabilityDistribution{
			Symbol:                         symbol,
			ExpiryDate:                     expiryDate,
			CalculationTimestamp:           calculationTimestamp,
			UnderlyingPrice:                underlyingPrice,
			StrikeProbabilities:            probs,
			MostLikelyPrice:                mostLikelyPrice,
			MedianPrice:                    medianPrice,
			ExpectedValue:                  mean,
			ExpectedMove:                   expectedMove,
			TailSkew:                       tailSkew,
			CumulativeProbabilityAboveSpot: aboveSpot,
			CumulativeProbabilityBelowSpot: belowSpot,
		},
	}
}

// EstimateImpliedProbabilityDistributions computes PDFs for all expirations.
func EstimateImpliedProbabilityDistributions(
	symbol string,
	underlyingPrice float64,
	options []floe.NormalizedOption,
	calculationTimestamp int64,
) []ImpliedProbabilityDistribution {
	expirationSet := make(map[int64]bool)
	for _, opt := range options {
		expirationSet[opt.ExpirationTimestamp] = true
	}
	expirations := make([]int64, 0, len(expirationSet))
	for exp := range expirationSet {
		expirations = append(expirations, exp)
	}
	sort.Slice(expirations, func(i, j int) bool { return expirations[i] < expirations[j] })

	var dists []ImpliedProbabilityDistribution
	for _, exp := range expirations {
		var calls []floe.NormalizedOption
		for _, opt := range options {
			if opt.ExpirationTimestamp == exp && opt.OptionType == floe.Call && opt.Bid > 0 && opt.Ask > 0 {
				calls = append(calls, opt)
			}
		}
		result := EstimateImpliedProbabilityDistribution(symbol, underlyingPrice, calls, calculationTimestamp)
		if result.Success {
			dists = append(dists, *result.Distribution)
		}
	}
	return dists
}

// GetProbabilityInRange returns the probability of finishing between two prices.
func GetProbabilityInRange(dist ImpliedProbabilityDistribution, lower, upper float64) float64 {
	p := 0.0
	for _, sp := range dist.StrikeProbabilities {
		if sp.Strike >= lower && sp.Strike <= upper {
			p += sp.Probability
		}
	}
	return p
}

// GetCumulativeProbability returns P(S ≤ price).
func GetCumulativeProbability(dist ImpliedProbabilityDistribution, price float64) float64 {
	p := 0.0
	for _, sp := range dist.StrikeProbabilities {
		if sp.Strike <= price {
			p += sp.Probability
		}
	}
	return p
}

// GetQuantile returns the strike at the given probability quantile.
func GetQuantile(dist ImpliedProbabilityDistribution, probability float64) float64 {
	if len(dist.StrikeProbabilities) == 0 {
		return 0
	}
	if probability <= 0 {
		return dist.StrikeProbabilities[0].Strike
	}
	if probability >= 1 {
		return dist.StrikeProbabilities[len(dist.StrikeProbabilities)-1].Strike
	}

	cum := 0.0
	for _, sp := range dist.StrikeProbabilities {
		cum += sp.Probability
		if cum >= probability {
			return sp.Strike
		}
	}
	return dist.StrikeProbabilities[len(dist.StrikeProbabilities)-1].Strike
}
