package rv

import (
	"math"
	"testing"
)

func TestComputeRealizedVolatility_BasicSeries(t *testing.T) {
	// Simulate a price series with known log returns.
	obs := []PriceObservation{
		{Price: 100, Timestamp: 0},
		{Price: 101, Timestamp: 60000},    // 1 min
		{Price: 100.5, Timestamp: 120000}, // 2 min
		{Price: 102, Timestamp: 180000},   // 3 min
		{Price: 101, Timestamp: 240000},   // 4 min
	}

	result := ComputeRealizedVolatility(obs)

	if result.NumObservations != 5 {
		t.Errorf("expected 5 observations, got %d", result.NumObservations)
	}
	if result.NumReturns != 4 {
		t.Errorf("expected 4 returns, got %d", result.NumReturns)
	}
	if result.RealizedVolatility <= 0 {
		t.Errorf("expected positive RV, got %f", result.RealizedVolatility)
	}
	if result.QuadraticVariation <= 0 {
		t.Errorf("expected positive QV, got %f", result.QuadraticVariation)
	}
	if result.ElapsedMinutes <= 0 {
		t.Errorf("expected positive elapsed minutes, got %f", result.ElapsedMinutes)
	}
}

func TestComputeRealizedVolatility_TooFewObs(t *testing.T) {
	obs := []PriceObservation{
		{Price: 100, Timestamp: 0},
	}
	result := ComputeRealizedVolatility(obs)
	if result.RealizedVolatility != 0 {
		t.Errorf("expected 0 RV for single observation, got %f", result.RealizedVolatility)
	}
}

func TestComputeRealizedVolatility_FiltersInvalid(t *testing.T) {
	obs := []PriceObservation{
		{Price: 100, Timestamp: 0},
		{Price: -1, Timestamp: 60000},             // invalid
		{Price: math.NaN(), Timestamp: 120000},     // invalid
		{Price: math.Inf(1), Timestamp: 180000},    // invalid
		{Price: 101, Timestamp: 240000},
	}
	result := ComputeRealizedVolatility(obs)
	if result.NumObservations != 2 {
		t.Errorf("expected 2 valid observations, got %d", result.NumObservations)
	}
}

func TestComputeRealizedVolatility_ZeroElapsed(t *testing.T) {
	obs := []PriceObservation{
		{Price: 100, Timestamp: 1000},
		{Price: 101, Timestamp: 1000}, // same timestamp
	}
	result := ComputeRealizedVolatility(obs)
	if result.RealizedVolatility != 0 {
		t.Errorf("expected 0 RV for zero elapsed time, got %f", result.RealizedVolatility)
	}
}
