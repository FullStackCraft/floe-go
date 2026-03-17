# `floe-go`

[![Go Reference](https://pkg.go.dev/badge/github.com/FullStackCraft/floe-go.svg)](https://pkg.go.dev/github.com/FullStackCraft/floe-go) ![Go Version](https://img.shields.io/badge/Go-1.24.5-blue?style=flat-square&logo=go) ![Repo](https://img.shields.io/badge/GitHub-floe--go-181717?style=flat-square&logo=github)

Zero-dependency Go packages for options analytics: Black-Scholes pricing, Greeks, IV surfaces, dealer exposures, implied PDFs, hedge-flow modeling, and volatility diagnostics.

This is the Go port of `floe`, designed for backend services, analytics pipelines, and trading/fintech systems.

## Quick Start / Documentation / Examples

- Docs + examples site: [fullstackcraft.github.io/floe-go](https://fullstackcraft.github.io/floe-go/)
- API docs: [pkg.go.dev/github.com/FullStackCraft/floe-go](https://pkg.go.dev/github.com/FullStackCraft/floe-go)
- Repository: [github.com/FullStackCraft/floe-go](https://github.com/FullStackCraft/floe-go)

## Features

- **Black-Scholes Pricing** - Pricing + first/second/third-order Greeks
- **Implied Volatility & Surfaces** - IV inversion and smile smoothing
- **Dealer Exposure Metrics** - Canonical, state-weighted, and flow-delta variants
- **Hedge Flow Analysis** - Regime params, impulse curve, charm integral, pressure cloud
- **Implied PDF** - Risk-neutral probability density estimation + exposure adjustments
- **Model-Free IV / RV** - Variance-swap style IV and realized volatility
- **Vol Response Model** - IV response residual z-score classification
- **Zero Dependencies** - Lightweight and easy to embed

## Current Scope

`floe-go` currently focuses on analytics packages. Broker streaming clients/adapters from the TypeScript package are not yet ported in this repository.

## Installation

```bash
go get github.com/FullStackCraft/floe-go
```

## Example

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
		Volatility:    0.20,
		RiskFreeRate:  0.05,
		DividendYield: 0.01,
		OptionType:    floe.Call,
	}

	price := blackscholes.BlackScholes(params)
	greeks := blackscholes.CalculateGreeks(params)

	fmt.Printf("Price: %.2f\n", price)
	fmt.Printf("Delta: %.5f\n", greeks.Delta)
	fmt.Printf("Gamma: %.5f\n", greeks.Gamma)
}
```

## Dataset API Client Example

```go
package main

import (
	"context"
	"fmt"

	"github.com/FullStackCraft/floe-go/apiclient"
)

func main() {
	client := apiclient.NewApiClient("YOUR_API_KEY", nil)
	ctx := context.Background()

	hindsight, err := client.GetHindsightData(ctx, apiclient.HindsightDataRequest{
		StartDate:     "2026-03-01",
		EndDate:       "2026-03-16",
		Country:       "US",
		MinVolatility: 2,
		Event:         "CPI",
	})
	if err != nil {
		panic(err)
	}
	fmt.Printf("hindsight rows: %d\n", len(hindsight))

	rows, err := client.GetDealerMinuteSurfaces(ctx, apiclient.DealerMinuteSurfacesRequest{
		Symbol:    "SPY",
		TradeDate: "2026-03-10",
	})
	if err != nil {
		panic(err)
	}
	fmt.Printf("dealer minute rows: %d\n", len(rows))

	statsRows, err := client.GetAMTSessionStats(ctx, apiclient.AMTRequest{
		Symbol:    "NQ",
		SessionID: "2026-03-10",
	})
	if err != nil {
		panic(err)
	}
	fmt.Printf("amt session rows: %d\n", len(statsRows))

	eventRows, err := client.GetAMTEvents(ctx, apiclient.AMTRequest{
		Symbol:    "NQ",
		SessionID: "2026-03-10",
	})
	if err != nil {
		panic(err)
	}
	fmt.Printf("amt event rows: %d\n", len(eventRows))
}
```

## Package Layout

- `blackscholes` - pricing, Greeks, implied volatility
- `volatility` - IV surfaces and smoothing
- `exposure` - GEX/VEX/CEX variants
- `hedgeflow` - impulse/charm/regime/pressure cloud
- `impliedpdf` - implied distributions and adjusted PDFs
- `iv` - model-free implied volatility
- `rv` - realized volatility
- `volresponse` - IV response z-score model
- `statistics` - normal CDF/PDF helpers

## Notes

- Most time-sensitive APIs take explicit Unix-millisecond timestamps.
- `blackscholes.CalculateImpliedVolatility` returns IV in percent units (for example `20` means `20%`).
- Chain-driven modules use `floe.OptionChain` and `floe.NormalizedOption`.

## TypeScript Version

Looking for the TypeScript package? See [@fullstackcraftllc/floe](https://fullstackcraft.github.io/floe/).

## Licensing

This project follows the same dual-license model as `floe`:

- **MIT License** for personal, educational, and non-commercial use
- **Commercial License** required for business/commercial use

For commercial licensing, contact [hi@fullstackcraft.com](mailto:hi@fullstackcraft.com).

## Contributing

Contributions are welcome. Please open an issue or PR.

## Credits

Copyright © 2026 [Full Stack Craft LLC](https://fullstackcraft.com)
