---
title: Dataset API Clients
description: Query hindsight, dealer, AMT, and options screener datasets with apiclient.NewApiClient.
order: 7
---

## Create the client

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

  hindsightEvents, err := client.GetHindsightData(ctx, apiclient.HindsightDataRequest{
    StartDate:     "2026-03-01",
    EndDate:       "2026-03-16",
    Country:       "US",
    MinVolatility: 2,
    Event:         "CPI",
  })
  if err != nil {
    panic(err)
  }
  fmt.Printf("hindsight rows: %d\n", len(hindsightEvents))

  sample, err := client.GetHindsightSample(ctx)
  if err != nil {
    panic(err)
  }
  fmt.Printf("sample rows: %d\n", len(sample))

  minuteRows, err := client.GetDealerMinuteSurfaces(ctx, apiclient.DealerMinuteSurfacesRequest{
    Symbol:    "SPY",
    TradeDate: "2026-03-10",
  })
  if err != nil {
    panic(err)
  }
  fmt.Printf("dealer minute rows: %d\n", len(minuteRows))

  sessionRows, err := client.GetAMTSessionStats(ctx, apiclient.AMTRequest{
    Symbol:    "NQ",
    SessionID: "2026-03-10",
  })
  if err != nil {
    panic(err)
  }
  fmt.Printf("amt session rows: %d\n", len(sessionRows))

  eventRows, err := client.GetAMTEvents(ctx, apiclient.AMTRequest{
    Symbol:    "NQ",
    SessionID: "2026-03-10",
  })
  if err != nil {
    panic(err)
  }
  fmt.Printf("amt event rows: %d\n", len(eventRows))

  // Wheel Screener
  wheelData, err := client.GetWheelScreenerData(ctx, apiclient.OptionsScreenerRequest{
    Strategy: "CC",
    ExtraParams: map[string]string{
      "page_size": "10",
      "min_score": "70",
      "sector":    "Technology",
    },
  })
  if err != nil {
    panic(err)
  }
  fmt.Printf("wheel screener rows: %d, total: %d\n", len(wheelData.Data), wheelData.Total)

  // LEAPS Screener
  leapsData, err := client.GetLeapsScreenerData(ctx, apiclient.OptionsScreenerRequest{
    Strategy: "LC",
    ExtraParams: map[string]string{
      "min_dte":   "180",
      "max_delta": "0.7",
    },
  })
  if err != nil {
    panic(err)
  }
  fmt.Printf("leaps screener rows: %d, total: %d\n", len(leapsData.Data), leapsData.Total)

  // Option Screener
  optionData, err := client.GetOptionScreenerData(ctx, apiclient.OptionsScreenerRequest{
    Strategy: "CDS",
    ExtraParams: map[string]string{
      "search":    "AAPL",
      "page_size": "25",
    },
  })
  if err != nil {
    panic(err)
  }
  fmt.Printf("option screener rows: %d, total: %d\n", len(optionData.Data), optionData.Total)
}
```
