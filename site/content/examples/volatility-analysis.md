---
title: IV vs RV Analysis
description: Compare model-free implied volatility with realized volatility and vol-response z-scores.
order: 6
---

## 1. Compute Model-Free IV

```go
near := iv.ComputeVarianceSwapIV(nearTermOptions, spot, 0.05, nowMs)
fmt.Printf("Near-term IV: %.2f%%\\n", near.ImpliedVolatility*100)
```

## 2. Interpolate to Constant Maturity (Optional)

```go
targetDays := 1
interpolated := iv.ComputeImpliedVolatility(
  nearTermOptions,
  spot,
  0.05,
  nowMs,
  farTermOptions,
  &targetDays,
)

fmt.Printf("Interpolated IV: %.2f%%\\n", interpolated.ImpliedVolatility*100)
```

## 3. Compute Realized Volatility from Ticks

```go
obs := []rv.PriceObservation{
  {Price: 600.10, Timestamp: float64(nowMs - 300000)},
  {Price: 600.25, Timestamp: float64(nowMs - 240000)},
  {Price: 599.80, Timestamp: float64(nowMs - 180000)},
  {Price: 600.50, Timestamp: float64(nowMs - 120000)},
  {Price: 601.10, Timestamp: float64(nowMs - 60000)},
  {Price: 600.90, Timestamp: float64(nowMs)},
}

realized := rv.ComputeRealizedVolatility(obs)
fmt.Printf("RV: %.2f%%\\n", realized.RealizedVolatility*100)
```

## 4. Track the IV-RV Spread

```go
spreadPts := (near.ImpliedVolatility - realized.RealizedVolatility) * 100
fmt.Printf("IV-RV: %.2f pts\\n", spreadPts)
```

## 5. Vol Response Z-Score

```go
series := []volresponse.VolResponseObservation{
  volresponse.BuildVolResponseObservation(0.21, 0.18, 600.4, nowMs-2000, 0.20, 600.0),
  volresponse.BuildVolResponseObservation(0.22, 0.19, 600.9, nowMs-1000, 0.21, 600.4),
  volresponse.BuildVolResponseObservation(0.23, 0.20, 601.2, nowMs, 0.22, 600.9),
}

result := volresponse.ComputeVolResponseZScore(
  series,
  volresponse.VolResponseConfig{MinObservations: 3},
)

fmt.Printf("Signal: %s | Z-score: %.2f\\n", result.Signal, result.ZScore)
```
