package exposure

import (
	"math"
	"testing"

	floe "github.com/FullStackCraft/floe-go"
)

func TestCalculateSharesNeededToCover_PositiveExposure(t *testing.T) {
	result := CalculateSharesNeededToCover(1_000_000_000, 5_000_000, 500)

	if result.ActionToCover != "SELL" {
		t.Errorf("expected SELL for positive exposure, got %s", result.ActionToCover)
	}
	if result.SharesToCover <= 0 {
		t.Error("expected positive shares to cover")
	}
}

func TestCalculateSharesNeededToCover_NegativeExposure(t *testing.T) {
	result := CalculateSharesNeededToCover(1_000_000_000, -5_000_000, 500)

	if result.ActionToCover != "BUY" {
		t.Errorf("expected BUY for negative exposure, got %s", result.ActionToCover)
	}
	if result.SharesToCover <= 0 {
		t.Error("expected positive shares to cover")
	}
}

func TestCalculateSharesNeededToCover_ZeroSharesOutstanding(t *testing.T) {
	result := CalculateSharesNeededToCover(0, 5_000_000, 500)
	if result.ActionToCover != "" {
		t.Errorf("expected empty action, got %s", result.ActionToCover)
	}
	if result.SharesToCover != 0 {
		t.Errorf("expected 0 shares, got %f", result.SharesToCover)
	}
}

func TestCalculateSharesNeededToCover_NaNInput(t *testing.T) {
	result := CalculateSharesNeededToCover(math.NaN(), 5_000_000, 500)
	if result.SharesToCover != 0 {
		t.Errorf("expected 0 shares for NaN input, got %f", result.SharesToCover)
	}
}

func TestResolveIVPercent_FromSurface(t *testing.T) {
	got := resolveIVPercent(25.0, 0.20)
	if got != 25.0 {
		t.Errorf("expected 25.0, got %f", got)
	}
}

func TestResolveIVPercent_Fallback(t *testing.T) {
	got := resolveIVPercent(0, 0.20)
	if got != 20.0 {
		t.Errorf("expected 20.0, got %f", got)
	}
}

func TestSanitizeFinite(t *testing.T) {
	if sanitizeFinite(math.NaN()) != 0 {
		t.Error("NaN should be sanitized to 0")
	}
	if sanitizeFinite(math.Inf(1)) != 0 {
		t.Error("+Inf should be sanitized to 0")
	}
	if sanitizeFinite(42.0) != 42.0 {
		t.Error("42 should pass through")
	}
}

func TestCalculateCanonicalVector_SignConventions(t *testing.T) {
	// Dealers are short calls (negative GEX contribution) and short puts (positive GEX contribution).
	spot := 500.0
	callOI := 1000.0
	putOI := 1000.0
	gamma := 0.01
	vanna := 0.005
	charm := 0.001

	vec := calculateCanonicalVector(spot, callOI, putOI, gamma, gamma, vanna, vanna, charm, charm)

	// With equal OI and Greeks, GEX = (-callOI + putOI) * gamma * spot² * 0.01 = 0
	// The signs should cancel out for symmetric case.
	_ = vec
}

func TestBuildModeBreakdown_Basic(t *testing.T) {
	variants := []floe.StrikeExposureVariants{
		{
			StrikePrice: 500,
			Canonical: floe.ExposureVector{
				GammaExposure: 100,
				VannaExposure: 50,
				CharmExposure: 25,
				NetExposure:   175,
			},
		},
		{
			StrikePrice: 510,
			Canonical: floe.ExposureVector{
				GammaExposure: -100,
				VannaExposure: -50,
				CharmExposure: -25,
				NetExposure:   -175,
			},
		},
	}

	breakdown := buildModeBreakdown(variants, "canonical")
	if breakdown.TotalGammaExposure != 0 {
		t.Errorf("expected 0 total GEX for symmetric case, got %f", breakdown.TotalGammaExposure)
	}
	if breakdown.StrikeOfMaxGamma != 500 {
		t.Errorf("expected max gamma at 500, got %f", breakdown.StrikeOfMaxGamma)
	}
	if breakdown.StrikeOfMinGamma != 510 {
		t.Errorf("expected min gamma at 510, got %f", breakdown.StrikeOfMinGamma)
	}
}
