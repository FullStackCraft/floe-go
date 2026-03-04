package volresponse

import (
	"math"
	"testing"
)

func generateObservations(n int) []VolResponseObservation {
	obs := make([]VolResponseObservation, n)
	for i := 0; i < n; i++ {
		t := float64(i)
		obs[i] = VolResponseObservation{
			Timestamp:     int64(i * 60000),
			DeltaIV:       0.001 * math.Sin(t*0.1),              // oscillating IV changes
			SpotReturn:    0.002 * math.Cos(t*0.1),              // oscillating returns
			AbsSpotReturn: math.Abs(0.002 * math.Cos(t*0.1)),
			RVLevel:       0.15 + 0.01*math.Sin(t*0.05),
			IVLevel:       0.20 + 0.01*math.Cos(t*0.05),
		}
	}
	return obs
}

func TestBuildVolResponseObservation(t *testing.T) {
	obs := BuildVolResponseObservation(
		0.21, 0.15, 501.0, 120000,
		0.20, 500.0,
	)

	expectedDelta := 0.21 - 0.20
	if math.Abs(obs.DeltaIV-expectedDelta) > 1e-10 {
		t.Errorf("deltaIV = %f, want %f", obs.DeltaIV, expectedDelta)
	}

	expectedReturn := math.Log(501.0 / 500.0)
	if math.Abs(obs.SpotReturn-expectedReturn) > 1e-10 {
		t.Errorf("spotReturn = %f, want %f", obs.SpotReturn, expectedReturn)
	}

	if obs.AbsSpotReturn != math.Abs(expectedReturn) {
		t.Errorf("absSpotReturn = %f, want %f", obs.AbsSpotReturn, math.Abs(expectedReturn))
	}

	if obs.Timestamp != 120000 {
		t.Errorf("timestamp = %d, want 120000", obs.Timestamp)
	}
}

func TestComputeVolResponseZScore_InsufficientData(t *testing.T) {
	obs := generateObservations(10) // less than default min of 30
	config := VolResponseConfig{}

	result := ComputeVolResponseZScore(obs, config)

	if result.IsValid {
		t.Error("expected invalid for insufficient data")
	}
	if result.Signal != "insufficient_data" {
		t.Errorf("expected insufficient_data signal, got %s", result.Signal)
	}
	if result.NumObservations != 10 {
		t.Errorf("expected 10 observations, got %d", result.NumObservations)
	}
}

func TestComputeVolResponseZScore_ValidModel(t *testing.T) {
	obs := generateObservations(50)
	config := VolResponseConfig{}

	result := ComputeVolResponseZScore(obs, config)

	if !result.IsValid {
		t.Error("expected valid model with 50 observations")
	}
	if result.NumObservations != 50 {
		t.Errorf("expected 50 observations, got %d", result.NumObservations)
	}
	if result.RSquared < 0 || result.RSquared > 1 {
		t.Errorf("R-squared %f outside [0, 1]", result.RSquared)
	}
	if result.ResidualStdDev < 0 {
		t.Errorf("expected non-negative residual std dev, got %f", result.ResidualStdDev)
	}
	if result.Signal != "vol_bid" && result.Signal != "vol_offered" && result.Signal != "neutral" {
		t.Errorf("unexpected signal: %s", result.Signal)
	}
	if math.IsNaN(result.ZScore) || math.IsInf(result.ZScore, 0) {
		t.Errorf("z-score is not finite: %f", result.ZScore)
	}
}

func TestComputeVolResponseZScore_EmptyObservations(t *testing.T) {
	result := ComputeVolResponseZScore(nil, VolResponseConfig{})

	if result.IsValid {
		t.Error("expected invalid for nil observations")
	}
	if result.Signal != "insufficient_data" {
		t.Errorf("expected insufficient_data, got %s", result.Signal)
	}
}

func TestComputeVolResponseZScore_CustomThresholds(t *testing.T) {
	obs := generateObservations(50)
	config := VolResponseConfig{
		MinObservations:     20,
		VolBidThreshold:     0.5,
		VolOfferedThreshold: -0.5,
	}

	result := ComputeVolResponseZScore(obs, config)

	if !result.IsValid {
		t.Error("expected valid model")
	}
	if result.MinObservations != 20 {
		t.Errorf("expected 20 min observations, got %d", result.MinObservations)
	}
}
