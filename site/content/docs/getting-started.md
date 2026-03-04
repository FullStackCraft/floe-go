---
title: Getting Started
description: Install floe and start running options analytics in Go.
order: 1
---

## Installation

Install the module:

```bash
go get github.com/FullStackCraft/floe-go
```

## Quick Start

```go
package main

import (
  "fmt"

  floe "github.com/FullStackCraft/floe-go"
  "github.com/FullStackCraft/floe-go/blackscholes"
)

func main() {
  params := floe.BlackScholesParams{
    Spot:          100,
    Strike:        105,
    TimeToExpiry:  0.25,
    RiskFreeRate:  0.05,
    Volatility:    0.20,
    DividendYield: 0.01,
    OptionType:    floe.Call,
  }

  price := blackscholes.BlackScholes(params)
  greeks := blackscholes.CalculateGreeks(params)

  fmt.Printf("Price: %.2f\\n", price)
  fmt.Printf("Delta: %.5f\\n", greeks.Delta)
  fmt.Printf("Gamma: %.5f\\n", greeks.Gamma)
  fmt.Printf("Theta/day: %.5f\\n", greeks.Theta)
}
```

## Package Layout

`floe` is organized by focused Go packages:

1. `blackscholes` for pricing, Greeks, implied volatility, and time-to-expiry helpers.
2. `volatility` for IV surfaces and smile smoothing.
3. `exposure` for canonical/state-weighted/flow-delta dealer exposures.
4. `hedgeflow` for impulse curve, charm integral, and pressure cloud analysis.
5. `impliedpdf` for risk-neutral distribution estimation.
6. `iv` and `rv` for model-free IV vs realized volatility workflows.
7. `volresponse` for IV response z-score classification.

## Notes

- Time-sensitive APIs take explicit timestamps in milliseconds (`time.Now().UnixMilli()`).
- `blackscholes.CalculateImpliedVolatility` returns percent IV (for example `20` means `20%`).
- Most chain-based functions consume `floe.OptionChain` and `floe.NormalizedOption`.

## Next Steps

- Read the [API Reference](/documentation/api-reference)
- Explore [Recipes](/documentation/recipes)
- Run end-to-end patterns in [Examples](/examples)
