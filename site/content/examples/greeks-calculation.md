---
title: Greeks Calculation
description: Calculate full Greeks output for risk and sensitivity analysis.
order: 2
---

## Full Greeks Profile

```go
package main

import (
  "fmt"

  floe "github.com/FullStackCraft/floe-go"
  "github.com/FullStackCraft/floe-go/blackscholes"
)

func main() {
  g := blackscholes.CalculateGreeks(floe.BlackScholesParams{
    Spot:          100,
    Strike:        100,
    TimeToExpiry:  0.25,
    RiskFreeRate:  0.05,
    Volatility:    0.20,
    DividendYield: 0.01,
    OptionType:    floe.Call,
  })

  fmt.Printf("Price: %.2f\\n", g.Price)
  fmt.Printf("Delta: %.5f | Gamma: %.5f | Theta/day: %.5f\\n", g.Delta, g.Gamma, g.Theta)
  fmt.Printf("Vega: %.5f | Rho: %.5f\\n", g.Vega, g.Rho)
  fmt.Printf("Vanna: %.5f | Charm: %.5f | Volga: %.5f\\n", g.Vanna, g.Charm, g.Volga)
  fmt.Printf("Speed: %.5f | Zomma: %.5f | Color: %.5f | Ultima: %.5f\\n", g.Speed, g.Zomma, g.Color, g.Ultima)
}
```
