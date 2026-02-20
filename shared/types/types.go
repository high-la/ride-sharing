package types

type Coordinate struct {
	Latitude  float64 `json:"latitude"`
	Longitude float64 `json:"longitude"`
}

type Route struct {
	Distance float64  `json:"distance"`
	Duration float64  `json:"duration"`
	Geometry Geometry `json:"geometry"`
}

type Geometry struct {
	Coordinates [][]float64 `json:"coordinates"`
	Type        string      `json:"type"`
}
