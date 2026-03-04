// Package exposure computes dealer gamma, vanna, and charm exposures
// (GEX, VEX, CEX) in three variants: canonical, state-weighted, and flow-delta.
package exposure

import (
	"fmt"
	"math"
	"sort"

	floe "github.com/FullStackCraft/floe-go"
	"github.com/FullStackCraft/floe-go/blackscholes"
	"github.com/FullStackCraft/floe-go/volatility"
)

// SharesCoverResult holds the result of computing shares needed to cover exposure.
type SharesCoverResult struct {
	ActionToCover        string
	SharesToCover        float64
	ImpliedMoveToCover   float64
	ResultingSpotToCover float64
}

// CalculateGammaVannaCharmExposures computes canonical, state-weighted, and
// flow-delta exposure variants for each expiration in the option chain.
func CalculateGammaVannaCharmExposures(
	chain floe.OptionChain,
	ivSurfaces []floe.IVSurface,
	opts floe.ExposureCalculationOptions,
) []floe.ExposureVariantsPerExpiry {
	spot := chain.Spot
	asOfTimestamp := opts.AsOfTimestamp
	if asOfTimestamp == 0 {
		// Caller must provide; there is no Date.now() equivalent here.
		// Return empty if not set.
		return nil
	}

	// Collect unique expirations.
	expirationSet := make(map[int64]bool)
	for _, opt := range chain.Options {
		expirationSet[opt.ExpirationTimestamp] = true
	}
	expirations := make([]int64, 0, len(expirationSet))
	for exp := range expirationSet {
		expirations = append(expirations, exp)
	}
	sort.Slice(expirations, func(i, j int) bool { return expirations[i] < expirations[j] })

	// Index puts by expiration:strike for fast lookup.
	putsByKey := make(map[string]floe.NormalizedOption)
	for _, opt := range chain.Options {
		if opt.OptionType == floe.Put {
			key := optionKey(opt.ExpirationTimestamp, opt.Strike)
			putsByKey[key] = opt
		}
	}

	var results []floe.ExposureVariantsPerExpiry

	for _, exp := range expirations {
		if exp < asOfTimestamp {
			continue
		}

		tte := float64(exp-asOfTimestamp) / floe.MillisecondsPerYear
		if tte <= 0 {
			continue
		}

		dteInDays := math.Max(tte*floe.DaysPerYear, 0)
		var strikeVariants []floe.StrikeExposureVariants

		for _, callOpt := range chain.Options {
			if callOpt.ExpirationTimestamp != exp || callOpt.OptionType != floe.Call {
				continue
			}

			putOpt, ok := putsByKey[optionKey(exp, callOpt.Strike)]
			if !ok {
				continue
			}

			// Resolve IV from surface with fallback.
			callIV := resolveIVPercent(
				volatility.GetIVForStrike(ivSurfaces, exp, floe.Call, callOpt.Strike),
				callOpt.ImpliedVolatility,
			)
			putIV := resolveIVPercent(
				volatility.GetIVForStrike(ivSurfaces, exp, floe.Put, putOpt.Strike),
				putOpt.ImpliedVolatility,
			)

			callGreeks := blackscholes.CalculateGreeks(floe.BlackScholesParams{
				Spot:          spot,
				Strike:        callOpt.Strike,
				TimeToExpiry:  tte,
				Volatility:    callIV / 100.0,
				RiskFreeRate:  chain.RiskFreeRate,
				DividendYield: chain.DividendYield,
				OptionType:    floe.Call,
			})
			putGreeks := blackscholes.CalculateGreeks(floe.BlackScholesParams{
				Spot:          spot,
				Strike:        putOpt.Strike,
				TimeToExpiry:  tte,
				Volatility:    putIV / 100.0,
				RiskFreeRate:  chain.RiskFreeRate,
				DividendYield: chain.DividendYield,
				OptionType:    floe.Put,
			})

			callOI := sanitizeFinite(callOpt.OpenInterest)
			putOI := sanitizeFinite(putOpt.OpenInterest)

			canonical := calculateCanonicalVector(
				spot, callOI, putOI,
				callGreeks.Gamma, putGreeks.Gamma,
				callGreeks.Vanna, putGreeks.Vanna,
				callGreeks.Charm, putGreeks.Charm,
			)

			stateWeighted := calculateStateWeightedVector(
				spot, callOI, putOI,
				callGreeks.Vanna, putGreeks.Vanna,
				callGreeks.Charm, putGreeks.Charm,
				callIV, putIV,
				dteInDays, canonical.GammaExposure,
			)

			callFlowDelta := resolveFlowDeltaOI(callOpt.OpenInterest, callOpt.LiveOpenInterest)
			putFlowDelta := resolveFlowDeltaOI(putOpt.OpenInterest, putOpt.LiveOpenInterest)
			flowDelta := calculateCanonicalVector(
				spot, callFlowDelta, putFlowDelta,
				callGreeks.Gamma, putGreeks.Gamma,
				callGreeks.Vanna, putGreeks.Vanna,
				callGreeks.Charm, putGreeks.Charm,
			)

			strikeVariants = append(strikeVariants, floe.StrikeExposureVariants{
				StrikePrice:   callOpt.Strike,
				Canonical:     canonical,
				StateWeighted: stateWeighted,
				FlowDelta:     flowDelta,
			})
		}

		if len(strikeVariants) == 0 {
			continue
		}

		results = append(results, floe.ExposureVariantsPerExpiry{
			SpotPrice:              spot,
			Expiration:             exp,
			Canonical:              buildModeBreakdown(strikeVariants, "canonical"),
			StateWeighted:          buildModeBreakdown(strikeVariants, "stateWeighted"),
			FlowDelta:              buildModeBreakdown(strikeVariants, "flowDelta"),
			StrikeExposureVariants: strikeVariants,
		})
	}

	return results
}

// CalculateSharesNeededToCover computes the shares dealers need to trade
// to neutralize net exposure.
func CalculateSharesNeededToCover(sharesOutstanding, totalNetExposure, underlyingMark float64) SharesCoverResult {
	actionToCover := "BUY"
	if totalNetExposure > 0 {
		actionToCover = "SELL"
	}

	if sharesOutstanding == 0 || math.IsNaN(sharesOutstanding) || math.IsInf(sharesOutstanding, 0) {
		return SharesCoverResult{ResultingSpotToCover: underlyingMark}
	}
	if underlyingMark == 0 || math.IsNaN(underlyingMark) || math.IsInf(underlyingMark, 0) {
		return SharesCoverResult{ResultingSpotToCover: underlyingMark}
	}

	sharesNeeded := -totalNetExposure / underlyingMark
	impliedChange := (sharesNeeded / sharesOutstanding) * 100.0
	resultingPrice := underlyingMark * (1 + impliedChange/100)

	return SharesCoverResult{
		ActionToCover:        actionToCover,
		SharesToCover:        math.Abs(sharesNeeded),
		ImpliedMoveToCover:   impliedChange,
		ResultingSpotToCover: resultingPrice,
	}
}

// ---- Internal helpers ----

func optionKey(expiration int64, strike float64) string {
	return fmt.Sprintf("%d:%.4f", expiration, strike)
}

func calculateCanonicalVector(
	spot, callOI, putOI,
	callGamma, putGamma,
	callVanna, putVanna,
	callCharm, putCharm float64,
) floe.ExposureVector {
	gex := -callOI*callGamma*(spot*100.0)*spot*0.01 +
		putOI*putGamma*(spot*100.0)*spot*0.01

	vex := -callOI*callVanna*(spot*100.0)*0.01 +
		putOI*putVanna*(spot*100.0)*0.01

	cex := -callOI*callCharm*(spot*100.0) +
		putOI*putCharm*(spot*100.0)

	return sanitizeVector(floe.ExposureVector{
		GammaExposure: gex,
		VannaExposure: vex,
		CharmExposure: cex,
		NetExposure:   gex + vex + cex,
	})
}

func calculateStateWeightedVector(
	spot, callOI, putOI,
	callVanna, putVanna,
	callCharm, putCharm,
	callIVPercent, putIVPercent,
	dteInDays, canonicalGEX float64,
) floe.ExposureVector {
	callIVLevel := math.Max(callIVPercent*0.01, 0)
	putIVLevel := math.Max(putIVPercent*0.01, 0)

	gammaExposure := canonicalGEX

	vannaExposure := -callOI*callVanna*(spot*100.0)*0.01*callIVLevel +
		putOI*putVanna*(spot*100.0)*0.01*putIVLevel

	canonicalCharm := -callOI*callCharm*(spot*100.0) +
		putOI*putCharm*(spot*100.0)
	charmExposure := canonicalCharm * math.Max(dteInDays, 0)

	return sanitizeVector(floe.ExposureVector{
		GammaExposure: gammaExposure,
		VannaExposure: vannaExposure,
		CharmExposure: charmExposure,
		NetExposure:   gammaExposure + vannaExposure + charmExposure,
	})
}

func resolveFlowDeltaOI(openInterest float64, liveOI *float64) float64 {
	if liveOI == nil || math.IsInf(*liveOI, 0) || math.IsNaN(*liveOI) {
		return 0
	}
	return sanitizeFinite(*liveOI - openInterest)
}

func resolveIVPercent(ivFromSurface, optionIVDecimal float64) float64 {
	if !math.IsInf(ivFromSurface, 0) && !math.IsNaN(ivFromSurface) && ivFromSurface > 0 {
		return ivFromSurface
	}
	fallback := optionIVDecimal * 100.0
	if !math.IsInf(fallback, 0) && !math.IsNaN(fallback) && fallback > 0 {
		return fallback
	}
	return 0
}

func buildModeBreakdown(variants []floe.StrikeExposureVariants, mode string) floe.ExposureModeBreakdown {
	if len(variants) == 0 {
		return floe.ExposureModeBreakdown{}
	}

	exposures := make([]floe.StrikeExposure, len(variants))
	for i, v := range variants {
		var vec floe.ExposureVector
		switch mode {
		case "canonical":
			vec = v.Canonical
		case "stateWeighted":
			vec = v.StateWeighted
		case "flowDelta":
			vec = v.FlowDelta
		}
		exposures[i] = floe.StrikeExposure{
			StrikePrice:   v.StrikePrice,
			GammaExposure: vec.GammaExposure,
			VannaExposure: vec.VannaExposure,
			CharmExposure: vec.CharmExposure,
			NetExposure:   vec.NetExposure,
		}
	}

	var totalGEX, totalVEX, totalCEX float64
	for _, e := range exposures {
		totalGEX += e.GammaExposure
		totalVEX += e.VannaExposure
		totalCEX += e.CharmExposure
	}
	totalNet := totalGEX + totalVEX + totalCEX

	// Sort copies for max/min tracking.
	byGamma := make([]floe.StrikeExposure, len(exposures))
	copy(byGamma, exposures)
	sort.Slice(byGamma, func(i, j int) bool { return byGamma[i].GammaExposure > byGamma[j].GammaExposure })

	byVanna := make([]floe.StrikeExposure, len(exposures))
	copy(byVanna, exposures)
	sort.Slice(byVanna, func(i, j int) bool { return byVanna[i].VannaExposure > byVanna[j].VannaExposure })

	byCharm := make([]floe.StrikeExposure, len(exposures))
	copy(byCharm, exposures)
	sort.Slice(byCharm, func(i, j int) bool { return byCharm[i].CharmExposure > byCharm[j].CharmExposure })

	byNet := make([]floe.StrikeExposure, len(exposures))
	copy(byNet, exposures)
	sort.Slice(byNet, func(i, j int) bool { return byNet[i].NetExposure > byNet[j].NetExposure })

	return floe.ExposureModeBreakdown{
		TotalGammaExposure: sanitizeFinite(totalGEX),
		TotalVannaExposure: sanitizeFinite(totalVEX),
		TotalCharmExposure: sanitizeFinite(totalCEX),
		TotalNetExposure:   sanitizeFinite(totalNet),
		StrikeOfMaxGamma:   byGamma[0].StrikePrice,
		StrikeOfMinGamma:   byGamma[len(byGamma)-1].StrikePrice,
		StrikeOfMaxVanna:   byVanna[0].StrikePrice,
		StrikeOfMinVanna:   byVanna[len(byVanna)-1].StrikePrice,
		StrikeOfMaxCharm:   byCharm[0].StrikePrice,
		StrikeOfMinCharm:   byCharm[len(byCharm)-1].StrikePrice,
		StrikeOfMaxNet:     byNet[0].StrikePrice,
		StrikeOfMinNet:     byNet[len(byNet)-1].StrikePrice,
		StrikeExposures:    byNet,
	}
}

func sanitizeVector(v floe.ExposureVector) floe.ExposureVector {
	return floe.ExposureVector{
		GammaExposure: sanitizeFinite(v.GammaExposure),
		VannaExposure: sanitizeFinite(v.VannaExposure),
		CharmExposure: sanitizeFinite(v.CharmExposure),
		NetExposure:   sanitizeFinite(v.NetExposure),
	}
}

func sanitizeFinite(v float64) float64 {
	if math.IsInf(v, 0) || math.IsNaN(v) {
		return 0
	}
	return v
}
