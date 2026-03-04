package hedgeflow

import floe "github.com/FullStackCraft/floe-go"

// AnalyzeHedgeFlow computes a complete hedge flow analysis for a single
// expiration, combining the hedge impulse curve (conditional: what happens
// if price moves) with the charm integral (unconditional: what happens
// from time passage alone).
func AnalyzeHedgeFlow(
	exposures floe.ExposurePerExpiry,
	ivSurface floe.IVSurface,
	impulseConfig HedgeImpulseConfig,
	charmConfig CharmIntegralConfig,
	computedAt int64,
) HedgeFlowAnalysis {
	regimeParams := DeriveRegimeParams(ivSurface, exposures.SpotPrice)
	impulseCurve := ComputeHedgeImpulseCurve(exposures, ivSurface, impulseConfig, computedAt)
	charmIntegral := ComputeCharmIntegral(exposures, charmConfig, computedAt)

	return HedgeFlowAnalysis{
		ImpulseCurve:  impulseCurve,
		CharmIntegral: charmIntegral,
		RegimeParams:  regimeParams,
	}
}
