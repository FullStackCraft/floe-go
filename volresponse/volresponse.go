// Package volresponse computes a vol response z-score that detects unusual
// implied volatility behavior relative to spot/RV dynamics using OLS
// regression with ridge regularization.
package volresponse

import (
	"math"
)

const (
	numFeatures = 5
	ridgeLambda = 1e-8
)

// VolResponseObservation is a single observation for the regression model.
type VolResponseObservation struct {
	Timestamp     int64
	DeltaIV       float64
	SpotReturn    float64
	AbsSpotReturn float64
	RVLevel       float64
	IVLevel       float64
}

// VolResponseCoefficients are the fitted regression coefficients.
type VolResponseCoefficients struct {
	Intercept     float64
	BetaReturn    float64
	BetaAbsReturn float64
	BetaRV        float64
	BetaIVLevel   float64
}

// VolResponseConfig configures the vol response model.
type VolResponseConfig struct {
	MinObservations     int
	VolBidThreshold     float64
	VolOfferedThreshold float64
}

// VolResponseResult is the complete output of the vol response model.
type VolResponseResult struct {
	IsValid         bool
	MinObservations int
	NumObservations int
	Coefficients    VolResponseCoefficients
	RSquared        float64
	ResidualStdDev  float64
	ExpectedDeltaIV float64
	ObservedDeltaIV float64
	Residual        float64
	ZScore          float64
	Signal          string // "vol_bid", "vol_offered", "neutral", "insufficient_data"
	Timestamp       int64
}

// BuildVolResponseObservation creates an observation from consecutive readings.
func BuildVolResponseObservation(
	currentIV, currentRV, currentSpot float64,
	currentTimestamp int64,
	previousIV, previousSpot float64,
) VolResponseObservation {
	deltaIV := currentIV - previousIV
	spotReturn := math.Log(currentSpot / previousSpot)

	return VolResponseObservation{
		Timestamp:     currentTimestamp,
		DeltaIV:       deltaIV,
		SpotReturn:    spotReturn,
		AbsSpotReturn: math.Abs(spotReturn),
		RVLevel:       currentRV,
		IVLevel:       currentIV,
	}
}

// ComputeVolResponseZScore fits an expanding-window OLS regression and
// computes the z-score of the most recent IV residual.
//
//	deltaIV(t) ~ a + b1*return + b2*|return| + b3*RV + b4*IV_level
//
// z >> 0 means vol is bid relative to baseline (stress / demand).
// z << 0 means vol is offered relative to baseline (supply / crush).
func ComputeVolResponseZScore(
	observations []VolResponseObservation,
	config VolResponseConfig,
) VolResponseResult {
	if config.MinObservations == 0 {
		config.MinObservations = 30
	}
	if config.VolBidThreshold == 0 {
		config.VolBidThreshold = 1.5
	}
	if config.VolOfferedThreshold == 0 {
		config.VolOfferedThreshold = -1.5
	}

	empty := VolResponseCoefficients{}
	n := len(observations)

	if n < config.MinObservations {
		var ts int64
		var obsIV float64
		if n > 0 {
			ts = observations[n-1].Timestamp
			obsIV = observations[n-1].DeltaIV
		}
		return VolResponseResult{
			IsValid:         false,
			MinObservations: config.MinObservations,
			NumObservations: n,
			Coefficients:    empty,
			ObservedDeltaIV: obsIV,
			Signal:          "insufficient_data",
			Timestamp:       ts,
		}
	}

	// Build design matrix X and response y.
	X := make([][]float64, n)
	y := make([]float64, n)
	for i, obs := range observations {
		X[i] = []float64{1, obs.SpotReturn, obs.AbsSpotReturn, obs.RVLevel, obs.IVLevel}
		y[i] = obs.DeltaIV
	}

	ols := solveOLS(X, y)
	if ols == nil {
		return VolResponseResult{
			IsValid:         false,
			MinObservations: config.MinObservations,
			NumObservations: n,
			Coefficients:    empty,
			ObservedDeltaIV: observations[n-1].DeltaIV,
			Signal:          "insufficient_data",
			Timestamp:       observations[n-1].Timestamp,
		}
	}

	lastObs := observations[n-1]
	lastX := X[n-1]

	expectedDeltaIV := 0.0
	for j := 0; j < numFeatures; j++ {
		expectedDeltaIV += ols.beta[j] * lastX[j]
	}

	residual := lastObs.DeltaIV - expectedDeltaIV
	zScore := 0.0
	if ols.residualStdDev > 0 {
		zScore = residual / ols.residualStdDev
	}

	signal := "neutral"
	if zScore > config.VolBidThreshold {
		signal = "vol_bid"
	} else if zScore < config.VolOfferedThreshold {
		signal = "vol_offered"
	}

	return VolResponseResult{
		IsValid:         true,
		MinObservations: config.MinObservations,
		NumObservations: n,
		Coefficients: VolResponseCoefficients{
			Intercept:     ols.beta[0],
			BetaReturn:    ols.beta[1],
			BetaAbsReturn: ols.beta[2],
			BetaRV:        ols.beta[3],
			BetaIVLevel:   ols.beta[4],
		},
		RSquared:        ols.rSquared,
		ResidualStdDev:  ols.residualStdDev,
		ExpectedDeltaIV: expectedDeltaIV,
		ObservedDeltaIV: lastObs.DeltaIV,
		Residual:        residual,
		ZScore:          zScore,
		Signal:          signal,
		Timestamp:       lastObs.Timestamp,
	}
}

// ---- OLS solver via normal equations with ridge regularization ----

type olsResult struct {
	beta           []float64
	residuals      []float64
	rSquared       float64
	residualStdDev float64
}

func solveOLS(X [][]float64, y []float64) *olsResult {
	n := len(X)
	p := numFeatures

	// X'X (p x p)
	XtX := make([][]float64, p)
	for i := 0; i < p; i++ {
		XtX[i] = make([]float64, p)
		for j := 0; j < p; j++ {
			s := 0.0
			for k := 0; k < n; k++ {
				s += X[k][i] * X[k][j]
			}
			XtX[i][j] = s
		}
	}

	// Ridge penalty (skip intercept).
	for i := 1; i < p; i++ {
		XtX[i][i] += ridgeLambda
	}

	// X'y (p x 1)
	Xty := make([]float64, p)
	for i := 0; i < p; i++ {
		s := 0.0
		for k := 0; k < n; k++ {
			s += X[k][i] * y[k]
		}
		Xty[i] = s
	}

	// Augmented matrix [XtX | Xty]
	aug := make([][]float64, p)
	for i := 0; i < p; i++ {
		aug[i] = make([]float64, p+1)
		copy(aug[i], XtX[i])
		aug[i][p] = Xty[i]
	}

	// Gauss-Jordan elimination with partial pivoting.
	for col := 0; col < p; col++ {
		maxVal := math.Abs(aug[col][col])
		maxRow := col
		for row := col + 1; row < p; row++ {
			v := math.Abs(aug[row][col])
			if v > maxVal {
				maxVal = v
				maxRow = row
			}
		}
		if maxVal < 1e-14 {
			return nil
		}
		if maxRow != col {
			aug[col], aug[maxRow] = aug[maxRow], aug[col]
		}

		pivot := aug[col][col]
		for j := col; j <= p; j++ {
			aug[col][j] /= pivot
		}
		for row := 0; row < p; row++ {
			if row == col {
				continue
			}
			factor := aug[row][col]
			for j := col; j <= p; j++ {
				aug[row][j] -= factor * aug[col][j]
			}
		}
	}

	beta := make([]float64, p)
	for i := 0; i < p; i++ {
		beta[i] = aug[i][p]
		if math.IsInf(beta[i], 0) || math.IsNaN(beta[i]) {
			return nil
		}
	}

	// Compute residuals and stats.
	residuals := make([]float64, n)
	yMean := 0.0
	for i := 0; i < n; i++ {
		yMean += y[i]
	}
	yMean /= float64(n)

	ssRes := 0.0
	ssTot := 0.0
	for i := 0; i < n; i++ {
		predicted := 0.0
		for j := 0; j < p; j++ {
			predicted += beta[j] * X[i][j]
		}
		residuals[i] = y[i] - predicted
		ssRes += residuals[i] * residuals[i]
		ssTot += (y[i] - yMean) * (y[i] - yMean)
	}

	rSquared := 0.0
	if ssTot > 0 {
		rSquared = math.Max(0, 1-ssRes/ssTot)
	}

	dof := math.Max(float64(n-p), 1)
	residualStdDev := math.Sqrt(ssRes / dof)

	return &olsResult{
		beta:           beta,
		residuals:      residuals,
		rSquared:       rSquared,
		residualStdDev: residualStdDev,
	}
}
