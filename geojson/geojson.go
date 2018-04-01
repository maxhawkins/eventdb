package geojson

import (
	"encoding/json"
	"math"
)

// DefaultSegments controls the number of segments output in the geometry created
// by CircleGeom.
var DefaultSegments float64 = 20

// EarthRadiusM is the approximate radius of the earth in meters
const EarthRadiusM float64 = 6378137.0

// Haversine computes the distance in meters across the world's surface between two lat/lng coordinates.
func Haversine(lonFrom float64, latFrom float64, lonTo float64, latTo float64) (distanceM float64) {
	var deltaLat = (latTo - latFrom) * (math.Pi / 180)
	var deltaLon = (lonTo - lonFrom) * (math.Pi / 180)

	var a = math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(latFrom*(math.Pi/180))*math.Cos(latTo*(math.Pi/180))*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	var c = 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	distanceM = EarthRadiusM * c

	return
}

// CircleGeom outputs a GeoJSON geometry representing a circle of radius
// radiusM meters centered at (cLat, cLng)
func CircleGeom(cLat, cLng, radiusM float64) string {
	// Based on https://gist.github.com/mashbridge/7331812

	var coords [][]float64

	// Convert to radians
	cLat *= (2.0 * math.Pi) / 360.0
	cLng *= (2.0 * math.Pi) / 360.0

	// Distance along the "true course radial"
	// http://www.edwilliams.org/avform.htm#LL
	d := radiusM / EarthRadiusM

	f := func(p float64) []float64 {
		lat := math.Asin(
			math.Sin(cLat)*math.Cos(d) +
				math.Cos(cLat)*math.Sin(d)*math.Cos(p))

		dLng := math.Atan2(
			math.Sin(p)*math.Sin(d)*math.Cos(cLat),
			math.Cos(d)-math.Sin(cLat)*math.Sin(lat))

		lng := math.Mod(
			cLng-dLng+math.Pi,
			2.0*math.Pi,
		) - math.Pi

		// Convert back to degrees
		lat *= 360.0 / (2.0 * math.Pi)
		lng *= 360.0 / (2.0 * math.Pi)

		return []float64{lng, lat}
	}

	step := (2.0 * math.Pi) / DefaultSegments
	for p := 0.0; p > -2*math.Pi; p -= step {
		coords = append(coords, f(p))
	}
	coords = append(coords, f(0))

	js, _ := json.Marshal(map[string]interface{}{
		"type":        "Polygon",
		"coordinates": [][][]float64{coords},
	})
	return string(js)
}
