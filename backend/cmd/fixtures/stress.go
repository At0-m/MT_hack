package main

import "fmt"

func fixtureColumns(stress bool) []string {
	columns := []string{"demand_base", "boost"}
	if stress {
		for i := 2; i < 256; i++ {
			columns = append(columns, fmt.Sprintf("f%d", i))
		}
	}
	return columns
}

func fixtureFeatures(stress bool) map[string]float64 {
	features := map[string]float64{"demand_base": 600, "boost": 10}
	if stress {
		for i := 2; i < 256; i++ {
			features[fmt.Sprintf("f%d", i)] = 0
		}
	}
	return features
}

func fixtureGolden(width int) [][]float32 {
	rows := [][]float32{make([]float32, width), make([]float32, width), make([]float32, width)}
	rows[0][0], rows[0][1] = 600, 10
	rows[2][0], rows[2][1] = -1, 2
	return rows
}
