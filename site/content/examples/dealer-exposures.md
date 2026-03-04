---
title: Dealer Exposures
description: Compute canonical, state-weighted, and flow-delta exposure vectors.
order: 3
---

## Build Chain, Surfaces, and Exposure Variants

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
  expiry := time.Now().Add(14 * 24 * time.Hour).UnixMilli()

  options := []floe.NormalizedOption{
    {Strike: 440, ExpirationTimestamp: expiry, OptionType: floe.Call, Bid: 13.0, Ask: 13.4, Mark: 13.2, OpenInterest: 21000, ImpliedVolatility: 0.19},
    {Strike: 440, ExpirationTimestamp: expiry, OptionType: floe.Put, Bid: 2.7, Ask: 3.1, Mark: 2.9, OpenInterest: 19000, ImpliedVolatility: 0.20},
    {Strike: 445, ExpirationTimestamp: expiry, OptionType: floe.Call, Bid: 9.0, Ask: 9.2, Mark: 9.1, OpenInterest: 26000, ImpliedVolatility: 0.18},
    {Strike: 445, ExpirationTimestamp: expiry, OptionType: floe.Put, Bid: 5.0, Ask: 5.2, Mark: 5.1, OpenInterest: 24000, ImpliedVolatility: 0.19},
    {Strike: 450, ExpirationTimestamp: expiry, OptionType: floe.Call, Bid: 5.8, Ask: 6.0, Mark: 5.9, OpenInterest: 33000, ImpliedVolatility: 0.17},
    {Strike: 450, ExpirationTimestamp: expiry, OptionType: floe.Put, Bid: 8.2, Ask: 8.4, Mark: 8.3, OpenInterest: 30000, ImpliedVolatility: 0.18},
  }

  chain := floe.OptionChain{
    Symbol:        "SPY",
    Spot:          447.5,
    RiskFreeRate:  0.05,
    DividendYield: 0.01,
    Options:       options,
  }

  surfaces := volatility.GetIVSurfaces(floe.SmoothingTotalVariance, chain, now)
  variants := exposure.CalculateGammaVannaCharmExposures(
    chain,
    surfaces,
    floe.ExposureCalculationOptions{AsOfTimestamp: now},
  )

  for _, exp := range variants {
    fmt.Printf("\nExpiry %d\\n", exp.Expiration)
    fmt.Printf("Canonical Net: %.0f\\n", exp.Canonical.TotalNetExposure)
    fmt.Printf("State-Weighted Net: %.0f\\n", exp.StateWeighted.TotalNetExposure)
    fmt.Printf("Flow-Delta Net: %.0f\\n", exp.FlowDelta.TotalNetExposure)
    fmt.Printf("Max Gamma Strike: %.0f\\n", exp.Canonical.StrikeOfMaxGamma)
  }
}
```

## Estimate Shares Needed to Rebalance

```go
cover := exposure.CalculateSharesNeededToCover(900_000_000, -4_000_000, 447.5)
fmt.Println(cover.ActionToCover)
fmt.Println(cover.SharesToCover)
fmt.Println(cover.ImpliedMoveToCover)
```
