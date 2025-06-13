package tools

import "fmt"

func PointToWKT(lat, lon float64) string {
	return fmt.Sprintf("SRID=4326;POINT(%f %f)", lon, lat)
}
