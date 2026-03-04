// Package volatility builds implied-volatility surfaces from an option chain
// and provides smoothing via total-variance cubic-spline interpolation with
// convexity enforcement.
package volatility

import (
	"math"
	"sort"

	floe "github.com/FullStackCraft/floe-go"
	"github.com/FullStackCraft/floe-go/blackscholes"
)

// ivFloor is the minimum IV (in percent) we accept when filtering valid points.
const ivFloor = 1.5

// minPointsForSmoothing is the minimum number of valid IV data points required
// before we attempt total-variance smoothing.
const minPointsForSmoothing = 5

// --------------------------------------------------------------------------
// Public API
// --------------------------------------------------------------------------

// GetIVSurfaces builds IV surfaces for every expiration in the option chain.
// It computes raw implied volatility via Black-Scholes, then optionally
// smooths each smile using the requested smoothing model.
//
// asOfTimestamp is the reference time in milliseconds (replaces Date.now()
// from the TypeScript implementation). Only non-expired expirations are
// processed when smoothing is applied.
func GetIVSurfaces(smoothingModel floe.SmoothingModel, chain floe.OptionChain, asOfTimestamp int64) []floe.IVSurface {
	// Group options by (expiration, optionType).
	type groupKey struct {
		expiration int64
		optType    floe.OptionType
	}
	groups := make(map[groupKey][]floe.NormalizedOption)

	for _, opt := range chain.Options {
		key := groupKey{expiration: opt.ExpirationTimestamp, optType: opt.OptionType}
		groups[key] = append(groups[key], opt)
	}

	var surfaces []floe.IVSurface

	for key, opts := range groups {
		// Sort options by strike.
		sort.Slice(opts, func(i, j int) bool {
			return opts[i].Strike < opts[j].Strike
		})

		strikes := make([]float64, len(opts))
		rawIVs := make([]float64, len(opts))

		tte := blackscholes.GetTimeToExpirationInYears(key.expiration, asOfTimestamp)

		for i, opt := range opts {
			strikes[i] = opt.Strike

			iv := blackscholes.CalculateImpliedVolatility(
				opt.Mark,
				chain.Spot,
				opt.Strike,
				chain.RiskFreeRate,
				chain.DividendYield,
				tte,
				opt.OptionType,
			)
			rawIVs[i] = iv
		}

		smoothedIVs := make([]float64, len(rawIVs))
		copy(smoothedIVs, rawIVs)

		// Apply smoothing for non-expired expirations.
		if smoothingModel == floe.SmoothingTotalVariance && key.expiration > asOfTimestamp {
			// Filter to valid IV points (above the floor).
			var validStrikes []float64
			var validIVs []float64
			var validIndices []int

			for i, iv := range rawIVs {
				if iv > ivFloor {
					validStrikes = append(validStrikes, strikes[i])
					validIVs = append(validIVs, rawIVs[i])
					validIndices = append(validIndices, i)
				}
			}

			if len(validStrikes) >= minPointsForSmoothing {
				// Compute T in years from (expiration - asOfTimestamp).
				T := float64(key.expiration-asOfTimestamp) / floe.MillisecondsPerYear

				smoothed := SmoothTotalVarianceSmile(validStrikes, validIVs, T)

				// Copy smoothed values back to the corresponding positions.
				for j, idx := range validIndices {
					smoothedIVs[idx] = smoothed[j]
				}
			}
		}

		surfaces = append(surfaces, floe.IVSurface{
			ExpirationDate: key.expiration,
			PutCall:        key.optType,
			Strikes:        strikes,
			RawIVs:         rawIVs,
			SmoothedIVs:    smoothedIVs,
		})
	}

	// Sort surfaces for deterministic output: by expiration, then by option type.
	sort.Slice(surfaces, func(i, j int) bool {
		if surfaces[i].ExpirationDate != surfaces[j].ExpirationDate {
			return surfaces[i].ExpirationDate < surfaces[j].ExpirationDate
		}
		return surfaces[i].PutCall < surfaces[j].PutCall
	})

	return surfaces
}

// GetIVForStrike looks up the smoothed IV for a specific strike on a given
// expiration and option type. Returns 0 if no matching surface or strike is
// found.
func GetIVForStrike(ivSurfaces []floe.IVSurface, expiration int64, optionType floe.OptionType, strike float64) float64 {
	for _, surface := range ivSurfaces {
		if surface.ExpirationDate != expiration || surface.PutCall != optionType {
			continue
		}
		for i, s := range surface.Strikes {
			if s == strike {
				return surface.SmoothedIVs[i]
			}
		}
		return 0
	}
	return 0
}

// --------------------------------------------------------------------------
// Total-variance smoothing
// --------------------------------------------------------------------------

// SmoothTotalVarianceSmile smooths an IV smile using cubic-spline
// interpolation in total-variance space with convexity enforcement.
//
// strikes and ivs are parallel slices; ivs are in percent (e.g. 20.0 = 20%).
// T is the time to expiration in years.
//
// The procedure:
//  1. Convert IV percentages to decimals, then to total variance w = vol^2 * T.
//  2. Fit a natural cubic spline to w(K).
//  3. Evaluate the spline at the original strikes.
//  4. Enforce convexity via lower convex hull projection.
//  5. Convert back to IV percentages; fall back to raw IV where w <= 0.
func SmoothTotalVarianceSmile(strikes []float64, ivs []float64, T float64) []float64 {
	n := len(strikes)
	if n <= 2 {
		out := make([]float64, n)
		copy(out, ivs)
		return out
	}

	// Convert IV% to total variance.
	w := make([]float64, n)
	for i := 0; i < n; i++ {
		vol := ivs[i] / 100.0 // decimal
		w[i] = vol * vol * T
	}

	// Fit cubic spline on w(K).
	spline := newCubicSpline(strikes, w)

	// Evaluate spline at each original strike.
	wSmooth := make([]float64, n)
	for i := 0; i < n; i++ {
		wSmooth[i] = spline.eval(strikes[i])
	}

	// Enforce convexity.
	wConvex := enforceConvexity(strikes, wSmooth)

	// Convert back to IV%.
	result := make([]float64, n)
	for i := 0; i < n; i++ {
		if wConvex[i] <= 0 {
			result[i] = ivs[i] // fallback to raw
		} else {
			result[i] = math.Sqrt(wConvex[i]/T) * 100.0
		}
	}
	return result
}

// --------------------------------------------------------------------------
// Natural cubic spline
// --------------------------------------------------------------------------

// cubicSpline holds the coefficients for a natural cubic spline.
type cubicSpline struct {
	x    []float64 // knots (sorted)
	a    []float64 // constant coefficients (equal to y values)
	b    []float64 // linear coefficients
	c    []float64 // quadratic coefficients
	d    []float64 // cubic coefficients
	n    int       // number of intervals (len(x) - 1)
}

// newCubicSpline constructs a natural cubic spline through the given (x, y)
// data points. The boundary conditions are S''(x_0) = S''(x_n) = 0.
func newCubicSpline(x, y []float64) *cubicSpline {
	n := len(x) - 1 // number of intervals

	a := make([]float64, n+1)
	copy(a, y)

	h := make([]float64, n)
	for i := 0; i < n; i++ {
		h[i] = x[i+1] - x[i]
	}

	// alpha: right-hand side for the tridiagonal system.
	alpha := make([]float64, n+1)
	for i := 1; i < n; i++ {
		alpha[i] = (3.0/h[i])*(a[i+1]-a[i]) - (3.0/h[i-1])*(a[i]-a[i-1])
	}

	// Solve tridiagonal system with natural boundary conditions.
	l := make([]float64, n+1)
	mu := make([]float64, n+1)
	z := make([]float64, n+1)
	l[0] = 1
	mu[0] = 0
	z[0] = 0

	for i := 1; i < n; i++ {
		l[i] = 2*(x[i+1]-x[i-1]) - h[i-1]*mu[i-1]
		mu[i] = h[i] / l[i]
		z[i] = (alpha[i] - h[i-1]*z[i-1]) / l[i]
	}

	l[n] = 1
	z[n] = 0

	b := make([]float64, n)
	c := make([]float64, n+1)
	d := make([]float64, n)

	c[n] = 0

	// Back substitution.
	for j := n - 1; j >= 0; j-- {
		c[j] = z[j] - mu[j]*c[j+1]
		b[j] = (a[j+1]-a[j])/h[j] - h[j]*(c[j+1]+2*c[j])/3.0
		d[j] = (c[j+1] - c[j]) / (3.0 * h[j])
	}

	return &cubicSpline{
		x: x,
		a: a,
		b: b,
		c: c,
		d: d,
		n: n,
	}
}

// eval evaluates the cubic spline at a given point xv using binary search
// to locate the correct interval.
func (s *cubicSpline) eval(xv float64) float64 {
	// Clamp to the spline domain.
	if xv <= s.x[0] {
		return s.a[0]
	}
	if xv >= s.x[s.n] {
		return s.a[s.n]
	}

	// Binary search for the interval [x[i], x[i+1]] containing xv.
	lo, hi := 0, s.n-1
	for lo <= hi {
		mid := (lo + hi) / 2
		if xv < s.x[mid] {
			hi = mid - 1
		} else if xv >= s.x[mid+1] {
			lo = mid + 1
		} else {
			lo = mid
			break
		}
	}

	i := lo
	dx := xv - s.x[i]
	return s.a[i] + s.b[i]*dx + s.c[i]*dx*dx + s.d[i]*dx*dx*dx
}

// --------------------------------------------------------------------------
// Convexity enforcement via lower convex hull
// --------------------------------------------------------------------------

// point2D is a simple 2-D point used in convex hull computation.
type point2D struct {
	x, w float64
}

// enforceConvexity projects the total-variance curve onto the lower convex
// hull of the (strike, w) points, then linearly interpolates the hull values
// back to the original strike positions.
func enforceConvexity(x, w []float64) []float64 {
	n := len(x)
	if n <= 2 {
		out := make([]float64, n)
		copy(out, w)
		return out
	}

	// Build lower convex hull.
	points := make([]point2D, n)
	for i := 0; i < n; i++ {
		points[i] = point2D{x: x[i], w: w[i]}
	}

	hull := make([]point2D, 0, n)
	for _, p := range points {
		for len(hull) >= 2 {
			h1 := hull[len(hull)-2]
			h2 := hull[len(hull)-1]
			// Cross product check: remove h2 if the turn from h1->h2->p is
			// not strictly left (i.e., the cross product is >= 0 means
			// collinear or right turn, which violates lower hull convexity).
			cross := (h2.x-h1.x)*(p.w-h1.w) - (h2.w-h1.w)*(p.x-h1.x)
			if cross >= 0 {
				// h2 is above the line from h1 to p; remove it.
				hull = hull[:len(hull)-1]
			} else {
				break
			}
		}
		hull = append(hull, p)
	}

	// Interpolate hull values back to original x positions.
	result := make([]float64, n)
	hi := 0
	for i := 0; i < n; i++ {
		xi := x[i]

		// Advance hull index so that hull[hi+1].x >= xi.
		for hi < len(hull)-2 && hull[hi+1].x < xi {
			hi++
		}

		if hi >= len(hull)-1 {
			// At or past last hull point; use its value.
			result[i] = hull[len(hull)-1].w
		} else {
			// Linear interpolation between hull[hi] and hull[hi+1].
			p0 := hull[hi]
			p1 := hull[hi+1]
			dx := p1.x - p0.x
			if dx == 0 {
				result[i] = p0.w
			} else {
				t := (xi - p0.x) / dx
				result[i] = p0.w + t*(p1.w-p0.w)
			}
		}
	}

	return result
}
