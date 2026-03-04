// Package blackscholes provides Black-Scholes-Merton option pricing,
// complete Greeks calculation (first through third order), and
// implied volatility inversion via bisection method.
package blackscholes

import (
	"math"

	floe "github.com/FullStackCraft/floe-go"
	"github.com/FullStackCraft/floe-go/statistics"
)

// round rounds a float to the given number of decimal places.
func round(value float64, decimals int) float64 {
	factor := math.Pow(10, float64(decimals))
	return math.Round(value*factor) / factor
}

// zeroGreeks returns a Greeks struct with all fields set to zero.
func zeroGreeks() floe.Greeks {
	return floe.Greeks{}
}

// BlackScholes calculates the option price using the Black-Scholes model.
func BlackScholes(params floe.BlackScholesParams) float64 {
	return CalculateGreeks(params).Price
}

// CalculateGreeks computes the complete set of option Greeks using the
// Black-Scholes-Merton model, including all first, second, and third-order Greeks.
func CalculateGreeks(params floe.BlackScholesParams) floe.Greeks {
	S := params.Spot
	K := params.Strike
	t := params.TimeToExpiry
	vol := params.Volatility
	r := params.RiskFreeRate
	q := params.DividendYield

	// Safety checks
	if t < 0 {
		return zeroGreeks()
	}
	if vol <= 0 || S <= 0 || t <= 0 {
		return zeroGreeks()
	}

	// Calculate d1 and d2
	sqrtT := math.Sqrt(t)
	d1 := (math.Log(S/K) + (r-q+vol*vol/2)*t) / (vol * sqrtT)
	d2 := d1 - vol*sqrtT

	// Probability functions
	nd1 := statistics.NormalPDF(d1)
	Nd1 := statistics.CumulativeNormalDistribution(d1)
	Nd2 := statistics.CumulativeNormalDistribution(d2)
	eqt := math.Exp(-q * t)
	ert := math.Exp(-r * t)

	if params.OptionType == floe.Call {
		return calculateCallGreeks(S, K, r, q, t, vol, d1, d2, nd1, Nd1, Nd2, eqt, ert)
	}
	return calculatePutGreeks(S, K, r, q, t, vol, d1, d2, nd1, Nd1, Nd2, eqt, ert)
}

func calculateCallGreeks(S, K, r, q, t, vol, d1, d2, nd1, Nd1, Nd2, eqt, ert float64) floe.Greeks {
	sqrtT := math.Sqrt(t)

	// Price
	price := S*eqt*Nd1 - K*ert*Nd2

	// First-order Greeks
	delta := eqt * Nd1
	gamma := (eqt * nd1) / (S * vol * sqrtT)
	theta := -(S*vol*eqt*nd1)/(2*sqrtT) - r*K*ert*Nd2 + q*S*eqt*Nd1
	vega := S * eqt * sqrtT * nd1
	rho := K * t * ert * Nd2

	// Second-order Greeks
	vanna := -eqt * nd1 * (d2 / vol)
	charm := -q*eqt*Nd1 - (eqt*nd1*(2*(r-q)*t-d2*vol*sqrtT))/(2*t*vol*sqrtT)
	volga := vega * ((d1 * d2) / (S * vol))
	speed := nd1 / (S * vol)
	zomma := (nd1 * d1) / (S * vol * vol)

	// Third-order Greeks
	color := -(d1 * d2 * nd1) / (vol * vol)
	ultima := (d1 * d2 * d2 * nd1) / (vol * vol * vol)

	return floe.Greeks{
		Price:  round(price, 2),
		Delta:  round(delta, 5),
		Gamma:  round(gamma, 5),
		Theta:  round(theta/floe.DaysPerYear, 5),
		Vega:   round(vega*0.01, 5),
		Rho:    round(rho*0.01, 5),
		Charm:  round(charm/floe.DaysPerYear, 5),
		Vanna:  round(vanna, 5),
		Volga:  round(volga, 5),
		Speed:  round(speed, 5),
		Zomma:  round(zomma, 5),
		Color:  round(color, 5),
		Ultima: round(ultima, 5),
	}
}

func calculatePutGreeks(S, K, r, q, t, vol, d1, d2, nd1, Nd1, Nd2, eqt, ert float64) floe.Greeks {
	sqrtT := math.Sqrt(t)

	NmD1 := statistics.CumulativeNormalDistribution(-d1)
	NmD2 := statistics.CumulativeNormalDistribution(-d2)

	// Price
	price := K*ert*NmD2 - S*eqt*NmD1

	// First-order Greeks
	delta := -eqt * NmD1
	gamma := (eqt * nd1) / (S * vol * sqrtT) // Same as call
	theta := -(S*vol*eqt*nd1)/(2*sqrtT) + r*K*ert*NmD2 - q*S*eqt*NmD1
	vega := S * eqt * sqrtT * nd1 // Same as call
	rho := -K * t * ert * NmD2

	// Second-order Greeks
	vanna := -eqt * nd1 * (d2 / vol) // Same as call
	charm := -q*eqt*NmD1 - (eqt*nd1*(2*(r-q)*t-d2*vol*sqrtT))/(2*t*vol*sqrtT)
	volga := vega * ((d1 * d2) / (S * vol)) // Same as call
	speed := (nd1 * d1 * d1) / vol
	zomma := ((1 + d1*d2) * nd1) / (vol * vol * sqrtT)

	// Third-order Greeks
	color := ((1 - d1*d2) * nd1) / S
	ultima := (t * S * nd1 * d1 * d1) / vol

	return floe.Greeks{
		Price:  round(price, 2),
		Delta:  round(delta, 5),
		Gamma:  round(gamma, 5),
		Theta:  round(theta/floe.DaysPerYear, 5),
		Vega:   round(vega*0.01, 5),
		Rho:    round(rho*0.01, 5),
		Charm:  round(charm/floe.DaysPerYear, 5),
		Vanna:  round(vanna, 5),
		Volga:  round(volga, 5),
		Speed:  round(speed, 5),
		Zomma:  round(zomma, 5),
		Color:  round(color, 5),
		Ultima: round(ultima, 5),
	}
}

// CalculateImpliedVolatility recovers implied volatility from an observed
// option price using the bisection method. Returns IV as a percentage
// (e.g. 20.0 for 20%).
func CalculateImpliedVolatility(
	price, spot, strike, riskFreeRate, dividendYield, timeToExpiry float64,
	optionType floe.OptionType,
) float64 {
	if price <= 0 || spot <= 0 || strike <= 0 || timeToExpiry <= 0 {
		return 0
	}

	// Calculate intrinsic value
	var intrinsic float64
	if optionType == floe.Call {
		intrinsic = math.Max(0, spot*math.Exp(-dividendYield*timeToExpiry)-strike*math.Exp(-riskFreeRate*timeToExpiry))
	} else {
		intrinsic = math.Max(0, strike*math.Exp(-riskFreeRate*timeToExpiry)-spot*math.Exp(-dividendYield*timeToExpiry))
	}

	extrinsic := price - intrinsic
	if extrinsic <= 0.01 {
		return 1.0 // 1% IV floor
	}

	// Bisection search
	low := 0.0001
	high := 5.0 // 500% volatility
	var mid float64

	for i := 0; i < 100; i++ {
		mid = 0.5 * (low + high)

		modelPrice := BlackScholes(floe.BlackScholesParams{
			Spot:          spot,
			Strike:        strike,
			TimeToExpiry:  timeToExpiry,
			Volatility:    mid,
			RiskFreeRate:  riskFreeRate,
			DividendYield: dividendYield,
			OptionType:    optionType,
		})

		diff := modelPrice - price
		if math.Abs(diff) < 1e-6 {
			return mid * 100.0
		}

		if diff > 0 {
			high = mid
		} else {
			low = mid
		}
	}

	return 0.5 * (low + high) * 100.0
}

// GetTimeToExpirationInYears converts an expiration timestamp (ms) to years from now.
func GetTimeToExpirationInYears(expirationTimestamp int64, now int64) float64 {
	ms := float64(expirationTimestamp - now)
	return ms / floe.MillisecondsPerYear
}
