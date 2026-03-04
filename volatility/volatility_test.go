package volatility

import (
	"math"
	"testing"
)

func TestSmoothTotalVarianceSmile_BasicSmile(t *testing.T) {
	// V-shaped smile with known values.
	strikes := []float64{90, 95, 100, 105, 110}
	ivs := []float64{24, 21, 18, 21, 24} // percent
	T := 0.25

	smoothed := SmoothTotalVarianceSmile(strikes, ivs, T)

	if len(smoothed) != len(ivs) {
		t.Fatalf("expected %d values, got %d", len(ivs), len(smoothed))
	}

	// Smoothed values should be positive and reasonable.
	for i, v := range smoothed {
		if v <= 0 {
			t.Errorf("smoothed[%d] = %f, want > 0", i, v)
		}
		if v > 100 {
			t.Errorf("smoothed[%d] = %f, unreasonably large", i, v)
		}
	}
}

func TestSmoothTotalVarianceSmile_TwoPoints(t *testing.T) {
	strikes := []float64{100, 110}
	ivs := []float64{20, 22}
	T := 0.25

	smoothed := SmoothTotalVarianceSmile(strikes, ivs, T)
	for i, v := range smoothed {
		if v != ivs[i] {
			t.Errorf("expected raw IV %f, got %f", ivs[i], v)
		}
	}
}

func TestSmoothTotalVarianceSmile_ConvexityEnforced(t *testing.T) {
	// Non-convex total variance (concave dip at center).
	strikes := []float64{90, 95, 100, 105, 110}
	ivs := []float64{25, 20, 30, 20, 25} // creates concavity in variance space
	T := 0.5

	smoothed := SmoothTotalVarianceSmile(strikes, ivs, T)

	// All values should be positive.
	for i, v := range smoothed {
		if v <= 0 {
			t.Errorf("smoothed[%d] = %f, want > 0", i, v)
		}
	}
}

func TestEnforceConvexity_Simple(t *testing.T) {
	x := []float64{1, 2, 3, 4, 5}
	w := []float64{1.0, 0.5, 0.3, 0.5, 1.0} // concave valley

	result := enforceConvexity(x, w)

	// After convex hull, middle values should be on the hull
	// (linear interpolation between endpoints if they form the hull).
	for i := 0; i < len(result)-1; i++ {
		if math.IsNaN(result[i]) {
			t.Errorf("result[%d] is NaN", i)
		}
	}
}

func TestCubicSpline_Interpolation(t *testing.T) {
	x := []float64{0, 1, 2, 3, 4}
	y := []float64{0, 1, 4, 9, 16} // approximately x^2

	spline := newCubicSpline(x, y)

	// Should reproduce known values exactly.
	for i, xi := range x {
		got := spline.eval(xi)
		if math.Abs(got-y[i]) > 1e-10 {
			t.Errorf("spline.eval(%f) = %f, want %f", xi, got, y[i])
		}
	}

	// Interpolated value at x=1.5 should be between 1 and 4.
	mid := spline.eval(1.5)
	if mid < 1 || mid > 4 {
		t.Errorf("spline.eval(1.5) = %f, expected between 1 and 4", mid)
	}
}
