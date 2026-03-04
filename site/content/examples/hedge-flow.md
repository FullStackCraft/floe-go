---
title: Hedge Flow Analysis
description: Combine impulse-curve and charm-integral analytics for one expiration.
order: 5
---

## Compute the Hedge Impulse Curve

```go
curve := hedgeflow.ComputeHedgeImpulseCurve(
  canonicalExposure,
  callSurface,
  hedgeflow.HedgeImpulseConfig{
    RangePercent:       3,
    StepPercent:        0.05,
    KernelWidthStrikes: 2,
  },
  time.Now().UnixMilli(),
)

fmt.Println("Regime:", curve.Regime)
fmt.Println("Impulse at spot:", curve.ImpulseAtSpot)
for _, zc := range curve.ZeroCrossings {
  fmt.Println("Flip level:", zc.Price, zc.Direction)
}
```

## Compute the Charm Integral

```go
charm := hedgeflow.ComputeCharmIntegral(
  canonicalExposure,
  hedgeflow.CharmIntegralConfig{TimeStepMinutes: 15},
  time.Now().UnixMilli(),
)

fmt.Println("Minutes to expiry:", charm.MinutesRemaining)
fmt.Println("Total charm to close:", charm.TotalCharmToClose)
fmt.Println("Direction:", charm.Direction)
```

## Full Combined Analysis

```go
analysis := hedgeflow.AnalyzeHedgeFlow(
  canonicalExposure,
  callSurface,
  hedgeflow.HedgeImpulseConfig{},
  hedgeflow.CharmIntegralConfig{},
  time.Now().UnixMilli(),
)

fmt.Println("Market regime:", analysis.RegimeParams.Regime)
fmt.Println("Impulse regime:", analysis.ImpulseCurve.Regime)
fmt.Println("Charm direction:", analysis.CharmIntegral.Direction)
```
