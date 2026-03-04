---
title: Implied Probability Distribution
description: Estimate risk-neutral strike distributions from call option prices.
order: 4
---

## Estimate a Single-Expiry Distribution

```go
package main

import (
  "fmt"
  "time"

  floe "github.com/FullStackCraft/floe-go"
  "github.com/FullStackCraft/floe-go/impliedpdf"
)

func main() {
  expiry := time.Now().Add(7 * 24 * time.Hour).UnixMilli()

  calls := []floe.NormalizedOption{
    {Strike: 490, ExpirationTimestamp: expiry, OptionType: floe.Call, Bid: 15.2, Ask: 15.5},
    {Strike: 495, ExpirationTimestamp: expiry, OptionType: floe.Call, Bid: 11.4, Ask: 11.7},
    {Strike: 500, ExpirationTimestamp: expiry, OptionType: floe.Call, Bid: 8.1, Ask: 8.4},
    {Strike: 505, ExpirationTimestamp: expiry, OptionType: floe.Call, Bid: 5.3, Ask: 5.6},
    {Strike: 510, ExpirationTimestamp: expiry, OptionType: floe.Call, Bid: 3.1, Ask: 3.4},
  }

  result := impliedpdf.EstimateImpliedProbabilityDistribution(
    "QQQ",
    502.5,
    calls,
    time.Now().UnixMilli(),
  )
  if !result.Success {
    panic(result.Error)
  }

  dist := *result.Distribution
  fmt.Printf("Mode: %.0f\\n", dist.MostLikelyPrice)
  fmt.Printf("Median: %.0f\\n", dist.MedianPrice)
  fmt.Printf("Expected Move: %.2f\\n", dist.ExpectedMove)
  fmt.Printf("Tail Skew: %.3f\\n", dist.TailSkew)
}
```

## Query Range and Quantiles

```go
pRange := impliedpdf.GetProbabilityInRange(dist, 495, 510)
pBelow := impliedpdf.GetCumulativeProbability(dist, 495)
p90 := impliedpdf.GetQuantile(dist, 0.90)

fmt.Printf("P(495-510): %.2f%%\\n", pRange*100)
fmt.Printf("P(<=495): %.2f%%\\n", pBelow*100)
fmt.Printf("90th percentile: %.0f\\n", p90)
```
