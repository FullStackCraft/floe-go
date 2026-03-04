package impliedpdf

import (
	"math"
	"testing"

	floe "github.com/FullStackCraft/floe-go"
)

func makeCallOptions() []floe.NormalizedOption {
	exp := int64(1000000 + 30*86400*1000)
	return []floe.NormalizedOption{
		{Strike: 490, OptionType: floe.Call, Bid: 12.0, Ask: 12.5, Mark: 12.25, ExpirationTimestamp: exp},
		{Strike: 495, OptionType: floe.Call, Bid: 8.0, Ask: 8.5, Mark: 8.25, ExpirationTimestamp: exp},
		{Strike: 500, OptionType: floe.Call, Bid: 5.0, Ask: 5.5, Mark: 5.25, ExpirationTimestamp: exp},
		{Strike: 505, OptionType: floe.Call, Bid: 3.0, Ask: 3.5, Mark: 3.25, ExpirationTimestamp: exp},
		{Strike: 510, OptionType: floe.Call, Bid: 1.5, Ask: 2.0, Mark: 1.75, ExpirationTimestamp: exp},
	}
}

func TestEstimateImpliedPDF_Basic(t *testing.T) {
	calls := makeCallOptions()
	result := EstimateImpliedProbabilityDistribution("QQQ", 500, calls, 1000000)

	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	dist := result.Distribution
	if dist.Symbol != "QQQ" {
		t.Errorf("expected QQQ, got %s", dist.Symbol)
	}

	// Probabilities should sum to ~1.
	sum := 0.0
	for _, sp := range dist.StrikeProbabilities {
		sum += sp.Probability
	}
	if math.Abs(sum-1.0) > 0.01 {
		t.Errorf("probabilities sum to %f, expected ~1.0", sum)
	}

	// Mode should be in the strike range.
	if dist.MostLikelyPrice < 490 || dist.MostLikelyPrice > 510 {
		t.Errorf("mode %f outside strike range", dist.MostLikelyPrice)
	}

	// Expected move should be positive.
	if dist.ExpectedMove <= 0 {
		t.Error("expected positive expected move")
	}
}

func TestEstimateImpliedPDF_TooFewOptions(t *testing.T) {
	calls := makeCallOptions()[:2]
	result := EstimateImpliedProbabilityDistribution("QQQ", 500, calls, 1000000)

	if result.Success {
		t.Error("expected failure with < 3 options")
	}
}

func TestGetProbabilityInRange(t *testing.T) {
	result := EstimateImpliedProbabilityDistribution("QQQ", 500, makeCallOptions(), 1000000)
	if !result.Success {
		t.Fatal("failed to estimate PDF")
	}

	dist := *result.Distribution
	// Full range should be close to 1.
	fullProb := GetProbabilityInRange(dist, 0, 10000)
	if math.Abs(fullProb-1.0) > 0.01 {
		t.Errorf("full range probability %f, expected ~1.0", fullProb)
	}

	// Empty range should be 0.
	emptyProb := GetProbabilityInRange(dist, 0, 1)
	if emptyProb != 0 {
		t.Errorf("expected 0 for range below all strikes, got %f", emptyProb)
	}
}

func TestGetCumulativeProbability(t *testing.T) {
	result := EstimateImpliedProbabilityDistribution("QQQ", 500, makeCallOptions(), 1000000)
	if !result.Success {
		t.Fatal("failed to estimate PDF")
	}
	dist := *result.Distribution

	cumLow := GetCumulativeProbability(dist, 480)
	cumHigh := GetCumulativeProbability(dist, 520)

	if cumLow > cumHigh {
		t.Error("cumulative probability should be monotonically increasing")
	}
	if math.Abs(cumHigh-1.0) > 0.01 {
		t.Errorf("cumulative at max strike should be ~1.0, got %f", cumHigh)
	}
}

func TestGetQuantile(t *testing.T) {
	result := EstimateImpliedProbabilityDistribution("QQQ", 500, makeCallOptions(), 1000000)
	if !result.Success {
		t.Fatal("failed to estimate PDF")
	}
	dist := *result.Distribution

	q50 := GetQuantile(dist, 0.5)
	if q50 < 490 || q50 > 510 {
		t.Errorf("50th percentile %f outside expected range", q50)
	}

	q0 := GetQuantile(dist, 0)
	if q0 != dist.StrikeProbabilities[0].Strike {
		t.Error("0th percentile should be the lowest strike")
	}
}
