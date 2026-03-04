---
title: Black-Scholes Pricing
description: Price calls and puts in Go with Black-Scholes-Merton.
order: 1
---

## Basic Pricing

```go
package main

import (
  "fmt"

  floe "github.com/FullStackCraft/floe-go"
  "github.com/FullStackCraft/floe-go/blackscholes"
)

func main() {
  common := floe.BlackScholesParams{
    Spot:         100,
    Strike:       105,
    TimeToExpiry: 0.25,
    RiskFreeRate: 0.05,
    Volatility:   0.20,
  }

  call := blackscholes.BlackScholes(floe.BlackScholesParams{
    Spot:         common.Spot,
    Strike:       common.Strike,
    TimeToExpiry: common.TimeToExpiry,
    RiskFreeRate: common.RiskFreeRate,
    Volatility:   common.Volatility,
    OptionType:   floe.Call,
  })

  put := blackscholes.BlackScholes(floe.BlackScholesParams{
    Spot:         common.Spot,
    Strike:       common.Strike,
    TimeToExpiry: common.TimeToExpiry,
    RiskFreeRate: common.RiskFreeRate,
    Volatility:   common.Volatility,
    OptionType:   floe.Put,
  })

  fmt.Printf("Call: %.2f\\n", call)
  fmt.Printf("Put: %.2f\\n", put)
}
```

## With Dividend Yield

```go
price := blackscholes.BlackScholes(floe.BlackScholesParams{
  Spot:          100,
  Strike:        100,
  TimeToExpiry:  0.50,
  RiskFreeRate:  0.05,
  Volatility:    0.25,
  DividendYield: 0.02,
  OptionType:    floe.Call,
})
```
