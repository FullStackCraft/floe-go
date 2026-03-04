package hedgeflow

import (
	"math"

	floe "github.com/FullStackCraft/floe-go"
)

const (
	skewToCorrScale  = 0.15
	volOfVolScale    = 2.0
)

// DeriveRegimeParams extracts market regime parameters from an IV surface.
func DeriveRegimeParams(ivSurface floe.IVSurface, spot float64) RegimeParams {
	strikes := ivSurface.Strikes
	ivs := ivSurface.SmoothedIVs

	atmIV := InterpolateIVAtStrike(strikes, ivs, spot) / 100.0
	skew := calculateSkewAtSpot(strikes, ivs, spot)
	impliedCorr := skewToCorrelation(skew)
	curvature := calculateCurvatureAtSpot(strikes, ivs, spot)
	volOfVol := curvatureToVolOfVol(curvature, atmIV)
	regime := ivToRegime(atmIV)

	return RegimeParams{
		AtmIV:                 atmIV,
		ImpliedSpotVolCorr:    impliedCorr,
		ImpliedVolOfVol:       volOfVol,
		Regime:                regime,
		ExpectedDailySpotMove: atmIV / math.Sqrt(252),
		ExpectedDailyVolMove:  volOfVol / math.Sqrt(252),
	}
}

// InterpolateIVAtStrike linearly interpolates IV at an arbitrary strike.
func InterpolateIVAtStrike(strikes, ivs []float64, targetStrike float64) float64 {
	if len(strikes) == 0 || len(ivs) == 0 {
		return 20
	}
	if len(strikes) == 1 {
		return ivs[0]
	}

	if targetStrike <= strikes[0] {
		return ivs[0]
	}
	if targetStrike >= strikes[len(strikes)-1] {
		return ivs[len(ivs)-1]
	}

	lower, upper := 0, len(strikes)-1
	for i := 0; i < len(strikes)-1; i++ {
		if strikes[i] <= targetStrike && strikes[i+1] >= targetStrike {
			lower = i
			upper = i + 1
			break
		}
	}

	dK := strikes[upper] - strikes[lower]
	if dK == 0 {
		return ivs[lower]
	}
	t := (targetStrike - strikes[lower]) / dK
	return ivs[lower] + t*(ivs[upper]-ivs[lower])
}

func skewToCorrelation(skew float64) float64 {
	return math.Max(-0.95, math.Min(0.5, skew*skewToCorrScale))
}

func curvatureToVolOfVol(curvature, atmIV float64) float64 {
	return math.Sqrt(math.Abs(curvature)) * volOfVolScale * atmIV
}

func ivToRegime(atmIV float64) MarketRegime {
	if atmIV < 0.15 {
		return RegimeCalm
	}
	if atmIV < 0.20 {
		return RegimeNormal
	}
	if atmIV < 0.35 {
		return RegimeStressed
	}
	return RegimeCrisis
}

func calculateSkewAtSpot(strikes, ivs []float64, spot float64) float64 {
	if len(strikes) < 2 {
		return 0
	}

	lowerIdx, upperIdx := 0, len(strikes)-1
	for i := 0; i < len(strikes)-1; i++ {
		if strikes[i] <= spot && strikes[i+1] >= spot {
			lowerIdx = i
			upperIdx = i + 1
			break
		}
	}

	dIV := ivs[upperIdx] - ivs[lowerIdx]
	dK := strikes[upperIdx] - strikes[lowerIdx]
	if dK <= 0 {
		return 0
	}
	return (dIV / dK) * spot
}

func calculateCurvatureAtSpot(strikes, ivs []float64, spot float64) float64 {
	if len(strikes) < 3 {
		return 0
	}

	centerIdx := 0
	for i := 0; i < len(strikes); i++ {
		if math.Abs(strikes[i]-spot) < math.Abs(strikes[centerIdx]-spot) {
			centerIdx = i
		}
	}

	if centerIdx == 0 || centerIdx == len(strikes)-1 {
		return 0
	}

	h := (strikes[centerIdx+1] - strikes[centerIdx-1]) / 2
	if h <= 0 {
		return 0
	}

	ivMinus := ivs[centerIdx-1]
	iv := ivs[centerIdx]
	ivPlus := ivs[centerIdx+1]

	return ((ivPlus - 2*iv + ivMinus) / (h * h)) * spot * spot
}
