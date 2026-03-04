package iv

import (
	"testing"

	floe "github.com/FullStackCraft/floe-go"
)

func makeOption(strike float64, optType floe.OptionType, bid, ask float64, expiration int64) floe.NormalizedOption {
	return floe.NormalizedOption{
		Strike:              strike,
		OptionType:          optType,
		Bid:                 bid,
		Ask:                 ask,
		Mark:                (bid + ask) / 2,
		ExpirationTimestamp: expiration,
	}
}

func TestComputeVarianceSwapIV_EmptyOptions(t *testing.T) {
	result := ComputeVarianceSwapIV(nil, 500, 0.05, 1000000)
	if result.ImpliedVolatility != 0 {
		t.Errorf("expected 0 IV for empty options, got %f", result.ImpliedVolatility)
	}
}

func TestComputeVarianceSwapIV_BasicChain(t *testing.T) {
	exp := int64(1000000 + 30*86400*1000) // 30 days out
	asOf := int64(1000000)

	options := []floe.NormalizedOption{
		makeOption(490, floe.Put, 1.0, 1.2, exp),
		makeOption(495, floe.Put, 2.0, 2.4, exp),
		makeOption(500, floe.Call, 5.0, 5.5, exp),
		makeOption(500, floe.Put, 4.8, 5.3, exp),
		makeOption(505, floe.Call, 2.0, 2.4, exp),
		makeOption(510, floe.Call, 1.0, 1.2, exp),
	}

	result := ComputeVarianceSwapIV(options, 500, 0.05, asOf)

	if result.ImpliedVolatility <= 0 {
		t.Errorf("expected positive IV, got %f", result.ImpliedVolatility)
	}
	if result.Forward <= 0 {
		t.Errorf("expected positive forward, got %f", result.Forward)
	}
	if result.NumStrikes == 0 {
		t.Error("expected some contributing strikes")
	}
	if result.TimeToExpiry <= 0 {
		t.Errorf("expected positive T, got %f", result.TimeToExpiry)
	}
}

func TestComputeImpliedVolatility_SingleTerm(t *testing.T) {
	exp := int64(1000000 + 30*86400*1000)
	asOf := int64(1000000)

	options := []floe.NormalizedOption{
		makeOption(490, floe.Put, 1.0, 1.2, exp),
		makeOption(500, floe.Call, 5.0, 5.5, exp),
		makeOption(500, floe.Put, 4.8, 5.3, exp),
		makeOption(510, floe.Call, 1.0, 1.2, exp),
	}

	result := ComputeImpliedVolatility(options, 500, 0.05, asOf, nil, nil)

	if result.IsInterpolated {
		t.Error("expected single-term, not interpolated")
	}
	if result.ImpliedVolatility <= 0 {
		t.Errorf("expected positive IV, got %f", result.ImpliedVolatility)
	}
	if result.FarTerm != nil {
		t.Error("expected nil far term for single-term")
	}
}

func TestComputeImpliedVolatility_TwoTerm(t *testing.T) {
	asOf := int64(1000000)
	exp1 := int64(1000000 + 20*86400*1000) // 20 days
	exp2 := int64(1000000 + 50*86400*1000) // 50 days

	near := []floe.NormalizedOption{
		makeOption(490, floe.Put, 1.0, 1.2, exp1),
		makeOption(500, floe.Call, 4.0, 4.5, exp1),
		makeOption(500, floe.Put, 3.8, 4.3, exp1),
		makeOption(510, floe.Call, 0.8, 1.0, exp1),
	}
	far := []floe.NormalizedOption{
		makeOption(490, floe.Put, 2.0, 2.4, exp2),
		makeOption(500, floe.Call, 7.0, 7.5, exp2),
		makeOption(500, floe.Put, 6.8, 7.3, exp2),
		makeOption(510, floe.Call, 2.0, 2.4, exp2),
	}

	targetDays := 30
	result := ComputeImpliedVolatility(near, 500, 0.05, asOf, far, &targetDays)

	if !result.IsInterpolated {
		t.Error("expected interpolated result")
	}
	if result.FarTerm == nil {
		t.Error("expected non-nil far term")
	}
	if result.ImpliedVolatility <= 0 {
		t.Errorf("expected positive interpolated IV, got %f", result.ImpliedVolatility)
	}
}
