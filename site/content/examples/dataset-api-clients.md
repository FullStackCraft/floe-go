---
title: Dataset API Clients
description: Query hindsight, dealer, and AMT datasets with apiclient.NewApiClient.
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
}
```
