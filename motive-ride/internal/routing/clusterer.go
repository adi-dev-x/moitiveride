package routing

import (
	"math"

	"github.com/motive-ride/motive-ride/internal/domain"
)

// dbscanEpsilonKm is the max distance between two employees to be considered neighbours.
const dbscanEpsilonKm = 2.0

// Clusterer groups employees by geographic proximity using DBSCAN.
// DBSCAN is preferred over K-Means because it discovers the number of clusters
// automatically and handles outliers (lone employees far from others) as singletons.
type Clusterer struct {
	maxVehicleCapacity int
}

func NewClusterer(maxVehicleCapacity int) *Clusterer {
	return &Clusterer{maxVehicleCapacity: maxVehicleCapacity}
}

// Cluster runs DBSCAN then applies merge/split adjustments.
func (c *Clusterer) Cluster(employees []domain.Employee) []Cluster {
	if len(employees) == 0 {
		return nil
	}

	points := toPoints(employees)
	labels := dbscan(points, dbscanEpsilonKm)

	raw := collectClusters(employees, labels)
	adjusted := c.splitLarge(c.mergeSmall(raw))
	return adjusted
}

// dbscan assigns a cluster label to each point.
// Label -1 means noise (treated as a singleton cluster).
func dbscan(points []Point, eps float64) []int {
	n := len(points)
	labels := make([]int, n)
	for i := range labels {
		labels[i] = -1
	}

	clusterID := 0
	visited := make([]bool, n)

	for i := 0; i < n; i++ {
		if visited[i] {
			continue
		}
		visited[i] = true
		neighbours := rangeQuery(points, i, eps)

		// minPoints = 1: every point is a core point
		labels[i] = clusterID
		seeds := make([]int, len(neighbours))
		copy(seeds, neighbours)

		for len(seeds) > 0 {
			q := seeds[0]
			seeds = seeds[1:]

			if !visited[q] {
				visited[q] = true
				qNeighbours := rangeQuery(points, q, eps)
				seeds = append(seeds, qNeighbours...)
			}
			if labels[q] == -1 {
				labels[q] = clusterID
			}
		}
		clusterID++
	}

	return labels
}

func rangeQuery(points []Point, idx int, eps float64) []int {
	var result []int
	for j, p := range points {
		if j != idx && haversineKm(points[idx].Lat, points[idx].Lng, p.Lat, p.Lng) <= eps {
			result = append(result, j)
		}
	}
	return result
}

func collectClusters(employees []domain.Employee, labels []int) []Cluster {
	groupMap := make(map[int][]domain.Employee)
	for i, label := range labels {
		if label == -1 {
			// noise → singleton
			groupMap[-(i + 1)] = []domain.Employee{employees[i]}
		} else {
			groupMap[label] = append(groupMap[label], employees[i])
		}
	}

	clusters := make([]Cluster, 0, len(groupMap))
	for _, emps := range groupMap {
		clusters = append(clusters, Cluster{Employees: emps})
	}
	return clusters
}

// mergeSmall merges clusters smaller than 3 with their nearest compatible neighbour.
func (c *Clusterer) mergeSmall(clusters []Cluster) []Cluster {
	const minSize = 3
	result := make([]Cluster, 0, len(clusters))
	merged := make([]bool, len(clusters))

	for i := 0; i < len(clusters); i++ {
		if merged[i] {
			continue
		}
		if len(clusters[i].Employees) >= minSize {
			result = append(result, clusters[i])
			continue
		}

		// find nearest cluster that won't overflow max capacity when merged
		bestJ := -1
		bestDist := math.MaxFloat64
		for j := i + 1; j < len(clusters); j++ {
			if merged[j] {
				continue
			}
			combined := len(clusters[i].Employees) + len(clusters[j].Employees)
			if combined > c.maxVehicleCapacity {
				continue
			}
			d := clusterDistance(clusters[i], clusters[j])
			if d < bestDist {
				bestDist = d
				bestJ = j
			}
		}

		if bestJ >= 0 {
			merged[bestJ] = true
			merged[i] = true
			result = append(result, Cluster{
				Employees: append(clusters[i].Employees, clusters[bestJ].Employees...),
			})
		} else {
			result = append(result, clusters[i])
		}
	}

	return result
}

// splitLarge recursively splits clusters that exceed maxVehicleCapacity.
func (c *Clusterer) splitLarge(clusters []Cluster) []Cluster {
	var result []Cluster
	for _, cl := range clusters {
		if len(cl.Employees) <= c.maxVehicleCapacity {
			result = append(result, cl)
			continue
		}
		// Split into two halves by geographic median
		halves := splitByMedianLng(cl.Employees)
		result = append(result, c.splitLarge(halves)...)
	}
	return result
}

// splitByMedianLng divides employees into two groups around the median longitude.
func splitByMedianLng(employees []domain.Employee) []Cluster {
	mid := len(employees) / 2
	return []Cluster{
		{Employees: employees[:mid]},
		{Employees: employees[mid:]},
	}
}

// clusterDistance is the centroid-to-centroid distance between two clusters.
func clusterDistance(a, b Cluster) float64 {
	ac := centroid(a.Employees)
	bc := centroid(b.Employees)
	return haversineKm(ac.Lat, ac.Lng, bc.Lat, bc.Lng)
}

func centroid(employees []domain.Employee) Point {
	var lat, lng float64
	for _, e := range employees {
		lat += e.HomeLat
		lng += e.HomeLng
	}
	n := float64(len(employees))
	return Point{Lat: lat / n, Lng: lng / n}
}

func toPoints(employees []domain.Employee) []Point {
	pts := make([]Point, len(employees))
	for i, e := range employees {
		pts[i] = Point{Lat: e.HomeLat, Lng: e.HomeLng, Idx: i}
	}
	return pts
}
