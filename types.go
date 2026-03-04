// Package floe provides options analytics including pricing, Greeks,
// dealer exposure analysis, hedge flow modeling, and volatility tools.
//
// This is the Go port of the TypeScript @fullstackcraftllc/floe library,
// serving as its server-side complement.
package floe

// OptionType represents call or put.
type OptionType string

const (
	Call OptionType = "call"
	Put  OptionType = "put"
)

// SmoothingModel controls how IV surfaces are smoothed.
type SmoothingModel string

const (
	SmoothingTotalVariance SmoothingModel = "totalvariance"
	SmoothingNone          SmoothingModel = "none"
)

// Time constants.
const (
	MillisecondsPerYear float64 = 31536000000
	MillisecondsPerDay  float64 = 86400000
	MinutesPerYear      float64 = 525600
	MinutesPerDay       float64 = 1440
	DaysPerYear         float64 = 365
)

// BlackScholesParams holds inputs for Black-Scholes pricing.
type BlackScholesParams struct {
	Spot          float64    // Current price of the underlying
	Strike        float64    // Strike price of the option
	TimeToExpiry  float64    // Time to expiration in years
	Volatility    float64    // Implied volatility (annualized, decimal e.g. 0.20)
	RiskFreeRate  float64    // Risk-free interest rate (annualized, decimal)
	OptionType    OptionType // Call or Put
	DividendYield float64    // Dividend yield (annualized, decimal)
}

// Greeks holds the complete set of option Greeks.
type Greeks struct {
	Price  float64 // Theoretical option price
	Delta  float64 // dPrice/dSpot
	Gamma  float64 // d²Price/dSpot²
	Theta  float64 // dPrice/dTime (per day)
	Vega   float64 // dPrice/dVol (per 1% change)
	Rho    float64 // dPrice/dRate (per 1% change)
	Charm  float64 // dDelta/dTime (per day)
	Vanna  float64 // dDelta/dVol
	Volga  float64 // dVega/dVol
	Speed  float64 // dGamma/dSpot
	Zomma  float64 // dGamma/dVol
	Color  float64 // dGamma/dTime
	Ultima float64 // dVolga/dVol
}

// NormalizedTicker is a broker-agnostic ticker quote.
type NormalizedTicker struct {
	Symbol    string  // Underlying symbol
	Spot      float64 // Current spot price
	Bid       float64 // Current bid
	BidSize   float64 // Bid size
	Ask       float64 // Current ask
	AskSize   float64 // Ask size
	Last      float64 // Last traded price
	Volume    float64 // Cumulative volume
	Timestamp int64   // Timestamp in milliseconds
}

// NormalizedOption is a broker-agnostic option quote.
type NormalizedOption struct {
	OCCSymbol            string     // OCC-formatted symbol
	Underlying           string     // Underlying ticker
	Strike               float64    // Strike price
	Expiration           string     // Expiration date (ISO 8601)
	ExpirationTimestamp  int64      // Expiration timestamp in ms
	OptionType           OptionType // Call or Put
	Bid                  float64    // Current bid
	BidSize              float64    // Bid size
	Ask                  float64    // Current ask
	AskSize              float64    // Ask size
	Mark                 float64    // Mid price
	Last                 float64    // Last trade price
	Volume               float64    // Trading volume
	OpenInterest         float64    // Open interest
	LiveOpenInterest     *float64   // Live intraday OI (nil if unavailable)
	ImpliedVolatility    float64    // IV as decimal
	Timestamp            int64      // Quote timestamp in ms
	Greeks               *Greeks    // Pre-calculated Greeks (optional)
}

// OptionChain holds a complete options chain with market context.
type OptionChain struct {
	Symbol        string             // Underlying symbol
	Spot          float64            // Current spot price
	RiskFreeRate  float64            // Risk-free rate (decimal)
	DividendYield float64            // Dividend yield (decimal)
	Options       []NormalizedOption // All options
}

// IVSurface holds an IV surface for one expiration and option type.
type IVSurface struct {
	ExpirationDate int64      // Expiration timestamp in ms
	PutCall        OptionType // Call or Put
	Strikes        []float64  // Sorted strike prices
	RawIVs         []float64  // Raw calculated IVs (as percentages)
	SmoothedIVs    []float64  // Smoothed IVs (as percentages)
}

// StrikeExposure holds exposure metrics at a single strike.
type StrikeExposure struct {
	StrikePrice    float64 // Strike price
	GammaExposure  float64 // GEX at this strike
	VannaExposure  float64 // VEX at this strike
	CharmExposure  float64 // CEX at this strike
	NetExposure    float64 // Sum of GEX + VEX + CEX
}

// ExposureVector holds a single mode's exposure values.
type ExposureVector struct {
	GammaExposure float64
	VannaExposure float64
	CharmExposure float64
	NetExposure   float64
}

// StrikeExposureVariants holds per-strike exposure in all three variants.
type StrikeExposureVariants struct {
	StrikePrice   float64
	Canonical     ExposureVector
	StateWeighted ExposureVector
	FlowDelta     ExposureVector
}

// ExposureModeBreakdown holds the full breakdown for one exposure mode.
type ExposureModeBreakdown struct {
	TotalGammaExposure float64
	TotalVannaExposure float64
	TotalCharmExposure float64
	TotalNetExposure   float64
	StrikeOfMaxGamma   float64
	StrikeOfMinGamma   float64
	StrikeOfMaxVanna   float64
	StrikeOfMinVanna   float64
	StrikeOfMaxCharm   float64
	StrikeOfMinCharm   float64
	StrikeOfMaxNet     float64
	StrikeOfMinNet     float64
	StrikeExposures    []StrikeExposure
}

// ExposureVariantsPerExpiry holds all three exposure variants for one expiration.
type ExposureVariantsPerExpiry struct {
	SpotPrice              float64
	Expiration             int64
	Canonical              ExposureModeBreakdown
	StateWeighted          ExposureModeBreakdown
	FlowDelta              ExposureModeBreakdown
	StrikeExposureVariants []StrikeExposureVariants
}

// ExposurePerExpiry is a flattened view used by hedge flow calculations.
type ExposurePerExpiry struct {
	SpotPrice          float64
	Expiration         int64
	TotalGammaExposure float64
	TotalVannaExposure float64
	TotalCharmExposure float64
	TotalNetExposure   float64
	StrikeOfMaxGamma   float64
	StrikeOfMinGamma   float64
	StrikeOfMaxVanna   float64
	StrikeOfMinVanna   float64
	StrikeOfMaxCharm   float64
	StrikeOfMinCharm   float64
	StrikeOfMaxNet     float64
	StrikeOfMinNet     float64
	StrikeExposures    []StrikeExposure
}

// ExposureCalculationOptions configures exposure calculations.
type ExposureCalculationOptions struct {
	AsOfTimestamp int64 // Reference timestamp in ms (0 = use current time)
}
