package geo

import (
	"math"
	"testing"
)

func TestDistanceM(t *testing.T) {
	tests := []struct {
		name                   string
		lat1, lng1, lat2, lng2 float64
		want, tolerance        float64
	}{
		{"same point", 52.0894, 5.1101, 52.0894, 5.1101, 0, 0.001},
		// Utrecht Centraal to Amsterdam Centraal, about 35.2 km as the crow flies.
		{"Utrecht to Amsterdam", 52.0894, 5.1101, 52.3789, 4.9003, 35220, 100},
		{"one degree of latitude", 52, 5, 53, 5, 111195, 50},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DistanceM(tt.lat1, tt.lng1, tt.lat2, tt.lng2)
			if math.Abs(got-tt.want) > tt.tolerance {
				t.Errorf("DistanceM = %.1f, want %.1f ± %.1f", got, tt.want, tt.tolerance)
			}
		})
	}
}

func TestHeadingDiff(t *testing.T) {
	tests := []struct{ a, b, want float64 }{
		{0, 0, 0},
		{10, 350, 20},
		{350, 10, 20},
		{90, 270, 180},
		{0, 720, 0},
		{45, 100, 55},
	}
	for _, tt := range tests {
		if got := HeadingDiff(tt.a, tt.b); math.Abs(got-tt.want) > 1e-9 {
			t.Errorf("HeadingDiff(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
		}
	}
}
