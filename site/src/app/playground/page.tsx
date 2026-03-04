"use client";

import Link from "next/link";
import { useState } from "react";

const EXAMPLES = {
  "black-scholes": {
    title: "Black-Scholes Pricing",
    description: "Price calls and puts with the Go Black-Scholes package.",
    code: `package main

import (
  "fmt"

  floe "github.com/FullStackCraft/floe-go"
  "github.com/FullStackCraft/floe-go/blackscholes"
)

func main() {
  params := floe.BlackScholesParams{
    Spot:         100,
    Strike:       105,
    TimeToExpiry: 0.25,
    RiskFreeRate: 0.05,
    Volatility:   0.20,
    OptionType:   floe.Call,
  }

  callPrice := blackscholes.BlackScholes(params)
  putPrice := blackscholes.BlackScholes(floe.BlackScholesParams{
    Spot:         params.Spot,
    Strike:       params.Strike,
    TimeToExpiry: params.TimeToExpiry,
    RiskFreeRate: params.RiskFreeRate,
    Volatility:   params.Volatility,
    OptionType:   floe.Put,
  })

  fmt.Printf("Call: %.2f\\n", callPrice)
  fmt.Printf("Put: %.2f\\n", putPrice)
}`,
  },
  greeks: {
    title: "Greeks",
    description: "Compute first, second, and third order Greeks.",
    code: `package main

import (
  "fmt"

  floe "github.com/FullStackCraft/floe-go"
  "github.com/FullStackCraft/floe-go/blackscholes"
)

func main() {
  greeks := blackscholes.CalculateGreeks(floe.BlackScholesParams{
    Spot:          100,
    Strike:        105,
    TimeToExpiry:  0.25,
    RiskFreeRate:  0.05,
    Volatility:    0.20,
    DividendYield: 0.01,
    OptionType:    floe.Call,
  })

  fmt.Printf("Price: %.2f\\n", greeks.Price)
  fmt.Printf("Delta: %.5f\\n", greeks.Delta)
  fmt.Printf("Gamma: %.5f\\n", greeks.Gamma)
  fmt.Printf("Theta/day: %.5f\\n", greeks.Theta)
  fmt.Printf("Vanna: %.5f\\n", greeks.Vanna)
  fmt.Printf("Charm/day: %.5f\\n", greeks.Charm)
}`,
  },
  "iv-surfaces": {
    title: "IV Surfaces",
    description: "Build smoothed IV surfaces for calls and puts.",
    code: `package main

import (
  "fmt"
  "time"

  floe "github.com/FullStackCraft/floe-go"
  "github.com/FullStackCraft/floe-go/volatility"
)

func main() {
  now := time.Now().UnixMilli()
  expiry := time.Now().Add(30 * 24 * time.Hour).UnixMilli()

  options := []floe.NormalizedOption{
    {Strike: 95, ExpirationTimestamp: expiry, OptionType: floe.Call, Bid: 7.1, Ask: 7.3, Mark: 7.2, ImpliedVolatility: 0.22},
    {Strike: 100, ExpirationTimestamp: expiry, OptionType: floe.Call, Bid: 4.8, Ask: 5.0, Mark: 4.9, ImpliedVolatility: 0.20},
    {Strike: 105, ExpirationTimestamp: expiry, OptionType: floe.Call, Bid: 3.0, Ask: 3.2, Mark: 3.1, ImpliedVolatility: 0.19},
    {Strike: 95, ExpirationTimestamp: expiry, OptionType: floe.Put, Bid: 1.0, Ask: 1.2, Mark: 1.1, ImpliedVolatility: 0.24},
    {Strike: 100, ExpirationTimestamp: expiry, OptionType: floe.Put, Bid: 2.7, Ask: 2.9, Mark: 2.8, ImpliedVolatility: 0.21},
    {Strike: 105, ExpirationTimestamp: expiry, OptionType: floe.Put, Bid: 5.6, Ask: 5.8, Mark: 5.7, ImpliedVolatility: 0.20},
  }

  chain := floe.OptionChain{
    Symbol:        "SPY",
    Spot:          100,
    RiskFreeRate:  0.05,
    DividendYield: 0.01,
    Options:       options,
  }

  surfaces := volatility.GetIVSurfaces(floe.SmoothingTotalVariance, chain, now)
  fmt.Printf("Built %d surfaces\\n", len(surfaces))
}`,
  },
  "dealer-exposures": {
    title: "Dealer Exposures",
    description: "Calculate canonical, state-weighted, and flow-delta exposures.",
    code: `package main

import (
  "fmt"
  "time"

  floe "github.com/FullStackCraft/floe-go"
  "github.com/FullStackCraft/floe-go/exposure"
  "github.com/FullStackCraft/floe-go/volatility"
)

func main() {
  now := time.Now().UnixMilli()
  expiry := time.Now().Add(7 * 24 * time.Hour).UnixMilli()

  options := []floe.NormalizedOption{
    {Strike: 440, ExpirationTimestamp: expiry, OptionType: floe.Call, Bid: 13.1, Ask: 13.3, Mark: 13.2, OpenInterest: 21000, ImpliedVolatility: 0.19},
    {Strike: 440, ExpirationTimestamp: expiry, OptionType: floe.Put, Bid: 2.8, Ask: 3.0, Mark: 2.9, OpenInterest: 18000, ImpliedVolatility: 0.20},
    {Strike: 445, ExpirationTimestamp: expiry, OptionType: floe.Call, Bid: 9.0, Ask: 9.2, Mark: 9.1, OpenInterest: 28000, ImpliedVolatility: 0.18},
    {Strike: 445, ExpirationTimestamp: expiry, OptionType: floe.Put, Bid: 4.9, Ask: 5.1, Mark: 5.0, OpenInterest: 25000, ImpliedVolatility: 0.19},
    {Strike: 450, ExpirationTimestamp: expiry, OptionType: floe.Call, Bid: 5.7, Ask: 5.9, Mark: 5.8, OpenInterest: 36000, ImpliedVolatility: 0.17},
    {Strike: 450, ExpirationTimestamp: expiry, OptionType: floe.Put, Bid: 8.3, Ask: 8.5, Mark: 8.4, OpenInterest: 34000, ImpliedVolatility: 0.18},
  }

  chain := floe.OptionChain{Symbol: "SPY", Spot: 447.5, RiskFreeRate: 0.05, DividendYield: 0.01, Options: options}
  surfaces := volatility.GetIVSurfaces(floe.SmoothingTotalVariance, chain, now)
  variants := exposure.CalculateGammaVannaCharmExposures(chain, surfaces, floe.ExposureCalculationOptions{AsOfTimestamp: now})

  for _, exp := range variants {
    fmt.Printf("Expiry %d | Canonical Net: %.0f | StateWeighted Net: %.0f\\n", exp.Expiration, exp.Canonical.TotalNetExposure, exp.StateWeighted.TotalNetExposure)
  }
}`,
  },
  "implied-pdf": {
    title: "Implied PDF",
    description: "Estimate risk-neutral distribution and pull range probabilities.",
    code: `package main

import (
  "fmt"
  "time"

  floe "github.com/FullStackCraft/floe-go"
  "github.com/FullStackCraft/floe-go/impliedpdf"
)

func main() {
  expiry := time.Now().Add(14 * 24 * time.Hour).UnixMilli()
  calls := []floe.NormalizedOption{
    {Strike: 490, ExpirationTimestamp: expiry, OptionType: floe.Call, Bid: 15.2, Ask: 15.5},
    {Strike: 495, ExpirationTimestamp: expiry, OptionType: floe.Call, Bid: 11.4, Ask: 11.7},
    {Strike: 500, ExpirationTimestamp: expiry, OptionType: floe.Call, Bid: 8.1, Ask: 8.4},
    {Strike: 505, ExpirationTimestamp: expiry, OptionType: floe.Call, Bid: 5.3, Ask: 5.6},
    {Strike: 510, ExpirationTimestamp: expiry, OptionType: floe.Call, Bid: 3.1, Ask: 3.4},
  }

  result := impliedpdf.EstimateImpliedProbabilityDistribution("QQQ", 502.5, calls, time.Now().UnixMilli())
  if !result.Success {
    panic(result.Error)
  }

  dist := *result.Distribution
  rangeProb := impliedpdf.GetProbabilityInRange(dist, 495, 510)
  p90 := impliedpdf.GetQuantile(dist, 0.90)

  fmt.Printf("Mode: %.0f\\n", dist.MostLikelyPrice)
  fmt.Printf("Expected Move: %.2f\\n", dist.ExpectedMove)
  fmt.Printf("P(495-510): %.2f%%\\n", rangeProb*100)
  fmt.Printf("90th percentile: %.0f\\n", p90)
}`,
  },
  "iv-vs-rv": {
    title: "IV vs RV",
    description: "Compare model-free implied vol with realized vol and vol-response z-score.",
    code: `package main

import (
  "fmt"
  "time"

  floe "github.com/FullStackCraft/floe-go"
  "github.com/FullStackCraft/floe-go/iv"
  "github.com/FullStackCraft/floe-go/rv"
  "github.com/FullStackCraft/floe-go/volresponse"
)

func main() {
  now := time.Now().UnixMilli()
  expiry := time.Now().Add(6 * time.Hour).UnixMilli()

  options := []floe.NormalizedOption{
    {Strike: 595, ExpirationTimestamp: expiry, OptionType: floe.Call, Bid: 5.2, Ask: 5.4},
    {Strike: 595, ExpirationTimestamp: expiry, OptionType: floe.Put, Bid: 0.5, Ask: 0.7},
    {Strike: 600, ExpirationTimestamp: expiry, OptionType: floe.Call, Bid: 2.1, Ask: 2.3},
    {Strike: 600, ExpirationTimestamp: expiry, OptionType: floe.Put, Bid: 2.0, Ask: 2.2},
    {Strike: 605, ExpirationTimestamp: expiry, OptionType: floe.Call, Bid: 0.7, Ask: 0.9},
    {Strike: 605, ExpirationTimestamp: expiry, OptionType: floe.Put, Bid: 5.3, Ask: 5.5},
  }

  ivResult := iv.ComputeVarianceSwapIV(options, 600, 0.05, now)

  ticks := []rv.PriceObservation{
    {Price: 600.0, Timestamp: float64(now - 300000)},
    {Price: 600.6, Timestamp: float64(now - 240000)},
    {Price: 599.8, Timestamp: float64(now - 180000)},
    {Price: 600.9, Timestamp: float64(now - 120000)},
    {Price: 601.2, Timestamp: float64(now - 60000)},
    {Price: 600.7, Timestamp: float64(now)},
  }
  rvResult := rv.ComputeRealizedVolatility(ticks)

  obs := make([]volresponse.VolResponseObservation, 0)
  obs = append(obs, volresponse.BuildVolResponseObservation(0.22, 0.19, 600.8, now-2000, 0.20, 600.1))
  obs = append(obs, volresponse.BuildVolResponseObservation(0.23, 0.20, 601.1, now-1000, 0.22, 600.8))
  obs = append(obs, volresponse.BuildVolResponseObservation(0.24, 0.21, 601.4, now, 0.23, 601.1))

  // For production use, feed 30+ observations.
  z := volresponse.ComputeVolResponseZScore(obs, volresponse.VolResponseConfig{MinObservations: 3})

  fmt.Printf("IV: %.2f%% | RV: %.2f%% | Spread: %.2f pts\\n", ivResult.ImpliedVolatility*100, rvResult.RealizedVolatility*100, (ivResult.ImpliedVolatility-rvResult.RealizedVolatility)*100)
  fmt.Printf("Vol response signal: %s (z=%.2f)\\n", z.Signal, z.ZScore)
}`,
  },
} as const;

type ExampleKey = keyof typeof EXAMPLES;

export default function PlaygroundPage() {
  const [activeExample, setActiveExample] = useState<ExampleKey>("black-scholes");
  const [copied, setCopied] = useState(false);

  const example = EXAMPLES[activeExample];

  const handleCopy = async () => {
    await navigator.clipboard.writeText(example.code);
    setCopied(true);
    setTimeout(() => setCopied(false), 1500);
  };

  return (
    <main className="min-h-screen bg-gray-50">
      <header className="bg-white border-b border-gray-200">
        <div className="max-w-7xl mx-auto px-4 py-4 flex items-center justify-between">
          <div className="flex items-center gap-6">
            <Link href="/" className="font-mono text-2xl font-bold text-[#00ADD8]">
              floe
            </Link>
            <span className="text-gray-300">|</span>
            <h1 className="text-lg font-medium">Playground</h1>
          </div>
          <nav className="flex gap-4">
            <Link href="/documentation" className="text-gray-600 hover:text-black transition-colors">
              Docs
            </Link>
            <Link href="/examples" className="text-gray-600 hover:text-black transition-colors">
              Examples
            </Link>
          </nav>
        </div>
      </header>

      <div className="max-w-7xl mx-auto px-4 py-8">
        <div className="mb-6">
          <div className="flex flex-wrap gap-2">
            {(Object.keys(EXAMPLES) as ExampleKey[]).map((key) => (
              <button
                key={key}
                onClick={() => setActiveExample(key)}
                className={`px-4 py-2 rounded-lg font-medium transition-colors ${
                  activeExample === key
                    ? "bg-[#00ADD8] text-white"
                    : "bg-white border border-gray-200 text-gray-700 hover:border-gray-400"
                }`}
              >
                {EXAMPLES[key].title}
              </button>
            ))}
          </div>
          <p className="mt-3 text-gray-600">{example.description}</p>
        </div>

        <div className="rounded-lg overflow-hidden border border-gray-200 shadow-sm bg-[#0b1720]">
          <div className="flex items-center justify-between px-4 py-3 border-b border-[#1d3645] bg-[#122331]">
            <span className="font-mono text-xs text-[#9ecde0]">main.go</span>
            <button
              onClick={handleCopy}
              className="text-xs px-2 py-1 rounded bg-[#00ADD8] text-white hover:bg-[#0087AB] transition-colors cursor-pointer"
            >
              {copied ? "Copied" : "Copy"}
            </button>
          </div>
          <pre className="p-5 overflow-x-auto text-sm leading-6 text-[#d7e7ef]">
            <code>{example.code}</code>
          </pre>
        </div>

        <div className="mt-8 bg-white rounded-lg border border-gray-200 p-6">
          <h2 className="font-mono text-lg font-semibold mb-3">Tips</h2>
          <ul className="text-gray-600 space-y-2 text-sm">
            <li>• These snippets are designed for server-side Go workflows</li>
            <li>
              • Import from <code className="bg-gray-100 px-1 rounded">github.com/FullStackCraft/floe-go</code> and package submodules
            </li>
            <li>
              • Check the <Link href="/documentation" className="text-[#00ADD8] hover:underline">documentation</Link> for package-level API details
            </li>
            <li>• Pass explicit timestamps (for example <code className="bg-gray-100 px-1 rounded">time.Now().UnixMilli()</code>) where required</li>
          </ul>
        </div>
      </div>
    </main>
  );
}
