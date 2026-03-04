package statistics

import (
	"math"
	"testing"
)

func TestCumulativeNormalDistribution(t *testing.T) {
	tests := []struct {
		name string
		x    float64
		want float64
		tol  float64
	}{
		{"zero", 0.0, 0.5, 1e-4},
		{"positive one", 1.0, 0.8413, 1e-3},
		{"negative one", -1.0, 0.1587, 1e-3},
		{"positive two", 2.0, 0.9772, 1e-3},
		{"negative two", -2.0, 0.0228, 1e-3},
		{"large positive", 5.0, 1.0, 1e-4},
		{"large negative", -5.0, 0.0, 1e-4},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CumulativeNormalDistribution(tt.x)
			if math.Abs(got-tt.want) > tt.tol {
				t.Errorf("CumulativeNormalDistribution(%v) = %v, want %v (tol %v)", tt.x, got, tt.want, tt.tol)
			}
		})
	}
}

func TestNormalPDF(t *testing.T) {
	tests := []struct {
		name string
		x    float64
		want float64
		tol  float64
	}{
		{"zero", 0.0, 1.0 / math.Sqrt(2*math.Pi), 1e-6},
		{"positive one", 1.0, 0.2420, 1e-3},
		{"negative one", -1.0, 0.2420, 1e-3},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := NormalPDF(tt.x)
			if math.Abs(got-tt.want) > tt.tol {
				t.Errorf("NormalPDF(%v) = %v, want %v (tol %v)", tt.x, got, tt.want, tt.tol)
			}
		})
	}

	// NormalPDF should be symmetric
	if NormalPDF(1.5) != NormalPDF(-1.5) {
		t.Error("NormalPDF should be symmetric")
	}
}
