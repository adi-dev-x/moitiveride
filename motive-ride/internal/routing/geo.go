package routing

import "math"

// haversineKm computes the great-circle distance in kilometres between two lat/lng points.
func haversineKm(lat1, lng1, lat2, lng2 float64) float64 {
	const earthRadiusKm = 6371.0

	dLat := toRad(lat2 - lat1)
	dLng := toRad(lng2 - lng1)

	a := math.Sin(dLat/2)*math.Sin(dLat/2) +
		math.Cos(toRad(lat1))*math.Cos(toRad(lat2))*
			math.Sin(dLng/2)*math.Sin(dLng/2)

	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return earthRadiusKm * c
}

func toRad(deg float64) float64 {
	return deg * math.Pi / 180
}

// Point is a 2-D geographic coordinate.
type Point struct {
	Lat float64
	Lng float64
	Idx int // original index in the employee slice
}

// distanceMatrix builds an N×N pairwise Haversine distance matrix.
func distanceMatrix(points []Point) [][]float64 {
	n := len(points)
	m := make([][]float64, n)
	for i := range m {
		m[i] = make([]float64, n)
		for j := range m[i] {
			if i != j {
				m[i][j] = haversineKm(points[i].Lat, points[i].Lng, points[j].Lat, points[j].Lng)
			}
		}
	}
	return m
}
