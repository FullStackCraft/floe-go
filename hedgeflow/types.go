// Package hedgeflow provides hedge impulse curve analysis, charm integral
// computation, regime derivation, and pressure cloud generation.
package hedgeflow

// MarketRegime classifies the market state from the IV surface.
type MarketRegime string

const (
	RegimeCalm     MarketRegime = "calm"
	RegimeNormal   MarketRegime = "normal"
	RegimeStressed MarketRegime = "stressed"
	RegimeCrisis   MarketRegime = "crisis"
)

// ImpulseRegime classifies the impulse curve shape at spot.
type ImpulseRegime string

const (
	ImpulsePinned      ImpulseRegime = "pinned"
	ImpulseExpansion   ImpulseRegime = "expansion"
	ImpulseSqueezeUp   ImpulseRegime = "squeeze-up"
	ImpulseSqueezeDown ImpulseRegime = "squeeze-down"
	ImpulseNeutral     ImpulseRegime = "neutral"
)

// RegimeParams holds parameters derived from the IV surface.
type RegimeParams struct {
	AtmIV                 float64
	ImpliedSpotVolCorr    float64
	ImpliedVolOfVol       float64
	Regime                MarketRegime
	ExpectedDailySpotMove float64
	ExpectedDailyVolMove  float64
}

// HedgeImpulseConfig configures the hedge impulse curve computation.
type HedgeImpulseConfig struct {
	RangePercent       float64 // default 3
	StepPercent        float64 // default 0.05
	KernelWidthStrikes float64 // default 2
}

// HedgeImpulsePoint is a single point on the hedge impulse curve.
type HedgeImpulsePoint struct {
	Price   float64
	Gamma   float64
	Vanna   float64
	Impulse float64
}

// ZeroCrossing is where the impulse curve crosses zero.
type ZeroCrossing struct {
	Price     float64
	Direction string // "rising" or "falling"
}

// ImpulseExtremum is a local extremum of the impulse curve.
type ImpulseExtremum struct {
	Price   float64
	Impulse float64
	Type    string // "basin" (positive peak = attractor) or "peak" (negative trough = accelerator)
}

// DirectionalAsymmetry measures the asymmetry of impulse around spot.
type DirectionalAsymmetry struct {
	Upside                  float64
	Downside                float64
	IntegrationRangePercent float64
	Bias                    string  // "up", "down", "neutral"
	AsymmetryRatio          float64
}

// HedgeImpulseCurve is the complete impulse curve analysis result.
type HedgeImpulseCurve struct {
	Spot                  float64
	Expiration            int64
	ComputedAt            int64
	SpotVolCoupling       float64
	KernelWidth           float64
	StrikeSpacing         float64
	Curve                 []HedgeImpulsePoint
	ImpulseAtSpot         float64
	SlopeAtSpot           float64
	ZeroCrossings         []ZeroCrossing
	Extrema               []ImpulseExtremum
	Asymmetry             DirectionalAsymmetry
	Regime                ImpulseRegime
	NearestAttractorAbove *float64
	NearestAttractorBelow *float64
}

// CharmIntegralConfig configures the charm integral computation.
type CharmIntegralConfig struct {
	TimeStepMinutes float64 // default 15
}

// CharmBucket is a single time bucket of the charm integral.
type CharmBucket struct {
	MinutesRemaining float64
	InstantaneousCEX float64
	CumulativeCEX    float64
}

// StrikeContribution holds per-strike charm breakdown.
type StrikeContribution struct {
	Strike          float64
	CharmExposure   float64
	FractionOfTotal float64
}

// CharmIntegral is the complete charm integral result.
type CharmIntegral struct {
	Spot                float64
	Expiration          int64
	ComputedAt          int64
	MinutesRemaining    float64
	TotalCharmToClose   float64
	Direction           string // "buying", "selling", "neutral"
	Buckets             []CharmBucket
	StrikeContributions []StrikeContribution
}

// HedgeFlowAnalysis combines impulse curve and charm integral.
type HedgeFlowAnalysis struct {
	ImpulseCurve  HedgeImpulseCurve
	CharmIntegral CharmIntegral
	RegimeParams  RegimeParams
}

// ---- Pressure Cloud types ----

// HedgeContractEstimates holds expected hedge volume in futures contracts.
type HedgeContractEstimates struct {
	NQ  float64
	MNQ float64
	ES  float64
	MES float64
}

// PressureZone is a price zone where dealer hedging creates predictable flow.
type PressureZone struct {
	Center    float64
	Lower     float64
	Upper     float64
	Strength  float64
	Side      string // "above-spot" or "below-spot"
	TradeType string // "long" or "short"
	HedgeType string // "passive" or "aggressive"
}

// RegimeEdge marks a transition between mean-reverting and trend-amplifying behavior.
type RegimeEdge struct {
	Price          float64
	TransitionType string // "stable-to-unstable" or "unstable-to-stable"
}

// PressureLevel holds per-price detail for the pressure cloud.
type PressureLevel struct {
	Price                  float64
	StabilityScore         float64
	AccelerationScore      float64
	ExpectedHedgeContracts float64
	HedgeContracts         HedgeContractEstimates
	HedgeType              string // "passive" or "aggressive"
}

// PressureCloudConfig configures the pressure cloud computation.
type PressureCloudConfig struct {
	ContractMultiplier   float64 // default 20 (NQ)
	ReachabilityMultiple float64 // default 2.0
	ZoneThreshold        float64 // default 0.15
}

// PressureCloud is the complete pressure cloud analysis.
type PressureCloud struct {
	Spot              float64
	Expiration        int64
	ComputedAt        int64
	StabilityZones    []PressureZone
	AccelerationZones []PressureZone
	RegimeEdges       []RegimeEdge
	PriceLevels       []PressureLevel
}
