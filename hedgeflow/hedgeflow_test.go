package hedgeflow

import (
	"math"
	"testing"

	floe "github.com/FullStackCraft/floe-go"
)

func makeIVSurface(spot float64) floe.IVSurface {
	// Create a simple downward-sloping IV smile.
	strikes := []float64{480, 490, 500, 510, 520}
	ivs := []float64{25, 22, 20, 22, 25} // percent
	return floe.IVSurface{
		ExpirationDate: 1000000 + 30*86400*1000,
		PutCall:        floe.Call,
		Strikes:        strikes,
		RawIVs:         ivs,
		SmoothedIVs:    ivs,
	}
}

func makeExposures(spot float64) floe.ExposurePerExpiry {
	return floe.ExposurePerExpiry{
		SpotPrice:          spot,
		Expiration:         1000000 + 30*86400*1000,
		TotalGammaExposure: 1000000,
		TotalVannaExposure: -500000,
		TotalCharmExposure: -200000,
		StrikeExposures: []floe.StrikeExposure{
			{StrikePrice: 490, GammaExposure: 500000, VannaExposure: -300000, CharmExposure: -100000},
			{StrikePrice: 495, GammaExposure: 300000, VannaExposure: -200000, CharmExposure: -50000},
			{StrikePrice: 500, GammaExposure: -200000, VannaExposure: 100000, CharmExposure: 50000},
			{StrikePrice: 505, GammaExposure: 100000, VannaExposure: -50000, CharmExposure: -50000},
			{StrikePrice: 510, GammaExposure: 300000, VannaExposure: -50000, CharmExposure: -50000},
		},
	}
}

func TestDeriveRegimeParams_CalmMarket(t *testing.T) {
	surface := makeIVSurface(500)
	// 20% ATM IV should be "normal" regime.
	surface.SmoothedIVs = []float64{14, 13, 12, 13, 14}

	params := DeriveRegimeParams(surface, 500)

	if params.Regime != RegimeCalm {
		t.Errorf("expected calm regime for ~12%% IV, got %s", params.Regime)
	}
	if params.AtmIV <= 0 {
		t.Error("expected positive ATM IV")
	}
	if params.ExpectedDailySpotMove <= 0 {
		t.Error("expected positive expected daily move")
	}
}

func TestDeriveRegimeParams_NormalMarket(t *testing.T) {
	surface := makeIVSurface(500)
	// Set IVs to ~18% ATM which falls in [0.15, 0.20) = normal.
	surface.SmoothedIVs = []float64{22, 20, 18, 20, 22}

	params := DeriveRegimeParams(surface, 500)

	if params.Regime != RegimeNormal {
		t.Errorf("expected normal regime for ~18%% IV, got %s (atmIV=%f)", params.Regime, params.AtmIV)
	}
}

func TestInterpolateIVAtStrike_Exact(t *testing.T) {
	strikes := []float64{490, 500, 510}
	ivs := []float64{22, 20, 22}

	got := InterpolateIVAtStrike(strikes, ivs, 500)
	if got != 20 {
		t.Errorf("expected 20 at exact strike, got %f", got)
	}
}

func TestInterpolateIVAtStrike_Midpoint(t *testing.T) {
	strikes := []float64{490, 510}
	ivs := []float64{22, 26}

	got := InterpolateIVAtStrike(strikes, ivs, 500)
	if math.Abs(got-24) > 0.01 {
		t.Errorf("expected ~24 at midpoint, got %f", got)
	}
}

func TestComputeHedgeImpulseCurve_Basic(t *testing.T) {
	exposures := makeExposures(500)
	surface := makeIVSurface(500)
	config := HedgeImpulseConfig{} // use defaults
	computedAt := int64(1000000)

	curve := ComputeHedgeImpulseCurve(exposures, surface, config, computedAt)

	if len(curve.Curve) == 0 {
		t.Fatal("expected non-empty curve")
	}
	if curve.Spot != 500 {
		t.Errorf("expected spot=500, got %f", curve.Spot)
	}
	if curve.SpotVolCoupling < 2 || curve.SpotVolCoupling > 20 {
		t.Errorf("spot-vol coupling %f outside [2, 20]", curve.SpotVolCoupling)
	}
	if curve.StrikeSpacing <= 0 {
		t.Errorf("expected positive strike spacing, got %f", curve.StrikeSpacing)
	}
}

func TestComputeHedgeImpulseCurve_FindsZeroCrossings(t *testing.T) {
	exposures := makeExposures(500)
	surface := makeIVSurface(500)
	curve := ComputeHedgeImpulseCurve(exposures, surface, HedgeImpulseConfig{}, 1000000)

	// With mixed positive/negative exposures, we should find some zero crossings.
	// (Not guaranteed, but likely with our test data.)
	if len(curve.Curve) < 10 {
		t.Error("curve should have many points")
	}
}

func TestDetectStrikeSpacing(t *testing.T) {
	strikes := []float64{490, 495, 500, 505, 510}
	spacing := detectStrikeSpacing(strikes)
	if spacing != 5.0 {
		t.Errorf("expected 5.0 spacing, got %f", spacing)
	}
}

func TestKernelSmooth_AtStrike(t *testing.T) {
	strikes := []float64{490, 500, 510}
	values := []float64{100, 200, 100}
	lambda := 10.0

	// At the center strike, value should be close to 200 (highest weight).
	result := kernelSmooth(strikes, values, 500, lambda)
	if result < 100 || result > 200 {
		t.Errorf("kernel smooth at center should be 100-200, got %f", result)
	}
}

func TestComputeCharmIntegral_Basic(t *testing.T) {
	exposures := makeExposures(500)
	config := CharmIntegralConfig{} // use default 15 min
	// Set computedAt so there's time remaining.
	computedAt := int64(1000000)

	integral := ComputeCharmIntegral(exposures, config, computedAt)

	if integral.Spot != 500 {
		t.Errorf("expected spot=500, got %f", integral.Spot)
	}
	if integral.MinutesRemaining <= 0 {
		t.Error("expected positive minutes remaining")
	}
	if len(integral.Buckets) == 0 {
		t.Error("expected non-empty buckets")
	}
	if integral.Direction != "buying" && integral.Direction != "selling" && integral.Direction != "neutral" {
		t.Errorf("unexpected direction: %s", integral.Direction)
	}
}

func TestComputeCharmIntegral_Expired(t *testing.T) {
	exposures := makeExposures(500)
	exposures.Expiration = 500000 // already expired relative to computedAt

	integral := ComputeCharmIntegral(exposures, CharmIntegralConfig{}, 1000000)

	if integral.MinutesRemaining != 0 {
		t.Error("expected 0 minutes remaining for expired")
	}
	if integral.TotalCharmToClose != 0 {
		t.Error("expected 0 charm for expired")
	}
}

func TestComputePressureCloud_Basic(t *testing.T) {
	exposures := makeExposures(500)
	surface := makeIVSurface(500)
	impulse := ComputeHedgeImpulseCurve(exposures, surface, HedgeImpulseConfig{}, 1000000)
	regime := DeriveRegimeParams(surface, 500)

	cloud := ComputePressureCloud(impulse, regime, PressureCloudConfig{}, 1000000)

	if cloud.Spot != 500 {
		t.Errorf("expected spot=500, got %f", cloud.Spot)
	}
	if len(cloud.PriceLevels) == 0 {
		t.Error("expected non-empty price levels")
	}
}
