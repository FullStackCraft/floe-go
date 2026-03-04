package hedgeflow

import (
	"math"
	"sort"

	floe "github.com/FullStackCraft/floe-go"
)

// ComputeCharmIntegral computes the cumulative expected delta change from
// time decay (charm exposure) from now until expiration.
func ComputeCharmIntegral(
	exposures floe.ExposurePerExpiry,
	config CharmIntegralConfig,
	computedAt int64,
) CharmIntegral {
	if config.TimeStepMinutes == 0 {
		config.TimeStepMinutes = 15
	}

	spot := exposures.SpotPrice
	expiration := exposures.Expiration
	msRemaining := float64(expiration - computedAt)
	minutesRemaining := math.Max(0, msRemaining/60000.0)

	// Per-strike charm breakdown.
	var totalAbsCharm float64
	for _, s := range exposures.StrikeExposures {
		totalAbsCharm += math.Abs(s.CharmExposure)
	}

	var contributions []StrikeContribution
	for _, s := range exposures.StrikeExposures {
		if s.CharmExposure == 0 {
			continue
		}
		frac := 0.0
		if totalAbsCharm > 0 {
			frac = math.Abs(s.CharmExposure) / totalAbsCharm
		}
		contributions = append(contributions, StrikeContribution{
			Strike:          s.StrikePrice,
			CharmExposure:   s.CharmExposure,
			FractionOfTotal: frac,
		})
	}
	sort.Slice(contributions, func(i, j int) bool {
		return math.Abs(contributions[i].CharmExposure) > math.Abs(contributions[j].CharmExposure)
	})

	if minutesRemaining <= 0 {
		return CharmIntegral{
			Spot:                spot,
			Expiration:          expiration,
			ComputedAt:          computedAt,
			MinutesRemaining:    0,
			TotalCharmToClose:   0,
			Direction:           "neutral",
			Buckets:             nil,
			StrikeContributions: contributions,
		}
	}

	totalCEX := exposures.TotalCharmExposure
	var buckets []CharmBucket
	cumulativeCEX := 0.0
	step := config.TimeStepMinutes

	for t := minutesRemaining; t >= math.Max(1, step); t -= step {
		timeScaling := math.Sqrt(minutesRemaining / t)
		instantCEX := totalCEX * timeScaling
		bucketFraction := step / 390.0
		bucketContrib := instantCEX * bucketFraction
		cumulativeCEX += bucketContrib

		buckets = append(buckets, CharmBucket{
			MinutesRemaining: t,
			InstantaneousCEX: instantCEX,
			CumulativeCEX:    cumulativeCEX,
		})
	}

	direction := "neutral"
	if cumulativeCEX > 0 {
		direction = "buying"
	} else if cumulativeCEX < 0 {
		direction = "selling"
	}

	return CharmIntegral{
		Spot:                spot,
		Expiration:          expiration,
		ComputedAt:          computedAt,
		MinutesRemaining:    minutesRemaining,
		TotalCharmToClose:   cumulativeCEX,
		Direction:           direction,
		Buckets:             buckets,
		StrikeContributions: contributions,
	}
}
