---
title: Recipes
description: Practical Go workflows combining floe packages.
order: 3
---

## Build IV Surfaces and Dealer Exposures

```go
package main

import (
  "fmt"
  "time"

  floe "github.com/FullStackCraft/floe-go"
  "github.com/FullStackCraft/floe-go/exposure"
  "github.com/FullStackCraft/floe-go/volatility"
)

func main() {
  now := time.Now().UnixMilli()

  chain := floe.OptionChain{
    Symbol:        "SPY",
    Spot:          450.5,
    RiskFreeRate:  0.05,
    DividendYield: 0.01,
    Options:       loadOptions(),
  }

  surfaces := volatility.GetIVSurfaces(floe.SmoothingTotalVariance, chain, now)
  variants := exposure.CalculateGammaVannaCharmExposures(
    chain,
    surfaces,
    floe.ExposureCalculationOptions{AsOfTimestamp: now},
  )

  for _, exp := range variants {
    fmt.Printf("Exp %d | Canonical Net %.0f | Max Gamma K %.0f\\n", exp.Expiration, exp.Canonical.TotalNetExposure, exp.Canonical.StrikeOfMaxGamma)
  }
}

func loadOptions() []floe.NormalizedOption {
  return []floe.NormalizedOption{}
}
```

## Use Market Price -> IV -> Greeks

```go
package main

import (
  "fmt"
  "time"

  floe "github.com/FullStackCraft/floe-go"
  "github.com/FullStackCraft/floe-go/blackscholes"
)

func main() {
  now := time.Now().UnixMilli()
  expiry := time.Now().Add(30 * 24 * time.Hour).UnixMilli()

  tte := blackscholes.GetTimeToExpirationInYears(expiry, now)

  ivPercent := blackscholes.CalculateImpliedVolatility(
    2.50,  // market option price
    100.0, // spot
    105.0, // strike
    0.05,
    0.01,
    tte,
    floe.Call,
  )

  greeks := blackscholes.CalculateGreeks(floe.BlackScholesParams{
    Spot:          100,
    Strike:        105,
    TimeToExpiry:  tte,
    RiskFreeRate:  0.05,
    DividendYield: 0.01,
    Volatility:    ivPercent / 100.0,
    OptionType:    floe.Call,
  })

  fmt.Printf("IV: %.2f%%\\n", ivPercent)
  fmt.Printf("Price: %.2f | Delta: %.5f | Vega: %.5f\\n", greeks.Price, greeks.Delta, greeks.Vega)
}
```

## Run Combined Hedge Flow Analysis

```go
analysis := hedgeflow.AnalyzeHedgeFlow(
  canonicalExposure,
  callSurface,
  hedgeflow.HedgeImpulseConfig{},
  hedgeflow.CharmIntegralConfig{},
  time.Now().UnixMilli(),
)

fmt.Println(analysis.ImpulseCurve.Regime)
fmt.Println(analysis.CharmIntegral.Direction)
fmt.Println(analysis.RegimeParams.Regime)
```
