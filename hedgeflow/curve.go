package hedgeflow

import (
	"math"

	floe "github.com/FullStackCraft/floe-go"
)

// ComputeHedgeImpulseCurve computes the combined gamma-vanna hedge impulse
// curve across a price grid.
//
// H(S) = GEX_smoothed(S) - (k / S) * VEX_smoothed(S)
//
// Positive H means dealer buying dampens a move (mean-reversion).
// Negative H means dealer selling amplifies a move (trend acceleration).
func ComputeHedgeImpulseCurve(
	exposures floe.ExposurePerExpiry,
	ivSurface floe.IVSurface,
	config HedgeImpulseConfig,
	computedAt int64,
) HedgeImpulseCurve {
	applyDefaults(&config)

	spot := exposures.SpotPrice
	regimeParams := DeriveRegimeParams(ivSurface, spot)
	k := deriveSpotVolCoupling(regimeParams)

	// Extract strike-space data.
	strikes := make([]float64, len(exposures.StrikeExposures))
	gexValues := make([]float64, len(exposures.StrikeExposures))
	vexValues := make([]float64, len(exposures.StrikeExposures))
	for i, se := range exposures.StrikeExposures {
		strikes[i] = se.StrikePrice
		gexValues[i] = se.GammaExposure
		vexValues[i] = se.VannaExposure
	}

	strikeSpacing := detectStrikeSpacing(strikes)
	lambda := config.KernelWidthStrikes * strikeSpacing

	// Build price grid.
	gridMin := spot * (1 - config.RangePercent/100)
	gridMax := spot * (1 + config.RangePercent/100)
	gridStep := spot * (config.StepPercent / 100)

	var curve []HedgeImpulsePoint
	for price := gridMin; price <= gridMax; price += gridStep {
		gamma := kernelSmooth(strikes, gexValues, price, lambda)
		vanna := kernelSmooth(strikes, vexValues, price, lambda)
		impulse := gamma - (k/price)*vanna

		curve = append(curve, HedgeImpulsePoint{
			Price:   price,
			Gamma:   gamma,
			Vanna:   vanna,
			Impulse: impulse,
		})
	}

	impulseAtSpot := interpolateImpulseAtPrice(curve, spot)
	slopeAtSpot := computeSlopeAtPrice(curve, spot)
	zeroCrossings := findZeroCrossings(curve)
	extrema := findExtrema(curve)
	asymmetry := computeAsymmetry(curve, spot, 0.5)
	regime := classifyRegime(impulseAtSpot, slopeAtSpot, asymmetry, curve, spot)

	// Find nearest attractors.
	var nearestAbove, nearestBelow *float64
	for _, e := range extrema {
		if e.Type == "basin" && e.Price > spot {
			if nearestAbove == nil || e.Price < *nearestAbove {
				p := e.Price
				nearestAbove = &p
			}
		}
		if e.Type == "basin" && e.Price < spot {
			if nearestBelow == nil || e.Price > *nearestBelow {
				p := e.Price
				nearestBelow = &p
			}
		}
	}

	return HedgeImpulseCurve{
		Spot:                  spot,
		Expiration:            exposures.Expiration,
		ComputedAt:            computedAt,
		SpotVolCoupling:       k,
		KernelWidth:           lambda,
		StrikeSpacing:         strikeSpacing,
		Curve:                 curve,
		ImpulseAtSpot:         impulseAtSpot,
		SlopeAtSpot:           slopeAtSpot,
		ZeroCrossings:         zeroCrossings,
		Extrema:               extrema,
		Asymmetry:             asymmetry,
		Regime:                regime,
		NearestAttractorAbove: nearestAbove,
		NearestAttractorBelow: nearestBelow,
	}
}

func applyDefaults(c *HedgeImpulseConfig) {
	if c.RangePercent == 0 {
		c.RangePercent = 3
	}
	if c.StepPercent == 0 {
		c.StepPercent = 0.05
	}
	if c.KernelWidthStrikes == 0 {
		c.KernelWidthStrikes = 2
	}
}

// detectStrikeSpacing finds the modal (most common) strike spacing.
func detectStrikeSpacing(strikes []float64) float64 {
	if len(strikes) < 2 {
		return 1
	}

	gapCounts := make(map[float64]int)
	for i := 1; i < len(strikes); i++ {
		gap := math.Abs(strikes[i] - strikes[i-1])
		if gap > 0 {
			rounded := math.Round(gap*100) / 100
			gapCounts[rounded]++
		}
	}

	if len(gapCounts) == 0 {
		return 1
	}

	var modalGap float64
	maxCount := 0
	for gap, count := range gapCounts {
		if count > maxCount {
			maxCount = count
			modalGap = gap
		}
	}
	return modalGap
}

// deriveSpotVolCoupling computes k from the IV surface.
// k = -ρ × atmIV × √252, clamped [2, 20].
func deriveSpotVolCoupling(rp RegimeParams) float64 {
	k := -rp.ImpliedSpotVolCorr * rp.AtmIV * math.Sqrt(252)
	return math.Max(2, math.Min(20, k))
}

// kernelSmooth applies Gaussian kernel smoothing to map strike-space
// exposures into price-space.
func kernelSmooth(strikes, values []float64, evalPrice, lambda float64) float64 {
	var weightedSum, weightSum float64
	for i := range strikes {
		dist := (strikes[i] - evalPrice) / lambda
		w := math.Exp(-(dist * dist))
		weightedSum += values[i] * w
		weightSum += w
	}
	if weightSum > 0 {
		return weightedSum / weightSum
	}
	return 0
}

func interpolateImpulseAtPrice(curve []HedgeImpulsePoint, price float64) float64 {
	if len(curve) == 0 {
		return 0
	}
	if price <= curve[0].Price {
		return curve[0].Impulse
	}
	if price >= curve[len(curve)-1].Price {
		return curve[len(curve)-1].Impulse
	}

	for i := 0; i < len(curve)-1; i++ {
		if curve[i].Price <= price && curve[i+1].Price >= price {
			t := (price - curve[i].Price) / (curve[i+1].Price - curve[i].Price)
			return curve[i].Impulse + t*(curve[i+1].Impulse-curve[i].Impulse)
		}
	}
	return 0
}

func computeSlopeAtPrice(curve []HedgeImpulsePoint, price float64) float64 {
	if len(curve) < 3 {
		return 0
	}
	step := curve[1].Price - curve[0].Price
	above := interpolateImpulseAtPrice(curve, price+step)
	below := interpolateImpulseAtPrice(curve, price-step)
	return (above - below) / (2 * step)
}

func findZeroCrossings(curve []HedgeImpulsePoint) []ZeroCrossing {
	var crossings []ZeroCrossing
	for i := 0; i < len(curve)-1; i++ {
		a := curve[i].Impulse
		b := curve[i+1].Impulse
		if a*b < 0 {
			t := math.Abs(a) / (math.Abs(a) + math.Abs(b))
			crossPrice := curve[i].Price + t*(curve[i+1].Price-curve[i].Price)
			dir := "rising"
			if b < a {
				dir = "falling"
			}
			crossings = append(crossings, ZeroCrossing{
				Price:     crossPrice,
				Direction: dir,
			})
		}
	}
	return crossings
}

func findExtrema(curve []HedgeImpulsePoint) []ImpulseExtremum {
	var extrema []ImpulseExtremum
	for i := 1; i < len(curve)-1; i++ {
		prev := curve[i-1].Impulse
		curr := curve[i].Impulse
		next := curve[i+1].Impulse

		if curr > prev && curr > next && curr > 0 {
			extrema = append(extrema, ImpulseExtremum{
				Price:   curve[i].Price,
				Impulse: curr,
				Type:    "basin",
			})
		}
		if curr < prev && curr < next && curr < 0 {
			extrema = append(extrema, ImpulseExtremum{
				Price:   curve[i].Price,
				Impulse: curr,
				Type:    "peak",
			})
		}
	}
	return extrema
}

func computeAsymmetry(curve []HedgeImpulsePoint, spot, integrationRangePercent float64) DirectionalAsymmetry {
	rangePrice := spot * (integrationRangePercent / 100)
	var step float64
	if len(curve) > 1 {
		step = curve[1].Price - curve[0].Price
	} else {
		step = 1
	}

	var upsideIntegral, downsideIntegral float64
	for _, pt := range curve {
		if pt.Price > spot && pt.Price <= spot+rangePrice {
			upsideIntegral += pt.Impulse * step
		}
		if pt.Price < spot && pt.Price >= spot-rangePrice {
			downsideIntegral += pt.Impulse * step
		}
	}

	bias := "neutral"
	threshold := math.Max(math.Abs(upsideIntegral), math.Abs(downsideIntegral)) * 0.1
	if upsideIntegral < downsideIntegral-threshold {
		bias = "up"
	} else if downsideIntegral < upsideIntegral-threshold {
		bias = "down"
	}

	denom := math.Abs(downsideIntegral)
	if denom < 1e-10 {
		denom = 1e-10
	}

	return DirectionalAsymmetry{
		Upside:                  upsideIntegral,
		Downside:                downsideIntegral,
		IntegrationRangePercent: integrationRangePercent,
		Bias:                    bias,
		AsymmetryRatio:          math.Abs(upsideIntegral) / denom,
	}
}

func classifyRegime(impulseAtSpot, slopeAtSpot float64, asymmetry DirectionalAsymmetry, curve []HedgeImpulsePoint, spot float64) ImpulseRegime {
	var sumAbs float64
	for _, pt := range curve {
		sumAbs += math.Abs(pt.Impulse)
	}
	meanAbsImpulse := sumAbs / float64(len(curve))
	if meanAbsImpulse == 0 {
		return ImpulseNeutral
	}

	normalized := impulseAtSpot / meanAbsImpulse

	if normalized > 0.5 {
		return ImpulsePinned
	}
	if normalized < -0.3 {
		switch asymmetry.Bias {
		case "up":
			return ImpulseSqueezeUp
		case "down":
			return ImpulseSqueezeDown
		default:
			return ImpulseExpansion
		}
	}

	if asymmetry.Bias == "up" && asymmetry.AsymmetryRatio > 1.5 {
		return ImpulseSqueezeUp
	}
	if asymmetry.Bias == "down" && asymmetry.AsymmetryRatio > 1.5 {
		return ImpulseSqueezeDown
	}

	return ImpulseNeutral
}
