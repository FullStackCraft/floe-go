---
title: API Reference
description: Core Go packages and function signatures in floe.
order: 2
---

## Pricing (`blackscholes`)

### `BlackScholes`

```go
price := blackscholes.BlackScholes(floe.BlackScholesParams{
  Spot: 100, Strike: 105, TimeToExpiry: 0.25,
  RiskFreeRate: 0.05, Volatility: 0.20,
  OptionType: floe.Call,
})
```

### `CalculateGreeks`

```go
g := blackscholes.CalculateGreeks(params)
fmt.Println(g.Price, g.Delta, g.Gamma, g.Theta, g.Vega, g.Rho)
fmt.Println(g.Vanna, g.Charm, g.Volga, g.Speed, g.Zomma, g.Color, g.Ultima)
```

### `CalculateImpliedVolatility`

```go
ivPercent := blackscholes.CalculateImpliedVolatility(
  3.50,   // option price
  100,    // spot
  105,    // strike
  0.05,   // risk-free rate
  0.01,   // dividend yield
  0.25,   // time to expiry (years)
  floe.Call,
)
```

### `GetTimeToExpirationInYears`

```go
tte := blackscholes.GetTimeToExpirationInYears(expirationMs, time.Now().UnixMilli())
```

## Statistics (`statistics`)

```go
cdf := statistics.CumulativeNormalDistribution(1.96)
pdf := statistics.NormalPDF(0)
```

## Volatility Surfaces (`volatility`)

### `GetIVSurfaces`

```go
surfaces := volatility.GetIVSurfaces(
  floe.SmoothingTotalVariance,
  chain,
  time.Now().UnixMilli(),
)
```

### `GetIVForStrike`

```go
ivAtK := volatility.GetIVForStrike(surfaces, expiryMs, floe.Call, 450)
```

### `SmoothTotalVarianceSmile`

```go
smoothed := volatility.SmoothTotalVarianceSmile(
  []float64{440, 445, 450, 455, 460},
  []float64{23, 21, 19, 20, 22},
  0.08,
)
```

## Dealer Exposure (`exposure`)

### `CalculateGammaVannaCharmExposures`

```go
variants := exposure.CalculateGammaVannaCharmExposures(
  chain,
  surfaces,
  floe.ExposureCalculationOptions{AsOfTimestamp: time.Now().UnixMilli()},
)

for _, v := range variants {
  fmt.Println(v.Expiration, v.Canonical.TotalNetExposure)
}
```

### `CalculateSharesNeededToCover`

```go
cover := exposure.CalculateSharesNeededToCover(900_000_000, totalNetExposure, spot)
fmt.Println(cover.ActionToCover, cover.SharesToCover, cover.ImpliedMoveToCover)
```

## Hedge Flow (`hedgeflow`)

### `ComputeHedgeImpulseCurve`

```go
curve := hedgeflow.ComputeHedgeImpulseCurve(
  canonicalExposure,
  callSurface,
  hedgeflow.HedgeImpulseConfig{RangePercent: 3, StepPercent: 0.05, KernelWidthStrikes: 2},
  time.Now().UnixMilli(),
)
```

### `ComputeCharmIntegral`

```go
charm := hedgeflow.ComputeCharmIntegral(
  canonicalExposure,
  hedgeflow.CharmIntegralConfig{TimeStepMinutes: 15},
  time.Now().UnixMilli(),
)
```

### `AnalyzeHedgeFlow`

```go
analysis := hedgeflow.AnalyzeHedgeFlow(
  canonicalExposure,
  callSurface,
  hedgeflow.HedgeImpulseConfig{},
  hedgeflow.CharmIntegralConfig{},
  time.Now().UnixMilli(),
)
```

## Implied Probability (`impliedpdf`)

### `EstimateImpliedProbabilityDistribution`

```go
result := impliedpdf.EstimateImpliedProbabilityDistribution(
  "QQQ",
  502.5,
  callOptions,
  time.Now().UnixMilli(),
)
```

### `EstimateImpliedProbabilityDistributions`

```go
dists := impliedpdf.EstimateImpliedProbabilityDistributions(
  "QQQ",
  502.5,
  allOptions,
  time.Now().UnixMilli(),
)
```

### Query helpers

```go
prob := impliedpdf.GetProbabilityInRange(dist, 495, 510)
cum := impliedpdf.GetCumulativeProbability(dist, 500)
q90 := impliedpdf.GetQuantile(dist, 0.90)
```

## IV vs RV (`iv`, `rv`, `volresponse`)

### Model-free IV

```go
near := iv.ComputeVarianceSwapIV(nearTermOptions, spot, 0.05, time.Now().UnixMilli())
targetDays := 1
interp := iv.ComputeImpliedVolatility(nearTermOptions, spot, 0.05, time.Now().UnixMilli(), farTermOptions, &targetDays)
```

### Realized volatility

```go
rvResult := rv.ComputeRealizedVolatility(observations)
```

### Vol response z-score

```go
obs := volresponse.BuildVolResponseObservation(currentIV, currentRV, currentSpot, nowMs, prevIV, prevSpot)
result := volresponse.ComputeVolResponseZScore(series, volresponse.VolResponseConfig{})
```
