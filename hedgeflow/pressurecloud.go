package hedgeflow

import (
	"math"
	"sort"
)

// Product multipliers for contract conversion.
var productMultipliers = map[string]float64{
	"NQ":  20,
	"MNQ": 2,
	"ES":  50,
	"MES": 5,
}

// ComputePressureCloud translates the raw impulse curve into actionable
// trading zones: stability zones (mean-reverting), acceleration zones
// (trend-amplifying), and regime edges (behavior transitions).
func ComputePressureCloud(
	impulseCurve HedgeImpulseCurve,
	regimeParams RegimeParams,
	config PressureCloudConfig,
	computedAt int64,
) PressureCloud {
	if config.ContractMultiplier == 0 {
		config.ContractMultiplier = 20
	}
	if config.ReachabilityMultiple == 0 {
		config.ReachabilityMultiple = 2.0
	}
	if config.ZoneThreshold == 0 {
		config.ZoneThreshold = 0.15
	}

	spot := impulseCurve.Spot
	expectedMove := regimeParams.ExpectedDailySpotMove * spot

	priceLevels := computePriceLevels(
		impulseCurve.Curve, spot, expectedMove,
		config.ReachabilityMultiple, config.ContractMultiplier,
	)

	stabilityZones := extractStabilityZones(
		impulseCurve.Extrema, impulseCurve.Curve, spot,
		expectedMove, config.ReachabilityMultiple, config.ZoneThreshold,
	)

	accelerationZones := extractAccelerationZones(
		impulseCurve.Extrema, impulseCurve.Curve, spot,
		expectedMove, config.ReachabilityMultiple, config.ZoneThreshold,
	)

	regimeEdges := convertZeroCrossingsToEdges(impulseCurve.ZeroCrossings, spot)

	return PressureCloud{
		Spot:              spot,
		Expiration:        impulseCurve.Expiration,
		ComputedAt:        computedAt,
		StabilityZones:    stabilityZones,
		AccelerationZones: accelerationZones,
		RegimeEdges:       regimeEdges,
		PriceLevels:       priceLevels,
	}
}

func computePriceLevels(
	curve []HedgeImpulsePoint, spot, expectedMove, reachabilityMultiple, contractMultiplier float64,
) []PressureLevel {
	reachRange := expectedMove * reachabilityMultiple

	levels := make([]PressureLevel, len(curve))
	for i, pt := range curve {
		distance := math.Abs(pt.Price - spot)
		proximity := math.Exp(-math.Pow(distance/reachRange, 2))

		var stabilityScore, accelerationScore float64
		if pt.Impulse > 0 {
			stabilityScore = pt.Impulse * proximity
		} else {
			accelerationScore = math.Abs(pt.Impulse) * proximity
		}

		contractDenom := contractMultiplier * spot * 0.01
		var hedgeContracts float64
		if contractDenom > 0 {
			hedgeContracts = pt.Impulse / contractDenom
		}

		hedgeType := "passive"
		if pt.Impulse < 0 {
			hedgeType = "aggressive"
		}

		levels[i] = PressureLevel{
			Price:                  pt.Price,
			StabilityScore:         stabilityScore,
			AccelerationScore:      accelerationScore,
			ExpectedHedgeContracts: sanitize(hedgeContracts),
			HedgeContracts:         computeHedgeContractEstimates(pt.Impulse, spot),
			HedgeType:              hedgeType,
		}
	}
	return levels
}

func extractStabilityZones(
	extrema []ImpulseExtremum, curve []HedgeImpulsePoint,
	spot, expectedMove, reachabilityMultiple, zoneThreshold float64,
) []PressureZone {
	var basins []ImpulseExtremum
	for _, e := range extrema {
		if e.Type == "basin" {
			basins = append(basins, e)
		}
	}
	if len(basins) == 0 {
		return nil
	}

	reachRange := expectedMove * reachabilityMultiple
	maxImpulse := 1e-10
	for _, b := range basins {
		if math.Abs(b.Impulse) > maxImpulse {
			maxImpulse = math.Abs(b.Impulse)
		}
	}

	var zones []PressureZone
	for _, basin := range basins {
		if math.Abs(basin.Impulse)/maxImpulse < zoneThreshold {
			continue
		}

		proximity := math.Exp(-math.Pow(math.Abs(basin.Price-spot)/reachRange, 2))
		rawStrength := (math.Abs(basin.Impulse) / maxImpulse) * proximity

		halfPeak := basin.Impulse * 0.5
		lower, upper := findZoneBounds(curve, basin.Price, halfPeak)

		side := "below-spot"
		if basin.Price >= spot {
			side = "above-spot"
		}
		tradeType := "long"
		if side == "above-spot" {
			tradeType = "short"
		}

		zones = append(zones, PressureZone{
			Center:    basin.Price,
			Lower:     lower,
			Upper:     upper,
			Strength:  math.Min(1, rawStrength),
			Side:      side,
			TradeType: tradeType,
			HedgeType: "passive",
		})
	}

	sort.Slice(zones, func(i, j int) bool { return zones[i].Strength > zones[j].Strength })
	return zones
}

func extractAccelerationZones(
	extrema []ImpulseExtremum, curve []HedgeImpulsePoint,
	spot, expectedMove, reachabilityMultiple, zoneThreshold float64,
) []PressureZone {
	var peaks []ImpulseExtremum
	for _, e := range extrema {
		if e.Type == "peak" {
			peaks = append(peaks, e)
		}
	}
	if len(peaks) == 0 {
		return nil
	}

	reachRange := expectedMove * reachabilityMultiple
	maxImpulse := 1e-10
	for _, p := range peaks {
		if math.Abs(p.Impulse) > maxImpulse {
			maxImpulse = math.Abs(p.Impulse)
		}
	}

	var zones []PressureZone
	for _, peak := range peaks {
		if math.Abs(peak.Impulse)/maxImpulse < zoneThreshold {
			continue
		}

		proximity := math.Exp(-math.Pow(math.Abs(peak.Price-spot)/reachRange, 2))
		rawStrength := (math.Abs(peak.Impulse) / maxImpulse) * proximity

		halfTrough := peak.Impulse * 0.5
		lower, upper := findZoneBounds(curve, peak.Price, halfTrough)

		side := "below-spot"
		if peak.Price >= spot {
			side = "above-spot"
		}
		tradeType := "short"
		if side == "above-spot" {
			tradeType = "long"
		}

		zones = append(zones, PressureZone{
			Center:    peak.Price,
			Lower:     lower,
			Upper:     upper,
			Strength:  math.Min(1, rawStrength),
			Side:      side,
			TradeType: tradeType,
			HedgeType: "aggressive",
		})
	}

	sort.Slice(zones, func(i, j int) bool { return zones[i].Strength > zones[j].Strength })
	return zones
}

func findZoneBounds(curve []HedgeImpulsePoint, centerPrice, thresholdImpulse float64) (float64, float64) {
	centerIdx := 0
	minDist := math.Inf(1)
	for i, pt := range curve {
		d := math.Abs(pt.Price - centerPrice)
		if d < minDist {
			minDist = d
			centerIdx = i
		}
	}

	isPositive := thresholdImpulse > 0

	lowerIdx := centerIdx
	for i := centerIdx - 1; i >= 0; i-- {
		if isPositive {
			if curve[i].Impulse < thresholdImpulse {
				lowerIdx = i
				break
			}
		} else {
			if curve[i].Impulse > thresholdImpulse {
				lowerIdx = i
				break
			}
		}
		lowerIdx = i
	}

	upperIdx := centerIdx
	for i := centerIdx + 1; i < len(curve); i++ {
		if isPositive {
			if curve[i].Impulse < thresholdImpulse {
				upperIdx = i
				break
			}
		} else {
			if curve[i].Impulse > thresholdImpulse {
				upperIdx = i
				break
			}
		}
		upperIdx = i
	}

	return curve[lowerIdx].Price, curve[upperIdx].Price
}

func convertZeroCrossingsToEdges(crossings []ZeroCrossing, spot float64) []RegimeEdge {
	edges := make([]RegimeEdge, len(crossings))
	for i, c := range crossings {
		isBelow := c.Price < spot
		var transType string
		if c.Direction == "falling" {
			if isBelow {
				transType = "stable-to-unstable"
			} else {
				transType = "unstable-to-stable"
			}
		} else {
			if isBelow {
				transType = "unstable-to-stable"
			} else {
				transType = "stable-to-unstable"
			}
		}
		edges[i] = RegimeEdge{
			Price:          c.Price,
			TransitionType: transType,
		}
	}
	return edges
}

func impulseToContracts(impulse, multiplier, spot float64) float64 {
	denom := multiplier * spot * 0.01
	if denom > 0 {
		return sanitize(impulse / denom)
	}
	return 0
}

func computeHedgeContractEstimates(impulse, spot float64) HedgeContractEstimates {
	return HedgeContractEstimates{
		NQ:  impulseToContracts(impulse, productMultipliers["NQ"], spot),
		MNQ: impulseToContracts(impulse, productMultipliers["MNQ"], spot),
		ES:  impulseToContracts(impulse, productMultipliers["ES"], spot),
		MES: impulseToContracts(impulse, productMultipliers["MES"], spot),
	}
}

func sanitize(v float64) float64 {
	if math.IsInf(v, 0) || math.IsNaN(v) {
		return 0
	}
	return v
}
