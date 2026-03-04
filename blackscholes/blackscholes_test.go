package blackscholes

import (
	"math"
	"testing"

	floe "github.com/FullStackCraft/floe-go"
)

func TestBlackScholesCallPrice(t *testing.T) {
	price := BlackScholes(floe.BlackScholesParams{
		Spot:         100,
		Strike:       100,
		TimeToExpiry: 1,
		Volatility:   0.20,
		RiskFreeRate: 0.05,
		OptionType:   floe.Call,
	})

	// Expected ~10.45 from Black-Scholes calculator
	if price <= 10 || price >= 11 {
		t.Errorf("ATM call price = %v, expected between 10 and 11", price)
	}
}

func TestBlackScholesPutPrice(t *testing.T) {
	price := BlackScholes(floe.BlackScholesParams{
		Spot:         100,
		Strike:       100,
		TimeToExpiry: 1,
		Volatility:   0.20,
		RiskFreeRate: 0.05,
		OptionType:   floe.Put,
	})

	// Expected ~5.57
	if price <= 5 || price >= 6 {
		t.Errorf("ATM put price = %v, expected between 5 and 6", price)
	}
}

func TestBlackScholesZeroTime(t *testing.T) {
	price := BlackScholes(floe.BlackScholesParams{
		Spot:         110,
		Strike:       100,
		TimeToExpiry: 0,
		Volatility:   0.20,
		RiskFreeRate: 0.05,
		OptionType:   floe.Call,
	})

	if price != 0 {
		t.Errorf("zero time should return 0, got %v", price)
	}
}

func TestPutCallParity(t *testing.T) {
	S := 100.0
	K := 105.0
	r := 0.05
	q := 0.02
	T := 0.25
	vol := 0.20

	callPrice := BlackScholes(floe.BlackScholesParams{
		Spot: S, Strike: K, TimeToExpiry: T,
		Volatility: vol, RiskFreeRate: r,
		DividendYield: q, OptionType: floe.Call,
	})

	putPrice := BlackScholes(floe.BlackScholesParams{
		Spot: S, Strike: K, TimeToExpiry: T,
		Volatility: vol, RiskFreeRate: r,
		DividendYield: q, OptionType: floe.Put,
	})

	// C - P = S*exp(-qT) - K*exp(-rT)
	lhs := callPrice - putPrice
	rhs := S*math.Exp(-q*T) - K*math.Exp(-r*T)

	if math.Abs(lhs-rhs) > 0.01 {
		t.Errorf("put-call parity violated: C-P=%v, S*e^(-qT)-K*e^(-rT)=%v", lhs, rhs)
	}
}

func TestDeepITMCall(t *testing.T) {
	price := BlackScholes(floe.BlackScholesParams{
		Spot: 150, Strike: 100, TimeToExpiry: 0.25,
		Volatility: 0.20, RiskFreeRate: 0.05,
		OptionType: floe.Call,
	})

	if price <= 49 || price >= 52 {
		t.Errorf("deep ITM call = %v, expected 49-52", price)
	}
}

func TestDeepOTMPut(t *testing.T) {
	price := BlackScholes(floe.BlackScholesParams{
		Spot: 150, Strike: 100, TimeToExpiry: 0.25,
		Volatility: 0.20, RiskFreeRate: 0.05,
		OptionType: floe.Put,
	})

	if price >= 0.01 {
		t.Errorf("deep OTM put = %v, expected < 0.01", price)
	}
}

func TestCalculateGreeksCallDelta(t *testing.T) {
	greeks := CalculateGreeks(floe.BlackScholesParams{
		Spot: 100, Strike: 100, TimeToExpiry: 0.25,
		Volatility: 0.20, RiskFreeRate: 0.05,
		OptionType: floe.Call,
	})

	if greeks.Delta <= 0.5 || greeks.Delta >= 0.65 {
		t.Errorf("ATM call delta = %v, expected 0.5-0.65", greeks.Delta)
	}
}

func TestCalculateGreeksPutDelta(t *testing.T) {
	greeks := CalculateGreeks(floe.BlackScholesParams{
		Spot: 100, Strike: 100, TimeToExpiry: 0.25,
		Volatility: 0.20, RiskFreeRate: 0.05,
		OptionType: floe.Put,
	})

	if greeks.Delta >= 0 || greeks.Delta <= -0.5 {
		t.Errorf("ATM put delta = %v, expected between -0.5 and 0", greeks.Delta)
	}
}

func TestCallPutGammaEqual(t *testing.T) {
	params := floe.BlackScholesParams{
		Spot: 100, Strike: 100, TimeToExpiry: 0.25,
		Volatility: 0.20, RiskFreeRate: 0.05,
	}

	params.OptionType = floe.Call
	callGreeks := CalculateGreeks(params)

	params.OptionType = floe.Put
	putGreeks := CalculateGreeks(params)

	if math.Abs(callGreeks.Gamma-putGreeks.Gamma) > 0.0001 {
		t.Errorf("call gamma %v != put gamma %v", callGreeks.Gamma, putGreeks.Gamma)
	}
}

func TestCallPutVegaEqual(t *testing.T) {
	params := floe.BlackScholesParams{
		Spot: 100, Strike: 100, TimeToExpiry: 0.25,
		Volatility: 0.20, RiskFreeRate: 0.05,
	}

	params.OptionType = floe.Call
	callGreeks := CalculateGreeks(params)

	params.OptionType = floe.Put
	putGreeks := CalculateGreeks(params)

	if math.Abs(callGreeks.Vega-putGreeks.Vega) > 0.0001 {
		t.Errorf("call vega %v != put vega %v", callGreeks.Vega, putGreeks.Vega)
	}
}

func TestNegativeTheta(t *testing.T) {
	greeks := CalculateGreeks(floe.BlackScholesParams{
		Spot: 100, Strike: 100, TimeToExpiry: 0.25,
		Volatility: 0.20, RiskFreeRate: 0.05,
		OptionType: floe.Call,
	})

	if greeks.Theta >= 0 {
		t.Errorf("theta = %v, expected negative", greeks.Theta)
	}
}

func TestPositiveVega(t *testing.T) {
	greeks := CalculateGreeks(floe.BlackScholesParams{
		Spot: 100, Strike: 100, TimeToExpiry: 0.25,
		Volatility: 0.20, RiskFreeRate: 0.05,
		OptionType: floe.Call,
	})

	if greeks.Vega <= 0 {
		t.Errorf("vega = %v, expected positive", greeks.Vega)
	}
}

func TestCallPositiveRho(t *testing.T) {
	greeks := CalculateGreeks(floe.BlackScholesParams{
		Spot: 100, Strike: 100, TimeToExpiry: 0.25,
		Volatility: 0.20, RiskFreeRate: 0.05,
		OptionType: floe.Call,
	})

	if greeks.Rho <= 0 {
		t.Errorf("call rho = %v, expected positive", greeks.Rho)
	}
}

func TestPutNegativeRho(t *testing.T) {
	greeks := CalculateGreeks(floe.BlackScholesParams{
		Spot: 100, Strike: 100, TimeToExpiry: 0.25,
		Volatility: 0.20, RiskFreeRate: 0.05,
		OptionType: floe.Put,
	})

	if greeks.Rho >= 0 {
		t.Errorf("put rho = %v, expected negative", greeks.Rho)
	}
}

func TestInvalidParamsReturnZero(t *testing.T) {
	greeks := CalculateGreeks(floe.BlackScholesParams{
		Spot: 100, Strike: 100, TimeToExpiry: 0.25,
		Volatility: 0, // Invalid
		RiskFreeRate: 0.05, OptionType: floe.Call,
	})

	if greeks.Delta != 0 || greeks.Gamma != 0 {
		t.Error("invalid params should return zero greeks")
	}
}

func TestSecondOrderGreeksExist(t *testing.T) {
	greeks := CalculateGreeks(floe.BlackScholesParams{
		Spot: 100, Strike: 100, TimeToExpiry: 0.25,
		Volatility: 0.20, RiskFreeRate: 0.05,
		OptionType: floe.Call,
	})

	// Vanna and charm should be non-zero for ATM options
	if greeks.Vanna == 0 {
		t.Error("vanna should be non-zero")
	}
	if greeks.Charm == 0 {
		t.Error("charm should be non-zero")
	}
}

func TestImpliedVolatilityRecovery(t *testing.T) {
	knownVol := 0.25
	S := 100.0
	K := 105.0
	r := 0.05
	q := 0.02
	T := 0.25

	price := BlackScholes(floe.BlackScholesParams{
		Spot: S, Strike: K, TimeToExpiry: T,
		Volatility: knownVol, RiskFreeRate: r,
		DividendYield: q, OptionType: floe.Call,
	})

	iv := CalculateImpliedVolatility(price, S, K, r, q, T, floe.Call)

	// IV returned as percentage
	if math.Abs(iv-knownVol*100) > 0.5 {
		t.Errorf("recovered IV = %v%%, expected ~%v%%", iv, knownVol*100)
	}
}

func TestImpliedVolatilityPut(t *testing.T) {
	knownVol := 0.30
	S := 100.0
	K := 95.0
	r := 0.05
	q := 0.01
	T := 0.5

	price := BlackScholes(floe.BlackScholesParams{
		Spot: S, Strike: K, TimeToExpiry: T,
		Volatility: knownVol, RiskFreeRate: r,
		DividendYield: q, OptionType: floe.Put,
	})

	iv := CalculateImpliedVolatility(price, S, K, r, q, T, floe.Put)

	if math.Abs(iv-knownVol*100) > 0.5 {
		t.Errorf("recovered put IV = %v%%, expected ~%v%%", iv, knownVol*100)
	}
}

func TestImpliedVolatilityZeroPrice(t *testing.T) {
	iv := CalculateImpliedVolatility(0, 100, 105, 0.05, 0, 0.25, floe.Call)
	if iv != 0 {
		t.Errorf("zero price should return 0 IV, got %v", iv)
	}
}

func TestImpliedVolatilityBelowIntrinsic(t *testing.T) {
	iv := CalculateImpliedVolatility(0.50, 110, 100, 0.05, 0, 0.25, floe.Call)
	if iv != 1.0 {
		t.Errorf("below-intrinsic price should return 1%% IV floor, got %v", iv)
	}
}

func TestImpliedVolatilityHighVol(t *testing.T) {
	knownVol := 0.80
	S := 100.0
	K := 100.0
	r := 0.05
	q := 0.0
	T := 1.0

	price := BlackScholes(floe.BlackScholesParams{
		Spot: S, Strike: K, TimeToExpiry: T,
		Volatility: knownVol, RiskFreeRate: r,
		DividendYield: q, OptionType: floe.Call,
	})

	iv := CalculateImpliedVolatility(price, S, K, r, q, T, floe.Call)

	if math.Abs(iv-knownVol*100) > 1.0 {
		t.Errorf("high vol recovery: IV = %v%%, expected ~%v%%", iv, knownVol*100)
	}
}
