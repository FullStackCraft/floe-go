package impliedpdf

import (
	"math"
	"sort"

	floe "github.com/FullStackCraft/floe-go"
)

// ---- Configuration types ----

// ExposureAdjustmentConfig configures exposure-based PDF adjustments.
type ExposureAdjustmentConfig struct {
	Gamma GammaConfig
	Vanna VannaConfig
	Charm CharmAdjConfig
}

// GammaConfig controls gamma-based kurtosis adjustment.
type GammaConfig struct {
	Enabled           bool
	AttractorStrength float64
	RepellentStrength float64
	Threshold         float64
	DecayRate         float64
}

// VannaConfig controls vanna-based tail adjustment.
type VannaConfig struct {
	Enabled            bool
	SpotVolBeta        float64
	MaxTailMultiplier  float64
	FeedbackIterations int
}

// CharmAdjConfig controls charm-based mean shift.
type CharmAdjConfig struct {
	Enabled     bool
	TimeHorizon string  // "intraday", "daily", "weekly"
	ShiftScale  float64
}

// AdjustedPDFResult holds baseline and adjusted PDFs with comparison metrics.
type AdjustedPDFResult struct {
	Baseline       ImpliedProbabilityDistribution
	Adjusted       ImpliedProbabilityDistribution
	GammaModifiers []float64
	VannaModifiers []float64
	CharmShift     float64
	Comparison     PDFComparison
}

// PDFComparison metrics between baseline and adjusted distributions.
type PDFComparison struct {
	MeanShift        float64
	MeanShiftPercent float64
	StdDevChange     float64
	TailSkewChange   float64
	LeftTail         TailComparison
	RightTail        TailComparison
	DominantFactor   string // "gamma", "vanna", "charm", "none"
}

// TailComparison compares a tail quantile between baseline and adjusted.
type TailComparison struct {
	Baseline float64
	Adjusted float64
	Ratio    float64
}

// AdjustmentLevel describes a significant adjustment at a strike.
type AdjustmentLevel struct {
	Strike       float64
	BaselineProb float64
	AdjustedProb float64
	Edge         float64
}

// ---- Preset configurations ----

// DefaultAdjustmentConfig is tuned for SPX-like indices.
var DefaultAdjustmentConfig = ExposureAdjustmentConfig{
	Gamma: GammaConfig{Enabled: true, AttractorStrength: 0.3, RepellentStrength: 0.3, Threshold: 1_000_000, DecayRate: 2.0},
	Vanna: VannaConfig{Enabled: true, SpotVolBeta: -3.0, MaxTailMultiplier: 2.5, FeedbackIterations: 3},
	Charm: CharmAdjConfig{Enabled: true, TimeHorizon: "daily", ShiftScale: 1.0},
}

// LowVolConfig is for calm, grinding markets.
var LowVolConfig = ExposureAdjustmentConfig{
	Gamma: GammaConfig{Enabled: true, AttractorStrength: 0.4, RepellentStrength: 0.2, Threshold: 500_000, DecayRate: 1.5},
	Vanna: VannaConfig{Enabled: true, SpotVolBeta: -2.0, MaxTailMultiplier: 1.5, FeedbackIterations: 2},
	Charm: CharmAdjConfig{Enabled: true, TimeHorizon: "daily", ShiftScale: 1.5},
}

// CrisisConfig is for high-volatility/crisis markets.
var CrisisConfig = ExposureAdjustmentConfig{
	Gamma: GammaConfig{Enabled: true, AttractorStrength: 0.1, RepellentStrength: 0.5, Threshold: 2_000_000, DecayRate: 3.0},
	Vanna: VannaConfig{Enabled: true, SpotVolBeta: -5.0, MaxTailMultiplier: 3.0, FeedbackIterations: 5},
	Charm: CharmAdjConfig{Enabled: true, TimeHorizon: "intraday", ShiftScale: 0.5},
}

// OpexConfig is for OPEX week.
var OpexConfig = ExposureAdjustmentConfig{
	Gamma: GammaConfig{Enabled: true, AttractorStrength: 0.5, RepellentStrength: 0.4, Threshold: 1_000_000, DecayRate: 2.5},
	Vanna: VannaConfig{Enabled: true, SpotVolBeta: -3.0, MaxTailMultiplier: 2.0, FeedbackIterations: 3},
	Charm: CharmAdjConfig{Enabled: true, TimeHorizon: "intraday", ShiftScale: 2.0},
}

// ---- Core functions ----

// EstimateExposureAdjustedPDF computes a flow-adjusted implied PDF.
func EstimateExposureAdjustedPDF(
	symbol string,
	underlyingPrice float64,
	callOptions []floe.NormalizedOption,
	exposures floe.ExposurePerExpiry,
	config *ExposureAdjustmentConfig,
	calculationTimestamp int64,
) (*AdjustedPDFResult, error) {
	cfg := DefaultAdjustmentConfig
	if config != nil {
		cfg = *config
	}

	baselineResult := EstimateImpliedProbabilityDistribution(symbol, underlyingPrice, callOptions, calculationTimestamp)
	if !baselineResult.Success {
		return nil, &pdfError{baselineResult.Error}
	}
	baseline := *baselineResult.Distribution

	// Gamma modifiers.
	var gammaModifiers []float64
	if cfg.Gamma.Enabled {
		gammaModifiers = calculateGammaModifiers(baseline.StrikeProbabilities, exposures, underlyingPrice, cfg.Gamma)
	} else {
		gammaModifiers = ones(len(baseline.StrikeProbabilities))
	}

	// Vanna modifiers.
	var vannaModifiers []float64
	if cfg.Vanna.Enabled {
		vannaModifiers = calculateVannaModifiers(baseline.StrikeProbabilities, exposures, underlyingPrice, cfg.Vanna)
	} else {
		vannaModifiers = ones(len(baseline.StrikeProbabilities))
	}

	// Charm shift.
	charmShift := 0.0
	if cfg.Charm.Enabled {
		charmShift = calculateCharmShift(exposures, underlyingPrice, cfg.Charm)
	}

	// Apply modifiers.
	adjusted := applyModifiers(baseline.StrikeProbabilities, gammaModifiers, vannaModifiers, charmShift)
	normalized := normalizeProbabilities(adjusted)
	adjustedDist := recalculateStats(baseline, normalized, underlyingPrice, calculationTimestamp)
	comparison := calculateComparison(baseline, adjustedDist, gammaModifiers, vannaModifiers, charmShift)

	return &AdjustedPDFResult{
		Baseline:       baseline,
		Adjusted:       adjustedDist,
		GammaModifiers: gammaModifiers,
		VannaModifiers: vannaModifiers,
		CharmShift:     charmShift,
		Comparison:     comparison,
	}, nil
}

// GetEdgeAtPrice returns the probability difference between adjusted and baseline.
func GetEdgeAtPrice(result AdjustedPDFResult, price float64) float64 {
	return GetCumulativeProbability(result.Adjusted, price) - GetCumulativeProbability(result.Baseline, price)
}

// GetSignificantAdjustmentLevels returns strikes with significant probability shifts.
func GetSignificantAdjustmentLevels(result AdjustedPDFResult, threshold float64) []AdjustmentLevel {
	if threshold == 0 {
		threshold = 0.01
	}
	var levels []AdjustmentLevel
	for i, bp := range result.Baseline.StrikeProbabilities {
		ap := 0.0
		if i < len(result.Adjusted.StrikeProbabilities) {
			ap = result.Adjusted.StrikeProbabilities[i].Probability
		}
		edge := ap - bp.Probability
		if math.Abs(edge) >= threshold {
			levels = append(levels, AdjustmentLevel{
				Strike:       bp.Strike,
				BaselineProb: bp.Probability,
				AdjustedProb: ap,
				Edge:         edge,
			})
		}
	}
	sort.Slice(levels, func(i, j int) bool { return math.Abs(levels[i].Edge) > math.Abs(levels[j].Edge) })
	return levels
}

// ---- Internal ----

func calculateGammaModifiers(probs []StrikeProbability, exposures floe.ExposurePerExpiry, spot float64, cfg GammaConfig) []float64 {
	maxGex := 1.0
	for _, e := range exposures.StrikeExposures {
		if math.Abs(e.GammaExposure) > maxGex {
			maxGex = math.Abs(e.GammaExposure)
		}
	}

	mods := make([]float64, len(probs))
	for i, p := range probs {
		mod := 1.0
		for _, e := range exposures.StrikeExposures {
			if math.Abs(e.GammaExposure) < cfg.Threshold {
				continue
			}
			dist := math.Abs(p.Strike-e.StrikePrice) / spot
			influence := 1 / (1 + cfg.DecayRate*dist*dist)
			normGex := e.GammaExposure / maxGex

			if e.GammaExposure > 0 {
				mod *= 1 + cfg.AttractorStrength*normGex*influence
			} else {
				mod *= 1 - cfg.RepellentStrength*math.Abs(normGex)*influence
			}
		}
		mods[i] = math.Max(0.1, math.Min(3.0, mod))
	}
	return mods
}

func calculateVannaModifiers(probs []StrikeProbability, exposures floe.ExposurePerExpiry, spot float64, cfg VannaConfig) []float64 {
	var vannaBelow, vannaAbove float64
	for _, e := range exposures.StrikeExposures {
		if e.StrikePrice < spot {
			vannaBelow += e.VannaExposure
		} else {
			vannaAbove += e.VannaExposure
		}
	}

	mods := make([]float64, len(probs))
	for i, p := range probs {
		mod := 1.0
		movePercent := (p.Strike - spot) / spot

		if movePercent < 0 {
			ivSpike := -movePercent * math.Abs(cfg.SpotVolBeta)
			vannaFlow := vannaBelow * ivSpike
			if vannaFlow < 0 {
				cumEffect := 0.0
				flow := math.Abs(vannaFlow)
				for iter := 0; iter < cfg.FeedbackIterations; iter++ {
					cumEffect += flow
					flow *= 0.5
				}
				effectScale := cumEffect / (spot * 1_000_000)
				mod = 1 + math.Min(cfg.MaxTailMultiplier-1, effectScale)
			}
		} else if movePercent > 0 {
			ivCompress := movePercent * math.Abs(cfg.SpotVolBeta) * 0.5
			vannaFlow := vannaAbove * (-ivCompress)
			if vannaFlow > 0 {
				effectScale := vannaFlow / (spot * 1_000_000)
				mod = math.Max(0.5, 1-effectScale*0.5)
			}
		}
		mods[i] = mod
	}
	return mods
}

func calculateCharmShift(exposures floe.ExposurePerExpiry, spot float64, cfg CharmAdjConfig) float64 {
	timeMult := 1.0
	switch cfg.TimeHorizon {
	case "intraday":
		timeMult = 0.25
	case "daily":
		timeMult = 1.0
	case "weekly":
		timeMult = 5.0
	}

	flowImpact := 0.001 * spot
	priceShift := (exposures.TotalCharmExposure / 1_000_000_000) * flowImpact * timeMult
	return priceShift * cfg.ShiftScale
}

func applyModifiers(probs []StrikeProbability, gamma, vanna []float64, charmShift float64) []StrikeProbability {
	result := make([]StrikeProbability, len(probs))
	for i, sp := range probs {
		result[i] = StrikeProbability{
			Strike:      sp.Strike + charmShift,
			Probability: sp.Probability * gamma[i] * vanna[i],
		}
	}
	return result
}

func normalizeProbabilities(probs []StrikeProbability) []StrikeProbability {
	sum := 0.0
	for _, sp := range probs {
		sum += sp.Probability
	}
	result := make([]StrikeProbability, len(probs))
	if sum < 1e-9 {
		uniform := 1.0 / float64(len(probs))
		for i := range probs {
			result[i] = StrikeProbability{Strike: probs[i].Strike, Probability: uniform}
		}
		return result
	}
	for i, sp := range probs {
		result[i] = StrikeProbability{Strike: sp.Strike, Probability: sp.Probability / sum}
	}
	return result
}

func recalculateStats(baseline ImpliedProbabilityDistribution, probs []StrikeProbability, underlyingPrice float64, ts int64) ImpliedProbabilityDistribution {
	mode := probs[0].Strike
	maxP := 0.0
	for _, sp := range probs {
		if sp.Probability > maxP {
			maxP = sp.Probability
			mode = sp.Strike
		}
	}

	cum := 0.0
	median := probs[len(probs)/2].Strike
	for _, sp := range probs {
		cum += sp.Probability
		if cum >= 0.5 {
			median = sp.Strike
			break
		}
	}

	mean := 0.0
	for _, sp := range probs {
		mean += sp.Strike * sp.Probability
	}

	variance := 0.0
	for _, sp := range probs {
		d := sp.Strike - mean
		variance += d * d * sp.Probability
	}

	var leftTail, rightTail float64
	for _, sp := range probs {
		if sp.Strike < mean {
			leftTail += sp.Probability
		} else {
			rightTail += sp.Probability
		}
	}

	var below, above float64
	for _, sp := range probs {
		if sp.Strike < underlyingPrice {
			below += sp.Probability
		} else if sp.Strike > underlyingPrice {
			above += sp.Probability
		}
	}

	return ImpliedProbabilityDistribution{
		Symbol:                         baseline.Symbol,
		ExpiryDate:                     baseline.ExpiryDate,
		CalculationTimestamp:           ts,
		UnderlyingPrice:                underlyingPrice,
		StrikeProbabilities:            probs,
		MostLikelyPrice:                mode,
		MedianPrice:                    median,
		ExpectedValue:                  mean,
		ExpectedMove:                   math.Sqrt(variance),
		TailSkew:                       rightTail / math.Max(leftTail, 1e-9),
		CumulativeProbabilityAboveSpot: above,
		CumulativeProbabilityBelowSpot: below,
	}
}

func calculateComparison(baseline, adjusted ImpliedProbabilityDistribution, gamma, vanna []float64, charmShift float64) PDFComparison {
	b5 := GetQuantile(baseline, 0.05)
	b95 := GetQuantile(baseline, 0.95)
	a5 := GetQuantile(adjusted, 0.05)
	a95 := GetQuantile(adjusted, 0.95)

	gammaEffect := maxFloat(gamma) - minFloat(gamma)
	vannaEffect := maxFloat(vanna) - minFloat(vanna)
	charmEffect := math.Abs(charmShift) / baseline.UnderlyingPrice

	dominant := "none"
	maxE := math.Max(gammaEffect, math.Max(vannaEffect, charmEffect))
	if maxE > 0.01 {
		switch {
		case gammaEffect == maxE:
			dominant = "gamma"
		case vannaEffect == maxE:
			dominant = "vanna"
		default:
			dominant = "charm"
		}
	}

	safeDiv := func(a, b float64) float64 {
		if b == 0 {
			return 0
		}
		return a / b
	}

	return PDFComparison{
		MeanShift:        adjusted.ExpectedValue - baseline.ExpectedValue,
		MeanShiftPercent: ((adjusted.ExpectedValue - baseline.ExpectedValue) / baseline.UnderlyingPrice) * 100,
		StdDevChange:     adjusted.ExpectedMove - baseline.ExpectedMove,
		TailSkewChange:   adjusted.TailSkew - baseline.TailSkew,
		LeftTail:         TailComparison{Baseline: b5, Adjusted: a5, Ratio: safeDiv(a5, b5)},
		RightTail:        TailComparison{Baseline: b95, Adjusted: a95, Ratio: safeDiv(a95, b95)},
		DominantFactor:   dominant,
	}
}

func ones(n int) []float64 {
	out := make([]float64, n)
	for i := range out {
		out[i] = 1
	}
	return out
}

func maxFloat(s []float64) float64 {
	if len(s) == 0 {
		return 0
	}
	m := s[0]
	for _, v := range s[1:] {
		if v > m {
			m = v
		}
	}
	return m
}

func minFloat(s []float64) float64 {
	if len(s) == 0 {
		return 0
	}
	m := s[0]
	for _, v := range s[1:] {
		if v < m {
			m = v
		}
	}
	return m
}

type pdfError struct{ msg string }

func (e *pdfError) Error() string { return e.msg }
