// Package geo has the small amount of spherical geometry the matcher needs.
package geo

import "math"

const earthRadiusM = 6371008.8

// DistanceM returns the great-circle distance in metres between two WGS84 points.
func DistanceM(lat1, lng1, lat2, lng2 float64) float64 {
	φ1, φ2 := rad(lat1), rad(lat2)
	dφ, dλ := rad(lat2-lat1), rad(lng2-lng1)
	a := math.Sin(dφ/2)*math.Sin(dφ/2) + math.Cos(φ1)*math.Cos(φ2)*math.Sin(dλ/2)*math.Sin(dλ/2)
	return 2 * earthRadiusM * math.Asin(math.Min(1, math.Sqrt(a)))
}

// HeadingDiff returns the smallest angle in degrees (0 to 180) between two compass headings.
func HeadingDiff(a, b float64) float64 {
	d := math.Mod(math.Abs(a-b), 360)
	if d > 180 {
		d = 360 - d
	}
	return d
}

func rad(deg float64) float64 { return deg * math.Pi / 180 }
